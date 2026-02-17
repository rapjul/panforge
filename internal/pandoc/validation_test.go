package pandoc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateYamlMetadata(t *testing.T) {
	// Parallel tests need their own temp directories or careful setup.
	// We'll move tmpDir creation inside the test loop for isolation.
	tests := []struct {
		name    string
		setup   func(t *testing.T) (map[string]interface{}, string) // returns meta and baseDir (or cleanup dir)
		wantErr bool
	}{
		{
			name: "Valid file",
			setup: func(t *testing.T) (map[string]interface{}, string) {
				d := t.TempDir()
				f := filepath.Join(d, "style.css")
				if err := os.WriteFile(f, []byte("body {}"), 0600); err != nil {
					t.Fatal(err)
				}
				return map[string]interface{}{"css": f}, ""
			},
			wantErr: false,
		},
		{
			name: "Missing file",
			setup: func(t *testing.T) (map[string]interface{}, string) {
				return map[string]interface{}{"css": "missing_style.css"}, ""
			},
			wantErr: true,
		},
		{
			name: "Valid list of files",
			setup: func(t *testing.T) (map[string]interface{}, string) {
				d := t.TempDir()
				f := filepath.Join(d, "style.css")
				if err := os.WriteFile(f, []byte("body {}"), 0600); err != nil {
					t.Fatal(err)
				}
				return map[string]interface{}{"css": []interface{}{f, f}}, ""
			},
			wantErr: false,
		},
		{
			name: "List with missing file",
			setup: func(t *testing.T) (map[string]interface{}, string) {
				d := t.TempDir()
				f := filepath.Join(d, "style.css")
				if err := os.WriteFile(f, []byte("body {}"), 0600); err != nil {
					t.Fatal(err)
				}
				return map[string]interface{}{"css": []interface{}{f, "missing.css"}}, ""
			},
			wantErr: true,
		},
		{
			name: "Valid URL",
			setup: func(t *testing.T) (map[string]interface{}, string) {
				return map[string]interface{}{"include-in-header": "https://example.com/header.html"}, ""
			},
			wantErr: false,
		},
		{
			name: "Output dir exists",
			setup: func(t *testing.T) (map[string]interface{}, string) {
				d := t.TempDir()
				return map[string]interface{}{"output": filepath.Join(d, "out.html")}, ""
			},
			wantErr: false,
		},
		{
			name: "Output parent dir missing",
			setup: func(t *testing.T) (map[string]interface{}, string) {
				d := t.TempDir()
				return map[string]interface{}{"output": filepath.Join(d, "subdir", "out.html")}, ""
			},
			wantErr: true,
		},
		{
			name: "Mixed valid URL and missing file in list",
			setup: func(t *testing.T) (map[string]interface{}, string) {
				return map[string]interface{}{"include-in-header": []interface{}{"https://example.com/header.html", "missing.html"}}, ""
			},
			wantErr: true,
		},
		{
			name: "Valid resource-path string",
			setup: func(t *testing.T) (map[string]interface{}, string) {
				d := t.TempDir()
				return map[string]interface{}{"resource-path": d}, ""
			},
			wantErr: false,
		},
		{
			name: "Valid resource-path list",
			setup: func(t *testing.T) (map[string]interface{}, string) {
				d := t.TempDir()
				return map[string]interface{}{"resource-path": []interface{}{d}}, ""
			},
			wantErr: false,
		},
		{
			name: "Invalid resource-path string",
			setup: func(t *testing.T) (map[string]interface{}, string) {
				return map[string]interface{}{"resource-path": "/non/existent/path"}, ""
			},
			wantErr: true,
		},
		{
			name: "Invalid resource-path list",
			setup: func(t *testing.T) (map[string]interface{}, string) {
				d := t.TempDir()
				return map[string]interface{}{"resource-path": []interface{}{d, "/non/existent/path"}}, ""
			},
			wantErr: true,
		},
		{
			name: "Valid multiple resource-path string",
			setup: func(t *testing.T) (map[string]interface{}, string) {
				d := t.TempDir()
				return map[string]interface{}{"resource-path": d + string(filepath.ListSeparator) + d}, ""
			},
			wantErr: false,
		},
		// New Test Case for Invalid Type in List
		{
			name: "Invalid type in list",
			setup: func(t *testing.T) (map[string]interface{}, string) {
				return map[string]interface{}{"css": []interface{}{"style.css", 123}}, ""
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			meta, _ := tt.setup(t)
			err := ValidateMetadata(meta, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
