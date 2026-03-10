package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// UserSettings represents user-specific settings
type UserSettings struct {
	HostName       string `json:"host_name,omitempty"`
	CityCode       string `json:"city_code,omitempty"`
	AutoHideSidebar bool  `json:"auto_hide_sidebar,omitempty"`
	WeatherBg      bool   `json:"weather_bg,omitempty"`
	BannerPosition string `json:"banner_position,omitempty"`
	ShowIRC        bool   `json:"show_irc,omitempty"`
	IRCNick        string `json:"irc_nick,omitempty"`
	FontSize       string `json:"font_size,omitempty"`
	Theme          string `json:"theme,omitempty"`
	Bell           bool   `json:"bell,omitempty"`
}

// UserSettingsManager manages user settings
type UserSettingsManager struct {
	filePath string
	mu       sync.RWMutex
	settings UserSettings
}

var userSettingsManager *UserSettingsManager

// InitUserSettings initializes the user settings manager
func InitUserSettings() *UserSettingsManager {
	homeDir, _ := os.UserHomeDir()
	filePath := filepath.Join(homeDir, ".config", "gotty", "user_settings.json")

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	os.MkdirAll(dir, 0700)

	userSettingsManager = &UserSettingsManager{
		filePath: filePath,
	}
	userSettingsManager.load()
	return userSettingsManager
}

// GetUserSettingsManager returns the global user settings manager
func GetUserSettingsManager() *UserSettingsManager {
	return userSettingsManager
}

// load reads settings from file
func (m *UserSettingsManager) load() {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		// File doesn't exist, use defaults
		m.settings = UserSettings{
			ShowIRC:        true,
			IRCNick:        "user",
			FontSize:       "14",
			Theme:          "default",
			BannerPosition: "bottom",
			Bell:           true,
		}
		return
	}

	json.Unmarshal(data, &m.settings)
}

// save writes settings to file
func (m *UserSettingsManager) save() error {
	data, err := json.MarshalIndent(m.settings, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(m.filePath, data, 0600)
}

// Get returns the current settings
func (m *UserSettingsManager) Get() UserSettings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.settings
}

// Update updates the settings
func (m *UserSettingsManager) Update(settings UserSettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.settings = settings
	return m.save()
}

// UpdatePartial updates specific fields
func (m *UserSettingsManager) UpdatePartial(updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Apply updates
	if v, ok := updates["host_name"].(string); ok {
		m.settings.HostName = v
	}
	if v, ok := updates["city_code"].(string); ok {
		m.settings.CityCode = v
	}
	if v, ok := updates["auto_hide_sidebar"].(bool); ok {
		m.settings.AutoHideSidebar = v
	}
	if v, ok := updates["weather_bg"].(bool); ok {
		m.settings.WeatherBg = v
	}
	if v, ok := updates["banner_position"].(string); ok {
		m.settings.BannerPosition = v
	}
	if v, ok := updates["show_irc"].(bool); ok {
		m.settings.ShowIRC = v
	}
	if v, ok := updates["irc_nick"].(string); ok {
		m.settings.IRCNick = v
	}
	if v, ok := updates["font_size"].(string); ok {
		m.settings.FontSize = v
	}
	if v, ok := updates["theme"].(string); ok {
		m.settings.Theme = v
	}
	if v, ok := updates["bell"].(bool); ok {
		m.settings.Bell = v
	}

	return m.save()
}

// handleUserSettings handles GET and POST for user settings
func (server *Server) handleUserSettings(w http.ResponseWriter, r *http.Request) {
	manager := GetUserSettingsManager()
	if manager == nil {
		http.Error(w, "Settings not initialized", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		settings := manager.Get()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)

	case http.MethodPost:
		var settings UserSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if err := manager.Update(settings); err != nil {
			http.Error(w, "Failed to save settings", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
