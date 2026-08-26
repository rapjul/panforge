package main

import (
	"bytes"
	"runtime/debug"
	"testing"
)

// TestInitVersion verifies that initVersion correctly populates version and commit from BuildInfo.
func TestInitVersion(t *testing.T) {
	origVersion := version
	origCommit := commit
	defer func() {
		version = origVersion
		commit = origCommit
	}()

	t.Run("nil BuildInfo", func(t *testing.T) {
		version = "dev"
		commit = "none"
		initVersion(nil)
		if version != "dev" || commit != "none" {
			t.Errorf("expected version='dev', commit='none', got version=%q, commit=%q", version, commit)
		}
	})

	t.Run("valid module version and revision", func(t *testing.T) {
		version = "dev"
		commit = "none"
		info := &debug.BuildInfo{
			Main: debug.Module{
				Version: "v0.3.0",
			},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "1234567890abcdef"},
			},
		}
		initVersion(info)
		if version != "v0.3.0" {
			t.Errorf("expected version='v0.3.0', got %q", version)
		}
		if commit != "1234567" {
			t.Errorf("expected commit='1234567', got %q", commit)
		}
	})

	t.Run("does not overwrite ldflags version and commit", func(t *testing.T) {
		version = "v1.0.0"
		commit = "custom"
		info := &debug.BuildInfo{
			Main: debug.Module{
				Version: "v0.3.0",
			},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "1234567890abcdef"},
			},
		}
		initVersion(info)
		if version != "v1.0.0" {
			t.Errorf("expected version='v1.0.0', got %q", version)
		}
		if commit != "custom" {
			t.Errorf("expected commit='custom', got %q", commit)
		}
	})

	t.Run("ignores devel version", func(t *testing.T) {
		version = "dev"
		commit = "none"
		info := &debug.BuildInfo{
			Main: debug.Module{
				Version: "(devel)",
			},
		}
		initVersion(info)
		if version != "dev" {
			t.Errorf("expected version='dev', got %q", version)
		}
	})
}

// TestFormatVersion tests version string formatting for development and release builds.
func TestFormatVersion(t *testing.T) {
	tests := []struct {
		name     string
		v        string
		commit   string
		expected string
	}{
		{
			name:     "Dev build with commit hash",
			v:        "dev",
			commit:   "abc1234",
			expected: "dev (commit: abc1234)",
		},
		{
			name:     "Dev build with none commit",
			v:        "dev",
			commit:   "none",
			expected: "dev (commit: none)",
		},
		{
			name:     "Tagged release version",
			v:        "v1.0.0",
			commit:   "abc1234",
			expected: "v1.0.0",
		},
		{
			name:     "Semver version string",
			v:        "0.2.1",
			commit:   "",
			expected: "0.2.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVersion(tt.v, tt.commit)
			if got != tt.expected {
				t.Errorf("formatVersion(%q, %q) = %q, want %q", tt.v, tt.commit, got, tt.expected)
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
