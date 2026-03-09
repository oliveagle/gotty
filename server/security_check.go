package server

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// SecurityCheckResult represents the result of a security check
type SecurityCheckResult struct {
	Name     string
	Status   string // "PASS", "WARN", "FAIL"
	Message  string
	Critical bool
}

// SecurityChecker performs security configuration checks
type SecurityChecker struct {
	options *Options
	results []SecurityCheckResult
}

// NewSecurityChecker creates a new security checker
func NewSecurityChecker(options *Options) *SecurityChecker {
	return &SecurityChecker{
		options: options,
		results: make([]SecurityCheckResult, 0),
	}
}

// RunChecks executes all security checks
func (sc *SecurityChecker) RunChecks() []SecurityCheckResult {
	sc.checkAuthentication()
	sc.checkTLS()
	sc.checkNetworkBinding()
	sc.checkFilePermissions()
	sc.checkRandomURL()
	sc.checkWritePermission()
	sc.checkWebAuthnConfig()

	return sc.results
}

// checkAuthentication verifies authentication configuration
func (sc *SecurityChecker) checkAuthentication() {
	if !sc.options.EnableAuth {
		sc.results = append(sc.results, SecurityCheckResult{
			Name:     "Authentication",
			Status:   "WARN",
			Message:  "Authentication is disabled. Terminal access is publicly accessible.",
			Critical: true,
		})
	} else {
		sc.results = append(sc.results, SecurityCheckResult{
			Name:     "Authentication",
			Status:   "PASS",
			Message:  "WebAuthn authentication is enabled.",
			Critical: false,
		})

		// Check if credentials exist
		if sc.options.WebAuthnDataDir != "" {
			dataDir := expandHome(sc.options.WebAuthnDataDir)
			credFile := filepath.Join(dataDir, "webauthn_user.json")
			if _, err := os.Stat(credFile); os.IsNotExist(err) {
				sc.results = append(sc.results, SecurityCheckResult{
					Name:     "WebAuthn Credentials",
					Status:   "WARN",
					Message:  "No WebAuthn credentials registered. First user to connect can register a passkey.",
					Critical: false,
				})
			}
		}
	}
}

// checkTLS verifies TLS configuration
func (sc *SecurityChecker) checkTLS() {
	if !sc.options.EnableTLS {
		// Check if binding to localhost only (development scenario)
		isLocalhost := sc.options.Address == "127.0.0.1" ||
			sc.options.Address == "localhost" ||
			sc.options.Address == "::1" ||
			sc.options.Address == ""

		if sc.options.EnableAuth {
			if isLocalhost {
				// Localhost development: downgrade to WARN
				sc.results = append(sc.results, SecurityCheckResult{
					Name:     "TLS/Encryption",
					Status:   "WARN",
					Message:  "TLS disabled (localhost only). Consider enabling TLS for production.",
					Critical: false,
				})
			} else {
				// Public binding: this is critical
				sc.results = append(sc.results, SecurityCheckResult{
					Name:     "TLS/Encryption",
					Status:   "FAIL",
					Message:  "TLS is disabled with authentication enabled. Credentials transmitted in plaintext!",
					Critical: true,
				})
			}
		} else {
			sc.results = append(sc.results, SecurityCheckResult{
				Name:     "TLS/Encryption",
				Status:   "WARN",
				Message:  "TLS is disabled. Consider enabling TLS for encrypted communication.",
				Critical: false,
			})
		}
	} else {
		sc.results = append(sc.results, SecurityCheckResult{
			Name:     "TLS/Encryption",
			Status:   "PASS",
			Message:  "TLS is enabled for encrypted communication.",
			Critical: false,
		})

		// Check certificate file permissions
		crtFile := expandHome(sc.options.TLSCrtFile)
		if err := sc.checkFilePermission(crtFile, "TLS Certificate"); err != nil {
			sc.results = append(sc.results, SecurityCheckResult{
				Name:     "TLS Certificate File",
				Status:   "WARN",
				Message:  fmt.Sprintf("Certificate file issue: %v", err),
				Critical: false,
			})
		}

		// Check key file permissions - should be 0600
		keyFile := expandHome(sc.options.TLSKeyFile)
		if info, err := os.Stat(keyFile); err == nil {
			mode := info.Mode().Perm()
			if mode != 0600 {
				sc.results = append(sc.results, SecurityCheckResult{
					Name:     "TLS Key Permissions",
					Status:   "WARN",
					Message:  fmt.Sprintf("TLS key file has permissions %o, recommended: 0600", mode),
					Critical: true,
				})
			}
		}
	}
}

