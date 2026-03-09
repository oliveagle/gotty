package utils

import (
	"bytes"
)

// StripJSONCComments removes comments from JSONC content
// Supports:
// - Single line comments: // comment
// - Multi-line comments: /* comment */
func StripJSONCComments(content []byte) []byte {
	var result bytes.Buffer
	i := 0
	n := len(content)
	inString := false

	for i < n {
		c := content[i]

		// Handle string literals
		if inString {
			result.WriteByte(c)
			if c == '\\' && i+1 < n {
				// Escape sequence, write next char too
				i++
				result.WriteByte(content[i])
			} else if c == '"' {
				inString = false
			}
			i++
			continue
		}

		// Check for string start
		if c == '"' {
			result.WriteByte(c)
			inString = true
			i++
			continue
		}

		// Check for single-line comment
		if c == '/' && i+1 < n && content[i+1] == '/' {
			// Skip until end of line
			i += 2
			for i < n && content[i] != '\n' && content[i] != '\r' {
				i++
			}
			continue
		}

		// Check for multi-line comment
		if c == '/' && i+1 < n && content[i+1] == '*' {
			i += 2
			for i < n {
				if content[i] == '*' && i+1 < n && content[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}

		// Regular character
		result.WriteByte(c)
		i++
	}

	return result.Bytes()
}
