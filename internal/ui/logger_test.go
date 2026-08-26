package ui_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/rapjul/panforge/internal/ui"
)

// TestNewLogger_Configurations tests logger level and handler creation across all flag combinations.
func TestNewLogger_Configurations(t *testing.T) {
	tests := []struct {
		name       string
		verbose    bool
		quiet      bool
		jsonOutput bool
		wantDebug  bool
		wantInfo   bool
		wantError  bool
	}{
		{
			name:       "Default (Info level)",
			verbose:    false,
			quiet:      false,
			jsonOutput: false,
			wantDebug:  false,
			wantInfo:   true,
			wantError:  true,
		},
		{
			name:       "Verbose mode (Debug level)",
			verbose:    true,
			quiet:      false,
			jsonOutput: false,
			wantDebug:  true,
			wantInfo:   true,
			wantError:  true,
		},
		{
			name:       "Quiet mode (Error level)",
			verbose:    false,
			quiet:      true,
			jsonOutput: false,
			wantDebug:  false,
			wantInfo:   false,
			wantError:  true,
		},
		{
			name:       "JSON mode",
			verbose:    false,
			quiet:      false,
			jsonOutput: true,
			wantDebug:  false,
			wantInfo:   true,
			wantError:  true,
		},
		{
			name:       "JSON mode overrides verbose/quiet",
			verbose:    true,
			quiet:      true,
			jsonOutput: true,
			wantDebug:  false,
			wantInfo:   true,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := ui.NewLogger(tt.verbose, tt.quiet, tt.jsonOutput)
			if logger == nil {
				t.Fatal("NewLogger returned nil")
			}

			ctx := context.Background()
			handler := logger.Handler()

			if got := handler.Enabled(ctx, slog.LevelDebug); got != tt.wantDebug {
				t.Errorf("LevelDebug enabled = %v, want %v", got, tt.wantDebug)
			}
			if got := handler.Enabled(ctx, slog.LevelInfo); got != tt.wantInfo {
				t.Errorf("LevelInfo enabled = %v, want %v", got, tt.wantInfo)
			}
			if got := handler.Enabled(ctx, slog.LevelError); got != tt.wantError {
				t.Errorf("LevelError enabled = %v, want %v", got, tt.wantError)
			}
		})
	}
}
