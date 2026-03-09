package server

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/oliveagle/gotty/pkg/randomstring"
	"github.com/oliveagle/gotty/summary"
)

// Session represents an active terminal session
type Session struct {
	ID           string
	Title        string
	Subtitle     string // AI-generated short summary
	WorkDir      string // Working directory
	ParentID     string // Parent session ID, empty for root sessions
	WorkspaceID  string // Workspace ID, empty for default workspace
	IsFolder     bool   // True if this is a folder (container) not a real session
	Order        int    // Sort order within parent
	ActiveTab    string // Current active zellij tab name
	CreatedAt    time.Time
	LastActiveAt time.Time // Last activity time
	Slave        Slave
	// For persistent backends, store the session ID for re-attachment
	// Slave will be nil after first disconnect

	// BackendSessionName stores the backend session name (e.g., zellij session name)
	// This is needed to kill the session even after Slave is nil
	BackendSessionName string

	// Output buffer for subtitle generation
	outputBuffer *summary.RingBuffer
	outputMu     sync.RWMutex

	// Track last output hash for change detection
	lastOutputHash string
}

// sessionMetadata is used for JSON serialization of session data
type sessionMetadata struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	ParentID    string `json:"parent_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	IsFolder    bool   `json:"is_folder"`
	Order       int    `json:"order"`
	WorkDir     string `json:"workdir,omitempty"`
	ActiveTab   string `json:"active_tab,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// SessionManager manages terminal sessions
type SessionManager struct {
	sessions     map[string]*Session
	mu           sync.RWMutex
	factory      Factory
	metadataFile string // Path to metadata file for persistence
	nextOrder    int    // Next order number for new sessions
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	// Default metadata file location
	homeDir, _ := os.UserHomeDir()
	metadataFile := filepath.Join(homeDir, ".config", "gotty", "sessions.json")

	// Ensure directory exists with secure permissions (0700 = owner only)
	dir := filepath.Dir(metadataFile)
	if err := os.MkdirAll(dir, 0700); err == nil {
		// Ensure file exists with empty array if not present
		// Use 0600 for secure file permissions (owner read/write only)
		if _, err := os.Stat(metadataFile); os.IsNotExist(err) {
			os.WriteFile(metadataFile, []byte("[]"), 0600)
		}
	}

	return &SessionManager{
		sessions:     make(map[string]*Session),
		metadataFile: metadataFile,
		nextOrder:    1,
	}
}

// SetMetadataFile sets a custom metadata file path
func (sm *SessionManager) SetMetadataFile(path string) {
	sm.metadataFile = path
}

// saveMetadata saves session metadata to file
func (sm *SessionManager) saveMetadata() {
	if sm.metadataFile == "" {
		return
	}

	// Create directory if needed with secure permissions
	dir := filepath.Dir(sm.metadataFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return
	}

	// Collect metadata
	metadata := make([]sessionMetadata, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		metadata = append(metadata, sessionMetadata{
			ID:          s.ID,
			Title:       s.Title,
			ParentID:    s.ParentID,
			WorkspaceID: s.WorkspaceID,
			IsFolder:    s.IsFolder,
			Order:       s.Order,
			WorkDir:     s.WorkDir,
			ActiveTab:   s.ActiveTab,
			CreatedAt:   s.CreatedAt.Format(time.RFC3339),
		})
	}

	// Write to file with secure permissions (0600 = owner read/write only)
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(sm.metadataFile, data, 0600)
}

