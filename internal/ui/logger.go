// Package ui provides formatting, styling, and logging utilities for the panforge application.
package ui

import (
	"log/slog"
	"os"
)

// NewLogger creates and configures a slog.Logger based on verbose, quiet, and jsonOutput settings.
//
// Parameters:
//   - verbose: if true and jsonOutput is false, enables LevelDebug logging
//   - quiet: if true and jsonOutput is false, restricts logging to LevelError
//   - jsonOutput: if true, uses slog.NewJSONHandler outputting to os.Stderr
//
// Returns:
//   - *slog.Logger: a configured logger instance
func NewLogger(verbose bool, quiet bool, jsonOutput bool) *slog.Logger {
	var handler slog.Handler
	if jsonOutput {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		level := slog.LevelInfo
		if verbose {
			level = slog.LevelDebug
		} else if quiet {
			level = slog.LevelError
		}
		handler = NewPrettyHandler(level)
	}
	return slog.New(handler)
}
