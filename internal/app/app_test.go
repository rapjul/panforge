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
	// Use os.WriteFile to specify permissions and ensure content
	tmpFile, err := os.CreateTemp("", "test-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	if err := tmpFile.Close(); err != nil { // Handle close error
		t.Fatalf("Failed to close temp file: %v", err)
	}

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
	_, _ = w.Write([]byte(inputContent))
	_ = w.Close()
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

// TestRun_ExecutionError verifies error propagation when the executor encounters a failure.
func TestRun_ExecutionError(t *testing.T) {
	executor := &MockExecutor{ShouldFail: true}
	opts := options.Options{
		Targets: []string{"html"},
	}

	tmpFile, _ := os.CreateTemp("", "test-*.md")
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_ = tmpFile.Close()

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	args := []string{tmpFile.Name()}

	err := app.Run(context.Background(), cmd, args, opts, executor)
	if err == nil {
		t.Error("Expected app.Run to fail when executor fails, but it succeeded")
	}
}

// TestRun_BatchProcessing verifies processing of multiple input files in a single invocation.
func TestRun_BatchProcessing(t *testing.T) {
	tmp1, err := os.CreateTemp("", "batch-test-1-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmp1.Name()) }()
	_, _ = tmp1.WriteString("# Doc 1\n")
	_ = tmp1.Close()

	tmp2, err := os.CreateTemp("", "batch-test-2-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmp2.Name()) }()
	_, _ = tmp2.WriteString("# Doc 2\n")
	_ = tmp2.Close()

	executor := &TestExecutor{}
	opts := options.Options{
		DryRun:  true,
		Targets: []string{"html"},
	}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	args := []string{tmp1.Name(), tmp2.Name()}
	err = app.Run(context.Background(), cmd, args, opts, executor)
	if err != nil {
		t.Fatalf("app.Run failed in batch mode: %v", err)
	}
}

// TestRun_BatchProcessing_FixedOutputError verifies error on fixed output filename with multiple inputs.
func TestRun_BatchProcessing_FixedOutputError(t *testing.T) {
	executor := &TestExecutor{}
	opts := options.Options{
		DryRun:  true,
		Targets: []string{"html"},
		Output:  "fixed-output.html", // Fixed output name without template variables
	}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	args := []string{"doc1.md", "doc2.md"}
	err := app.Run(context.Background(), cmd, args, opts, executor)
	if err == nil {
		t.Error("Expected error when using fixed --output with multiple input files, got nil")
	}
	if !strings.Contains(err.Error(), "cannot specify a fixed output filename") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestRun_ExplicitInputFlags verifies inputs provided through --input / -i flag.
func TestRun_ExplicitInputFlags(t *testing.T) {
	tmp, err := os.CreateTemp("", "flag-test-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, _ = tmp.WriteString("# Doc via Flag\n")
	_ = tmp.Close()

	executor := &TestExecutor{}
	opts := options.Options{
		DryRun:  true,
		Targets: []string{"html"},
		Inputs:  []string{tmp.Name()},
	}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err = app.Run(context.Background(), cmd, []string{}, opts, executor)
	if err != nil {
		t.Fatalf("app.Run failed with explicit inputs flag: %v", err)
	}
}

// TestRun_NoInputFile verifies handling when no input files are specified.
func TestRun_NoInputFile(t *testing.T) {
	executor := &TestExecutor{}
	opts := options.Options{
		Targets: []string{"html"},
	}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := app.Run(context.Background(), cmd, []string{}, opts, executor)
	if err == nil {
		t.Error("Expected error when no input file is found with targets set, got nil")
	}
}

// TestRun_FlagsAndOptions verifies execution with logging, concurrency, relative output, and verbose flags.
func TestRun_FlagsAndOptions(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "flags-test-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_, _ = tmpFile.WriteString("# Document with Flags\n")
	_ = tmpFile.Close()

	logFile, err := os.CreateTemp("", "panforge-test-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(logFile.Name()) }()
	_ = logFile.Close()

	executor := &TestExecutor{}
	opts := options.Options{
		DryRun:         true,
		Targets:        []string{"html"},
		Log:            logFile.Name(),
		Concurrency:    2,
		RelativeOutput: true,
		Verbose:        true,
	}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err = app.Run(context.Background(), cmd, []string{tmpFile.Name()}, opts, executor)
	if err != nil {
		t.Fatalf("app.Run failed with flags and options: %v", err)
	}

	// Verify log file content
	logContent, _ := os.ReadFile(logFile.Name())
	if !strings.Contains(string(logContent), "panforge calling: pandoc") {
		t.Errorf("Expected log file to contain command call, got: %s", string(logContent))
	}
}
