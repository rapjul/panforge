package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/rapjul/panforge/internal/utils"
)

// TestPrintToolCheckRow_Table verifies formatting and status calculation for tool check results.
func TestPrintToolCheckRow_Table(t *testing.T) {
	tests := []struct {
		name         string
		res          utils.CheckResult
		wantFound    bool
		wantContains []string
	}{
		{
			name: "Found tool with short version",
			res: utils.CheckResult{
				Name:    "pandoc",
				Found:   true,
				Version: "pandoc 3.1.2",
			},
			wantFound:    true,
			wantContains: []string{"pandoc", "FOUND", "pandoc 3.1.2"},
		},
		{
			name: "Found tool with long version string (truncated)",
			res: utils.CheckResult{
				Name:    "pdflatex",
				Found:   true,
				Version: "pdfTeX 3.141592653-2.6-1.40.26 (TeX Live 2024) (preloaded format=pdflatex 2024.3.12) kpathsea version 6.4.0",
			},
			wantFound:    true,
			wantContains: []string{"pdflatex", "FOUND", "..."},
		},
		{
			name: "Found tool with path only",
			res: utils.CheckResult{
				Name:  "tectonic",
				Found: true,
				Path:  "/usr/local/bin/tectonic",
			},
			wantFound:    true,
			wantContains: []string{"tectonic", "FOUND", "/usr/local/bin/tectonic"},
		},
		{
			name: "Missing tool with error",
			res: utils.CheckResult{
				Name:  "xelatex",
				Found: false,
				Error: errors.New("executable file not found in $PATH"),
			},
			wantFound:    false,
			wantContains: []string{"xelatex", "MISSING", "executable file not found in $PATH"},
		},
		{
			name: "Missing tool without error",
			res: utils.CheckResult{
				Name:  "typst",
				Found: false,
			},
			wantFound:    false,
			wantContains: []string{"typst", "MISSING", "not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			gotFound := printToolCheckRow(&buf, tt.res)
			if gotFound != tt.wantFound {
				t.Errorf("printToolCheckRow() returned found = %v, want %v", gotFound, tt.wantFound)
			}
			out := buf.String()
			for _, exp := range tt.wantContains {
				if !strings.Contains(out, exp) {
					t.Errorf("Output %q does not contain expected substring %q", out, exp)
				}
			}
		})
	}
}

// TestNewRootCmd_Flags verifies flag configuration and alias presence on rootCmd.
func TestNewRootCmd_Flags(t *testing.T) {
	cmd, opts := newRootCmd()
	if cmd == nil || opts == nil {
		t.Fatal("newRootCmd returned nil")
	}

	// Verify flags exist
	expectedFlags := []string{
		"input", "to", "target", "all", "output", "force", "dry-run",
		"verbose", "quiet", "log", "json", "concurrency", "relative-output", "watch",
	}
	for _, f := range expectedFlags {
		if flag := cmd.Flags().Lookup(f); flag == nil {
			t.Errorf("Expected flag --%s on rootCmd, but was not found", f)
		}
	}

	// Verify short flags
	if flag := cmd.Flags().ShorthandLookup("F"); flag == nil || flag.Name != "force" {
		t.Errorf("Expected short flag -F to bind to force")
	}
	if flag := cmd.Flags().ShorthandLookup("n"); flag == nil || flag.Name != "dry-run" {
		t.Errorf("Expected short flag -n to bind to dry-run")
	}
	if flag := cmd.Flags().ShorthandLookup("i"); flag == nil || flag.Name != "input" {
		t.Errorf("Expected short flag -i to bind to input")
	}
	if flag := cmd.Flags().ShorthandLookup("t"); flag == nil || flag.Name != "to" {
		t.Errorf("Expected short flag -t to bind to to")
	}
}

// TestNewRootCmd_Subcommands verifies init and check subcommands are attached.
func TestNewRootCmd_Subcommands(t *testing.T) {
	cmd, _ := newRootCmd()

	var initFound, checkFound bool
	for _, sub := range cmd.Commands() {
		if sub.Name() == "init" {
			initFound = true
		}
		if sub.Name() == "check" {
			checkFound = true
		}
	}

	if !initFound {
		t.Error("init subcommand was not found on rootCmd")
	}
	if !checkFound {
		t.Error("check subcommand was not found on rootCmd")
	}
}

// TestNewRootCmd_HelpOutput verifies help generation.
func TestNewRootCmd_HelpOutput(t *testing.T) {
	cmd, _ := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute() with --help failed: %v", err)
	}

	helpText := out.String()
	if len(helpText) == 0 {
		t.Error("Expected non-empty help text")
	}
}

// TestNewRootCmd_ValidArgsCompletion verifies filename extension completion for rootCmd.
func TestNewRootCmd_ValidArgsCompletion(t *testing.T) {
	cmd, _ := newRootCmd()

	// Initial argument completion
	completions, directive := cmd.ValidArgsFunction(cmd, []string{}, "")
	if len(completions) != 2 || completions[0] != "md" || completions[1] != "markdown" {
		t.Errorf("Unexpected completions: %v", completions)
	}
	_ = directive

	// After arguments already present
	completions, _ = cmd.ValidArgsFunction(cmd, []string{"doc.md"}, "")
	if completions != nil {
		t.Errorf("Expected nil completions when args are already present, got %v", completions)
	}
}

// TestNewRootCmd_FlagCompletion verifies custom flag completion functions.
func TestNewRootCmd_FlagCompletion(t *testing.T) {
	cmd, _ := newRootCmd()

	// Test watch flag completion
	watchFlag := cmd.Flags().Lookup("watch")
	if watchFlag == nil {
		t.Fatal("Expected --watch flag")
	}
	fn, _ := cmd.GetFlagCompletionFunc("watch")
	if fn != nil {
		comps, _ := fn(cmd, []string{}, "")
		if len(comps) != 2 || comps[0] != "true" || comps[1] != "false" {
			t.Errorf("Unexpected watch completions: %v", comps)
		}
	}

	// Test to flag completion
	toFlag := cmd.Flags().Lookup("to")
	if toFlag == nil {
		t.Fatal("Expected --to flag")
	}
	toFn, _ := cmd.GetFlagCompletionFunc("to")
	if toFn != nil {
		_, _ = toFn(cmd, []string{}, "")
	}
}
