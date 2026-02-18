package pandoc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// pathType defines the type of validation required for a path key.
type pathType int

const (
	ptFile pathType = iota
	ptDir
	ptParentDir
)

// KnownPathKeys maps metadata keys to their expected path type.
var KnownPathKeys = map[string]pathType{
	// Files
	"epub-cover-image":       ptFile,
	"css":                    ptFile,
	"c":                      ptFile,
	"epub-fonts":             ptFile,
	"bibliography":           ptFile,
	"template":               ptFile,
	"reference-doc":          ptFile,
	"include-in-header":      ptFile,
	"include-before-body":    ptFile,
	"include-after-body":     ptFile,
	"csl":                    ptFile,
	"citation-abbreviations": ptFile,

	// Directories
	"data-dir": ptDir,

	// Output parent directories
	"output": ptParentDir,

	"log":      ptParentDir,
	"log-file": ptParentDir,
}

// IgnoredMetadataKeys is a set of keys that should NOT be passed as flags to Pandoc.
// These are typically standard metadata keys that do not have corresponding CLI flags.
// Note: 'title' and 'author' are handled by the Config struct and usually don't end up in Generic,
// but we include them for safety.
var IgnoredMetadataKeys = map[string]bool{
	"title":       true,
	"author":      true,
	"date":        true,
	"subtitle":    true,
	"abstract":    true,
	"keywords":    true,
	"description": true,
	"creator":     true,
	"lang":        true,
	"subject":     true,
	"identifier":  true,
	"publisher":   true,
	"contributor": true,
	"coverage":    true,
	"rights":      true,
	"source":      true,
	"relation":    true,
	"type":        true,
	"format":      true,
	"license":     true,
	// Add other common metadata keys here
}

// ValidateMetadata checks that file paths in the metadata exist.
// It skips URLs and handles both single strings and lists of strings.
// baseDir is used to resolve relative paths. Absolute paths ignore baseDir.
//
// Parameters:
//   - meta: the metadata map to validate
//   - baseDir: the base directory for resolving relative paths
//
// Returns:
//   - error: an error if validation fails, nil otherwise
func ValidateMetadata(meta map[string]any, baseDir string) error {
	for k, v := range meta {
		if pt, ok := KnownPathKeys[k]; ok {
			var validator validatorFunc
			switch pt {
			case ptFile:
				validator = func(p string) error { return validateFile(p, baseDir) }
			case ptDir:
				validator = func(p string) error { return validateDir(p, baseDir) }
			case ptParentDir:
				validator = func(p string) error { return validateParentDir(p, baseDir) }
			default:
				return fmt.Errorf("unknown path type for key '%s'", k)
			}

			if err := validateGeneric(v, validator); err != nil {
				return fmt.Errorf("invalid path for key '%s': %w", k, err)
			}
		}

		// Check path lists (resource-path) separately as it has unique parsing logic
		if k == "resource-path" {
			if err := validatePathList(v, baseDir); err != nil {
				return fmt.Errorf("invalid resource path for key '%s': %w", k, err)
			}
		}
	}
	return nil
}

type validatorFunc func(string) error

// validateGeneric handles both string and list of strings for a given validator
//
// Parameters:
//   - v: the value to validate
//   - validator: the validator function to use
//
// Returns:
//   - error: an error if the value is not a string or list of strings
func validateGeneric(v any, validator validatorFunc) error {
	switch val := v.(type) {
	case string:
		return validator(val)
	case []any:
		for _, item := range val {
			s, ok := item.(string)
			if !ok {
				return fmt.Errorf("expected string in list, got %T", item)
			}
			if err := validator(s); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("expected string or list of strings, got %T", v)
	}
	return nil
}

// resolvePath resolves a path relative to the base directory.
//
// If the path is absolute, it returns the path as is.
// If the path is relative, it returns the path joined with the base directory.
// If the base directory is empty, it returns the path as is.
//
// Parameters:
//   - path: the path to resolve
//   - baseDir: the base directory for resolving relative paths
//
// Returns:
//   - string: the resolved path
func resolvePath(path, baseDir string) string {
	if baseDir != "" && !filepath.IsAbs(path) {
		return filepath.Join(baseDir, path)
	}
	return path
}

// validateFile checks if a file exists and is a regular file.
//
// Parameters:
//   - path: the path to the file to validate
//   - baseDir: the base directory for resolving relative paths
//
// Returns:
//   - error: an error if the file does not exist or is not a regular file
func validateFile(path, baseDir string) error {
	// Skip URLs
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return nil
	}

	fullPath := resolvePath(path, baseDir)
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", fullPath)
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("expected file but found directory: %s", fullPath)
	}
	return nil
}

