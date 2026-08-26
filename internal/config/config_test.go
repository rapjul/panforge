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

// TestLoadDefaultConfig_NotFound verifies behavior when a non-existent config name is requested.
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

// TestLoadDefaultConfig_ExplicitPath verifies loading a config file by explicit file path.
func TestLoadDefaultConfig_ExplicitPath(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "explicit_config_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := []byte("title: Explicit Title\n")
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpfile.Close()

	path, cfg, err := LoadDefaultConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadDefaultConfig with path failed: %v", err)
	}
	if path == "" {
		t.Error("Expected non-empty path returned")
	}
	if cfg.Title != "Explicit Title" {
		t.Errorf("cfg.Title = %v, want Explicit Title", cfg.Title)
	}

	// Missing explicit path should return error
	_, _, err = LoadDefaultConfig("./nonexistent_explicit_path_12345.yaml")
	if err == nil {
		t.Error("Expected error for non-existent explicit file path, got nil")
	}
}

// TestLoadDefaultConfig_LocalCandidates verifies loading .panforge.yaml in working directory.
func TestLoadDefaultConfig_LocalCandidates(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "panforge_local_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()
	_ = os.Chdir(tmpDir)

	localConfigFile := ".panforge.yaml"
	content := []byte("title: Local Project Config\n")
	if err := os.WriteFile(localConfigFile, content, 0600); err != nil {
		t.Fatal(err)
	}

	path, cfg, err := LoadDefaultConfig("default")
	if err != nil {
		t.Fatalf("LoadDefaultConfig failed for local file: %v", err)
	}
	if path == "" {
		t.Error("Expected loaded path to be non-empty")
	}
	if cfg.Title != "Local Project Config" {
		t.Errorf("cfg.Title = %v, want Local Project Config", cfg.Title)
	}
}

// TestLoadDefaultConfig_XDGDirectory verifies loading default.yaml from XDG directory.
func TestLoadDefaultConfig_XDGDirectory(t *testing.T) {
	xdgDir, err := os.MkdirTemp("", "panforge_xdg_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(xdgDir) }()

	panforgeDir := filepath.Join(xdgDir, "panforge")
	if err := os.MkdirAll(panforgeDir, 0700); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", xdgDir)

	// Create default.yaml inside XDG panforge dir
	content := []byte("title: XDG Default Title\n")
	if err := os.WriteFile(filepath.Join(panforgeDir, "default.yaml"), content, 0600); err != nil {
		t.Fatal(err)
	}

	path, cfg, err := LoadDefaultConfig("default")
	if err != nil {
		t.Fatalf("LoadDefaultConfig with XDG dir failed: %v", err)
	}
	if path == "" {
		t.Error("Expected loaded path to be non-empty")
	}
	if cfg.Title != "XDG Default Title" {
		t.Errorf("cfg.Title = %v, want XDG Default Title", cfg.Title)
	}
}

// TestLoadConfig_MissingFile verifies LoadConfig returns error on missing file.
func TestLoadConfig_MissingFile(t *testing.T) {
	_, _, err := LoadConfig("/nonexistent/path/to/missing_file.yaml")
	if err == nil {
		t.Error("Expected error when loading missing config file, got nil")
	}
}
