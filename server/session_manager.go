package server

import (
	"strings"
	"sync"
	"time"

	"github.com/oliveagle/gotty/pkg/randomstring"
)

// Session represents an active terminal session
type Session struct {
	ID        string
	Title     string
	Subtitle  string // AI-generated short summary
	CreatedAt time.Time
	Slave     Slave
	// For persistent backends, store the session ID for re-attachment
	// Slave will be nil after first disconnect
}

// SessionManager manages terminal sessions
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	factory  Factory
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// SetFactory sets the factory for creating new slaves
func (sm *SessionManager) SetFactory(factory Factory) {
	sm.factory = factory
}

// RestoreSessions restores sessions from persistent storage (e.g., zellij)
// This should be called after SetFactory
func (sm *SessionManager) RestoreSessions() {
	if sm.factory == nil || !sm.factory.IsPersistent() {
		return
	}

	// For zellij backend, restore sessions from zellij list
	if sm.factory.Name() == "zellij" {
		sm.restoreFromZellij()
	}
}

// restoreFromZellij restores gotty sessions from existing zellij sessions
func (sm *SessionManager) restoreFromZellij() {
	// Import zellij session listing
	sessions := listZellijSessions()
	for _, name := range sessions {
		// Only restore sessions with gotty- prefix
		if strings.HasPrefix(name, "gotty-") {
			id := strings.TrimPrefix(name, "gotty-")
			if id == "" {
				continue
			}
			// Check if already exists
			if _, exists := sm.sessions[id]; !exists {
				sm.sessions[id] = &Session{
					ID:        id,
					Title:     name,
					CreatedAt: time.Now(), // We don't have exact creation time
					Slave:     nil,         // Will be created on demand
				}
			}
		}
	}
}

// listZellijSessions lists all zellij session names
// This is implemented in the zellij backend package
var listZellijSessions func() []string

// SetZellijSessionLister sets the function used to list zellij sessions
func SetZellijSessionLister(fn func() []string) {
	listZellijSessions = fn
}

// Create creates a new session with the given slave
func (sm *SessionManager) Create(title string, slave Slave) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := randomstring.Generate(8)
	session := &Session{
		ID:        id,
		Title:     title,
		CreatedAt: time.Now(),
		Slave:     slave,
	}
	sm.sessions[id] = session
	return session
}

// CreateWithID creates a new session with a specific ID using the factory
// This is useful for persistent backends like zellij that need session ID to match
func (sm *SessionManager) CreateWithID(title string, params map[string][]string) (*Session, error) {
	if sm.factory == nil {
		return nil, nil
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := randomstring.Generate(8)

	slave, err := sm.factory.NewWithID(id, params)
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        id,
		Title:     title,
		CreatedAt: time.Now(),
		Slave:     slave,
	}
	sm.sessions[id] = session
	return session, nil
}

// Get returns a session by ID
func (sm *SessionManager) Get(id string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[id]
	return session, ok
}

// List returns all sessions
func (sm *SessionManager) List() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// Close closes and removes a session by ID
func (sm *SessionManager) Close(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[id]
	if !ok {
		return nil // Session not found, consider it closed
	}

	delete(sm.sessions, id)
	return session.Slave.Close()
}

// Count returns the number of active sessions
func (sm *SessionManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return len(sm.sessions)
}

// Rename updates the title of a session
func (sm *SessionManager) Rename(id string, newTitle string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[id]
	if !ok {
		return false
	}
	session.Title = newTitle
	return true
}

// UpdateSubtitle updates the AI-generated subtitle of a session
func (sm *SessionManager) UpdateSubtitle(id string, subtitle string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[id]
	if !ok {
		return false
	}
	session.Subtitle = subtitle
	return true
}
