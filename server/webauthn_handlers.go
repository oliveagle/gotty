package server

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
)

// jsonError sends a JSON error response
func jsonError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"success": "false",
		"message": message,
	})
}

// handleWebAuthnStatus returns the current WebAuthn status
func (server *Server) handleWebAuthnStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if server.webAuthnManager == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"enabled":       false,
			"message":       "WebAuthn not enabled",
			"has_auth":      false,
			"can_register":  false,
		})
		return
	}

	hasCredentials := server.webAuthnManager.HasCredentials()
	canRegister := server.webAuthnManager.CanRegister()
	requiresToken := hasCredentials && server.webAuthnManager.GetRegisterToken() != ""

	json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled":       true,
		"has_auth":      hasCredentials,
		"can_register":  canRegister,
		"requires_token": requiresToken,
		"message":       getStatusMessage(hasCredentials),
	})
}

func getStatusMessage(hasAuth bool) string {
	if hasAuth {
		return "WebAuthn ready. Click to authenticate."
	}
	return "No Passkey registered. Click to register."
}

// handleWebAuthnRegisterBegin starts the registration process
func (server *Server) handleWebAuthnRegisterBegin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if server.webAuthnManager == nil {
		jsonError(w, "WebAuthn not enabled", http.StatusBadRequest)
		return
	}

	// Check if registration is allowed
	if !server.webAuthnManager.CanRegister() {
		jsonError(w, "Registration is disabled. A passkey is already registered.", http.StatusForbidden)
		return
	}

	// If token is required, validate it
	if server.webAuthnManager.HasCredentials() && server.webAuthnManager.GetRegisterToken() != "" {
		token := r.URL.Query().Get("token")
		if token == "" {
			jsonError(w, "Registration token required. Use: /api/webauthn/register/begin?token=YOUR_TOKEN", http.StatusForbidden)
			return
		}
		if !server.webAuthnManager.ValidateRegisterToken(token) {
			jsonError(w, "Invalid registration token", http.StatusForbidden)
			return
		}
	}

	// Begin registration
	options, sessionData, err := server.webAuthnManager.BeginRegistration()
	if err != nil {
		log.Printf("[WebAuthn] Registration begin error: %v", err)
		jsonError(w, "Failed to begin registration", http.StatusInternalServerError)
		return
	}

	// Store session data
	sessionID := generateSessionID()
	server.webAuthnSessions.Store(sessionID, sessionData)

	// Return options with session ID
	response := map[string]interface{}{
		"session_id": sessionID,
		"options":    options,
	}

	json.NewEncoder(w).Encode(response)
}

// handleWebAuthnRegisterFinish completes the registration process
func (server *Server) handleWebAuthnRegisterFinish(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if server.webAuthnManager == nil {
		jsonError(w, "WebAuthn not enabled", http.StatusBadRequest)
		return
	}

	// Parse request
	var req struct {
		SessionID string          `json:"session_id"`
		Response  json.RawMessage `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get session data
	sessionData, exists := server.webAuthnSessions.Get(req.SessionID)
	if !exists {
		jsonError(w, "Session not found or expired", http.StatusBadRequest)
		return
	}
	defer server.webAuthnSessions.Delete(req.SessionID)

	// Parse credential creation response - use bytes.NewReader to wrap raw JSON
	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(req.Response))
	if err != nil {
		log.Printf("[WebAuthn] Parse credential error: %v", err)
		jsonError(w, "Failed to parse credential: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Finish registration
	if err := server.webAuthnManager.FinishRegistration(parsedResponse, sessionData); err != nil {
		log.Printf("[WebAuthn] Registration finish error: %v", err)
		jsonError(w, "Failed to complete registration: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[WebAuthn] Registration successful")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Passkey registered successfully",
	})
}

// handleWebAuthnLoginBegin starts the login process
func (server *Server) handleWebAuthnLoginBegin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if server.webAuthnManager == nil {
		jsonError(w, "WebAuthn not enabled", http.StatusBadRequest)
		return
	}

	if !server.webAuthnManager.HasCredentials() {
		jsonError(w, "No credentials registered", http.StatusBadRequest)
		return
	}

	// Begin login
	options, sessionData, err := server.webAuthnManager.BeginLogin()
	if err != nil {
		log.Printf("[WebAuthn] Login begin error: %v", err)
		jsonError(w, "Failed to begin login", http.StatusInternalServerError)
		return
	}

	// Store session data
	sessionID := generateSessionID()
	server.webAuthnSessions.Store(sessionID, sessionData)

	// Return options with session ID
	response := map[string]interface{}{
		"session_id": sessionID,
		"options":    options,
	}

	json.NewEncoder(w).Encode(response)
}

// handleWebAuthnLoginFinish completes the login process
func (server *Server) handleWebAuthnLoginFinish(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if server.webAuthnManager == nil {
		jsonError(w, "WebAuthn not enabled", http.StatusBadRequest)
		return
	}

	// Parse request
	var req struct {
		SessionID string          `json:"session_id"`
		Response  json.RawMessage `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get session data
	sessionData, exists := server.webAuthnSessions.Get(req.SessionID)
	if !exists {
		jsonError(w, "Session not found or expired", http.StatusBadRequest)
		return
	}
	defer server.webAuthnSessions.Delete(req.SessionID)

	// Parse credential assertion response - use ParseCredentialRequestResponseBytes
	parsedResponse, err := protocol.ParseCredentialRequestResponseBytes(req.Response)
	if err != nil {
		log.Printf("[WebAuthn] Parse assertion error: %v", err)
		jsonError(w, "Failed to parse assertion: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Finish login
	success, err := server.webAuthnManager.FinishLogin(parsedResponse, sessionData)
	if err != nil || !success {
		log.Printf("[WebAuthn] Login finish error: %v", err)
		jsonError(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	log.Printf("[WebAuthn] Login successful")

	// Create long-lived auth session
	authToken := server.authSessionMgr.CreateSession()
	log.Printf("[WebAuthn] Created auth session, TTL: %d hours", server.options.WebAuthnSessionTTL)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message":    "Authentication successful",
		"auth_token": authToken,
	})
}

// handleWebAuthnValidateToken validates an auth token
func (server *Server) handleWebAuthnValidateToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if server.authSessionMgr == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":   false,
			"message": "Auth sessions not enabled",
		})
		return
	}

	// Get token from Authorization header or query parameter
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	if token == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":   false,
			"message": "No token provided",
		})
		return
	}

	valid := server.authSessionMgr.ValidateToken(token)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":   valid,
		"message": func() string {
			if valid {
				return "Token is valid"
			}
			return "Token is invalid or expired"
		}(),
	})
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	return base64.RawURLEncoding.EncodeToString([]byte(randomString(16)))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	rand.Read(b) // Use crypto/rand for true randomness
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}
