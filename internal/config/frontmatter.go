package config

import (
	"strings"
)

// ExtractFrontmatter separates YAML frontmatter from the rest of the file content.
// It looks for a leading "---", followed by a closing "---" or "..." on a new line.
// If valid frontmatter is found, it returns the YAML byte slice.
// Otherwise, it returns the original data (treating the whole file as YAML).
func ExtractFrontmatter(data []byte) []byte {
	// Helper to check for frontmatter
	// We check if the file starts with ---
	// If so, we look for the closing delimiter (--- or ...)
	var yamlData = data

	hasFrontmatter := false
	if len(data) >= 3 && string(data[:3]) == "---" {
		// It starts with dash, check if it is followed by newline
		if len(data) > 3 {
			b := data[3]
			if b == '\n' || b == '\r' {
				hasFrontmatter = true
			}
		}
	}

	if hasFrontmatter {
		// Look for closing delimiter defined as \n--- or \n...
		// explicit check for newlines to ensure it is on a new line
		// We'll search from index 3
		remaining := data[3:]

		delims := []string{"\n---", "\r\n---", "\n...", "\r\n..."}

		// Find the earliest occurrence of any delimiter
		firstPos := len(remaining) + 100 // larger than any valid index
		found := false

		s := string(remaining)
		for _, delim := range delims {
			idx := strings.Index(s, delim)
			if idx != -1 {
				// Check content after delimiter to ensure it's a full line (or EOF)
				after := idx + len(delim)
				isEnd := false
				if after >= len(s) {
					isEnd = true
				} else {
					c := s[after]
					if c == '\n' || c == '\r' || c == ' ' {
						isEnd = true
					}
				}

				if isEnd {
					if idx < firstPos {
						firstPos = idx
						found = true
					}
				}
			}
		}

		if found {
			// slicing data[:3+firstPos] gets us the context *before* the closing delimiter.
			// This is valid YAML as it acts as a stream with one document.
			yamlData = data[:3+firstPos]
		}
	}
	return yamlData
}
