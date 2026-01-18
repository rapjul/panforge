package main_test

import (
	"context"
	"os"
	"testing"

	"github.com/spf13/cobra"

	"github.com/rapjul/panforge/internal/app"
	"github.com/rapjul/panforge/internal/options"
)

func TestCLI_GoodInput(t *testing.T) {
	inputFile := "test-files/good-test-input.md"

	// Files expected to be generated in CWD (which is ./test package dir during testing)
	expectedFiles := []string{
		"My Document.epub",
		"test-output.html",
		"test.pdf",
	}

	// Cleanup before execution
	for _, f := range expectedFiles {
		_ = os.Remove(f)
	}

	// Cleanup after execution
	defer func() {
		for _, f := range expectedFiles {
			_ = os.Remove(f)
		}
	}()

	// Mock command and options
	cmd := &cobra.Command{}
	opts := options.Options{
		// Default options (false/empty)
		// We want real execution, so no dry-run
	}

	// Arguments passed to Run (non-flag args)
	args := []string{inputFile}

	executor := &app.RealExecutor{}
	if err := app.Run(context.Background(), cmd, args, opts, executor); err != nil {
		t.Fatalf("app.Run failed: %v", err)
	}

	// ASSERT
	for _, f := range expectedFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("Expected output file '%s' was not created", f)
		} else if err != nil {
			t.Errorf("Error checking file '%s': %v", f, err)
		}
	}
}
