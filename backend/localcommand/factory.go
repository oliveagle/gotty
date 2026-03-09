package localcommand

import (
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/oliveagle/gotty/server"
)

type Options struct {
	CloseSignal  int `hcl:"close_signal" flagName:"close-signal" flagSName:"" flagDescribe:"Signal sent to the command process when gotty close it (default: SIGHUP)" default:"1"`
	CloseTimeout int `hcl:"close_timeout" flagName:"close-timeout" flagSName:"" flagDescribe:"Time in seconds to force kill process after client is disconnected (default: -1)" default:"-1"`
}

type Factory struct {
	command string
	argv    []string
	options *Options
	opts    []Option
}

func NewFactory(command string, argv []string, options *Options) (*Factory, error) {
	opts := []Option{WithCloseSignal(syscall.Signal(options.CloseSignal))}
	if options.CloseTimeout >= 0 {
		opts = append(opts, WithCloseTimeout(time.Duration(options.CloseTimeout)*time.Second))
	}

	return &Factory{
		command: command,
		argv:    argv,
		options: options,
		opts:    opts,
	}, nil
}

func (factory *Factory) Name() string {
	return "local command"
}

// validateArg validates a single argument for dangerous patterns
// Returns an error if the argument contains potentially dangerous content
func validateArg(arg string) error {
	// Block arguments that start with dangerous prefixes
	dangerousPrefixes := []string{
		"-e", "--execute", "-c", "--command",
		"--", // Option terminator that could allow injection
	}

	for _, prefix := range dangerousPrefixes {
		if arg == prefix || strings.HasPrefix(arg, prefix+"=") {
			return fmt.Errorf("argument not allowed: %s (potential command injection)", arg)
		}
	}

	// Block arguments containing shell metacharacters that could be dangerous
	// This is a conservative approach - legitimate arguments might be blocked
	dangerousChars := []string{
		";", "|", "&", "$", "`", "(", ")",
		"<", ">", "\n", "\r",
	}

	for _, char := range dangerousChars {
		if strings.Contains(arg, char) {
			return fmt.Errorf("argument contains forbidden character: %q", char)
		}
	}

	return nil
}

// sanitizeArgs validates and returns safe arguments
// Returns error if any argument fails validation
func sanitizeArgs(args []string) ([]string, error) {
	safe := make([]string, 0, len(args))
	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return nil, err
		}
		safe = append(safe, arg)
	}
	return safe, nil
}

func (factory *Factory) New(params map[string][]string) (server.Slave, error) {
	argv := make([]string, len(factory.argv))
	copy(argv, factory.argv)

	if params["arg"] != nil && len(params["arg"]) > 0 {
		// Validate and sanitize user-provided arguments
		safeArgs, err := sanitizeArgs(params["arg"])
		if err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		argv = append(argv, safeArgs...)
	}

	return New(factory.command, argv, factory.opts...)
}

// NewWithID implements Factory interface - for localcommand, it's the same as New
func (factory *Factory) NewWithID(sessionID string, params map[string][]string) (server.Slave, error) {
	return factory.New(params)
}

// IsPersistent returns false - local command sessions don't persist
func (factory *Factory) IsPersistent() bool {
	return false
}
