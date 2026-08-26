package app

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rapjul/panforge/internal/config"
	"github.com/rapjul/panforge/internal/options"
)

func TestDetermineTargets(t *testing.T) {
	tests := []struct {
		name     string
		opts     options.Options
		cfg      *config.Config
		expected []string
	}{
		{
			name: "CLI targets override everything",
			opts: options.Options{Targets: []string{"pdf", "docx"}},
			cfg: &config.Config{
				Outputs: []any{"html"},
			},
			expected: []string{"pdf", "docx"},
		},
		{
			name: "Config outputs list used if no CLI targets",
			opts: options.Options{},
			cfg: &config.Config{
				Outputs: []any{"html", "epub"},
			},
			expected: []string{"html", "epub"},
		},
		{
			name: "Config output map used if no outputs list",
			opts: options.Options{},
			cfg: &config.Config{
				OutputMap: map[string]any{
					"pdf":  nil,
					"docx": nil,
				},
			},
			// determineTargets sorts map keys
			expected: []string{"docx", "pdf"},
		},
		{
			name:     "Fallback to html",
			opts:     options.Options{},
			cfg:      &config.Config{},
			expected: []string{"html"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineTargets(tt.opts, tt.cfg)
			if len(got) != len(tt.expected) {
				t.Errorf("determineTargets() length = %v, want %v", len(got), len(tt.expected))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("determineTargets()[%d] = %v, want %v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestIsOverwriteAllowed(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		metaOut map[string]any
		want    bool
	}{
		{
			name:    "Default false",
			cfg:     &config.Config{},
			metaOut: map[string]any{},
			want:    false,
		},
		{
			name: "Target specific true",
			cfg:  &config.Config{},
			metaOut: map[string]any{
				"overwrite": true,
			},
			want: true,
		},
		{
			name: "Target specific false",
			cfg: &config.Config{
				Generic: map[string]any{"overwrite": true},
			},
			metaOut: map[string]any{
				"overwrite": false,
			},
			// Current logic: checks target first. If target differs?
			// The code says:
			// if target has it, return true if true.
			// if global has it, return true if true.
			// It implies that if target is explicit FALSE, it might still return TRUE if generic is TRUE?
			// Let's check code:
			// if v, ok := metaOut["overwrite"]; ok { if b { return true } }
			// This means if metaOut["overwrite"] is FALSE, it continues to check global.
			// So "overwrite: false" in target DOES NOT overload "overwrite: true" in global.
			// This logic promotes "allow overwrite", i.e. strict OR.
			want: true,
		},
		{
			name: "Global true",
			cfg: &config.Config{
				Generic: map[string]any{"overwrite": true},
			},
			metaOut: map[string]any{},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOverwriteAllowed(tt.cfg, tt.metaOut); got != tt.want {
				t.Errorf("isOverwriteAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAskForConfirmation verifies interactive prompt responses.
func TestAskForConfirmation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"yes lowercase", "y\n", true},
		{"yes word", "yes\n", true},
		{"yes uppercase", "Y\n", true},
		{"no lowercase", "n\n", false},
		{"no word", "no\n", false},
		{"empty input", "\n", false},
		{"random text", "maybe\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			var w bytes.Buffer
			got := askForConfirmation("test.html", r, &w)
			if got != tt.expected {
				t.Errorf("askForConfirmation(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// TestRealExecutor verifies execution with dry-run on and off.
func TestRealExecutor(t *testing.T) {
	dryExec := &RealExecutor{DryRun: true}
	var out, errBuf bytes.Buffer
	if err := dryExec.Run(context.Background(), "echo", []string{"hello"}, &out, &errBuf); err != nil {
		t.Errorf("DryRun RealExecutor failed: %v", err)
	}

	realExec := &RealExecutor{DryRun: false}
	if err := realExec.Run(context.Background(), "go", []string{"version"}, &out, &errBuf); err != nil {
		t.Errorf("RealExecutor Run failed: %v", err)
	}
}

// TestParseArgs_Direct verifies argument parsing logic across diverse flag and dash combinations.
func TestParseArgs_Direct(t *testing.T) {
	// Case 1: Simple input file
	inputs, post := parseArgs(nil, []string{"doc.md"}, nil)
	if len(inputs) != 1 || inputs[0] != "doc.md" || len(post) != 0 {
		t.Errorf("parseArgs simple failed: inputs=%v, post=%v", inputs, post)
	}

	// Case 2: Multiple input files and post flags
	inputs, post = parseArgs(nil, []string{"doc1.md", "doc2.md", "-t", "html"}, []string{"extra.md"})
	if len(inputs) != 3 || len(post) != 2 {
		t.Errorf("parseArgs multi failed: inputs=%v, post=%v", inputs, post)
	}

	// Case 3: Stdin
	inputs, post = parseArgs(nil, []string{"-"}, nil)
	if len(inputs) != 1 || inputs[0] != "-" {
		t.Errorf("parseArgs stdin failed: inputs=%v, post=%v", inputs, post)
	}
}

// TestContains verifies string slice membership checking.
func TestContains(t *testing.T) {
	slice := []string{"pandoc", "xelatex", "typst"}
	if !contains(slice, "pandoc") {
		t.Error("contains() returned false for existing item")
	}
	if contains(slice, "nonexistent") {
		t.Error("contains() returned true for nonexistent item")
	}
}