// checkNetworkBinding verifies network binding configuration
func (sc *SecurityChecker) checkNetworkBinding() {
	addr := sc.options.Address
	if addr == "0.0.0.0" || addr == "::" {
		sc.results = append(sc.results, SecurityCheckResult{
			Name:     "Network Binding",
			Status:   "WARN",
			Message:  "Server bound to all interfaces (0.0.0.0). Accessible from network.",
			Critical: false,
		})
	} else if addr == "127.0.0.1" || addr == "localhost" {
		sc.results = append(sc.results, SecurityCheckResult{
			Name:     "Network Binding",
			Status:   "PASS",
			Message:  "Server bound to localhost only. Not accessible from network.",
			Critical: false,
		})
	}
}

// checkFilePermissions verifies sensitive file permissions
func (sc *SecurityChecker) checkFilePermissions() {
	// Check WebAuthn data directory
	if sc.options.EnableAuth && sc.options.WebAuthnDataDir != "" {
		dataDir := expandHome(sc.options.WebAuthnDataDir)
		if err := sc.checkSecureDirectory(dataDir); err != nil {
			sc.results = append(sc.results, SecurityCheckResult{
				Name:     "WebAuthn Data Directory",
				Status:   "WARN",
				Message:  fmt.Sprintf("WebAuthn directory issue: %v", err),
				Critical: true,
			})
		} else {
			sc.results = append(sc.results, SecurityCheckResult{
				Name:     "WebAuthn Data Directory",
				Status:   "PASS",
				Message:  "WebAuthn data directory has secure permissions.",
				Critical: false,
			})
		}
	}

	// Check sessions file
	homeDir, _ := os.UserHomeDir()
	sessionsFile := filepath.Join(homeDir, ".config", "gotty", "sessions.json")
	if info, err := os.Stat(sessionsFile); err == nil {
		mode := info.Mode().Perm()
		if mode > 0600 {
			sc.results = append(sc.results, SecurityCheckResult{
				Name:     "Sessions File",
				Status:   "WARN",
				Message:  fmt.Sprintf("Sessions file has permissions %o, recommended: 0600", mode),
				Critical: false,
			})
		}
	}
}

// checkRandomURL verifies random URL configuration
func (sc *SecurityChecker) checkRandomURL() {
	if sc.options.EnableRandomUrl {
		if sc.options.RandomUrlLength < 16 {
			sc.results = append(sc.results, SecurityCheckResult{
				Name:     "Random URL Length",
				Status:   "WARN",
				Message:  fmt.Sprintf("Random URL length is %d, minimum recommended: 16", sc.options.RandomUrlLength),
				Critical: false,
			})
		} else {
			sc.results = append(sc.results, SecurityCheckResult{
				Name:     "Random URL Length",
				Status:   "PASS",
				Message:  fmt.Sprintf("Random URL length is %d characters.", sc.options.RandomUrlLength),
				Critical: false,
			})
		}
	}
}

