package main_test

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/rapjul/panforge/internal/app"
	"github.com/rapjul/panforge/internal/options"
)

func TestCLI_NoHeader_NoFlags(t *testing.T) {
	inputFile := "test-files/markdown-only-input.md"
	cmd := &cobra.Command{}
	opts := options.Options{
		Targets: []string{}, // Empty targets
	}
	args := []string{inputFile}
	executor := &app.RealExecutor{DryRun: true} // DryRun to avoid actual execution if it accidentally proceeds

	err := app.Run(context.Background(), cmd, args, opts, executor)
	if err == nil {
		t.Fatal("Expected error due to missing YAML header and no CLI targets, but got nil")
	}

	expectedMsg := "input file has no valid YAML header and no target format specified"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message containing '%s', got '%v'", expectedMsg, err)
	}
}

func TestCLI_NoHeader_WithFlags(t *testing.T) {
	inputFile := "test-files/markdown-only-input.md"
	cmd := &cobra.Command{}
	opts := options.Options{
		Targets: []string{"html"}, // CLI target specified
	}
	args := []string{inputFile}

	// mock executor? RealExecutor with DryRun is safer for now,
	// assuming app.Run doesn't do much else destructive.
	// But app.Run might try to actually run pandoc.
	// RealExecutor.Run checks DryRun and returns nil.
	executor := &app.RealExecutor{DryRun: true}

	err := app.Run(context.Background(), cmd, args, opts, executor)
	if err != nil {
		t.Fatalf("Expected no error when CLI target is specified, but got: %v", err)
	}
}
