package app

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/rapjul/panforge/internal/templates"
)

// InitOptions holds flags and arguments for the init command.
type InitOptions struct {
	// Filename specifies the optional destination file path.
	Filename string
	// Config triggers generation of the default configuration file.
	Config bool
	// Markdown triggers generation of a sample Markdown file.
	Markdown bool
	// Force enables overwriting existing files without prompt.
	Force bool
	// Formats is a list of targets to include in the scaffolded Markdown.
	Formats []string
}

// KnownFormats are the formats supported by the scaffold generator.
var KnownFormats = []string{DefaultFormat, "pdf", "epub", "docx"}

// RunInit executes the init command logic.
//
// Parameters:
//   - opts: the initialization options containing flags and settings
//
// Returns:
//   - error: an error if the initialization fails
func RunInit(opts InitOptions) error {
	if opts.Markdown {
		return createScaffold(opts)
	}

	// Default to config if no specific type selected, or if --config is explicit
	// We'll create it in the current directory as .panforge.yaml (or specified filename)
	return createConfig(opts)
}

// createConfig generates a default configuration file.
//
// Parameters:
//   - opts: the initialization options containing flags (e.g. Force, Filename)
//
// Returns:
//   - error: an error if the config file cannot be created
func createConfig(opts InitOptions) error {
	content, err := templates.GetConfigTemplate()
	if err != nil {
		return fmt.Errorf("failed to load config template: %w", err)
	}
	targetFile := opts.Filename
	if targetFile == "" {
		targetFile = ".panforge.yaml"
	}
	return createFile(targetFile, content, opts.Force)
}

// createScaffold generates a sample Markdown input file.
//
// Parameters:
//   - opts: the initialization options containing formatting preferences and optional filename
//
// Returns:
//   - error: an error if the scaffold file cannot be created
func createScaffold(opts InitOptions) error {
	tmplContent, err := templates.GetScaffoldTemplate()
	if err != nil {
		return fmt.Errorf("failed to load scaffold template: %w", err)
	}

	tmpl, err := template.New("scaffold").Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("failed to parse scaffold template: %w", err)
	}

	data := struct {
		Formats []string
	}{
		Formats: opts.Formats,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	targetFile := opts.Filename
	if targetFile == "" {
		targetFile = "input.md"
	}
	return createFile(targetFile, buf.String(), opts.Force)
}

// createFile writes content to a file.
//
// Parameters:
//   - filename: the name of the file to create
//   - content: the string content to write
//   - force: determines if existing files should be overwritten
//
// Returns:
//   - error: an error if the file cannot be written
func createFile(filename string, content string, force bool) error {
	// Check if file exists
	if _, err := os.Stat(filename); err == nil {
		if !force {
			return fmt.Errorf("file '%s' already exists (use --force to overwrite)", filename)
		}
	}

	// Write config file
	//nolint:gosec // G306: Expect WriteFile permissions to be 0600 or less (config file should be readable)
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	absPath, _ := filepath.Abs(filename)
	fmt.Printf("Created %s at %s\n", filename, absPath)
	return nil
}
