package server

import (
	"regexp"
	"strings"
)

// SensitiveDataRedaction provides utilities to redact sensitive information from logs
type SensitiveDataRedaction struct {
	// Patterns to match and redact
	patterns []*redactionPattern
}

type redactionPattern struct {
	name     string
	pattern  *regexp.Regexp
	replace  string
}

// NewSensitiveDataRedaction creates a new redaction utility
func NewSensitiveDataRedaction() *SensitiveDataRedaction {
	sdr := &SensitiveDataRedaction{
		patterns: make([]*redactionPattern, 0),
	}

	// Add common sensitive patterns
	sdr.addPattern("auth_token", `token["']?\s*[:=]\s*["']?([a-zA-Z0-9_-]{20,})["']?`, `token=***REDACTED***`)
	sdr.addPattern("bearer_token", `Bearer\s+([a-zA-Z0-9_-]{20,})`, `Bearer ***REDACTED***`)
	sdr.addPattern("password", `password["']?\s*[:=]\s*["']?([^\s"']+)["']?`, `password=***REDACTED***`)
	sdr.addPattern("api_key", `api[_-]?key["']?\s*[:=]\s*["']?([a-zA-Z0-9_-]{20,})["']?`, `api_key=***REDACTED***`)
	sdr.addPattern("secret", `secret["']?\s*[:=]\s*["']?([a-zA-Z0-9_-]{10,})["']?`, `secret=***REDACTED***`)
	sdr.addPattern("credential", `credential["']?\s*[:=]\s*["']?([a-zA-Z0-9_-]{10,})["']?`, `credential=***REDACTED***`)

	return sdr
}

// addPattern adds a new redaction pattern
func (sdr *SensitiveDataRedaction) addPattern(name, patternStr, replace string) {
	pattern := regexp.MustCompile(patternStr)
	sdr.patterns = append(sdr.patterns, &redactionPattern{
		name:    name,
		pattern: pattern,
		replace: replace,
	})
}

// Redact redacts sensitive information from a string
func (sdr *SensitiveDataRedaction) Redact(input string) string {
	result := input
	for _, rp := range sdr.patterns {
		result = rp.pattern.ReplaceAllString(result, rp.replace)
	}
	return result
}

// RedactURL redacts sensitive query parameters from URLs
func (sdr *SensitiveDataRedaction) RedactURL(url string) string {
	// Redact token parameter
	if strings.Contains(url, "token=") {
		re := regexp.MustCompile(`token=[^&]*`)
		url = re.ReplaceAllString(url, "token=***REDACTED***")
	}
	return url
}

// RedactHeaders redacts sensitive values from HTTP headers
func (sdr *SensitiveDataRedaction) RedactHeaders(headers map[string]string) map[string]string {
	result := make(map[string]string)
	sensitiveHeaders := map[string]bool{
		"Authorization":   true,
		"Cookie":          true,
		"Set-Cookie":      true,
		"X-Auth-Token":    true,
		"X-Api-Key":       true,
		"X-Session-Token": true,
	}

	for k, v := range headers {
		if sensitiveHeaders[k] {
			result[k] = "***REDACTED***"
		} else {
			result[k] = v
		}
	}
	return result
}

// Global redaction instance
var globalRedaction = NewSensitiveDataRedaction()

// RedactSensitive redacts sensitive data using the global redaction instance
func RedactSensitive(input string) string {
	return globalRedaction.Redact(input)
}

// RedactURLSensitive redacts sensitive data in URLs
func RedactURLSensitive(url string) string {
	return globalRedaction.RedactURL(url)
}
