package app_test

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/rapjul/panforge/internal/app"
	"github.com/rapjul/panforge/internal/options"
)

func TestRun_MissingFileValidation(t *testing.T) {
	// Create a temp file with invalid config (missing file)
	tmpFile, err := os.CreateTemp("", "test-invalid-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	content := `---
title: Test
output:
  epub:
    epub-cover-image: missing-image.jpg
---
# Hello
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Setup
	executor := &TestExecutor{}
	opts := options.Options{
		// Force epub target to trigger validation of epub-cover-image
		Targets: []string{"epub"},
	}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	// mock args with input file
	args := []string{tmpFile.Name()}

	// Run
	err = app.Run(context.Background(), cmd, args, opts, executor)

	// Assert
	if err == nil {
		t.Fatal("Expected app.Run to fail due to missing file, but it succeeded")
	}

	// The error message format depends on ValidateMetadata implementation
	// We expect something like: invalid path for key 'epub-cover-image': file not found: missing-image.jpg
	expectedPart := "invalid path for key 'epub-cover-image'"
	expectedPart2 := "missing-image.jpg"

	if !strings.Contains(err.Error(), expectedPart) || !strings.Contains(err.Error(), expectedPart2) {
		t.Errorf("Expected error to contain %q and %q, got %q", expectedPart, expectedPart2, err.Error())
	}
}
