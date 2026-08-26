// Package config handles proper configuration loading and validation.
package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
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
	Outputs []any `yaml:"outputs,omitempty"`
	// OutputMap allows detailed configuration per format.
	OutputMap map[string]any `yaml:"output,omitempty"`
	// FilenameTemplate for generating output filenames.
	FilenameTemplate string `yaml:"filename-template,omitempty"`
	// SlugifyFilename acts as a tri-state boolean (nil = unset).
	SlugifyFilename *bool `yaml:"slugify-filename,omitempty"`
	// Generic captures all other top-level keys as metadata.
	Generic map[string]any `yaml:",inline"`
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

	yamlData := ExtractFrontmatter(data)

	var cfg Config
	if err := yaml.Unmarshal(yamlData, &cfg); err != nil {
		return absPath, nil, fmt.Errorf("error parsing YAML in '%s': %w", absPath, err)
	}
	return absPath, &cfg, nil
}

// DataDirName returns the data directory for panforge.
// It follows the XDG Base Directory specification:
//  1. $XDG_CONFIG_HOME/panforge
//  2. Windows: %APPDATA%/panforge
//  3. macOS/Linux: ~/.config/panforge
//
// Returns:
//   - string: the path to the data directory
func DataDirName() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "panforge")
	}

	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "panforge")
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: failed to get user home directory: %v", err)
		return ""
	}
	return filepath.Join(home, ".config", "panforge")
}

// LoadDefaultConfig tries to load a default YAML configuration by name or path.
// If name is empty or "default", it checks:
//  1. Current working directory for .panforge.yaml, panforge.yaml, .panforge.yml, panforge.yml
//  2. Data directory (~/.config/panforge/default.yaml or %APPDATA%/panforge/default.yaml)
//
// Parameters:
//   - name: either a direct file path or the name of a config file in the default data directory
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

	// If looking for "default", check local project directory first
	if name == "default" {
		localCandidates := []string{
			".panforge.yaml",
			"panforge.yaml",
			".panforge.yml",
			"panforge.yml",
		}
		for _, candidate := range localCandidates {
			if _, err := os.Stat(candidate); err == nil {
				return LoadConfig(candidate)
			}
		}
	}

	// look in ~/.config/panforge/ (or platform equivalent)
	dataDir := DataDirName()
	if dataDir != "" {
		path := filepath.Join(dataDir, name+".yaml")
		if _, err := os.Stat(path); err == nil {
			return LoadConfig(path)
		}
		pathYml := filepath.Join(dataDir, name+".yml")
		if _, err := os.Stat(pathYml); err == nil {
			return LoadConfig(pathYml)
		}
	}
	return "", &Config{}, nil
}
