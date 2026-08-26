package main_test

import (
	"context"
	"os"
	"testing"

	"github.com/spf13/cobra"

	"github.com/rapjul/panforge/internal/app"
	"github.com/rapjul/panforge/internal/options"
)

// TestCLI_GoodInput verifies live pandoc conversion for all configured formats in frontmatter.
func TestCLI_GoodInput(t *testing.T) {
	inputFile := "test-files/good-test-input.md"

	// Files expected to be generated in CWD (which is ./test package dir during testing)
	expectedFiles := []string{
		"test-files/My Document.epub",
		"test-files/test-output.html",
		"test-files/test.pdf",
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
	opts := options.Options{}

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

// TestCLI_BatchProcessing verifies live pandoc batch conversion of multiple Markdown files.
func TestCLI_BatchProcessing(t *testing.T) {
	doc1 := "test-files/good-test-input.md"
	doc2 := "test-files/markdown-only-input.md"

	expectedFiles := []string{
		"test-files/test-output.html",
		"test-files/markdown-only-input_2026-08-26.html", // Or generated name
	}

	for _, f := range expectedFiles {
		_ = os.Remove(f)
	}
	defer func() {
		for _, f := range expectedFiles {
			_ = os.Remove(f)
		}
	}()

	cmd := &cobra.Command{}
	opts := options.Options{
		Targets: []string{"html"},
		Force:   true,
	}

	args := []string{doc1, doc2}
	executor := &app.RealExecutor{}
	if err := app.Run(context.Background(), cmd, args, opts, executor); err != nil {
		t.Fatalf("app.Run batch execution failed: %v", err)
	}

	// Verify at least doc1 output exists
	if _, err := os.Stat("test-files/test-output.html"); os.IsNotExist(err) {
		t.Errorf("Expected output file 'test-files/test-output.html' was not created")
	}
}

// TestCLI_PandocPassthroughFlags verifies live passthrough of flags to pandoc via --.
func TestCLI_PandocPassthroughFlags(t *testing.T) {
	inputFile := "test-files/good-test-input.md"
	outputFile := "test-passthrough.html"
	expectedPath := "test-files/test-passthrough.html"

	_ = os.Remove(expectedPath)
	defer func() { _ = os.Remove(expectedPath) }()

	cmd := &cobra.Command{}
	opts := options.Options{
		Targets: []string{"html"},
		Output:  outputFile,
		Force:   true,
	}

	// Pass arbitrary pandoc flags like --toc and --shift-heading-level-by
	args := []string{inputFile, "--toc", "--shift-heading-level-by=1"}
	executor := &app.RealExecutor{}
	if err := app.Run(context.Background(), cmd, args, opts, executor); err != nil {
		t.Fatalf("app.Run with passthrough flags failed: %v", err)
	}

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected output file '%s' was not created", expectedPath)
	}
}
