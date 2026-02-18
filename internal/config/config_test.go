package config

import (
	"os"
	"path/filepath"
	"runtime"
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
		m, ok := val.(map[string]any)
		if !ok {
			t.Error("html value is not a map")
		}
		if m["standalone"] != true {
			t.Errorf("html.standalone = %v, want true", m["standalone"])
		}
	}
}

func TestDataDirName(t *testing.T) {
	// 1. Test XDG_CONFIG_HOME
	xdgDir := "/tmp/xdg_test_config"
	t.Setenv("XDG_CONFIG_HOME", xdgDir)

	expected := filepath.Join(xdgDir, "panforge")
	if got := DataDirName(); got != expected {
		t.Errorf("DataDirName() with XDG_CONFIG_HOME = %v, want %v", got, expected)
	}

	// 2. Test Default Fallback (unset XDG_CONFIG_HOME)
	t.Setenv("XDG_CONFIG_HOME", "")

	// For Windows, we'd need to unset APPDATA too, but let's see current behavior.
	// We can't easily mock runtime.GOOS, so we test behavior on current platform.
	got := DataDirName()

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		expected = filepath.Join(appData, "panforge")
	} else {
		home, _ := os.UserHomeDir()
		expected = filepath.Join(home, ".config", "panforge")
	}

	if got != expected {
		t.Errorf("DataDirName() default = %v, want %v", got, expected)
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
