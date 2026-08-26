package app_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rapjul/panforge/internal/app"
	"github.com/rapjul/panforge/internal/options"
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
			gotFound := app.PrintToolCheckRow(&buf, tt.res)
			if gotFound != tt.wantFound {
				t.Errorf("PrintToolCheckRow() returned found = %v, want %v", gotFound, tt.wantFound)
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

// TestRunCheck_DefaultTools verifies default check execution without input files.
func TestRunCheck_DefaultTools(t *testing.T) {
	var stdout, stderr bytes.Buffer
	opts := options.Options{}

	err := app.RunCheck(context.Background(), &stdout, &stderr, nil, opts)
	if err != nil {
		t.Fatalf("RunCheck returned unexpected error: %v", err)
	}

	outStr := stdout.String()
	if !strings.Contains(outStr, "Tool") || !strings.Contains(outStr, "Status") || !strings.Contains(outStr, "Version/Path") {
		t.Errorf("Output missing table headers, got: %s", outStr)
	}

	for _, tool := range app.DefaultCheckTools {
		if !strings.Contains(outStr, tool) {
			t.Errorf("Expected tool %s in default check output", tool)
		}
	}
}

// TestRunCheck_WithInputFile tests checking dependencies for a specific markdown input file.
func TestRunCheck_WithInputFile(t *testing.T) {
	tmpDir := t.TempDir()
	docPath := filepath.Join(tmpDir, "doc.md")
	content := `---
outputs:
  - html
---
# Test Document
`
	if err := os.WriteFile(docPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	opts := options.Options{}

	_ = app.RunCheck(context.Background(), &stdout, &stderr, []string{docPath}, opts)
	outStr := stdout.String()

	if !strings.Contains(outStr, "pandoc") {
		t.Errorf("Expected pandoc in output, got: %s", outStr)
	}
}
