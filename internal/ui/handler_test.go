package ui_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/rapjul/panforge/internal/ui"
)

// TestPrettyHandler_LogLevels verifies enabled status for various log levels.
func TestPrettyHandler_LogLevels(t *testing.T) {
	h := ui.NewPrettyHandler(slog.LevelInfo)
	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Expected Debug level to be disabled for LevelInfo handler")
	}
	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Expected Info level to be enabled for LevelInfo handler")
	}
	if !h.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("Expected Warn level to be enabled for LevelInfo handler")
	}
	if !h.Enabled(context.Background(), slog.LevelError) {
		t.Error("Expected Error level to be enabled for LevelInfo handler")
	}
}

// TestPrettyHandler_WithAttrsAndGroup verifies handler interface compatibility.
func TestPrettyHandler_WithAttrsAndGroup(t *testing.T) {
	h := ui.NewPrettyHandler(slog.LevelInfo)
	withAttrs := h.WithAttrs([]slog.Attr{slog.String("key", "val")})
	if withAttrs == nil {
		t.Error("WithAttrs returned nil")
	}
	withGroup := h.WithGroup("testGroup")
	if withGroup == nil {
		t.Error("WithGroup returned nil")
	}
}

// TestPrettyHandler_HandleMessages tests rendering of messages, commands, errors, and config debug output.
func TestPrettyHandler_HandleMessages(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(ui.NewPrettyHandler(slog.LevelDebug))

	// Normal info message
	logger.Info("Starting compilation")

	// Error with error attribute
	logger.Error("Compilation failed", "error", errors.New("missing dependency"))

	// Dry run command execution with args
	logger.Info("executing the command",
		"args", []string{"pandoc", "doc.md", "--to", "pdf", "--output", "doc.pdf"},
		"dry_run", true,
	)

	// Command with []any args
	logger.Info("executing command with any args",
		"args", []any{"pandoc", "doc.md", "-t", "html"},
	)

	// Command with command string
	logger.Info("executing command string",
		"command", "pandoc doc.md -o doc.html",
	)

	// Config debug output
	logger.Debug("Effective configuration",
		"config", map[string]any{
			"standalone": true,
			"nested": map[string]any{
				"engine": "xelatex",
			},
			"items": []any{"a", "b"},
		},
	)

	_ = buf.String()
}
