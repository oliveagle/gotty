package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/oliveagle/gotty/pkg/homedir"
)

// WebAuthnUser represents a user for WebAuthn authentication
type WebAuthnUser struct {
	ID          []byte
	Name        string
	DisplayName string
	Credentials []webauthn.Credential
}

// WebAuthnID returns the user ID
func (u *WebAuthnUser) WebAuthnID() []byte {
	return u.ID
}

// WebAuthnName returns the username
func (u *WebAuthnUser) WebAuthnName() string {
	return u.Name
}

// WebAuthnDisplayName returns the display name
func (u *WebAuthnUser) WebAuthnDisplayName() string {
	return u.DisplayName
}

// WebAuthnIcon returns the user icon (not used)
func (u *WebAuthnUser) WebAuthnIcon() string {
	return ""
}

// WebAuthnCredentials returns the user's credentials
func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

// AddCredential adds a new credential to the user
func (u *WebAuthnUser) AddCredential(cred webauthn.Credential) {
	u.Credentials = append(u.Credentials, cred)
}

// WebAuthnManager manages WebAuthn authentication
type WebAuthnManager struct {
	webauthn *webauthn.WebAuthn
	user     *WebAuthnUser
	mu       sync.RWMutex
	dataFile string
}

// NewWebAuthnManager creates a new WebAuthn manager
func NewWebAuthnManager(displayName, hostname, dataDir string) (*WebAuthnManager, error) {
	// Expand data directory
	dataDir = homedir.Expand(dataDir)

	// Create data directory if not exists
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create WebAuthn data directory: %w", err)
	}

	// Parse hostname for WebAuthn RP ID
	rpID := hostname
	if rpID == "" {
		rpID = "localhost"
	}

	// Build origin
	origin := "https://" + rpID
	if rpID == "localhost" {
		origin = "http://localhost"
	}

	wconfig := &webauthn.Config{
		RPDisplayName: displayName,
		RPID:          rpID,
		RPOrigins:     []string{origin},
	}

	wn, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebAuthn instance: %w", err)
	}

	dataFile := filepath.Join(dataDir, "webauthn_user.json")

	mgr := &WebAuthnManager{
		webauthn: wn,
		user: &WebAuthnUser{
			ID:          generateUserID(),
			Name:        "gotty",
			DisplayName: "GoTTY User",
		},
		dataFile: dataFile,
	}

	// Load existing user data
	if err := mgr.loadUser(); err != nil {
		log.Printf("[WebAuthn] No existing user data, starting fresh: %v", err)
	}

	return mgr, nil
}

// generateUserID generates a random user ID
func generateUserID() []byte {
	id := make([]byte, 32)
	rand.Read(id)
	return id
}

// loadUser loads user data from file
func (m *WebAuthnManager) loadUser() error {
	data, err := os.ReadFile(m.dataFile)
	if err != nil {
		return err
	}

	var userData struct {
		ID          []byte                  `json:"id"`
		Name        string                  `json:"name"`
		DisplayName string                  `json:"display_name"`
		Credentials []webauthn.Credential   `json:"credentials"`
	}

	if err := json.Unmarshal(data, &userData); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.user.ID = userData.ID
	m.user.Name = userData.Name
	m.user.DisplayName = userData.DisplayName
	m.user.Credentials = userData.Credentials

	log.Printf("[WebAuthn] Loaded user with %d credentials", len(m.user.Credentials))
	return nil
}

// saveUser saves user data to file
func (m *WebAuthnManager) saveUser() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userData := struct {
		ID          []byte                `json:"id"`
		Name        string                `json:"name"`
		DisplayName string                `json:"display_name"`
		Credentials []webauthn.Credential `json:"credentials"`
	}{
		ID:          m.user.ID,
		Name:        m.user.Name,
		DisplayName: m.user.DisplayName,
		Credentials: m.user.Credentials,
	}

	data, err := json.MarshalIndent(userData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.dataFile, data, 0600)
}

// GetUser returns the WebAuthn user
func (m *WebAuthnManager) GetUser() *WebAuthnUser {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.user
}

