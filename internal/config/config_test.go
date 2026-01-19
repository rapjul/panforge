package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temp file
	content := []byte(`
title: Test Doc
author: Tester
output:
  html:
    standalone: true
`)
	tmpfile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }() // clean up

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	path, cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if path != tmpfile.Name() {
		// On some systems temp path might be symlinked, but usually it matches
		// Just check it's not empty
		if path == "" {
			t.Error("LoadConfig returned empty path")
		}
	}

	if cfg.Title != "Test Doc" {
		t.Errorf("Title = %v, want %v", cfg.Title, "Test Doc")
	}
	if cfg.Author != "Tester" {
		t.Errorf("Author = %v, want %v", cfg.Author, "Tester")
	}

	// check output map
	if val, ok := cfg.OutputMap["html"]; !ok {
		t.Error("OutputMap missing html key")
	} else {
		m, ok := val.(map[string]interface{})
		if !ok {
			t.Error("html value is not a map")
		}
		if m["standalone"] != true {
			t.Errorf("html.standalone = %v, want true", m["standalone"])
		}
	}
}

func TestLoadDefaultConfig_NotFound(t *testing.T) {
	// Should return empty config and no error if not found (per implementation)
	_, cfg, err := LoadDefaultConfig("nonexistent_config_file_12345")
	if err != nil {
		t.Errorf("LoadDefaultConfig returned error for missing file: %v", err)
	}
	if cfg == nil {
		t.Error("LoadDefaultConfig returned nil config")
	}
}
