package server

import (
	"fmt"
	"log"
	"os"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[37m"
)

// Log levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

var (
	logLevel = LogLevelInfo
	// Prefixes with colors
	prefixInfo  = fmt.Sprintf("%s[INFO]%s ", ColorGreen, ColorReset)
	prefixWarn  = fmt.Sprintf("%s[WARN]%s ", ColorYellow, ColorReset)
	prefixError = fmt.Sprintf("%s[ERROR]%s ", ColorRed, ColorReset)
	prefixDebug = fmt.Sprintf("%s[DEBUG]%s ", ColorGray, ColorReset)
)

// ColorLogger provides colored logging output
type ColorLogger struct {
	prefix string
}

// NewColorLogger creates a new colored logger
func NewColorLogger(prefix string) *ColorLogger {
	return &ColorLogger{prefix: prefix}
}

// SetLogLevel sets the global log level
func SetLogLevel(level LogLevel) {
	logLevel = level
}

// Info logs an info message (green)
func (l *ColorLogger) Info(format string, args ...interface{}) {
	if logLevel <= LogLevelInfo {
		log.Printf(prefixInfo+l.prefix+format, args...)
	}
}

// Warn logs a warning message (yellow)
func (l *ColorLogger) Warn(format string, args ...interface{}) {
	if logLevel <= LogLevelWarn {
		log.Printf(prefixWarn+l.prefix+format, args...)
	}
}

// Error logs an error message (red)
func (l *ColorLogger) Error(format string, args ...interface{}) {
	if logLevel <= LogLevelError {
		log.Printf(prefixError+l.prefix+format, args...)
	}
}

// Debug logs a debug message (gray)
func (l *ColorLogger) Debug(format string, args ...interface{}) {
	if logLevel <= LogLevelDebug {
		log.Printf(prefixDebug+l.prefix+format, args...)
	}
}

// Colored print functions for direct use
func printColored(color, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s%s%s\n", color, fmt.Sprintf(format, args...), ColorReset)
}

// PrintInfo prints info in green
func PrintInfo(format string, args ...interface{}) {
	printColored(ColorGreen, format, args...)
}

// PrintWarn prints warning in yellow
func PrintWarn(format string, args ...interface{}) {
	printColored(ColorYellow, format, args...)
}

// PrintError prints error in red
func PrintError(format string, args ...interface{}) {
	printColored(ColorRed, format, args...)
}

// PrintSuccess prints success in green with checkmark
func PrintSuccess(format string, args ...interface{}) {
	printColored(ColorGreen, "✓ "+format, args...)
}

// PrintFail prints failure in red with X
func PrintFail(format string, args ...interface{}) {
	printColored(ColorRed, "✗ "+format, args...)
}