// loadMetadata loads session metadata from file
func (sm *SessionManager) loadMetadata() map[string]sessionMetadata {
	if sm.metadataFile == "" {
		return nil
	}

	data, err := os.ReadFile(sm.metadataFile)
	if err != nil {
		return nil
	}

	var metadata []sessionMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil
	}

	// Convert to map for easy lookup
	result := make(map[string]sessionMetadata)
	for _, m := range metadata {
		result[m.ID] = m
	}
	return result
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
	// Load saved metadata
	metadata := sm.loadMetadata()

	// First, restore folders (they don't have zellij sessions)
	for id, m := range metadata {
		if m.IsFolder {
			if _, exists := sm.sessions[id]; !exists {
				session := &Session{
					ID:           id,
					Title:        m.Title,
					ParentID:     m.ParentID,
					WorkspaceID:  m.WorkspaceID,
					IsFolder:     true,
					Order:        m.Order,
					CreatedAt:    time.Now(),
					Slave:        nil,
					outputBuffer: nil, // Folders don't need output buffer
				}
				if t, err := time.Parse(time.RFC3339, m.CreatedAt); err == nil {
					session.CreatedAt = t
				}
				sm.sessions[id] = session
			}
		}
	}

	// Then, restore zellij sessions
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
				session := &Session{
					ID:                 id,
					Title:              name,
					CreatedAt:          time.Now(),
					Slave:              nil,
					BackendSessionName: name, // Store the full zellij session name
					outputBuffer:       summary.NewRingBuffer(16384),
				}

				// Apply saved metadata if exists
				if m, ok := metadata[id]; ok {
					session.Title = m.Title
					session.ParentID = m.ParentID
					session.WorkspaceID = m.WorkspaceID
					session.Order = m.Order
					session.WorkDir = m.WorkDir
					session.ActiveTab = m.ActiveTab
					if t, err := time.Parse(time.RFC3339, m.CreatedAt); err == nil {
						session.CreatedAt = t
					}
				}

				sm.sessions[id] = session
			}
		}
	}

	// Update nextOrder to be greater than any existing order
	for _, s := range sm.sessions {
		if s.Order >= sm.nextOrder {
			sm.nextOrder = s.Order + 1
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
	order := sm.nextOrder
	sm.nextOrder++
	now := time.Now()

	session := &Session{
		ID:           id,
		Title:        title,
		Order:        order,
		CreatedAt:    now,
		LastActiveAt: now,
		Slave:        slave,
		outputBuffer: summary.NewRingBuffer(16384), // 16KB buffer
	}
	sm.sessions[id] = session
	sm.saveMetadata()
	return session
}

// CreateChild creates a new child session under the given parent
func (sm *SessionManager) CreateChild(title string, parentID string, slave Slave) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := randomstring.Generate(8)
	order := sm.nextOrder
	sm.nextOrder++
	now := time.Now()

	session := &Session{
		ID:           id,
		Title:        title,
		ParentID:     parentID,
		Order:        order,
		CreatedAt:    now,
		LastActiveAt: now,
		Slave:        slave,
		outputBuffer: summary.NewRingBuffer(16384), // 16KB buffer
	}
	sm.sessions[id] = session
	sm.saveMetadata()
	return session
}

// CreateFolder creates a new folder (container for sessions)
func (sm *SessionManager) CreateFolder(title string, parentID string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := randomstring.Generate(8)
	order := sm.nextOrder
	sm.nextOrder++

	session := &Session{
		ID:           id,
		Title:        title,
		ParentID:     parentID,
		IsFolder:     true,
		Order:        order,
		CreatedAt:    time.Now(),
		Slave:        nil,
		outputBuffer: nil,
	}
	sm.sessions[id] = session
	sm.saveMetadata()
	return session
}

// MoveToFolder moves a session to a folder (or root if folderID is empty)
func (sm *SessionManager) MoveToFolder(sessionID string, folderID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return false
	}

	// Validate folder exists if folderID is provided
	if folderID != "" {
		folder, ok := sm.sessions[folderID]
		if !ok || !folder.IsFolder {
			return false
		}
		// Prevent moving a folder into itself or its descendants
		if session.IsFolder && sm.isDescendant(folderID, sessionID) {
			return false
		}
	}

	session.ParentID = folderID
	sm.saveMetadata()
	return true
}

