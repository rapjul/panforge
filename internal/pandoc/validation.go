package pandoc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
func ValidateMetadata(meta map[string]interface{}, baseDir string) error {
	// Helper to create a validator with baseDir
	makeFileValidator := func(base string) validatorFunc {
		return func(path string) error {
			return validateFile(path, base)
		}
	}
	makeDirValidator := func(base string) validatorFunc {
		return func(path string) error {
			return validateDir(path, base)
		}
	}
	makeParentDirValidator := func(base string) validatorFunc {
		return func(path string) error {
			return validateParentDir(path, base)
		}
	}

	validators := map[string]validatorFunc{
		// Files
		"epub-cover-image":       makeFileValidator(baseDir),
		"css":                    makeFileValidator(baseDir),
		"c":                      makeFileValidator(baseDir),
		"epub-fonts":             makeFileValidator(baseDir),
		"bibliography":           makeFileValidator(baseDir),
		"template":               makeFileValidator(baseDir),
		"reference-doc":          makeFileValidator(baseDir),
		"include-in-header":      makeFileValidator(baseDir),
		"include-before-body":    makeFileValidator(baseDir),
		"include-after-body":     makeFileValidator(baseDir),
		"csl":                    makeFileValidator(baseDir),
		"citation-abbreviations": makeFileValidator(baseDir),

		// Directories
		"data-dir": makeDirValidator(baseDir),

		// Output parent directories
		"output":   makeParentDirValidator(baseDir),
		"o":        makeParentDirValidator(baseDir),
		"log":      makeParentDirValidator(baseDir),
		"log-file": makeParentDirValidator(baseDir),
	}

	for k, v := range meta {
		if validator, ok := validators[k]; ok {
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
func validateGeneric(v interface{}, validator validatorFunc) error {
	switch val := v.(type) {
	case string:
		return validator(val)
	case []interface{}:
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
func validatePathList(v interface{}, baseDir string) error {
	makeValidator := func(base string) validatorFunc {
		return func(path string) error {
			return validateDir(path, base)
		}
	}
	valFunc := makeValidator(baseDir)

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
	case []interface{}:
		return validateGeneric(val, valFunc)
	default:
		return fmt.Errorf("expected string (path list) or list of strings, got %T", v)
	}
	return nil
}
