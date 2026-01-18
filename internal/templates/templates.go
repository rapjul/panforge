package templates

import (
	"embed"
	"io/fs"
)

//go:embed files/*
var files embed.FS

// GetConfigTemplate returns the default configuration template content
func GetConfigTemplate() (string, error) {
	return readFile("files/default.yaml")
}

// GetScaffoldTemplate returns the scaffold markdown template content
func GetScaffoldTemplate() (string, error) {
	return readFile("files/scaffold.md")
}

// readFile reads a file from the embedded filesystem.
//
// Parameters:
//   - `name`: the path to the file within the embedded system
func readFile(name string) (string, error) {
	data, err := fs.ReadFile(files, name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
