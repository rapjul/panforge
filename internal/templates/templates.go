// Package templates manages embedded resources and file generation.
package templates

import (
	"embed"
	"io/fs"
)

//go:embed files/*
var files embed.FS

// GetConfigTemplate returns the default configuration template content
//
// Returns:
//   - string: the content of the default configuration template
//   - error: an error if the default configuration template cannot be read
func GetConfigTemplate() (string, error) {
	return readFile("files/default.yaml")
}

// GetScaffoldTemplate returns the scaffold markdown template content
//
// Returns:
//   - string: the content of the scaffold template
//   - error: an error if the scaffold template cannot be read
func GetScaffoldTemplate() (string, error) {
	return readFile("files/scaffold.md")
}

// readFile reads a file from the embedded filesystem.
//
// Parameters:
//   - name: the path to the file within the embedded system
//
// Returns:
//   - string: the content of the file
//   - error: an error if the file cannot be read
func readFile(name string) (string, error) {
	data, err := fs.ReadFile(files, name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