// Reorder moves a session to a new position within a parent (folder or root)
// parentID specifies the target parent (empty string for root)
// afterID specifies which session to insert after (empty string means insert at beginning)
func (sm *SessionManager) Reorder(sessionID string, parentID string, afterID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return false
	}

	// Validate parentID (must be empty or a valid folder)
	if parentID != "" {
		parent, ok := sm.sessions[parentID]
		if !ok || !parent.IsFolder {
			return false
		}
		// Prevent moving a folder into itself or its descendants
		if session.IsFolder && sm.isDescendant(parentID, sessionID) {
			return false
		}
	}

	// Validate afterID if provided (must exist and be in the same parent)
	if afterID != "" {
		afterSession, ok := sm.sessions[afterID]
		if !ok {
			return false
		}
		// afterID must be in the same parent (or root)
		if afterSession.ParentID != parentID {
			return false
		}
	}

	// Get all siblings in the target parent (excluding the session being moved)
	var siblings []*Session
	for _, s := range sm.sessions {
		if s.ParentID == parentID && s.ID != sessionID {
			siblings = append(siblings, s)
		}
	}

	// Sort siblings by order
	sort.Slice(siblings, func(i, j int) bool {
		return siblings[i].Order < siblings[j].Order
	})

	// Find the position to insert
	insertPos := 0
	if afterID != "" {
		for i, s := range siblings {
			if s.ID == afterID {
				insertPos = i + 1
				break
			}
		}
	}

	// Insert the session at the correct position
	// Create a new slice with the session inserted
	newOrder := make([]*Session, 0, len(siblings)+1)
	newOrder = append(newOrder, siblings[:insertPos]...)
	newOrder = append(newOrder, session)
	newOrder = append(newOrder, siblings[insertPos:]...)

	// Reassign order values
	for i, s := range newOrder {
		s.Order = i + 1
	}

	// Update the session's parent
	session.ParentID = parentID

	// Update nextOrder if needed
	if len(newOrder) >= sm.nextOrder {
		sm.nextOrder = len(newOrder) + 1
	}

	sm.saveMetadata()
	return true
}

// isDescendant checks if target is a descendant of ancestor
func (sm *SessionManager) isDescendant(targetID string, ancestorID string) bool {
	current := sm.sessions[targetID]
	for current != nil {
		if current.ParentID == ancestorID {
			return true
		}
		if current.ParentID == "" {
			return false
		}
		current = sm.sessions[current.ParentID]
	}
	return false
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
	order := sm.nextOrder
	sm.nextOrder++
	now := time.Now()

	slave, err := sm.factory.NewWithID(id, params)
	if err != nil {
		return nil, err
	}

	// Determine backend session name for persistent backends (e.g., zellij)
	// For zellij, the session name is "gotty-{id}"
	backendSessionName := ""
	if sm.factory.IsPersistent() {
		backendSessionName = "gotty-" + id
	}

	session := &Session{
		ID:                 id,
		Title:              title,
		Order:              order,
		CreatedAt:          now,
		LastActiveAt:       now,
		Slave:              slave,
		BackendSessionName: backendSessionName,
		outputBuffer:       summary.NewRingBuffer(16384), // 16KB buffer
	}
	sm.sessions[id] = session
	sm.saveMetadata()
	return session, nil
}

// Get returns a session by ID
func (sm *SessionManager) Get(id string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[id]
	return session, ok
}

// List returns all sessions sorted by order, then by creation time for stable sorting
func (sm *SessionManager) List() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	// Sort by order, then by creation time for stable sorting
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].Order != sessions[j].Order {
			return sessions[i].Order < sessions[j].Order
		}
		return sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
	})
	return sessions
}

// GetRootSessions returns all root sessions (sessions without a parent) sorted by order, then by creation time
func (sm *SessionManager) GetRootSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0)
	for _, s := range sm.sessions {
		if s.ParentID == "" {
			sessions = append(sessions, s)
		}
	}
	// Sort by order, then by creation time for stable sorting
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].Order != sessions[j].Order {
			return sessions[i].Order < sessions[j].Order
		}
		return sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
	})
	return sessions
}

