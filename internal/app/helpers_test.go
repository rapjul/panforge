package app

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rapjul/panforge/internal/config"
	"github.com/rapjul/panforge/internal/options"
)

// errReader is an io.Reader that always returns an error.
type errReader struct{}

func (e *errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read failure")
}

// mockExecutor implements CommandExecutor for testing execution branches.
type mockExecutor struct {
	runFunc func(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error
}

func (m *mockExecutor) Run(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	if m.runFunc != nil {
		return m.runFunc(ctx, name, args, stdout, stderr)
	}
	return nil
}

// TestCreateStdinTempFile tests stdin ingestion and temporary file cleanup.
func TestCreateStdinTempFile(t *testing.T) {
	t.Run("Successful read and cleanup", func(t *testing.T) {
		inputData := "hello from stdin"
		r := strings.NewReader(inputData)

		path, cleanup, err := createStdinTempFile(r)
		if err != nil {
			t.Fatalf("createStdinTempFile failed: %v", err)
		}
		if cleanup == nil {
			t.Fatal("cleanup callback is nil")
		}

		// Verify file content
		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			t.Fatalf("failed to read created temp file: %v", err)
		}
		if string(data) != inputData {
			t.Errorf("got content %q, want %q", string(data), inputData)
		}

		// Execute cleanup and verify deletion
		cleanup()
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected temp file to be deleted by cleanup, stat err: %v", err)
		}
	})

	t.Run("Reader error branch", func(t *testing.T) {
		_, _, err := createStdinTempFile(&errReader{})
		if err == nil {
			t.Fatal("expected error from faulty reader, got nil")
		}
	})
}

// TestMergeDefaultConfig tests configuration merging across all fields.
func TestMergeDefaultConfig(t *testing.T) {
	t.Run("Nil safety", func(t *testing.T) {
		mergeDefaultConfig(nil, nil)
		cfg := &config.Config{}
		mergeDefaultConfig(cfg, nil)
		mergeDefaultConfig(nil, cfg)
	})

	t.Run("Merge missing fields", func(t *testing.T) {
		slugifyVal := true
		defaultCfg := &config.Config{
			Title:            "Default Title",
			FilenameTemplate: "{title}.{ext}",
			SlugifyFilename:  &slugifyVal,
			OutputMap: map[string]any{
				"pdf": map[string]any{"toc": true},
			},
			Generic: map[string]any{
				"lang": "en",
			},
		}

		targetCfg := &config.Config{
			Title: "Custom Title",
			OutputMap: map[string]any{
				"html": map[string]any{"standalone": true},
			},
			Generic: map[string]any{
				"author": "Tester",
			},
		}

		mergeDefaultConfig(targetCfg, defaultCfg)

		if targetCfg.Title != "Custom Title" {
			t.Errorf("got Title %q, want 'Custom Title'", targetCfg.Title)
		}
		if targetCfg.FilenameTemplate != "{title}.{ext}" {
			t.Errorf("got FilenameTemplate %q, want '{title}.{ext}'", targetCfg.FilenameTemplate)
		}
		if targetCfg.SlugifyFilename == nil || *targetCfg.SlugifyFilename != true {
			t.Errorf("SlugifyFilename was not copied from default")
		}
		if _, exists := targetCfg.OutputMap["pdf"]; !exists {
			t.Errorf("OutputMap missing 'pdf' from default")
		}
		if _, exists := targetCfg.Generic["lang"]; !exists {
			t.Errorf("Generic missing 'lang' from default")
		}
		if targetCfg.Generic["author"] != "Tester" {
			t.Errorf("Generic 'author' overwritten, got %v", targetCfg.Generic["author"])
		}
	})
}

