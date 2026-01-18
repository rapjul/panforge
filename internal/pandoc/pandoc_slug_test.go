package pandoc_test

import (
	"strings"
	"testing"

	"github.com/rapjul/panforge/internal/config"
	"github.com/rapjul/panforge/internal/pandoc"
	"github.com/rapjul/panforge/internal/utils"
)

func boolPtr(b bool) *bool { return &b }

func TestGenerateOutputFilename(t *testing.T) {
	dateStr := utils.FormatDate()
	// timeStr check is hard because it changes. We'll loose check it.

	cases := []struct {
		name     string
		cfg      *config.Config
		meta     map[string]interface{}
		fmt      string
		expected string // If empty, we check loose matching
	}{
		{
			name: "Default Template",
			cfg: &config.Config{
				Title: "My Title",
			},
			meta:     map[string]interface{}{},
			fmt:      "html",
			expected: "My Title_" + dateStr + ".html",
		},
		{
			name: "SlugifyFilename Implied False",
			cfg: &config.Config{
				Title:           "My Title",
				SlugifyFilename: nil,
			},
			meta:     map[string]interface{}{},
			fmt:      "html",
			expected: "My Title_" + dateStr + ".html",
		},
		{
			name: "SlugifyFilename Explicit True",
			cfg: &config.Config{
				Title:           "My Title",
				SlugifyFilename: boolPtr(true),
			},
			meta:     map[string]interface{}{},
			fmt:      "html",
			expected: "my-title-" + dateStr + ".html", // Dashes because slugify replaces underscores too
		},
		{
			name: "Custom Template",
			cfg: &config.Config{
				Title:            "My Title",
				Author:           "Jane Doe",
				FilenameTemplate: "{author-slug}-{title}.{ext}",
			},
			meta:     map[string]interface{}{},
			fmt:      "pdf",
			expected: "jane-doe-My Title.pdf",
		},
		{
			name: "Variable {title-slug}",
			cfg: &config.Config{
				Title:            "My Title",
				FilenameTemplate: "{title-slug}.{ext}",
			},
			meta:     map[string]interface{}{},
			fmt:      "markdown",
			expected: "my-title.md",
		},
		{
			name: "Variable {ext}",
			cfg: &config.Config{
				Title:            "Doc",
				FilenameTemplate: "file.{ext}",
			},
			meta:     map[string]interface{}{},
			fmt:      "epub",
			expected: "file.epub",
		},
		{
			name: "Sanitization Enforced",
			cfg: &config.Config{
				Title:            "Part 1/2",
				FilenameTemplate: "{title}",
			},
			meta:     map[string]interface{}{},
			fmt:      "html",
			expected: "", // "Part 1_2" (slash replaced)
		},
		{
			name: "Meta Override Slugify",
			cfg: &config.Config{
				Title:           "My Title",
				SlugifyFilename: boolPtr(false),
			},
			meta: map[string]interface{}{
				"slugify-filename": true,
			},
			fmt:      "html",
			expected: "my-title-" + dateStr + ".html", // Dashes
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pandoc.GenerateOutputFilename("input.md", tc.cfg, tc.meta, tc.fmt)

			if tc.expected != "" {
				if got != tc.expected {
					// Slight possibility: sanitization of expected string might vary?
					// But here we constructed simple expected strings.
					// For "Part 1/2", let's handle it manually.
					t.Errorf("expected %q, got %q", tc.expected, got)
				}
			} else {
				// Manual checks for complex cases
				if tc.name == "Sanitization Enforced" {
					if strings.Contains(got, "/") {
						t.Errorf("result %q contained unsafe char /", got)
					}
					if !strings.Contains(got, "Part 1_2") { // Basic check
						t.Errorf("expected sanitized title, got %q", got)
					}
				}
			}
		})
	}
}

func TestGenerateOutputFilename_Time(t *testing.T) {
	// Separate test for time since it varies
	cfg := &config.Config{
		Title:            "Time Test",
		FilenameTemplate: "{time}",
	}
	got := pandoc.GenerateOutputFilename("input.md", cfg, map[string]interface{}{}, "html")
	// Format is 15-04-05 (HH-MM-SS)
	// Validate length and format
	// Should be e.g. 14-30-01.html
	base := strings.TrimSuffix(got, ".html")
	// Expecting 8 chars: XX-XX-XX
	if len(base) != 8 {
		t.Errorf("expected time format length 8, got %q", base)
	}
	if !strings.Contains(base, "-") {
		t.Errorf("expected time format with dashes, got %q", base)
	}
}
