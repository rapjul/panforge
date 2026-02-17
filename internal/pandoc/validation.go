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
func ValidateMetadata(meta map[string]interface{}, baseDir string) error {
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

	for k, v := range meta {
		// Check files
		for _, key := range files {
			if k == key {
				if err := validateGeneric(v, makeFileValidator(baseDir)); err != nil {
					return fmt.Errorf("invalid path for key '%s': %w", k, err)
				}
			}
		}

		// Check dirs
		for _, key := range dirs {
			if k == key {
				if err := validateGeneric(v, makeDirValidator(baseDir)); err != nil {
					return fmt.Errorf("invalid directory for key '%s': %w", k, err)
				}
			}
		}

		// Check path lists
		for _, key := range pathLists {
			if k == key {
				if err := validatePathList(v, baseDir); err != nil {
					return fmt.Errorf("invalid resource path for key '%s': %w", k, err)
				}
			}
		}

		// Check parent dirs
		for _, key := range parentDirs {
			if k == key {
				if err := validateGeneric(v, makeParentDirValidator(baseDir)); err != nil {
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

func resolvePath(path, baseDir string) string {
	if baseDir != "" && !filepath.IsAbs(path) {
		return filepath.Join(baseDir, path)
	}
	return path
}

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

func validateParentDir(path, baseDir string) error {
	fullPath := resolvePath(path, baseDir)
	dir := filepath.Dir(fullPath)
	if dir == "." {
		return nil
	}
	return validateDir(dir, "")
}

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
	}
	return nil
}
