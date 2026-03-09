package server

import (
	"log"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a simple IP-based rate limiter
type RateLimiter struct {
	requests map[string]*clientInfo
	mu       sync.RWMutex
	limit    int           // Max requests per window
	window   time.Duration // Time window
}

type clientInfo struct {
	count     int
	firstSeen time.Time
	blocked   bool
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*clientInfo),
		limit:    limit,
		window:   window,
	}
	go rl.cleanupExpired()
	return rl
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	info, exists := rl.requests[ip]

	if !exists {
		rl.requests[ip] = &clientInfo{
			count:     1,
			firstSeen: now,
			blocked:   false,
		}
		return true
	}

	// Reset counter if window has passed
	if now.Sub(info.firstSeen) > rl.window {
		info.count = 1
		info.firstSeen = now
		info.blocked = false
		return true
	}

	// Check if limit exceeded
	if info.count >= rl.limit {
		if !info.blocked {
			log.Printf("[SECURITY] Rate limit exceeded for IP: %s (%d requests in %v)", ip, info.count, rl.window)
			info.blocked = true
		}
		return false
	}

	info.count++
	return true
}

// cleanupExpired periodically removes expired entries
func (rl *RateLimiter) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, info := range rl.requests {
			if now.Sub(info.firstSeen) > rl.window*2 {
				delete(rl.requests, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware creates a rate limiting middleware
func (server *Server) RateLimitMiddleware(next http.Handler) http.Handler {
	// Create rate limiters for different endpoints
	// API endpoints: 100 requests per minute
	apiLimiter := NewRateLimiter(100, time.Minute)
	// Auth endpoints: 10 requests per minute (stricter for security)
	authLimiter := NewRateLimiter(10, time.Minute)
	// WebSocket: 10 connections per minute
	wsLimiter := NewRateLimiter(10, time.Minute)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP (handle X-Forwarded-For for reverse proxies)
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			// Take the first IP in the chain
			if idx := len(forwarded); idx > 0 {
				for i, c := range forwarded {
					if c == ',' && i > 0 {
						ip = forwarded[:i]
						break
					}
				}
				if ip == forwarded {
					ip = forwarded
				}
			}
		}

		path := r.URL.Path

		// Select appropriate limiter based on endpoint
		var limiter *RateLimiter
		if contains(path, []string{"/api/webauthn/register", "/api/webauthn/login"}) {
			limiter = authLimiter
		} else if path == "/ws" {
			limiter = wsLimiter
		} else if len(path) >= 4 && path[:4] == "/api" {
			limiter = apiLimiter
		} else {
			// No rate limiting for static assets
			next.ServeHTTP(w, r)
			return
		}

		if !limiter.Allow(ip) {
			http.Error(w, `{"error": "rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// contains checks if a string contains any of the substrings
func contains(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) && s[:len(substr)] == substr {
			return true
		}
	}
	return false
}
