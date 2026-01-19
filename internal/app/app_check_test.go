package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rapjul/panforge/internal/options"
)

func TestGetRequiredTools(t *testing.T) {
	// Setup temp dir
	// tmpDir := t.TempDir() // Not used directly, using t.TempDir() in loop

	tests := []struct {
		name          string
		yamlContent   string
		opts          options.Options
		expectedTools []string
		excludedTools []string
	}{
		{
			name: "Case 1: Typst",
			yamlContent: `---
outputs:
  - typst
---`,
			opts:          options.Options{},
			expectedTools: []string{"pandoc", "typst"},
			excludedTools: []string{},
		},
		{
			name: "Case 2: PDF with Tectonic",
			yamlContent: `---
output:
  pdf:
    pdf-engine: tectonic
---`,
			opts:          options.Options{},
			expectedTools: []string{"pandoc", "tectonic"},
			excludedTools: []string{"pdflatex"},
		},
		{
			name: "Case 3: HTML only",
			yamlContent: `---
outputs:
  - html
---`,
			opts:          options.Options{},
			expectedTools: []string{"pandoc"},
			excludedTools: []string{"pdflatex", "tectonic", "typst"},
		},
		{
			name: "Case 4: PDF default engine",
			yamlContent: `---
outputs:
  - pdf
---`,
			opts:          options.Options{},
			expectedTools: []string{"pandoc", "pdflatex"},
			excludedTools: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "config.yaml")
			err := os.WriteFile(tmpFile, []byte(tt.yamlContent), 0600) //nolint:gosec // 0600 for tests
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			tools, err := GetRequiredTools(tmpFile, tt.opts)
			if err != nil {
				t.Fatalf("GetRequiredTools returned error: %v", err)
			}

			for _, expected := range tt.expectedTools {
				found := false
				for _, tool := range tools {
					if tool == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected tool %q not found in %v", expected, tools)
				}
			}

			for _, excluded := range tt.excludedTools {
				found := false
				for _, tool := range tools {
					if tool == excluded {
						found = true
						break
					}
				}
				if found {
					t.Errorf("excluded tool %q found in %v", excluded, tools)
				}
			}
		})
	}
}
