package utils

import "testing"

func TestSanitizeInternal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		osName   string
		expected string
	}{
		// Windows tests
		{"Windows: simple", "file.txt", "windows", "file.txt"},
		{"Windows: bad chars", "foo<bar>.txt", "windows", "foo_bar_.txt"},
		{"Windows: colon", "C:file.txt", "windows", "C_file.txt"},
		{"Windows: slash", "path/to/file", "windows", "path_to_file"},
		{"Windows: reserved CON", "CON", "windows", "_"},
		{"Windows: reserved LPT1", "LPT1", "windows", "_"},
		{"Windows: reserved lower", "aux", "windows", "_"}, // sanitize doesn't lower case before check? User code logic check.

		// Darwin tests
		{"Darwin: simple", "file.txt", "darwin", "file.txt"},
		{"Darwin: colon", "file:name.txt", "darwin", "file_name.txt"},
		{"Darwin: slash", "foo/bar", "darwin", "foo_bar"}, // Slash is always bad

		// Linux/Other tests
		{"Linux: simple", "file.txt", "linux", "file.txt"},
		{"Linux: colon allowed", "file:name.txt", "linux", "file:name.txt"},
		{"Linux: slash", "foo/bar", "linux", "foo_bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitize(tt.input, tt.osName)
			if got != tt.expected {
				t.Errorf("sanitize(%q, %q) = %q; want %q", tt.input, tt.osName, got, tt.expected)
			}
		})
	}
}