// checkWritePermission verifies write permission configuration
func (sc *SecurityChecker) checkWritePermission() {
	if sc.options.PermitWrite {
		if !sc.options.EnableAuth {
			sc.results = append(sc.results, SecurityCheckResult{
				Name:     "Write Permission",
				Status:   "FAIL",
				Message:  "Write permission enabled WITHOUT authentication. Anyone can execute commands!",
				Critical: true,
			})
		} else {
			sc.results = append(sc.results, SecurityCheckResult{
				Name:     "Write Permission",
				Status:   "PASS",
				Message:  "Write permission enabled with authentication.",
				Critical: false,
			})
		}
	} else {
		sc.results = append(sc.results, SecurityCheckResult{
			Name:     "Write Permission",
			Status:   "PASS",
			Message:  "Write permission disabled (read-only mode).",
			Critical: false,
		})
	}
}

// checkWebAuthnConfig verifies WebAuthn configuration
func (sc *SecurityChecker) checkWebAuthnConfig() {
	if !sc.options.EnableAuth {
		return
	}

	// Check session TTL
	if sc.options.WebAuthnSessionTTL > 168 { // More than 7 days
		sc.results = append(sc.results, SecurityCheckResult{
			Name:     "Session TTL",
			Status:   "WARN",
			Message:  fmt.Sprintf("Session TTL is %d hours (> 7 days). Consider shorter sessions.", sc.options.WebAuthnSessionTTL),
			Critical: false,
		})
	}

	// Check registration token
	if sc.options.WebAuthnRegisterToken == "" && sc.options.WebAuthnAllowRegister {
		sc.results = append(sc.results, SecurityCheckResult{
			Name:     "Registration Token",
			Status:   "WARN",
			Message:  "Registration allowed without token requirement. Anyone can register after first credential.",
			Critical: false,
		})
	}
}

// checkFilePermission checks if a file exists and has appropriate permissions
func (sc *SecurityChecker) checkFilePermission(path, name string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s file not found: %s", name, path)
	}

	mode := info.Mode().Perm()
	if mode > 0644 {
		return fmt.Errorf("permissions too open: %o", mode)
	}

	return nil
}

// checkSecureDirectory checks if a directory has secure permissions
func (sc *SecurityChecker) checkSecureDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	mode := info.Mode().Perm()
	if mode > 0700 {
		return fmt.Errorf("directory permissions too open: %o, should be 0700", mode)
	}

	return nil
}

// PrintReport prints the security check report with colors
func (sc *SecurityChecker) PrintReport() {
	fmt.Println("\n" + ColorCyan + "=== Security Configuration Check ===" + ColorReset)

	hasFailures := false
	hasWarnings := false

	for _, result := range sc.results {
		var statusIcon, statusText string
		if result.Status == "PASS" {
			statusIcon = ColorGreen + "✓" + ColorReset
			statusText = ColorGreen + "PASS" + ColorReset
		} else if result.Status == "WARN" {
			statusIcon = ColorYellow + "⚠" + ColorReset
			statusText = ColorYellow + "WARN" + ColorReset
			hasWarnings = true
		} else if result.Status == "FAIL" {
			statusIcon = ColorRed + "✗" + ColorReset
			statusText = ColorRed + "FAIL" + ColorReset
			hasFailures = true
		}

		criticalFlag := ""
		if result.Critical {
			criticalFlag = ColorRed + " [CRITICAL]" + ColorReset
		}

		fmt.Printf("  %s [%s]%s %s: %s\n", statusIcon, statusText, criticalFlag, result.Name, result.Message)
	}

	fmt.Println(ColorCyan + "====================================" + ColorReset)

	if hasFailures {
		fmt.Println(ColorRed + "[SECURITY] CRITICAL: Security check FAILED. Please review configuration." + ColorReset)
	} else if hasWarnings {
		fmt.Println(ColorYellow + "[SECURITY] WARNING: Security check completed with warnings." + ColorReset)
	} else {
		fmt.Println(ColorGreen + "[SECURITY] All security checks passed." + ColorReset)
	}
	fmt.Println()
}

// expandHome expands ~ to home directory
func expandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		usr, _ := user.Current()
		return filepath.Join(usr.HomeDir, path[1:])
	}
	return path
}