// HasCredentials returns true if user has registered credentials
func (m *WebAuthnManager) HasCredentials() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.user.Credentials) > 0
}

// BeginRegistration starts the WebAuthn registration process
func (m *WebAuthnManager) BeginRegistration() (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	m.mu.RLock()
	user := m.user
	m.mu.RUnlock()

	return m.webauthn.BeginRegistration(user)
}

// FinishRegistration completes the WebAuthn registration process
func (m *WebAuthnManager) FinishRegistration(parsedResponse *protocol.ParsedCredentialCreationData, sessionData *webauthn.SessionData) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cred, err := m.webauthn.CreateCredential(m.user, *sessionData, parsedResponse)
	if err != nil {
		return err
	}

	m.user.Credentials = append(m.user.Credentials, *cred)
	log.Printf("[WebAuthn] Registered new credential: %s", base64.RawURLEncoding.EncodeToString(cred.ID))

	// Save to file
	if err := m.saveUser(); err != nil {
		log.Printf("[WebAuthn] Failed to save user data: %v", err)
	}

	return nil
}

// BeginLogin starts the WebAuthn login process
func (m *WebAuthnManager) BeginLogin() (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	m.mu.RLock()
	user := m.user
	m.mu.RUnlock()

	if len(user.Credentials) == 0 {
		return nil, nil, fmt.Errorf("no credentials registered")
	}

	return m.webauthn.BeginLogin(user)
}

// FinishLogin completes the WebAuthn login process
func (m *WebAuthnManager) FinishLogin(parsedResponse *protocol.ParsedCredentialAssertionData, sessionData *webauthn.SessionData) (bool, error) {
	m.mu.RLock()
	user := m.user
	m.mu.RUnlock()

	_, err := m.webauthn.ValidateLogin(user, *sessionData, parsedResponse)
	if err != nil {
		return false, err
	}

	log.Printf("[WebAuthn] Successful login with credential: %s", base64.RawURLEncoding.EncodeToString(parsedResponse.RawID))
	return true, nil
}

// WebAuthnSessionData stores session data for WebAuthn flows
type WebAuthnSessionData struct {
	Challenge        string    `json:"challenge"`
	UserID           string    `json:"user_id"`
	CreatedAt        time.Time `json:"created_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	SessionDataBytes []byte    `json:"session_data"`
}

// SessionDataManager manages WebAuthn session data
type SessionDataManager struct {
	sessions map[string]*WebAuthnSessionData
	mu       sync.RWMutex
	ttl      time.Duration
}

// NewSessionDataManager creates a new session data manager
func NewSessionDataManager() *SessionDataManager {
	mgr := &SessionDataManager{
		sessions: make(map[string]*WebAuthnSessionData),
		ttl:      5 * time.Minute,
	}
	go mgr.cleanupExpired()
	return mgr
}

// Store stores session data
func (m *SessionDataManager) Store(sessionID string, sessionData *webauthn.SessionData) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, _ := json.Marshal(sessionData)
	m.sessions[sessionID] = &WebAuthnSessionData{
		Challenge:        string(sessionData.Challenge),
		UserID:           string(sessionData.UserID),
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(m.ttl),
		SessionDataBytes: data,
	}
}

// Get retrieves session data
func (m *SessionDataManager) Get(sessionID string) (*webauthn.SessionData, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists || time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	var sd webauthn.SessionData
	if err := json.Unmarshal(session.SessionDataBytes, &sd); err != nil {
		return nil, false
	}
	return &sd, true
}

// Delete removes session data
func (m *SessionDataManager) Delete(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

// cleanupExpired periodically removes expired sessions
func (m *SessionDataManager) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for id, session := range m.sessions {
			if now.After(session.ExpiresAt) {
				delete(m.sessions, id)
			}
		}
		m.mu.Unlock()
	}
}

// ParseWebAuthnOrigin parses an origin URL and extracts the host for RP ID
func ParseWebAuthnOrigin(origin string) (string, error) {
	u, err := url.Parse(origin)
	if err != nil {
		return "", err
	}
	return u.Host, nil
}
