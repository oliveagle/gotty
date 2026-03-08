package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/oliveagle/gotty/pkg/randomstring"
)

// WorkspaceManager manages workspaces
type WorkspaceManager struct {
	workspaces   map[string]*Workspace
	mu           sync.RWMutex
	metadataFile string // Path to metadata file for persistence
	activeID     string // Currently active workspace ID
	nextOrder    int    // Next order number for new workspaces
}

// NewWorkspaceManager creates a new workspace manager
func NewWorkspaceManager() *WorkspaceManager {
	// Default metadata file location
	homeDir, _ := os.UserHomeDir()
	metadataFile := filepath.Join(homeDir, ".config", "gotty", "workspaces.json")

	wm := &WorkspaceManager{
		workspaces:   make(map[string]*Workspace),
		metadataFile: metadataFile,
		nextOrder:    1,
	}

	// Load existing workspaces or create default
	wm.loadOrCreateDefault()

	return wm
}

// SetMetadataFile sets a custom metadata file path
func (wm *WorkspaceManager) SetMetadataFile(path string) {
	wm.metadataFile = path
}

// saveMetadata saves workspace metadata to file
func (wm *WorkspaceManager) saveMetadata() {
	if wm.metadataFile == "" {
		return
	}

	// Create directory if needed
	dir := filepath.Dir(wm.metadataFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	// Prepare data for saving
	type workspaceData struct {
		Workspaces []*Workspace `json:"workspaces"`
		ActiveID   string       `json:"active_id"`
	}

	data := workspaceData{
		Workspaces: make([]*Workspace, 0, len(wm.workspaces)),
		ActiveID:   wm.activeID,
	}

	for _, w := range wm.workspaces {
		data.Workspaces = append(data.Workspaces, w)
	}

	// Sort by order before saving
	sort.Slice(data.Workspaces, func(i, j int) bool {
		return data.Workspaces[i].Order < data.Workspaces[j].Order
	})

	// Write to file
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(wm.metadataFile, jsonData, 0644)
}

// loadOrCreateDefault loads workspaces from file or creates default
func (wm *WorkspaceManager) loadOrCreateDefault() {
	if wm.metadataFile == "" {
		wm.createDefaultWorkspace()
		return
	}

	data, err := os.ReadFile(wm.metadataFile)
	if err != nil {
		wm.createDefaultWorkspace()
		return
	}

	type workspaceData struct {
		Workspaces []*Workspace `json:"workspaces"`
		ActiveID   string       `json:"active_id"`
	}

	var loaded workspaceData
	if err := json.Unmarshal(data, &loaded); err != nil {
		wm.createDefaultWorkspace()
		return
	}

	// Load workspaces
	for _, w := range loaded.Workspaces {
		wm.workspaces[w.ID] = w
		if w.Order >= wm.nextOrder {
			wm.nextOrder = w.Order + 1
		}
	}

	// Set active workspace
	if loaded.ActiveID != "" {
		if _, exists := wm.workspaces[loaded.ActiveID]; exists {
			wm.activeID = loaded.ActiveID
		}
	}

	// Ensure default workspace exists
	if _, exists := wm.workspaces[DefaultWorkspaceID]; !exists {
		wm.createDefaultWorkspace()
	}

	// If no active workspace, set to default
	if wm.activeID == "" {
		wm.activeID = DefaultWorkspaceID
	}
}

// createDefaultWorkspace creates the default workspace
func (wm *WorkspaceManager) createDefaultWorkspace() {
	now := time.Now()
	defaultWorkspace := &Workspace{
		ID:         DefaultWorkspaceID,
		Name:       "Default",
		ColorTheme: DefaultColorTheme,
		Icon:       DefaultWorkspaceIcon(DefaultColorTheme),
		Order:      0,
		CreatedAt:  now,
	}
	wm.workspaces[DefaultWorkspaceID] = defaultWorkspace
	wm.activeID = DefaultWorkspaceID
	wm.saveMetadata()
}

// Create creates a new workspace
func (wm *WorkspaceManager) Create(name string, colorTheme string, icon string) *Workspace {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	id := randomstring.Generate(8)
	order := wm.nextOrder
	wm.nextOrder++

	if colorTheme == "" {
		colorTheme = DefaultColorTheme
	}
	if icon == "" {
		icon = DefaultWorkspaceIcon(colorTheme)
	}

	workspace := &Workspace{
		ID:         id,
		Name:       name,
		ColorTheme: colorTheme,
		Icon:       icon,
		Order:      order,
		CreatedAt:  time.Now(),
	}
	wm.workspaces[id] = workspace
	wm.saveMetadata()
	return workspace
}

// Get returns a workspace by ID
func (wm *WorkspaceManager) Get(id string) (*Workspace, bool) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	workspace, ok := wm.workspaces[id]
	return workspace, ok
}

// List returns all workspaces sorted by order
func (wm *WorkspaceManager) List() []*Workspace {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	workspaces := make([]*Workspace, 0, len(wm.workspaces))
	for _, w := range wm.workspaces {
		workspaces = append(workspaces, w)
	}
	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].Order < workspaces[j].Order
	})
	return workspaces
}

// Update updates a workspace
func (wm *WorkspaceManager) Update(id string, name string, colorTheme string, icon string) bool {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	workspace, ok := wm.workspaces[id]
	if !ok {
		return false
	}

	// Cannot modify default workspace's ID
	if id == DefaultWorkspaceID && name != "" {
		// Allow renaming default workspace
	}

	if name != "" {
		workspace.Name = name
	}
	if colorTheme != "" {
		workspace.ColorTheme = colorTheme
	}
	if icon != "" {
		workspace.Icon = icon
	}

	wm.saveMetadata()
	return true
}

// Delete deletes a workspace (sessions will be moved to default)
// Returns the deleted workspace ID for session migration
func (wm *WorkspaceManager) Delete(id string) (string, bool) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Cannot delete default workspace
	if id == DefaultWorkspaceID {
		return "", false
	}

	_, ok := wm.workspaces[id]
	if !ok {
		return "", false
	}

	// If deleting active workspace, switch to default
	if wm.activeID == id {
		wm.activeID = DefaultWorkspaceID
	}

	delete(wm.workspaces, id)
	wm.saveMetadata()
	return id, true
}

// GetActive returns the active workspace ID
func (wm *WorkspaceManager) GetActive() string {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.activeID
}

// SetActive sets the active workspace
func (wm *WorkspaceManager) SetActive(id string) bool {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, ok := wm.workspaces[id]; !ok {
		return false
	}

	wm.activeID = id
	wm.saveMetadata()
	return true
}

// Count returns the number of workspaces
func (wm *WorkspaceManager) Count() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return len(wm.workspaces)
}
