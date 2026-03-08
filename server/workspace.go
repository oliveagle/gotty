package server

import (
	"time"
)

// Workspace represents a top-level workspace for organizing sessions
type Workspace struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	ColorTheme string    `json:"color_theme"` // blue, green, purple, orange, red, cyan, pink, gray
	Icon       string    `json:"icon"`        // emoji
	Order      int       `json:"order"`
	CreatedAt  time.Time `json:"created_at"`
}

// DefaultWorkspaceID is the ID for the default workspace
const DefaultWorkspaceID = "default"

// DefaultColorTheme is the default color theme
const DefaultColorTheme = "blue"

// AvailableColorThemes returns all available color themes
func AvailableColorThemes() []string {
	return []string{"blue", "green", "purple", "orange", "red", "cyan", "pink", "gray"}
}

// ColorThemeToCSS returns the CSS variable value for a color theme
func ColorThemeToCSS(theme string) string {
	switch theme {
	case "blue":
		return "#4a9eff"
	case "green":
		return "#4ade80"
	case "purple":
		return "#a855f7"
	case "orange":
		return "#f59e0b"
	case "red":
		return "#ef4444"
	case "cyan":
		return "#06b6d4"
	case "pink":
		return "#ec4899"
	case "gray":
		return "#6b7280"
	default:
		return "#4a9eff"
	}
}

// DefaultWorkspaceIcon returns a default icon for a color theme
func DefaultWorkspaceIcon(theme string) string {
	switch theme {
	case "blue":
		return "💼" // work
	case "green":
		return "🏠" // home/personal
	case "purple":
		return "🎨" // creative
	case "orange":
		return "⚡" // important
	case "red":
		return "🔒" // private
	case "cyan":
		return "📚" // learning
	case "pink":
		return "🎮" // entertainment
	case "gray":
		return "📁" // archive
	default:
		return "📁"
	}
}
