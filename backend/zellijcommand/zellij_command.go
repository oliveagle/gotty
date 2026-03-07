package zellijcommand

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/creack/pty"
	"github.com/pkg/errors"

	"github.com/oliveagle/gotty/server"
)

const (
	DefaultCloseSignal  = syscall.SIGINT
	DefaultCloseTimeout = 10 * time.Second
)

type ZellijCommand struct {
	sessionName string
	command     string
	argv        []string

	closeSignal  syscall.Signal
	closeTimeout time.Duration

	cmd       *exec.Cmd
	pty       *os.File
	ptyClosed chan struct{}
}

// sessionExists checks if a zellij session with the given name exists
func sessionExists(name string) bool {
	cmd := exec.Command("zellij", "list-sessions")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Parse session name from output like "session-name [Created ...]"
		line = strings.TrimSpace(line)
		// Remove ANSI color codes
		line = stripANSI(line)
		if idx := strings.Index(line, " "); idx > 0 {
			line = line[:idx]
		}
		if line == name {
			return true
		}
	}
	return false
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	var buf bytes.Buffer
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		buf.WriteByte(s[i])
	}
	return buf.String()
}

// New creates a new ZellijCommand, connecting to existing session or creating a new one
func New(sessionName string, command string, argv []string, options ...Option) (*ZellijCommand, error) {
	var cmd *exec.Cmd
	exists := sessionExists(sessionName)

	if exists {
		// Attach to existing session via PTY
		cmd = exec.Command("zellij", "attach", sessionName)
	} else {
		// Create new session in background first, then attach
		// This avoids the panic when zellij can't get terminal attributes
		createCmd := exec.Command("zellij", "attach", "-c", "-b", sessionName)
		if err := createCmd.Run(); err != nil {
			// Session might have been created by another process, continue anyway
		}
		// Now attach to the session via PTY
		cmd = exec.Command("zellij", "attach", sessionName)
	}

	pty, err := pty.Start(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start zellij session `%s`", sessionName)
	}
	ptyClosed := make(chan struct{})

	zcmd := &ZellijCommand{
		sessionName: sessionName,
		command:     command,
		argv:        argv,

		closeSignal:  DefaultCloseSignal,
		closeTimeout: DefaultCloseTimeout,

		cmd:       cmd,
		pty:       pty,
		ptyClosed: ptyClosed,
	}

	for _, option := range options {
		option(zcmd)
	}

	// When the process is closed by the user,
	// close pty so that Read() on the pty breaks with an EOF.
	go func() {
		defer func() {
			zcmd.pty.Close()
			close(zcmd.ptyClosed)
		}()

		zcmd.cmd.Wait()
	}()

	return zcmd, nil
}

func (zcmd *ZellijCommand) Read(p []byte) (n int, err error) {
	return zcmd.pty.Read(p)
}

func (zcmd *ZellijCommand) Write(p []byte) (n int, err error) {
	return zcmd.pty.Write(p)
}

func (zcmd *ZellijCommand) Close() error {
	// For zellij, we don't kill the session on close
	// The session persists and can be reattached
	// Just close the PTY connection
	if zcmd.pty != nil {
		zcmd.pty.Close()
	}
	return nil
}

// KillSession kills the zellij session (for cleanup)
func (zcmd *ZellijCommand) KillSession() error {
	cmd := exec.Command("zellij", "delete-session", zcmd.sessionName)
	return cmd.Run()
}

func (zcmd *ZellijCommand) WindowTitleVariables() map[string]interface{} {
	return map[string]interface{}{
		"session": zcmd.sessionName,
		"command": zcmd.command,
		"argv":    zcmd.argv,
	}
}

func (zcmd *ZellijCommand) ResizeTerminal(width int, height int) error {
	window := struct {
		row uint16
		col uint16
		x   uint16
		y   uint16
	}{
		uint16(height),
		uint16(width),
		0,
		0,
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		zcmd.pty.Fd(),
		syscall.TIOCSWINSZ,
		uintptr(unsafe.Pointer(&window)),
	)
	if errno != 0 {
		return errno
	} else {
		return nil
	}
}

func (zcmd *ZellijCommand) closeTimeoutC() <-chan time.Time {
	if zcmd.closeTimeout >= 0 {
		return time.After(zcmd.closeTimeout)
	}

	return make(chan time.Time)
}

// GetSessionName returns the zellij session name
func (zcmd *ZellijCommand) GetSessionName() string {
	return zcmd.sessionName
}

// SessionInfo returns information about the zellij session
type SessionInfo struct {
	Name      string
	CreatedAt time.Time
}

// ListZellijSessions lists all zellij sessions
func ListZellijSessions() ([]SessionInfo, error) {
	cmd := exec.Command("zellij", "list-sessions")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list zellij sessions: %w", err)
	}

	var sessions []SessionInfo
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove ANSI color codes
		line = stripANSI(line)
		// Parse: "session-name [Created Xm ago]"
		if idx := strings.Index(line, " "); idx > 0 {
			name := line[:idx]
			sessions = append(sessions, SessionInfo{
				Name: name,
			})
		}
	}
	return sessions, nil
}

// GetSessionNames returns a list of all zellij session names
func GetSessionNames() []string {
	sessions, err := ListZellijSessions()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(sessions))
	for _, s := range sessions {
		names = append(names, s.Name)
	}
	return names
}

func init() {
	// Register the zellij session lister for session restoration
	server.SetZellijSessionLister(GetSessionNames)
}
