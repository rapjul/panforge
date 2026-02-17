// Package config handles proper configuration loading and validation.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the top-level structure of a YAML file or default config.
// It maps directly to the YAML keys used in configuration files.
type Config struct {
	// Title of the document.
	Title string `yaml:"title,omitempty"`
	// Author of the document.
	Author string `yaml:"author,omitempty"`
	// Outputs list (e.g., ["html", "pdf"]).
	Outputs []interface{} `yaml:"outputs,omitempty"`
	// OutputMap allows detailed configuration per format.
	OutputMap map[string]interface{} `yaml:"output,omitempty"`
	// FilenameTemplate for generating output filenames.
	FilenameTemplate string `yaml:"filename-template,omitempty"`
	// SlugifyFilename acts as a tri-state boolean (nil = unset).
	SlugifyFilename *bool `yaml:"slugify-filename,omitempty"`
	// Generic captures all other top-level keys as metadata.
	Generic map[string]interface{} `yaml:",inline"`
}

// LoadConfig loads the YAML configuration from a file.
//
// Parameters:
//   - `path`: the file path to the configuration file
//
// Returns:
//   - string: the absolute path of the loaded config file
//   - *Config: the parsed configuration struct
//   - error: any error encountered during loading or parsing
func LoadConfig(path string) (string, *Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path // fallback
	}
	//nolint:gosec // G304: Potential file inclusion via variable is intended behavior for CLI file arguments
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", nil, err
	}

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

	var cfg Config
	if err := yaml.Unmarshal(yamlData, &cfg); err != nil {
		return absPath, nil, fmt.Errorf("error parsing YAML in '%s': %w", absPath, err)
	}
	return absPath, &cfg, nil
}

// DataDirName returns the data directory for panforge.
// It checks APPDATA environment variable first, then defaults to ~/.panforge.
func DataDirName() string {
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "panforge")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".panforge")
}

// LoadDefaultConfig tries to load a default YAML configuration by name or path.
//
// Parameters:
//   - `name`: either a direct file path or the name of a config file in the default data directory
//
// Returns:
//   - string: absolute path of the loaded file
//   - *Config: parsed configuration
//   - error: error if file not found or invalid
func LoadDefaultConfig(name string) (string, *Config, error) {
	if name == "" {
		name = "default"
	}

	// check if name is a file path
	if strings.ContainsAny(name, "./\\") {
		if _, err := os.Stat(name); err == nil {
			return LoadConfig(name)
		}
		return "", nil, fmt.Errorf("could not find file %s", name)
	}

	// look in ~/.panforge/
	path := filepath.Join(DataDirName(), name+".yaml")
	if _, err := os.Stat(path); err == nil {
		return LoadConfig(path)
	}
	return "", &Config{}, nil
}