// TestResolveTargetConfig tests format normalization and metadata inheritance.
func TestResolveTargetConfig(t *testing.T) {
	t.Run("Nil config fallback", func(t *testing.T) {
		fmtStr, meta := resolveTargetConfig("pdf", nil)
		if fmtStr != "pdf" {
			t.Errorf("got fmtStr %q, want 'pdf'", fmtStr)
		}
		if meta == nil {
			t.Fatal("expected non-nil metaOut map")
		}
	})

	t.Run("Target in OutputMap with custom to format", func(t *testing.T) {
		cfg := &config.Config{
			OutputMap: map[string]any{
				"paper": map[string]any{
					"to":  "latex",
					"toc": true,
				},
			},
			Generic: map[string]any{
				"number-sections": true,
				"creator":         "IgnoredTool", // In IgnoredMetadataKeys
			},
		}

		fmtStr, meta := resolveTargetConfig("paper", cfg)
		if fmtStr != "latex" {
			t.Errorf("got fmtStr %q, want 'latex'", fmtStr)
		}
		if meta["toc"] != true {
			t.Errorf("expected toc=true in meta")
		}
		if meta["number-sections"] != true {
			t.Errorf("expected generic number-sections to be merged, got %v", meta["number-sections"])
		}
		if _, exists := meta["creator"]; exists {
			t.Errorf("expected blacklisted key 'creator' to be excluded")
		}
	})

	t.Run("Target in Generic map", func(t *testing.T) {
		cfg := &config.Config{
			Generic: map[string]any{
				"html": map[string]any{"standalone": true},
			},
		}

		fmtStr, meta := resolveTargetConfig("html", cfg)
		if fmtStr != "html" {
			t.Errorf("got fmtStr %q, want 'html'", fmtStr)
		}
		if meta["standalone"] != true {
			t.Errorf("expected standalone=true in meta")
		}
	})
}

// TestResolveOutputPath tests output filename generation and relative resolution rules.
func TestResolveOutputPath(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "doc.md")

	t.Run("Explicit output filename", func(t *testing.T) {
		explicit := filepath.Join(tmpDir, "custom.pdf")
		out, err := resolveOutputPath(inputFile, explicit, "pdf", &config.Config{}, nil, false)
		if err != nil {
			t.Fatalf("resolveOutputPath failed: %v", err)
		}
		if out != explicit {
			t.Errorf("got out %q, want %q", out, explicit)
		}
	})

	t.Run("Relative to input directory when relativeOutput is false", func(t *testing.T) {
		cfg := &config.Config{Title: "My Doc"}
		out, err := resolveOutputPath(inputFile, "", "pdf", cfg, nil, false)
		if err != nil {
			t.Fatalf("resolveOutputPath failed: %v", err)
		}
		if filepath.Dir(out) != tmpDir {
			t.Errorf("expected output dir %q, got %q", tmpDir, filepath.Dir(out))
		}
	})

	t.Run("Stdin temp file resolves against CWD", func(t *testing.T) {
		stdinPath := filepath.Join(tmpDir, "panforge-stdin-12345.md")
		cfg := &config.Config{Title: "Stdin Doc"}
		out, err := resolveOutputPath(stdinPath, "", "html", cfg, nil, false)
		if err != nil {
			t.Fatalf("resolveOutputPath failed: %v", err)
		}
		if out == "" {
			t.Fatal("empty output path returned")
		}
	})
}

// TestConfirmOverwrite tests overwrite decision branches and prompt bypasses.
func TestConfirmOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("data"), 0600); err != nil {
		t.Fatalf("failed to create existing test file: %v", err)
	}
	missingFile := filepath.Join(tmpDir, "missing.txt")

	t.Run("DryRun bypasses overwrite check", func(t *testing.T) {
		allowed, err := confirmOverwrite(existingFile, &config.Config{}, nil, options.Options{DryRun: true}, nil)
		if err != nil || !allowed {
			t.Errorf("expected allowed=true, err=nil, got allowed=%v, err=%v", allowed, err)
		}
	})

	t.Run("File does not exist", func(t *testing.T) {
		allowed, err := confirmOverwrite(missingFile, &config.Config{}, nil, options.Options{}, nil)
		if err != nil || !allowed {
			t.Errorf("expected allowed=true for missing file, got allowed=%v, err=%v", allowed, err)
		}
	})

	t.Run("Force flag allows overwrite", func(t *testing.T) {
		allowed, err := confirmOverwrite(existingFile, &config.Config{}, nil, options.Options{Force: true}, nil)
		if err != nil || !allowed {
			t.Errorf("expected allowed=true with Force, got allowed=%v, err=%v", allowed, err)
		}
	})

	t.Run("Watch mode allows overwrite", func(t *testing.T) {
		allowed, err := confirmOverwrite(existingFile, &config.Config{}, nil, options.Options{Watch: true}, nil)
		if err != nil || !allowed {
			t.Errorf("expected allowed=true with Watch, got allowed=%v, err=%v", allowed, err)
		}
	})

	t.Run("Target metadata overwrite: true allows overwrite", func(t *testing.T) {
		metaOut := map[string]any{"overwrite": true}
		allowed, err := confirmOverwrite(existingFile, &config.Config{}, metaOut, options.Options{}, nil)
		if err != nil || !allowed {
			t.Errorf("expected allowed=true with metadata overwrite, got allowed=%v, err=%v", allowed, err)
		}
	})

	t.Run("Generic config overwrite: true allows overwrite", func(t *testing.T) {
		cfg := &config.Config{Generic: map[string]any{"overwrite": true}}
		allowed, err := confirmOverwrite(existingFile, cfg, nil, options.Options{}, nil)
		if err != nil || !allowed {
			t.Errorf("expected allowed=true with generic overwrite, got allowed=%v, err=%v", allowed, err)
		}
	})
}

