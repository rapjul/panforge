package pandoc_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rapjul/panforge/internal/pandoc"
)

// TestResolveMetadataPaths tests path resolution for metadata maps.
func TestResolveMetadataPaths(t *testing.T) {
	baseDir := "/test/docs"

	meta := map[string]any{
		"css":              "style.css",
		"epub-cover-image": "cover.png",
		"bibliography":     []any{"ref1.bib", "ref2.bib", 123},
		"resource-path":    "img" + string(os.PathListSeparator) + "assets",
		"other-key":        "plain-value",
		"http-url":         "https://example.com/style.css",
		"data-dir":         "/absolute/path/data",
	}

	resolved := pandoc.ResolveMetadataPaths(meta, baseDir)

	// Check css
	if got, ok := resolved["css"].(string); !ok || got != filepath.Join(baseDir, "style.css") {
		t.Errorf("css = %v, want %v", resolved["css"], filepath.Join(baseDir, "style.css"))
	}

	// Check epub-cover-image
	if got, ok := resolved["epub-cover-image"].(string); !ok || got != filepath.Join(baseDir, "cover.png") {
		t.Errorf("epub-cover-image = %v, want %v", resolved["epub-cover-image"], filepath.Join(baseDir, "cover.png"))
	}

	// Check bibliography list
	bibList, ok := resolved["bibliography"].([]any)
	if !ok || len(bibList) != 3 {
		t.Fatalf("bibliography list invalid: %v", resolved["bibliography"])
	}
	if bibList[0] != filepath.Join(baseDir, "ref1.bib") {
		t.Errorf("bibliography[0] = %v, want %v", bibList[0], filepath.Join(baseDir, "ref1.bib"))
	}
	if bibList[1] != filepath.Join(baseDir, "ref2.bib") {
		t.Errorf("bibliography[1] = %v, want %v", bibList[1], filepath.Join(baseDir, "ref2.bib"))
	}
	if bibList[2] != 123 {
		t.Errorf("bibliography[2] = %v, want %v", bibList[2], 123)
	}

	// Check resource-path string
	resPath, ok := resolved["resource-path"].(string)
	if !ok {
		t.Fatalf("resource-path is not string: %v", resolved["resource-path"])
	}
	expectedParts := []string{filepath.Join(baseDir, "img"), filepath.Join(baseDir, "assets")}
	expectedResPath := strings.Join(expectedParts, string(os.PathListSeparator))
	if resPath != expectedResPath {
		t.Errorf("resource-path = %v, want %v", resPath, expectedResPath)
	}

	// Check resource-path as slice
	metaSliceRes := map[string]any{
		"resource-path": []any{"img", "assets"},
	}
	resolvedSliceRes := pandoc.ResolveMetadataPaths(metaSliceRes, baseDir)
	resList, ok := resolvedSliceRes["resource-path"].([]any)
	if !ok || len(resList) != 2 {
		t.Fatalf("resource-path slice invalid: %v", resolvedSliceRes["resource-path"])
	}
	if resList[0] != filepath.Join(baseDir, "img") {
		t.Errorf("resource-path[0] = %v, want %v", resList[0], filepath.Join(baseDir, "img"))
	}

	// Check URL handling
	metaURL := map[string]any{
		"css": "https://cdn.example.com/style.css",
	}
	resolvedURL := pandoc.ResolveMetadataPaths(metaURL, baseDir)
	if resolvedURL["css"] != "https://cdn.example.com/style.css" {
		t.Errorf("URL css = %v, want unchanged URL", resolvedURL["css"])
	}

	// Check absolute path handling
	absPath := "/var/styles/custom.css"
	metaAbs := map[string]any{
		"css": absPath,
	}
	resolvedAbs := pandoc.ResolveMetadataPaths(metaAbs, baseDir)
	if resolvedAbs["css"] != absPath {
		t.Errorf("Absolute css = %v, want %v", resolvedAbs["css"], absPath)
	}

	// Check other non-path key remains unmodified
	if resolved["other-key"] != "plain-value" {
		t.Errorf("other-key = %v, want plain-value", resolved["other-key"])
	}
}
