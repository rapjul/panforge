package pandoc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateYamlMetadata(t *testing.T) {
	// Create dummy files for valid tests
	tmpDir, err := os.MkdirTemp("", "validate_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	validFile := filepath.Join(tmpDir, "style.css")
	if err := os.WriteFile(validFile, []byte("body {}"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		meta    map[string]interface{}
		wantErr bool
	}{
		{
			name: "Valid file",
			meta: map[string]interface{}{
				"css": validFile,
			},
			wantErr: false,
		},
		{
			name: "Missing file",
			meta: map[string]interface{}{
				"css": "missing_style.css",
			},
			wantErr: true,
		},
		{
			name: "Valid list of files",
			meta: map[string]interface{}{
				"css": []interface{}{validFile, validFile},
			},
			wantErr: false,
		},
		{
			name: "List with missing file",
			meta: map[string]interface{}{
				"css": []interface{}{validFile, "missing.css"},
			},
			wantErr: true,
		},
		{
			name: "Valid URL",
			meta: map[string]interface{}{
				"include-in-header": "https://example.com/header.html",
			},
			wantErr: false,
		},
		{
			name: "Output dir exists",
			meta: map[string]interface{}{
				"output": filepath.Join(tmpDir, "out.html"),
			},
			wantErr: false,
		},
		{
			name: "Output parent dir missing",
			meta: map[string]interface{}{
				"output": filepath.Join(tmpDir, "subdir", "out.html"),
			},
			wantErr: true,
		},
		{
			name: "Mixed valid URL and missing file in list",
			meta: map[string]interface{}{
				"include-in-header": []interface{}{"https://example.com/header.html", "missing.html"},
			},
			wantErr: true,
		},
		{
			name: "Valid resource-path string",
			meta: map[string]interface{}{
				"resource-path": tmpDir,
			},
			wantErr: false,
		},
		{
			name: "Valid resource-path list",
			meta: map[string]interface{}{
				"resource-path": []interface{}{tmpDir},
			},
			wantErr: false,
		},
		{
			name: "Invalid resource-path string",
			meta: map[string]interface{}{
				"resource-path": "/non/existent/path",
			},
			wantErr: true,
		},
		{
			name: "Invalid resource-path list",
			meta: map[string]interface{}{
				"resource-path": []interface{}{tmpDir, "/non/existent/path"},
			},
			wantErr: true,
		},
		{
			name: "Valid multiple resource-path string",
			meta: map[string]interface{}{
				"resource-path": tmpDir + string(filepath.ListSeparator) + tmpDir,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMetadata(tt.meta, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