// TestBuildPandocCommand tests command argument generation and formatting.
func TestBuildPandocCommand(t *testing.T) {
	meta := map[string]any{
		"toc": true,
	}
	postArgs := []string{"-t", "html5", "--variable", "theme=dark"}

	args, cmdStr := buildPandocCommand("input file.md", "output doc.html", "html5", meta, ".", postArgs)

	if !strings.Contains(cmdStr, "pandoc") {
		t.Errorf("expected cmdStr to start with pandoc, got: %s", cmdStr)
	}
	if !strings.Contains(cmdStr, `"input file.md"`) {
		t.Errorf("expected quoted input file in cmdStr: %s", cmdStr)
	}
	if !strings.Contains(cmdStr, "--to html5") {
		t.Errorf("expected -t normalized to --to in cmdStr: %s", cmdStr)
	}

	// Verify args slice
	foundTo := false
	for _, arg := range args {
		if arg == "--to" {
			foundTo = true
			break
		}
	}
	if !foundTo {
		t.Errorf("expected --to in args, got: %v", args)
	}
}

// TestExecutePandoc tests execution failure handling and stderr tail extraction.
func TestExecutePandoc(t *testing.T) {
	t.Run("Success branch", func(t *testing.T) {
		mock := &mockExecutor{
			runFunc: func(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
				return nil
			},
		}
		err := executePandoc(context.Background(), mock, []string{"doc.md"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("Error with short stderr", func(t *testing.T) {
		mock := &mockExecutor{
			runFunc: func(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
				_, _ = stderr.Write([]byte("short error description\n"))
				return errors.New("exit status 1")
			},
		}
		err := executePandoc(context.Background(), mock, []string{"doc.md"})
		if err == nil || !strings.Contains(err.Error(), "short error description") {
			t.Errorf("expected formatted stderr error, got: %v", err)
		}
	})

	t.Run("Error with long (>5 line) stderr", func(t *testing.T) {
		longStderr := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\n"
		mock := &mockExecutor{
			runFunc: func(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
				_, _ = stderr.Write([]byte(longStderr))
				return errors.New("exit status 1")
			},
		}
		err := executePandoc(context.Background(), mock, []string{"doc.md"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		// Should contain line 8 but not line 1
		if !strings.Contains(err.Error(), "line 8") || strings.Contains(err.Error(), "line 1") {
			t.Errorf("expected last 5 lines extraction, got error: %v", err)
		}
	})
}

// TestResolvePDFEngine tests engine resolution priority.
func TestResolvePDFEngine(t *testing.T) {
	t.Run("Target meta override", func(t *testing.T) {
		meta := map[string]any{"pdf-engine": "xelatex"}
		generic := map[string]any{"pdf-engine": "lualatex"}
		if got := resolvePDFEngine(meta, generic); got != "xelatex" {
			t.Errorf("got %q, want 'xelatex'", got)
		}
	})

	t.Run("Generic fallback", func(t *testing.T) {
		meta := map[string]any{}
		generic := map[string]any{"pdf-engine": "tectonic"}
		if got := resolvePDFEngine(meta, generic); got != "tectonic" {
			t.Errorf("got %q, want 'tectonic'", got)
		}
	})

	t.Run("Default fallback", func(t *testing.T) {
		if got := resolvePDFEngine(nil, nil); got != "pdflatex" {
			t.Errorf("got %q, want 'pdflatex'", got)
		}
	})
}

// TestContainsHelper tests string slice search.
func TestContainsHelper(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}
	if !contains(slice, "banana") {
		t.Error("expected contains to return true for 'banana'")
	}
	if contains(slice, "orange") {
		t.Error("expected contains to return false for 'orange'")
	}
}