// GetChildren returns all child sessions of a given parent sorted by order, then by creation time
func (sm *SessionManager) GetChildren(parentID string) []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0)
	for _, s := range sm.sessions {
		if s.ParentID == parentID {
			sessions = append(sessions, s)
		}
	}
	// Sort by order, then by creation time for stable sorting
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].Order != sessions[j].Order {
			return sessions[i].Order < sessions[j].Order
		}
		return sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
	})
	return sessions
}

// HasChildren checks if a session has any children
func (sm *SessionManager) HasChildren(id string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, s := range sm.sessions {
		if s.ParentID == id {
			return true
		}
	}
	return false
}

// Close closes and removes a session by ID
func (sm *SessionManager) Close(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[id]
	if !ok {
		return nil // Session not found, consider it closed
	}

	// Remove children first (cascade delete)
	for _, child := range sm.sessions {
		if child.ParentID == id {
			delete(sm.sessions, child.ID)
		}
	}

	delete(sm.sessions, id)
	sm.saveMetadata()
	return session.Slave.Close()
}

// Kill permanently kills a session including the backend (e.g., zellij session)
func (sm *SessionManager) Kill(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[id]
	if !ok {
		return nil // Session not found
	}

	// Remove children first (cascade delete)
	for _, child := range sm.sessions {
		if child.ParentID == id {
			delete(sm.sessions, child.ID)
		}
	}

	delete(sm.sessions, id)
	sm.saveMetadata()

	var err error

	// Try to kill the backend session
	if session.Slave != nil {
		if killable, ok := session.Slave.(KillableSlave); ok {
			err = killable.KillSession()
		} else {
			// Fallback to regular close
			err = session.Slave.Close()
		}
	} else if session.BackendSessionName != "" {
		// If Slave is nil but we have the backend session name, kill directly
		err = killZellijSession(session.BackendSessionName)
	}

	return err
}

// killZellijSession kills a zellij session by name
func killZellijSession(name string) error {
	// Use --force to delete active sessions
	cmd := exec.Command("zellij", "delete-session", "--force", name)
	return cmd.Run()
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
	sm.saveMetadata()
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

// UpdateWorkDir updates the working directory of a session
func (sm *SessionManager) UpdateWorkDir(id string, workDir string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[id]
	if !ok {
		return false
	}
	session.WorkDir = workDir
	sm.saveMetadata()
	return true
}

// UpdateActivity updates the last active time of a session
func (sm *SessionManager) UpdateActivity(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[id]
	if !ok {
		return
	}
	session.LastActiveAt = time.Now()
}

// GetActivity returns the activity status of a session
// Returns: is_active (true if active within threshold), seconds since last activity
func (sm *SessionManager) GetActivity(id string) (bool, int64) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[id]
	if !ok {
		return false, 0
	}

	// Consider active if activity within last 30 seconds
	threshold := int64(30)
	secondsSinceActivity := int64(time.Since(session.LastActiveAt).Seconds())
	isActive := secondsSinceActivity < threshold

	return isActive, secondsSinceActivity
}

// CaptureOutput captures output to a session's buffer for subtitle generation
func (sm *SessionManager) CaptureOutput(id string, data []byte) {
	sm.mu.Lock()
	session, ok := sm.sessions[id]
	if !ok || session.outputBuffer == nil {
		sm.mu.Unlock()
		return
	}

	// Update activity time
	session.LastActiveAt = time.Now()

	session.outputMu.Lock()
	session.outputBuffer.Write(data)
	session.outputMu.Unlock()
	sm.mu.Unlock()
}

// GetOutputBuffer returns a copy of the session's output buffer
func (sm *SessionManager) GetOutputBuffer(id string) []byte {
	sm.mu.RLock()
	session, ok := sm.sessions[id]
	sm.mu.RUnlock()

	if !ok || session.outputBuffer == nil {
		return nil
	}

	session.outputMu.RLock()
	defer session.outputMu.RUnlock()
	return session.outputBuffer.Bytes()
}

