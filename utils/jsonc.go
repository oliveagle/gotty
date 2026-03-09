package utils

import (
	"bufio"
	"bytes"
	"strings"
)

// StripJSONCComments removes comments from JSONC content
// Supports:
// - Single line comments: // comment
// - Multi-line comments: /* comment */
// - Trailing comments: "key": "value" // comment
func StripJSONCComments(content []byte) []byte {
	var result bytes.Buffer
	reader := bytes.NewReader(content)
	scanner := bufio.NewReader(reader)

	inString := false
	inSingleComment := false
	inMultiComment :=
		false

	for {
		b, err := scanner.ReadByte()
		if err != nil {
			break
		}

		// Handle string literals (don't process comments inside strings)
		if !inSingleComment && !inMultiComment {
			if b == '"' && !inString {
				inString = true
				result.WriteByte(b)
				continue
			}
			if b == '"' && inString {
				inString = false
				result.WriteByte(b)
				continue
			}
			if inString {
				result.WriteByte(b)
				// Handle escaped quotes
				if b == '\\' {
					next, err := scanner.ReadByte()
					if err == nil {
						result.WriteByte(next)
					}
				}
				continue
			}
		}

		// Handle single-line comments
		if !inMultiComment && b == '/' {
			next, err := scanner.ReadByte()
			if err != nil {
				result.WriteByte(b)
				break
			}
			if next == '/' && !inSingleComment {
				inSingleComment = true
				continue
			}
			if next == '*' && !inSingleComment {
				inMultiComment = true
				continue
			}
			// Not a comment, write both bytes
			result.WriteByte(b)
			result.WriteByte(next)
			continue
		}

		// End of single-line comment
		if inSingleComment && (b == '\n' || b == '\r') {
			inSingleComment = false
			result.WriteByte(b)
			continue
		}

		// Inside single-line comment, skip
		if inSingleComment {
			continue
		}

		// End of multi-line comment
		if inMultiComment && b == '*' {
			next, err := scanner.ReadByte()
			if err != nil {
				break
			}
			if next == '/' {
				inMultiComment = false
				continue
			}
			// Not end of comment, continue skipping
			continue
		}

		// Inside multi-line comment, skip
		if inMultiComment {
			continue
		}

		result.WriteByte(b)
	}

	return result.Bytes()
}

// isJSONCFile checks if the file is a JSONC file based on extension
func isJSONCFile(filePath string) bool {
	return strings.HasSuffix(strings.ToLower(filePath), ".jsonc")
}

// isJSONFile checks if the file is a JSON file based on extension
func isJSONFile(filePath string) bool {
	return strings.HasSuffix(strings.ToLower(filePath), ".json")
}
