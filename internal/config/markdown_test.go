package config

import (
	"os"
	"testing"
)

func TestLoadConfig_MarkdownFrontmatter(t *testing.T) {
	// Markdown file with YAML frontmatter + content that is invalid YAML
	content := []byte(`---
title: My Story
date: 2023-01-01
---

## Chapter 1

**Start of story**
`)
	tmpfile, err := os.CreateTemp("", "repro_test_*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	_, cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed on markdown file: %v", err)
	}

	if cfg.Title != "My Story" {
		t.Errorf("Title = %v, want %v", cfg.Title, "My Story")
	}
}
