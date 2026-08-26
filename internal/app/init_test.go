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
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origWd) }()

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
		// Create input file
		_ = os.WriteFile("input.md", []byte("exists"), 0600)

		err := RunInit(InitOptions{Markdown: true})
		if err == nil {
			t.Error("expected error when file exists, got nil")
		}
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			t.Errorf("expected 'already exists' error, got %v", err)
		}
	})

	t.Run("OverwriteExistingFiles", func(t *testing.T) {
		// Create input file
		_ = os.WriteFile("input.md", []byte("exists"), 0600)

		err := RunInit(InitOptions{Markdown: true, Force: true})
		if err != nil {
			t.Errorf("expected no error with force, got %v", err)
		}

		content, _ := os.ReadFile("input.md")
		if !strings.Contains(string(content), "title: \"Untitled Document\"") {
			t.Errorf("file was not overwritten")
		}
	})

	t.Run("CustomFilename_Scaffold", func(t *testing.T) {
		err := RunInit(InitOptions{Markdown: true, Filename: "custom-doc.md"})
		if err != nil {
			t.Fatalf("RunInit failed with custom filename: %v", err)
		}

		content, err := os.ReadFile("custom-doc.md")
		if err != nil {
			t.Fatalf("custom-doc.md not created")
		}
		if !strings.Contains(string(content), "title: \"Untitled Document\"") {
			t.Errorf("unexpected content in custom scaffold file")
		}
	})

	t.Run("CustomFilename_Config", func(t *testing.T) {
		err := RunInit(InitOptions{Config: true, Filename: "custom-config.yaml"})
		if err != nil {
			t.Fatalf("RunInit failed with custom config filename: %v", err)
		}

		content, err := os.ReadFile("custom-config.yaml")
		if err != nil {
			t.Fatalf("custom-config.yaml not created")
		}
		if !strings.Contains(string(content), "Default Configuration for panforge") {
			t.Errorf("unexpected content in custom config file")
		}
	})
}