// validateDir checks if a directory exists and is a directory.
//
// Parameters:
//   - path: the path to the directory to validate
//   - baseDir: the base directory for resolving relative paths
//
// Returns:
//   - error: an error if the directory does not exist or is not a directory
func validateDir(path, baseDir string) error {
	fullPath := resolvePath(path, baseDir)
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory not found: %s", fullPath)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("expected directory but found file: %s", fullPath)
	}
	return nil
}

// validateParentDir checks if the parent directory of a path exists and is a directory.
//
// Parameters:
//   - path: the path to the directory to validate
//   - baseDir: the base directory for resolving relative paths
//
// Returns:
//   - error: an error if the parent directory does not exist or is not a directory
func validateParentDir(path, baseDir string) error {
	fullPath := resolvePath(path, baseDir)
	dir := filepath.Dir(fullPath)
	if dir == "." {
		return nil
	}
	return validateDir(dir, "")
}

// validatePathList checks if a list of paths are valid.
//
// Parameters:
//   - v: the list of paths to validate
//   - baseDir: the base directory for resolving relative paths
//
// Returns:
//   - error: an error if any path in the list is invalid
func validatePathList(v any, baseDir string) error {
	valFunc := func(p string) error { return validateDir(p, baseDir) }

	switch val := v.(type) {
	case string:
		// Split by list separator (colon on UNIX, semicolon on Windows)
		parts := filepath.SplitList(val)
		for _, p := range parts {
			if p == "" {
				continue
			}
			if err := valFunc(p); err != nil {
				return err
			}
		}
	case []any:
		return validateGeneric(val, valFunc)
	default:
		return fmt.Errorf("expected string (path list) or list of strings, got %T", v)
	}
	return nil
}

// ResolveMetadataPaths resolves relative paths in metadata to absolute paths based on baseDir.
// It returns a copy of the metadata with resolved paths.
//
// Parameters:
//   - meta: the metadata map to resolve
//   - baseDir: the base directory for resolving relative paths
//
// Returns:
//   - map[string]interface{}: a new metadata map with resolved paths
func ResolveMetadataPaths(meta map[string]any, baseDir string) map[string]any {
	resolved := make(map[string]any)
	for k, v := range meta {
		resolved[k] = v
	}

	for k, v := range meta {
		if _, ok := KnownPathKeys[k]; ok {
			resolved[k] = resolveGenericValue(v, baseDir)
		} else if k == "resource-path" {
			resolved[k] = resolveResourcePathValue(v, baseDir)
		}
	}

	return resolved
}

// resolveSinglePath is a helper to resolve a single path string.
func resolveSinglePath(path, baseDir string) string {
	// Skip URLs
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}

// resolveGenericValue resolves paths in a string or list of strings.
func resolveGenericValue(v any, baseDir string) any {
	if s, ok := v.(string); ok {
		return resolveSinglePath(s, baseDir)
	}
	if list, ok := v.([]any); ok {
		var resolvedList []any
		for _, item := range list {
			if s, ok := item.(string); ok {
				resolvedList = append(resolvedList, resolveSinglePath(s, baseDir))
			} else {
				resolvedList = append(resolvedList, item)
			}
		}
		return resolvedList
	}
	return v
}

// resolveResourcePathValue handles the special case of resource-path which can be
// a colon-separated string or a list.
func resolveResourcePathValue(v any, baseDir string) any {
	if s, ok := v.(string); ok {
		parts := filepath.SplitList(s)
		var resolvedParts []string
		for _, p := range parts {
			resolvedParts = append(resolvedParts, resolveSinglePath(p, baseDir))
		}
		return strings.Join(resolvedParts, string(os.PathListSeparator))
	}
	// If it's a list, it acts like a generic list of paths
	return resolveGenericValue(v, baseDir)
}
