package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
)

// ClipboardManager handles clipboard operations
type ClipboardManager struct {
	lastContent string
	mu          sync.RWMutex
}

// NewClipboardManager creates a new clipboard manager
func NewClipboardManager() *ClipboardManager {
	return &ClipboardManager{}
}

// GetClipboardContent returns the current system clipboard content
// Tries both clipboard and primary selection (for X11)
func (cm *ClipboardManager) GetClipboardContent() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Try xclip first (most common on Linux)
	// First try clipboard selection
	content, err := exec.Command("xclip", "-selection", "clipboard", "-o").Output()
	if err == nil && len(content) > 0 {
		log.Printf("[clipboard] Got content from xclip clipboard: %d bytes", len(content))
		return string(content)
	}

	// Try primary selection (mouse selection)
	content, err = exec.Command("xclip", "-selection", "primary", "-o").Output()
	if err == nil && len(content) > 0 {
		log.Printf("[clipboard] Got content from xclip primary: %d bytes", len(content))
		return string(content)
	}

	// Try xsel as fallback - clipboard
	content, err = exec.Command("xsel", "--clipboard", "--output").Output()
	if err == nil && len(content) > 0 {
		log.Printf("[clipboard] Got content from xsel clipboard: %d bytes", len(content))
		return string(content)
	}

	// Try xsel - primary
	content, err = exec.Command("xsel", "--primary", "--output").Output()
	if err == nil && len(content) > 0 {
		log.Printf("[clipboard] Got content from xsel primary: %d bytes", len(content))
		return string(content)
	}

	// Try pbpaste on macOS
	content, err = exec.Command("pbpaste").Output()
	if err == nil {
		log.Printf("[clipboard] Got content from pbpaste: %d bytes", len(content))
		return string(content)
	}

	// Try wl-paste on Wayland
	content, err = exec.Command("wl-paste").Output()
	if err == nil {
		log.Printf("[clipboard] Got content from wl-paste: %d bytes", len(content))
		return string(content)
	}

	log.Printf("[clipboard] No content found from any source")
	return ""
}

// SetClipboardContent sets the system clipboard content
func (cm *ClipboardManager) SetClipboardContent(content string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Try xclip first
	cmd := exec.Command("xclip", "-selection", "clipboard", "-i")
	cmd.Stdin = strings.NewReader(content)
	err := cmd.Run()
	if err == nil {
		cm.lastContent = content
		return nil
	}

	// Try xsel
	cmd = exec.Command("xsel", "--clipboard", "--input")
	cmd.Stdin = strings.NewReader(content)
	err = cmd.Run()
	if err == nil {
		cm.lastContent = content
		return nil
	}

	// Try pbcopy on macOS
	cmd = exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(content)
	err = cmd.Run()
	if err == nil {
		cm.lastContent = content
		return nil
	}

	// Try wl-copy on Wayland
	cmd = exec.Command("wl-copy")
	cmd.Stdin = strings.NewReader(content)
	err = cmd.Run()
	if err == nil {
		cm.lastContent = content
		return nil
	}

	return err
}

// handleClipboard handles clipboard API requests
func (server *Server) handleClipboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		// Get clipboard content from server
		content := server.clipboardManager.GetClipboardContent()
		json.NewEncoder(w).Encode(map[string]string{
			"content": content,
		})
	case "POST":
		// Set clipboard content on server
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", 400)
			return
		}
		if err := server.clipboardManager.SetClipboardContent(req.Content); err != nil {
			http.Error(w, "Failed to set clipboard", 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	default:
		http.Error(w, "Method not allowed", 405)
	}
}
