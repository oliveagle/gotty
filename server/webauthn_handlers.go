package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
)

// handleWebAuthnStatus returns the current WebAuthn status
func (server *Server) handleWebAuthnStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if server.webAuthnManager == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"enabled":  false,
			"message":  "WebAuthn not enabled",
			"has_auth": false,
		})
		return
	}

	hasCredentials := server.webAuthnManager.HasCredentials()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled":  true,
		"has_auth": hasCredentials,
		"message":  getStatusMessage(hasCredentials),
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
		http.Error(w, "WebAuthn not enabled", http.StatusBadRequest)
		return
	}

	// Begin registration
	options, sessionData, err := server.webAuthnManager.BeginRegistration()
	if err != nil {
		log.Printf("[WebAuthn] Registration begin error: %v", err)
		http.Error(w, "Failed to begin registration", http.StatusInternalServerError)
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
		http.Error(w, "WebAuthn not enabled", http.StatusBadRequest)
		return
	}

	// Parse request
	var req struct {
		SessionID string          `json:"session_id"`
		Response  json.RawMessage `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get session data
	sessionData, exists := server.webAuthnSessions.Get(req.SessionID)
	if !exists {
		http.Error(w, "Session not found or expired", http.StatusBadRequest)
		return
	}
	defer server.webAuthnSessions.Delete(req.SessionID)

	// Parse credential creation response - use bytes.NewReader to wrap raw JSON
	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(req.Response))
	if err != nil {
		log.Printf("[WebAuthn] Parse credential error: %v", err)
		http.Error(w, "Failed to parse credential", http.StatusBadRequest)
		return
	}

	// Finish registration
	if err := server.webAuthnManager.FinishRegistration(parsedResponse, sessionData); err != nil {
		log.Printf("[WebAuthn] Registration finish error: %v", err)
		http.Error(w, "Failed to complete registration", http.StatusBadRequest)
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
		http.Error(w, "WebAuthn not enabled", http.StatusBadRequest)
		return
	}

	if !server.webAuthnManager.HasCredentials() {
		http.Error(w, "No credentials registered", http.StatusBadRequest)
		return
	}

	// Begin login
	options, sessionData, err := server.webAuthnManager.BeginLogin()
	if err != nil {
		log.Printf("[WebAuthn] Login begin error: %v", err)
		http.Error(w, "Failed to begin login", http.StatusInternalServerError)
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
		http.Error(w, "WebAuthn not enabled", http.StatusBadRequest)
		return
	}

	// Parse request
	var req struct {
		SessionID string          `json:"session_id"`
		Response  json.RawMessage `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get session data
	sessionData, exists := server.webAuthnSessions.Get(req.SessionID)
	if !exists {
		http.Error(w, "Session not found or expired", http.StatusBadRequest)
		return
	}
	defer server.webAuthnSessions.Delete(req.SessionID)

	// Parse credential assertion response - use ParseCredentialRequestResponseBytes
	parsedResponse, err := protocol.ParseCredentialRequestResponseBytes(req.Response)
	if err != nil {
		log.Printf("[WebAuthn] Parse assertion error: %v", err)
		http.Error(w, "Failed to parse assertion", http.StatusBadRequest)
		return
	}

	// Finish login
	success, err := server.webAuthnManager.FinishLogin(parsedResponse, sessionData)
	if err != nil || !success {
		log.Printf("[WebAuthn] Login finish error: %v", err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	log.Printf("[WebAuthn] Login successful")

	// Return auth token (session ID that can be used for WebSocket auth)
	authToken := "webauthn:" + base64.StdEncoding.EncodeToString([]byte(req.SessionID))

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message":    "Authentication successful",
		"auth_token": authToken,
	})
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	return base64.RawURLEncoding.EncodeToString([]byte(randomString(16)))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
