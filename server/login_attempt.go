package server

import (
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// LoginAttempt tracks login attempts for an IP
type LoginAttempt struct {
	Count       int
	FirstAttempt time.Time
	LastAttempt  time.Time
	BlockedUntil time.Time
}

// LoginAttemptManager manages login attempts and blocks
type LoginAttemptManager struct {
	attempts    map[string]*LoginAttempt
	mu          sync.RWMutex
	maxAttempts int           // Max failed attempts before block
	blockWindow time.Duration // How long to block after max attempts
	attemptTTL  time.Duration // How long to remember attempts
}

// NewLoginAttemptManager creates a new login attempt manager
func NewLoginAttemptManager(maxAttempts int, blockWindow time.Duration) *LoginAttemptManager {
	lam := &LoginAttemptManager{
		attempts:    make(map[string]*LoginAttempt),
		maxAttempts: maxAttempts,
		blockWindow: blockWindow,
		attemptTTL:  15 * time.Minute,
	}
	go lam.cleanupExpired()
	return lam
}

// RecordFailure records a failed login attempt
func (lam *LoginAttemptManager) RecordFailure(ip string) {
	lam.mu.Lock()
	defer lam.mu.Unlock()

	now := time.Now()
	attempt, exists := lam.attempts[ip]

	if !exists {
		attempt = &LoginAttempt{
			Count:        1,
			FirstAttempt: now,
			LastAttempt:  now,
		}
		lam.attempts[ip] = attempt
		return
	}

	// Reset if TTL has passed
	if now.Sub(attempt.LastAttempt) > lam.attemptTTL {
		attempt.Count = 1
		attempt.FirstAttempt = now
		attempt.LastAttempt = now
		attempt.BlockedUntil = time.Time{}
	} else {
		attempt.Count++
		attempt.LastAttempt = now
	}

	// Block if exceeded max attempts
	if attempt.Count >= lam.maxAttempts {
		attempt.BlockedUntil = now.Add(lam.blockWindow)
		log.Printf("[SECURITY] IP %s blocked for %v due to %d failed login attempts",
			ip, lam.blockWindow, attempt.Count)
	}
}

// RecordSuccess records a successful login (clears attempts)
func (lam *LoginAttemptManager) RecordSuccess(ip string) {
	lam.mu.Lock()
	defer lam.mu.Unlock()

	if _, exists := lam.attempts[ip]; exists {
		delete(lam.attempts, ip)
		log.Printf("[SECURITY] Cleared login attempts for IP %s after successful login", ip)
	}
}

// IsBlocked checks if an IP is currently blocked
func (lam *LoginAttemptManager) IsBlocked(ip string) bool {
	lam.mu.RLock()
	defer lam.mu.RUnlock()

	attempt, exists := lam.attempts[ip]
	if !exists {
		return false
	}

	if attempt.BlockedUntil.IsZero() {
		return false
	}

	if time.Now().Before(attempt.BlockedUntil) {
		return true
	}

	return false
}

// GetRemainingBlockTime returns how long an IP is still blocked
func (lam *LoginAttemptManager) GetRemainingBlockTime(ip string) time.Duration {
	lam.mu.RLock()
	defer lam.mu.RUnlock()

	attempt, exists := lam.attempts[ip]
	if !exists || attempt.BlockedUntil.IsZero() {
		return 0
	}

	remaining := time.Until(attempt.BlockedUntil)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetAttemptCount returns the number of failed attempts for an IP
func (lam *LoginAttemptManager) GetAttemptCount(ip string) int {
	lam.mu.RLock()
	defer lam.mu.RUnlock()

	attempt, exists := lam.attempts[ip]
	if !exists {
		return 0
	}
	return attempt.Count
}

// cleanupExpired periodically removes expired entries
func (lam *LoginAttemptManager) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		lam.mu.Lock()
		now := time.Now()
		for ip, attempt := range lam.attempts {
			// Remove if block has expired and no recent attempts
			if !attempt.BlockedUntil.IsZero() && now.After(attempt.BlockedUntil) {
				delete(lam.attempts, ip)
			} else if now.Sub(attempt.LastAttempt) > lam.attemptTTL {
				delete(lam.attempts, ip)
			}
		}
		lam.mu.Unlock()
	}
}

// extractIP extracts the client IP from a request
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for reverse proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		for i, c := range xff {
			if c == ',' {
				return net.ParseIP(xff[:i]).String()
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
