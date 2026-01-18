package app

import (
	"os"
	"strings"
	"testing"
)

func TestRunInit(t *testing.T) {
	// Setup temp dir
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	t.Run("GenerateConfig", func(t *testing.T) {
		err := RunInit(InitOptions{Config: true})
		if err != nil {
			t.Fatalf("RunInit failed: %v", err)
		}

		content, err := os.ReadFile(".panforge.yaml")
		if err != nil {
			t.Fatalf("config file not created")
		}
		if !strings.Contains(string(content), "Default Configuration for panforge") {
			t.Errorf("unexpected content in config file")
		}
	})

	t.Run("GenerateScaffold", func(t *testing.T) {
		err := RunInit(InitOptions{Markdown: true})
		if err != nil {
			t.Fatalf("RunInit failed: %v", err)
		}

		content, err := os.ReadFile("input.md")
		if err != nil {
			t.Fatalf("input.md not created")
		}
		if !strings.Contains(string(content), "title: \"Untitled Document\"") {
			t.Errorf("unexpected content in scaffold file")
		}
	})

	t.Run("FileExistsError", func(t *testing.T) {
		// Create file first
		os.WriteFile("input.md", []byte("exists"), 0644)

		err := RunInit(InitOptions{Markdown: true})
		if err == nil {
			t.Error("expected error when file exists, got nil")
		}
	})

	t.Run("FileExistsForce", func(t *testing.T) {
		// Create file first
		os.WriteFile("input.md", []byte("exists"), 0644)

		err := RunInit(InitOptions{Markdown: true, Force: true})
		if err != nil {
			t.Errorf("expected no error with force, got %v", err)
		}

		content, _ := os.ReadFile("input.md")
		if !strings.Contains(string(content), "title: \"Untitled Document\"") {
			t.Errorf("file was not overwritten")
		}
	})
}
