package server

import (
	"crypto/rand"
	"crypto/subtle"
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
	webauthn       *webauthn.WebAuthn
	user           *WebAuthnUser
	mu             sync.RWMutex
	dataFile       string
	registerToken  string
	allowRegister  bool
}

// NewWebAuthnManager creates a new WebAuthn manager
func NewWebAuthnManager(displayName, hostname, dataDir, registerToken string, allowRegister bool) (*WebAuthnManager, error) {
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

	// Build origins - WebAuthn requires exact origin match including port
	// For localhost, allow both with and without common ports
	origins := []string{}
	if rpID == "localhost" {
		// Allow common localhost origins (both HTTP and HTTPS)
		origins = []string{
			"http://localhost",
			"http://localhost:13782", // default gotty port
			"http://localhost:8080",
			"http://127.0.0.1",
			"http://127.0.0.1:13782",
			"https://localhost",
			"https://localhost:13782",
			"https://127.0.0.1",
			"https://127.0.0.1:13782",
		}
	} else {
		// For custom hostname, use https with port
		origins = []string{
			"https://" + rpID,
			"https://" + rpID + ":13782", // default gotty port
		}
	}

	wconfig := &webauthn.Config{
		RPDisplayName: displayName,
		RPID:          rpID,
		RPOrigins:     origins,
	}

	wn, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebAuthn instance: %w", err)
	}

	dataFile := filepath.Join(dataDir, "webauthn_user.json")

	log.Printf("[WebAuthn] RP ID: %s, Allowed origins: %v", rpID, origins)

	mgr := &WebAuthnManager{
		webauthn:      wn,
		user: &WebAuthnUser{
			ID:          generateUserID(),
			Name:        "gotty",
			DisplayName: "GoTTY User",
		},
		dataFile:      dataFile,
		registerToken: registerToken,
		allowRegister: allowRegister,
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
// Note: This method does NOT acquire locks - caller must hold appropriate lock
func (m *WebAuthnManager) saveUser() error {
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

// CanRegister returns true if registration is allowed
// Registration is allowed if:
// 1. No credentials exist yet (first-time setup)
// 2. allowRegister flag is true AND no credentials exist
// 3. registerToken is set (requires token to register)
func (m *WebAuthnManager) CanRegister() bool {
	m.mu.RLock()
	hasCreds := len(m.user.Credentials) > 0
	m.mu.RUnlock()

	// If no credentials exist, always allow registration (first-time setup)
	if !hasCreds {
		return true
	}

	// If register token is set, allow registration with token
	if m.registerToken != "" {
		return true
	}

	// Otherwise, registration is blocked
	return false
}

// ValidateRegisterToken validates the registration token
// Uses constant-time comparison to prevent timing attacks
func (m *WebAuthnManager) ValidateRegisterToken(token string) bool {
	if m.registerToken == "" {
		return false // No token configured, registration not allowed after first credential
	}
	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(m.registerToken), []byte(token)) == 1
}

// GetRegisterToken returns the configured register token (for checking if token-based registration is enabled)
func (m *WebAuthnManager) GetRegisterToken() string {
	return m.registerToken
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
	SessionData *webauthn.SessionData
	CreatedAt   time.Time
	ExpiresAt   time.Time
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

	m.sessions[sessionID] = &WebAuthnSessionData{
		SessionData: sessionData,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(m.ttl),
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

	return session.SessionData, true
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

// AuthSession represents a long-lived authentication session
type AuthSession struct {
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// AuthSessionManager manages long-lived auth sessions
type AuthSessionManager struct {
	sessions map[string]*AuthSession
	mu       sync.RWMutex
	ttl      time.Duration
	dataFile string
}

// NewAuthSessionManager creates a new auth session manager
func NewAuthSessionManager(dataDir string, ttlHours int) *AuthSessionManager {
	ttl := time.Duration(ttlHours) * time.Hour
	dataFile := filepath.Join(dataDir, "auth_sessions.json")

	mgr := &AuthSessionManager{
		sessions: make(map[string]*AuthSession),
		ttl:      ttl,
		dataFile: dataFile,
	}

	// Load existing sessions
	mgr.loadSessions()
	go mgr.cleanupExpired()

	return mgr
}

// loadSessions loads sessions from file
func (m *AuthSessionManager) loadSessions() error {
	data, err := os.ReadFile(m.dataFile)
	if err != nil {
		return err
	}

	var sessions map[string]*AuthSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Only load non-expired sessions
	now := time.Now()
	for id, session := range sessions {
		if now.Before(session.ExpiresAt) {
			m.sessions[id] = session
		}
	}

	return nil
}

// saveSessions saves sessions to file
func (m *AuthSessionManager) saveSessions() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.MarshalIndent(m.sessions, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.dataFile, data, 0600)
}

// CreateSession creates a new auth session and returns the token
func (m *AuthSessionManager) CreateSession() string {
	token := base64.RawURLEncoding.EncodeToString(generateUserID()) // Reuse generateUserID for random bytes

	session := &AuthSession{
		Token:     token,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(m.ttl),
	}

	m.mu.Lock()
	m.sessions[token] = session
	m.mu.Unlock()

	m.saveSessions()

	return token
}

// ValidateToken validates an auth token
func (m *AuthSessionManager) ValidateToken(token string) bool {
	if token == "" {
		return false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[token]
	if !exists {
		return false
	}

	return time.Now().Before(session.ExpiresAt)
}

// DeleteSession removes a session
func (m *AuthSessionManager) DeleteSession(token string) {
	m.mu.Lock()
	delete(m.sessions, token)
	m.mu.Unlock()

	m.saveSessions()
}

// cleanupExpired periodically removes expired sessions
func (m *AuthSessionManager) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for id, session := range m.sessions {
			if now.After(session.ExpiresAt) {
				delete(m.sessions, id)
			}
		}
		m.mu.Unlock()
		m.saveSessions()
	}
}
