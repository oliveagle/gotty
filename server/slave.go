package server

import (
	"github.com/oliveagle/gotty/webtty"
)

// Slave is webtty.Slave with some additional methods.
type Slave interface {
	webtty.Slave

	Close() error
}

// KillableSlave is a Slave that can be permanently killed (e.g., zellij session)
type KillableSlave interface {
	Slave
	KillSession() error
}

type Factory interface {
	Name() string
	New(params map[string][]string) (Slave, error)
	// NewWithID creates a slave with a specific session ID (for persistent backends like zellij)
	// If the backend doesn't support it, this should behave like New()
	NewWithID(sessionID string, params map[string][]string) (Slave, error)
	// IsPersistent returns true if the backend supports reconnection to existing sessions
	// For persistent backends (like zellij), sessions survive client disconnect
	IsPersistent() bool
}
