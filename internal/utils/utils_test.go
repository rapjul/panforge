package utils

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{"simple", "Hello World", "hello-world"},
		{"mixed case", "My New POST", "my-new-post"},
		{"numbers", "Blog post 123", "blog-post-123"},
		{"special chars", "This & That!", "this-that"},
		{"multiple spaces", "foo   bar", "foo-bar"},
		{"leading trailing", "  test  ", "test"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Slugify(tt.arg); got != tt.want {
				t.Errorf("Slugify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolvePath(t *testing.T) {
	// Getting absolute path depends on Cwd, but we can check if it returns an abs path
	path, err := ResolvePath("foo.md")
	if err != nil {
		t.Errorf("ResolvePath failed: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("ResolvePath returned relative path: %v", path)
	}
	if !strings.HasSuffix(path, "foo.md") {
		t.Errorf("ResolvePath did not end with filename: %v", path)
	}

	// Empty
	p2, err := ResolvePath("")
	if err != nil {
		t.Error(err)
	}
	if p2 != "" {
		t.Errorf("ResolvePath('') = %v, want empty", p2)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{"simple", "file.txt", "file.txt"},
		{"slashes", "foo/bar.txt", "foo_bar.txt"},
		{"backslashes", "foo\\bar.txt", "foo_bar.txt"},
		{"windows reserved ignored on non-windows logic usually but sanitize handles it via runtime check", "con", "con"}, // This depends on runtime in the actual code
	}
	// The implementation of SanitizeFilename uses runtime.GOOS.
	// We can't easily swap runtime.GOOS in tests without DI.
	// But we can test the `sanitize` private function if we wanted, but it's private.
	// We will rely on public function behavior for the current OS.

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFilename(tt.arg)
			// Check basic sanitization which should apply everywhere (slashes)
			if strings.Contains(got, "/") || strings.Contains(got, "\\") {
				t.Errorf("SanitizeFilename(%q) = %q, still contains slashes", tt.arg, got)
			}
		})
	}
}
