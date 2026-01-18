package app

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rapjul/panforge/internal/options"
)

// MockExecutor records calls
type MockExecutor struct {
	mu    sync.Mutex
	calls []string
}

func (m *MockExecutor) Run(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, name+" "+args[len(args)-1]) // Store last arg (output filename) or similar
	return nil
}

func (m *MockExecutor) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func TestWatch(t *testing.T) {
	// Setup temp dir
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	// Create dummy input file
	inputFile := "test.md"
	content := `---
outputs:
  - html
---
# Hello
`
	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}

	absInput, _ := filepath.Abs(inputFile)

	mockExec := &MockExecutor{}
	opts := options.Options{
		Watch: true,
	}

	// Create a context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run Watch in goroutine
	done := make(chan error)
	go func() {
		done <- Watch(ctx, absInput, "", []string{}, opts, mockExec)
	}()

	// Wait for startup (initial run)
	// We expect 1 call initially
	time.Sleep(500 * time.Millisecond) // enough for initial process

	if count := mockExec.CallCount(); count != 1 {
		t.Fatalf("expected 1 initial call, got %d", count)
	}

	// Create "output" file to trigger overwrite check on next run
	// Note: Generic output filename logic will likely produce "Hello.html" based on title/file
	// For this test, lets just create a file that matches what we expect Process to find.
	// We need to look at what Process generates.
	// In the test setup, we didn't specify title, so it might use "test" from filename.
	// But we wrote "outputs: [html]".
	// Let's create `test.html` and `Hello.html` (since dummy content has # Hello) just to be sure.
	os.WriteFile("test.html", []byte("old"), 0644)
	os.WriteFile("Hello.html", []byte("old"), 0644)

	// Modify file
	f, err := os.OpenFile(inputFile, os.O_APPEND|os.O_WRONLY, 0644)

	if err != nil {
		t.Fatalf("failed to open file for append: %v", err)
	}
	if _, err := f.WriteString("\nNew content\n"); err != nil {
		t.Fatalf("failed to write to file: %v", err)
	}
	f.Close()

	// Wait for debounce (100ms) + processing time
	time.Sleep(500 * time.Millisecond)

	// We expect 2 calls now (initial + reload)
	if count := mockExec.CallCount(); count < 2 {
		t.Fatalf("expected at least 2 calls after modification, got %d", count)
	}

	// Cancel context to stop watcher
	cancel()
	err = <-done
	if err != nil {
		t.Errorf("Watch returned error: %v", err)
	}
}
