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

// MockExecutor allows simulating errors
type MockExecutor struct {
	ShouldFail bool
}

func (m *MockExecutor) Run(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	if m.ShouldFail {
		return context.DeadlineExceeded // just some error
	}
	return nil
}

// TestExecutor captures the command execution details
type TestExecutor struct {
	CapturedName string
	CapturedArgs []string
}

func (t *TestExecutor) Run(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	t.CapturedName = name
	t.CapturedArgs = args
	return nil
}

func TestRun_PostArgs_ToFlagConversion(t *testing.T) {
	// Create a temp file to simulate input
	tmpFile, err := os.CreateTemp("", "test-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Setup
	executor := &TestExecutor{}
	opts := options.Options{
		DryRun:  true,
		Targets: []string{"html"}, // Ensure at least one target so it runs
	}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	// Simulate post-args containing -t
	// The first arg is the input file, subsequent args are passed through
	args := []string{tmpFile.Name(), "-t", "docx", "--standalone"}

	err = app.Run(context.Background(), cmd, args, opts, executor)
	if err != nil {
		t.Fatalf("app.Run failed: %v", err)
	}

	// Verify that -t was converted to --to in the captured args
	foundTo := false
	foundDocx := false
	for i, arg := range executor.CapturedArgs {
		if arg == "--to" {
			foundTo = true
			if i+1 < len(executor.CapturedArgs) && executor.CapturedArgs[i+1] == "docx" {
				foundDocx = true
			}
		}
		if arg == "-t" {
			t.Errorf("Found '-t' flag in args, expected conversion to '--to'")
		}
	}

	if !foundTo {
		t.Errorf("Did not find '--to' flag in captured args: %v", executor.CapturedArgs)
	}
	if !foundDocx {
		t.Errorf("Did not find 'docx' value after '--to' flag: %v", executor.CapturedArgs)
	}
}

func TestRun_Stdin(t *testing.T) {
	// Setup
	executor := &TestExecutor{}
	opts := options.Options{
		Targets: []string{"html"}, // Minimal target
	}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	// Mock Stdin
	inputContent := "# Hello Stdin"
	r, w, _ := os.Pipe()
	w.Write([]byte(inputContent))
	w.Close()
	// Restore old stdin logic if we were replacing os.Stdin, but here we use cmd.SetIn
	cmd.SetIn(r)

	// args: use "-" for stdin
	args := []string{"-"}

	err := app.Run(context.Background(), cmd, args, opts, executor)
	if err != nil {
		t.Fatalf("app.Run failed with stdin: %v", err)
	}

	// Verify executor was called with a temp file
	if len(executor.CapturedArgs) == 0 {
		t.Fatal("Executor was not called")
	}
	firstArg := executor.CapturedArgs[0]
	// It should be a temp file path (containing "panforge-stdin-")
	if !strings.Contains(firstArg, "panforge-stdin-") {
		t.Errorf("Expected first arg to be a temp file path (containing 'panforge-stdin-'), got: %s", firstArg)
	}
}

func TestRun_ExecutionError(t *testing.T) {
	// Verify that if Executor returns error, Run returns error
	executor := &MockExecutor{ShouldFail: true}
	opts := options.Options{
		Targets: []string{"html"},
	}

	tmpFile, _ := os.CreateTemp("", "test-*.md")
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	args := []string{tmpFile.Name()}

	err := app.Run(context.Background(), cmd, args, opts, executor)
	if err == nil {
		t.Error("Expected app.Run to fail when executor fails, but it succeeded")
	}
}
