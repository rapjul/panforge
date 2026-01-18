package utils

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var slugRegex = regexp.MustCompile("[^a-z0-9]+")

// ResolvePath returns the absolute path for a given filename.
//
// Parameters:
//   - `path`: the input file path (relative or absolute)
//
// Returns:
//   - string: the absolute path
//   - error: any error encountered
func ResolvePath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path %s: %w", path, err)
	}
	return absPath, nil
}

// Slugify converts a title to a safe filename (simple ASCII fallback).
//
// Parameters:
//   - `title`: the string to slugify
//
// Returns:
//   - string: the slugified string
func Slugify(title string) string {
	if title == "" {
		return ""
	}

	s := strings.ToLower(strings.TrimSpace(title))
	// replace non-alphanum with dashes
	s = slugRegex.ReplaceAllString(s, "-")
	// remove leading/trailing dashes
	s = strings.Trim(s, "-")

	if s == "" {
		return ""
	}
	return s
}

// SanitizeFilename replaces dangerous characters with underscores.
// It preserves case and spaces but handles OS-specific constraints.
//
// Parameters:
//   - `name`: the filename to sanitize
//
// Returns:
//   - string: the sanitized filename
func SanitizeFilename(name string) string {
	return sanitize(name, runtime.GOOS)
}

// sanitize performs OS-specific sanitization.
//
// Parameters:
//   - `name`: the filename to sanitize
//   - `osName`: the operating system name (e.g., "windows", "darwin")
func sanitize(name, osName string) string {
	if name == "" {
		return ""
	}

	s := strings.TrimSpace(name)

	// universally unsafe for file paths
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")

	// macOS: colon is the path separator in Finder (HFS+ legacy but still visible in some contexts), usually displayed as / in terminal but treated as separator in UI sometimes.
	// But fundamentally on Unix, / is separator. : is allowed but often troublesome.
	if osName == "darwin" {
		s = strings.ReplaceAll(s, ":", "_")
	}

	// Windows sanitization: < > : " / \ | ? *, and reserved names
	if osName == "windows" {
		badChars := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
		for _, b := range badChars {
			s = strings.ReplaceAll(s, b, "_")
		}

		// Reserved names (check uppercase)
		badNames := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
		upperS := strings.ToUpper(s)
		for _, b := range badNames {
			if upperS == b {
				s = "_"
				break
			}
		}
	}

	return s
}

// FormatDate returns the current date in YYYY-MM-DD format.
func FormatDate() string {
	return time.Now().Format("2006-01-02")
}
