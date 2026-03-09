package server

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
)

// IPFilter provides IP-based access control
type IPFilter struct {
	whitelist    map[string]bool
	blacklist    map[string]bool
	mu           sync.RWMutex
	allowLocal   bool // Always allow localhost
	allowPrivate bool // Allow private IP ranges
}

// NewIPFilter creates a new IP filter
func NewIPFilter() *IPFilter {
	return &IPFilter{
		whitelist:    make(map[string]bool),
		blacklist:    make(map[string]bool),
		allowLocal:   true,  // Always allow localhost by default
		allowPrivate: true,  // Allow private IPs by default
	}
}

// AddToWhitelist adds an IP or CIDR to the whitelist
func (f *IPFilter) AddToWhitelist(ip string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.whitelist[ip] = true
	log.Printf("[SECURITY] Added %s to IP whitelist", ip)
}

// RemoveFromWhitelist removes an IP from the whitelist
func (f *IPFilter) RemoveFromWhitelist(ip string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.whitelist, ip)
	log.Printf("[SECURITY] Removed %s from IP whitelist", ip)
}

// AddToBlacklist adds an IP or CIDR to the blacklist
func (f *IPFilter) AddToBlacklist(ip string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.blacklist[ip] = true
	log.Printf("[SECURITY] Added %s to IP blacklist", ip)
}

// RemoveFromBlacklist removes an IP from the blacklist
func (f *IPFilter) RemoveFromBlacklist(ip string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.blacklist, ip)
	log.Printf("[SECURITY] Removed %s from IP blacklist", ip)
}

// SetAllowLocal sets whether to always allow localhost
func (f *IPFilter) SetAllowLocal(allow bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.allowLocal = allow
}

// SetAllowPrivate sets whether to allow private IP ranges
func (f *IPFilter) SetAllowPrivate(allow bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.allowPrivate = allow
}

// IsAllowed checks if an IP is allowed to access
func (f *IPFilter) IsAllowed(clientIP string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Parse the IP
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false // Invalid IP
	}

	// Check blacklist first (highest priority)
	for cidr := range f.blacklist {
		if f.matchCIDR(ip, cidr) {
			log.Printf("[SECURITY] IP %s rejected (blacklisted)", clientIP)
			return false
		}
	}

	// Check whitelist
	if len(f.whitelist) > 0 {
		for cidr := range f.whitelist {
			if f.matchCIDR(ip, cidr) {
				return true
			}
		}
		// If whitelist is non-empty and IP not in whitelist, reject
		// Exception: localhost if allowLocal is true
		if f.allowLocal && f.isLocalhost(ip) {
			return true
		}
		return false
	}

	// No whitelist configured - check defaults
	if f.allowLocal && f.isLocalhost(ip) {
		return true
	}

	if f.allowPrivate && f.isPrivateIP(ip) {
		return true
	}

	// If no restrictions, allow all
	if len(f.blacklist) == 0 && len(f.whitelist) == 0 {
		return true
	}

	return false
}

// matchCIDR checks if an IP matches a CIDR notation
func (f *IPFilter) matchCIDR(ip net.IP, cidr string) bool {
	// If it's a single IP (no /)
	if !strings.Contains(cidr, "/") {
		cidrIP := net.ParseIP(cidr)
		if cidrIP != nil {
			return ip.Equal(cidrIP)
		}
	}

	// Parse as CIDR
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return network.Contains(ip)
}

// isLocalhost checks if an IP is localhost
func (f *IPFilter) isLocalhost(ip net.IP) bool {
	return ip.IsLoopback()
}

// isPrivateIP checks if an IP is in a private range
func (f *IPFilter) isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	return false
}

// IPFilterMiddleware creates an IP filtering middleware
func (server *Server) IPFilterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip IP filtering for static assets and health checks
		path := r.URL.Path
		if strings.HasPrefix(path, "/js/") ||
			strings.HasPrefix(path, "/css/") ||
			path == "/favicon.png" ||
			path == "/auth_token.js" ||
			path == "/config.js" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip if IP filter is not configured (allow all)
		if server.ipFilter == nil {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := extractIP(r)

		if !server.ipFilter.IsAllowed(clientIP) {
			log.Printf("[SECURITY] Access denied for IP: %s path: %s", clientIP, path)
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetWhitelist returns the current whitelist
func (f *IPFilter) GetWhitelist() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	list := make([]string, 0, len(f.whitelist))
	for ip := range f.whitelist {
		list = append(list, ip)
	}
	return list
}

// GetBlacklist returns the current blacklist
func (f *IPFilter) GetBlacklist() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	list := make([]string, 0, len(f.blacklist))
	for ip := range f.blacklist {
		list = append(list, ip)
	}
	return list
}
