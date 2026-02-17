package pandoc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateMetadata checks that file paths in the metadata exist.
// It skips URLs and handles both single strings and lists of strings.
func ValidateMetadata(meta map[string]interface{}) error {
	// Map keys to their validation logic
	// ValidatorFile: inputs that must be files (or URLs if supported)
	files := []string{
		"epub-cover-image", "css", "c", "epub-fonts", "bibliography",
		"template", "reference-doc", "include-in-header",
		"include-before-body", "include-after-body",
		"csl", "citation-abbreviations",
	}

	// ValidatorDir: inputs that must be directories
	dirs := []string{"data-dir"}

	// ValidatorPathList: inputs that can be list of paths or colon-separated string
	pathLists := []string{"resource-path"}

	// ValidatorParentDir: output files where parent must exist
	parentDirs := []string{"output", "o", "log", "log-file"}

	for k, v := range meta {
		// Check files
		for _, key := range files {
			if k == key {
				if err := validateGeneric(v, validateFile); err != nil {
					return fmt.Errorf("invalid path for key '%s': %w", k, err)
				}
			}
		}

		// Check dirs
		for _, key := range dirs {
			if k == key {
				if err := validateGeneric(v, validateDir); err != nil {
					return fmt.Errorf("invalid directory for key '%s': %w", k, err)
				}
			}
		}

		// Check path lists
		for _, key := range pathLists {
			if k == key {
				if err := validatePathList(v); err != nil {
					return fmt.Errorf("invalid resource path for key '%s': %w", k, err)
				}
			}
		}

		// Check parent dirs
		for _, key := range parentDirs {
			if k == key {
				if err := validateGeneric(v, validateParentDir); err != nil {
					return fmt.Errorf("invalid output path for key '%s': %w", k, err)
				}
			}
		}
	}
	return nil
}

type validatorFunc func(string) error

// validateGeneric handles both string and list of strings for a given validator
func validateGeneric(v interface{}, validator validatorFunc) error {
	switch val := v.(type) {
	case string:
		return validator(val)
	case []interface{}:
		for _, item := range val {
			if s, ok := item.(string); ok {
				if err := validator(s); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateFile(path string) error {
	// Skip URLs
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return nil
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("expected file but found directory: %s", path)
	}
	return nil
}

func validateDir(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory not found: %s", path)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("expected directory but found file: %s", path)
	}
	return nil
}

func validateParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." {
		return nil
	}
	return validateDir(dir)
}

func validatePathList(v interface{}) error {
	switch val := v.(type) {
	case string:
		// Split by list separator (colon on UNIX, semicolon on Windows)
		parts := filepath.SplitList(val)
		for _, p := range parts {
			if p == "" {
				continue
			}
			if err := validateDir(p); err != nil {
				return err
			}
		}
	case []interface{}:
		return validateGeneric(val, validateDir)
	}
	return nil
}