// HasOutputChanged checks if the output has changed since last check
// Returns true if changed, and updates the internal hash
func (sm *SessionManager) HasOutputChanged(id string) bool {
	sm.mu.RLock()
	session, ok := sm.sessions[id]
	sm.mu.RUnlock()

	if !ok || session.outputBuffer == nil {
		return false
	}

	session.outputMu.Lock()
	defer session.outputMu.Unlock()

	output := session.outputBuffer.Bytes()
	if len(output) == 0 {
		return false
	}

	// Simple hash: use length as a quick check
	newHash := ""
	if len(output) > 0 {
		// Use last 100 bytes as a "fingerprint"
		fingerprint := output
		if len(fingerprint) > 100 {
			fingerprint = fingerprint[len(fingerprint)-100:]
		}
		newHash = string(fingerprint)
	}

	if session.lastOutputHash == newHash {
		return false
	}

	session.lastOutputHash = newHash
	return true
}

// ListByWorkspace returns all sessions in a specific workspace sorted by order
func (sm *SessionManager) ListByWorkspace(workspaceID string) []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0)
	for _, s := range sm.sessions {
		// Include sessions with matching workspaceID or empty workspaceID (legacy) for default workspace
		if s.WorkspaceID == workspaceID || (workspaceID == DefaultWorkspaceID && s.WorkspaceID == "") {
			sessions = append(sessions, s)
		}
	}
	// Sort by order, then by creation time for stable sorting
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].Order != sessions[j].Order {
			return sessions[i].Order < sessions[j].Order
		}
		return sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
	})
	return sessions
}

// MoveToWorkspace moves a session to a different workspace
// If the session is a folder, also moves all its children
func (sm *SessionManager) MoveToWorkspace(sessionID string, workspaceID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return false
	}

	// Validate workspaceID (empty string means default workspace)
	if workspaceID == "" {
		workspaceID = DefaultWorkspaceID
	}

	session.WorkspaceID = workspaceID

	// If this is a folder, also move all children (sessions with this folder as parent)
	if session.IsFolder {
		for _, s := range sm.sessions {
			if s.ParentID == sessionID {
				s.WorkspaceID = workspaceID
			}
		}
	}

	sm.saveMetadata()
	return true
}

// MigrateToWorkspace migrates all sessions without workspace to the specified workspace
func (sm *SessionManager) MigrateToWorkspace(workspaceID string) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	count := 0
	for _, s := range sm.sessions {
		if s.WorkspaceID == "" {
			s.WorkspaceID = workspaceID
			count++
		}
	}
	if count > 0 {
		sm.saveMetadata()
	}
	return count
}

// SetWorkspaceID sets the workspace ID for a new session
func (sm *SessionManager) SetWorkspaceID(sessionID string, workspaceID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return false
	}

	if workspaceID == "" {
		workspaceID = DefaultWorkspaceID
	}
	session.WorkspaceID = workspaceID
	sm.saveMetadata()
	return true
}

// UpdateActiveTab updates the active zellij tab for a session
func (sm *SessionManager) UpdateActiveTab(sessionID string, tabName string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return false
	}

	session.ActiveTab = tabName
	sm.saveMetadata()
	return true
}

// GetZellijActiveTab returns the active tab name from zellij dump-layout output
func GetZellijActiveTab() string {
	cmd := exec.Command("zellij", "action", "dump-layout")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse the layout to find the tab with focus=true
	// Format: tab name="xxx" focus=true
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Look for tab line with focus=true
		if strings.Contains(line, "tab") && strings.Contains(line, "focus=true") {
			// Extract name="xxx"
			nameStart := strings.Index(line, `name="`)
			if nameStart == -1 {
				continue
			}
			nameStart += 6 // len of `name="`
			nameEnd := strings.Index(line[nameStart:], `"`)
			if nameEnd == -1 {
				continue
			}
			return line[nameStart : nameStart+nameEnd]
		}
	}
	return ""
}
