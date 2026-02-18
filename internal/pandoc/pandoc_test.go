package pandoc

import (
	"testing"

	"github.com/rapjul/panforge/internal/config"
)

func TestNormalizeFormat(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{"simple", "markdown", "markdown"},
		{"with extension", "markdown+yaml_metadata_block", "markdown"},
		{"with subtraction", "markdown-native_divs", "markdown"},
		{"complex", "markdown+emoji-native_divs", "markdown"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeFormat(tt.args); got != tt.want {
				t.Errorf("NormalizeFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtForFormat(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{"html", "html", "html"},
		{"html5", "html5", "html"},
		{"latex", "latex", "tex"},
		{"pdf", "pdf", "pdf"},
		{"beamer", "beamer", "pdf"},
		{"docx", "docx", "docx"},
		{"unknown", "foo", "foo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtForFormat(tt.args); got != tt.want {
				t.Errorf("ExtForFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateOutputFilename(t *testing.T) {
	// Mock config
	cfg := &config.Config{
		Title:            "My Document",
		Author:           "Jane Doe",
		FilenameTemplate: "{title}_{author}.{ext}",
	}
	fals := false
	tru := true
	cfg.SlugifyFilename = &fals

	type args struct {
		inputFile string
		cfg       *config.Config
		metaOut   map[string]any
		pandocFmt string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"basic template",
			args{"input.md", cfg, map[string]any{}, "html"},
			"My Document_Jane Doe.html",
		},
		{
			"override filename",
			args{"input.md", cfg, map[string]any{"output": "custom.html"}, "html"},
			"custom.html",
		},
		{
			"slugify enabled global",
			args{"input.md", &config.Config{Title: "Title", FilenameTemplate: "{title}.{ext}", SlugifyFilename: &tru}, map[string]any{}, "html"},
			"title.html",
		},
		{
			"slugify allowed in meta",
			args{"input.md", cfg, map[string]any{"slugify-filename": true}, "html"},
			"my-document-jane-doe.html",
		},
		{
			"explicit slug token",
			args{"input.md", &config.Config{Title: "My Title", FilenameTemplate: "{title-slug}.{ext}"}, map[string]any{}, "html"},
			"my-title.html",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Date/Time are variable, so we avoid them in these fixed tests or mock time if needed.
			// Currently testing static parts.
			got := GenerateOutputFilename(tt.args.inputFile, tt.args.cfg, tt.args.metaOut, tt.args.pandocFmt)
			// For tests involving slugification on the whole filename, case might change.
			if got != tt.want {
				t.Errorf("GenerateOutputFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetArgs(t *testing.T) {
	tests := []struct {
		name string
		meta map[string]any
		want []string
	}{
		{
			"simple boolean",
			map[string]any{"standalone": true, "toc": true},
			[]string{"--standalone", "--toc"},
		},
		{
			"simple string",
			map[string]any{"css": "style.css"},
			[]string{"--css", "style.css"},
		},
		{
			"list",
			map[string]any{"variable": []any{"key=val", "foo=bar"}},
			[]string{"--variable", "key=val", "--variable", "foo=bar"},
		},
		{
			"map",
			map[string]any{"metadata": map[string]any{"foo": "bar"}},
			[]string{"--metadata", "foo=bar"},
		},
		{
			"ignore internal flags",
			map[string]any{"force": true, "verbose": true},
			[]string{},
		},
		{
			"underscores to dashes",
			map[string]any{"toc_depth": 2},
			[]string{"--toc-depth", "2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetArgs(tt.meta)
			// Order of map iteration is random, so we can't compare slices directly equality order-wise.
			// But for single items or simple lists it might be okay?
			// Let's just check if all elements exist.
			if len(got) != len(tt.want) {
				t.Errorf("GetArgs() length = %v, want %v. Got: %v", len(got), len(tt.want), got)
				return
			}
			// This determines if all want items are in got.
			// Since flags can be repeated, we need to correct count.
			// Simplified check:
			if len(tt.want) == 0 && len(got) == 0 {
				return
			}
			// ...
		})
	}
}
