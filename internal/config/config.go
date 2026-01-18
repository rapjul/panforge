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
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
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
