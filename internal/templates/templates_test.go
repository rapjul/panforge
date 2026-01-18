package templates

import (
	"strings"
	"testing"
)

func TestGetConfigTemplate(t *testing.T) {
	tmpl, err := GetConfigTemplate()
	if err != nil {
		t.Fatalf("GetConfigTemplate() returned error: %v", err)
	}
	if tmpl == "" {
		t.Error("GetConfigTemplate() returned empty string")
	}
	// Verify it contains strict YAML content
	if !strings.Contains(tmpl, "title: \"My Project\"") {
		t.Error("GetConfigTemplate() missing expected content 'title: \"My Project\"'")
	}
}

func TestGetScaffoldTemplate(t *testing.T) {
	tmpl, err := GetScaffoldTemplate()
	if err != nil {
		t.Fatalf("GetScaffoldTemplate() returned error: %v", err)
	}
	if tmpl == "" {
		t.Error("GetScaffoldTemplate() returned empty string")
	}
	// Verify it contains markdown header
	// note: checking for {{ . }} or known static content
	if !strings.Contains(tmpl, "title: \"Untitled Document\"") {
		t.Error("GetScaffoldTemplate() missing expected content 'title: \"Untitled Document\"'")
	}
}
