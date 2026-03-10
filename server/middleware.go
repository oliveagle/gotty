package server

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"time"
)

// Security audit log entry
type securityLogEntry struct {
	Timestamp   string
	RemoteAddr  string
	Method      string
	Path        string
	UserAgent   string
	Origin      string
	StatusCode  int
	Duration    time.Duration
	Suspicious  bool
	Reason      string
}

func (server *Server) wrapLogger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &logResponseWriter{w, 200}

		// Security audit: check for suspicious patterns
		suspicious, reason := server.detectSuspiciousRequest(r)

		handler.ServeHTTP(rw, r)

		duration := time.Since(start)

		// SECURITY: Redact sensitive data from URL before logging
		logPath := RedactURLSensitive(r.URL.Path)
		if r.URL.RawQuery != "" {
			logPath = RedactURLSensitive(logPath + "?" + r.URL.RawQuery)
		}

		// Log request (with sensitive data redacted)
		log.Printf("%s %d %s %s %v", r.RemoteAddr, rw.status, r.Method, logPath, duration)

		// Security audit logging for suspicious requests
		if suspicious {
			log.Printf("[SECURITY AUDIT] Suspicious request detected: ip=%s method=%s path=%s user-agent=%s reason=%s",
				r.RemoteAddr, r.Method, logPath, r.UserAgent(), reason)
		}

		// Log authentication failures with more detail
		if rw.status == http.StatusUnauthorized {
			log.Printf("[SECURITY AUDIT] Auth failure: ip=%s path=%s origin=%s",
				r.RemoteAddr, logPath, r.Header.Get("Origin"))
		}
	})
}

// detectSuspiciousRequest checks for common attack patterns
func (server *Server) detectSuspiciousRequest(r *http.Request) (bool, string) {
	path := r.URL.Path
	query := r.URL.RawQuery
	userAgent := r.UserAgent()

	// Check for path traversal attempts
	if strings.Contains(path, "..") || strings.Contains(path, "%2e%2e") {
		return true, "path_traversal"
	}

	// Check for SQL injection patterns (in URL)
	sqlPatterns := []string{"'", "\"", "--", ";", "union", "select", "insert", "delete", "drop", "exec(", "eval("}
	lowerQuery := strings.ToLower(query)
	lowerPath := strings.ToLower(path)
	for _, pattern := range sqlPatterns {
		if strings.Contains(lowerQuery, pattern) || strings.Contains(lowerPath, pattern) {
			return true, "sql_injection_pattern"
		}
	}

	// Check for XSS patterns
	xssPatterns := []string{"<script", "javascript:", "onerror=", "onload=", "onclick="}
	lowerUserAgent := strings.ToLower(userAgent)
	for _, pattern := range xssPatterns {
		if strings.Contains(lowerPath, pattern) || strings.Contains(lowerQuery, pattern) {
			return true, "xss_pattern"
		}
		// Also check user-agent for attack patterns
		if strings.Contains(lowerUserAgent, pattern) {
			return true, "malicious_user_agent"
		}
	}

	// Check for common scanner/bot user agents
	scannerPatterns := []string{"sqlmap", "nmap", "nikto", "masscan", "zgrab", "gobuster", "dirbuster", "wfuzz"}
	for _, pattern := range scannerPatterns {
		if strings.Contains(lowerUserAgent, pattern) {
			return true, "security_scanner"
		}
	}

	// Check for null byte injection
	if strings.Contains(path, "%00") || strings.Contains(query, "%00") {
		return true, "null_byte_injection"
	}

	// Check for overly long requests (potential buffer overflow attempt)
	if len(path) > 1000 || len(query) > 2000 {
		return true, "oversized_request"
	}

	return false, ""
}

func (server *Server) wrapHeaders(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// SECURITY: Don't expose server version
		w.Header().Set("Server", "GoTTY")

		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Cache-Control for API responses
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		}

		// Content-Security-Policy - relaxed for xterm.js and inline styles
		// Note: 'unsafe-inline' for style is needed for xterm.js dynamic styling
		// Note: 'unsafe-eval' is needed for html2canvas
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: blob:; " +
			"font-src 'self' data:; " +
			"connect-src 'self' ws: wss:; " +
			"frame-src * data: blob:; " +
			"frame-ancestors 'self'; " +
			"form-action 'self'; " +
			"base-uri 'self'"
		w.Header().Set("Content-Security-Policy", csp)

		// Referrer-Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy (formerly Feature-Policy)
		w.Header().Set("Permissions-Policy", "clipboard-read=(), clipboard-write=(self), geolocation=(), microphone=(), camera=()")

		// HSTS (HTTP Strict Transport Security) - only if TLS is enabled
		if server.options.EnableTLS {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// SECURITY: CORS headers - strict same-origin policy
		// Only allow same-origin requests by default
		origin := r.Header.Get("Origin")
		if origin != "" {
			host := r.Host
			// Check if origin matches host (same-origin)
			if strings.HasPrefix(origin, "http://"+host) || strings.HasPrefix(origin, "https://"+host) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
			} else if server.isAllowedOrigin(origin) {
				// Allow configured origins
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}
			// For non-allowed origins, don't set CORS headers (browser will block)
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// isAllowedOrigin checks if the origin is in the allowed list
func (server *Server) isAllowedOrigin(origin string) bool {
	// Allow localhost for development
	localhostPatterns := []string{
		"http://localhost",
		"https://localhost",
		"http://127.0.0.1",
		"https://127.0.0.1",
	}
	for _, pattern := range localhostPatterns {
		if strings.HasPrefix(origin, pattern) {
			return true
		}
	}
	return false
}

func (server *Server) wrapBasicAuth(handler http.Handler, credential string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.SplitN(r.Header.Get("Authorization"), " ", 2)

		if len(token) != 2 || strings.ToLower(token[0]) != "basic" {
			w.Header().Set("WWW-Authenticate", `Basic realm="GoTTY"`)
			http.Error(w, "Bad Request", http.StatusUnauthorized)
			return
		}

		payload, err := base64.StdEncoding.DecodeString(token[1])
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if credential != string(payload) {
			w.Header().Set("WWW-Authenticate", `Basic realm="GoTTY"`)
			http.Error(w, "authorization failed", http.StatusUnauthorized)
			return
		}

		log.Printf("Basic Authentication Succeeded: %s", r.RemoteAddr)
		handler.ServeHTTP(w, r)
	})
}
