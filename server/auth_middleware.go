package server

import (
	"log"
	"net/http"
	"strings"
)

// AuthMiddleware provides unified authentication for all protected endpoints
type AuthMiddleware struct {
	server *Server
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(server *Server) *AuthMiddleware {
	return &AuthMiddleware{server: server}
}

// Wrap wraps an http.Handler with authentication
func (m *AuthMiddleware) Wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.server.options.EnableAuth {
			// Auth not enabled, pass through
			next(w, r)
			return
		}

		// Check for valid auth token
		token := m.extractToken(r)
		if token == "" {
			log.Printf("[Auth] Missing token for %s %s", r.Method, r.URL.Path)
			http.Error(w, `{"error": "authentication required"}`, http.StatusUnauthorized)
			return
		}

		if !m.server.authSessionMgr.ValidateToken(token) {
			log.Printf("[Auth] Invalid token for %s %s", r.Method, r.URL.Path)
			http.Error(w, `{"error": "invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Token is valid, proceed
		next(w, r)
	}
}

// WrapWS wraps a WebSocket handler with authentication
func (m *AuthMiddleware) WrapWS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.server.options.EnableAuth {
			// Auth not enabled, pass through
			next(w, r)
			return
		}

		// For WebSocket, check token in query parameter
		token := r.URL.Query().Get("token")
		if token == "" {
			log.Printf("[Auth] Missing token for WebSocket %s", r.URL.Path)
			http.Error(w, `{"error": "authentication required"}`, http.StatusUnauthorized)
			return
		}

		if !m.server.authSessionMgr.ValidateToken(token) {
			log.Printf("[Auth] Invalid token for WebSocket %s", r.URL.Path)
			http.Error(w, `{"error": "invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Token is valid, proceed
		next(w, r)
	}
}

// extractToken extracts auth token from request
// Supports: Authorization header, query parameter, cookie
//
// SECURITY WARNING: Token passed via URL query parameter may be logged in:
// - Server access logs (nginx, apache, etc.)
// - Browser history
// - Referer headers when navigating to external sites
//
// RECOMMENDED: Use Authorization header or HttpOnly Cookie for production.
// Query parameter support is provided for compatibility only.
func (m *AuthMiddleware) extractToken(r *http.Request) string {
	// 1. Check Authorization header (Bearer token) - RECOMMENDED
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		return authHeader
	}

	// 2. Check cookie - RECOMMENDED (HttpOnly)
	cookie, err := r.Cookie("gotty_auth_token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// 3. Check query parameter - NOT RECOMMENDED (logged in URLs)
	// SECURITY: Log warning when token is passed via URL
	token := r.URL.Query().Get("token")
	if token != "" {
		log.Printf("[SECURITY] Warning: Token passed via URL query parameter - may be logged. Path: %s", r.URL.Path)
		return token
	}

	return ""
}

// RequireAuth is a helper that returns 401 if not authenticated
// Can be used in handlers that need inline auth check
func (server *Server) RequireAuth(w http.ResponseWriter, r *http.Request) bool {
	if !server.options.EnableAuth {
		return true
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}
	}

	if token == "" || !server.authSessionMgr.ValidateToken(token) {
		http.Error(w, `{"error": "authentication required"}`, http.StatusUnauthorized)
		return false
	}

	return true
}
