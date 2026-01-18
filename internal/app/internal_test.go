package app

import (
	"testing"

	"github.com/rapjul/panforge/internal/config"
	"github.com/rapjul/panforge/internal/options"
)

func TestDetermineTargets(t *testing.T) {
	tests := []struct {
		name     string
		opts     options.Options
		cfg      *config.Config
		expected []string
	}{
		{
			name: "CLI targets override everything",
			opts: options.Options{Targets: []string{"pdf", "docx"}},
			cfg: &config.Config{
				Outputs: []interface{}{"html"},
			},
			expected: []string{"pdf", "docx"},
		},
		{
			name: "Config outputs list used if no CLI targets",
			opts: options.Options{},
			cfg: &config.Config{
				Outputs: []interface{}{"html", "epub"},
			},
			expected: []string{"html", "epub"},
		},
		{
			name: "Config output map used if no outputs list",
			opts: options.Options{},
			cfg: &config.Config{
				OutputMap: map[string]interface{}{
					"pdf":  nil,
					"docx": nil,
				},
			},
			// determineTargets sorts map keys
			expected: []string{"docx", "pdf"},
		},
		{
			name:     "Fallback to html",
			opts:     options.Options{},
			cfg:      &config.Config{},
			expected: []string{"html"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineTargets(tt.opts, tt.cfg)
			if len(got) != len(tt.expected) {
				t.Errorf("determineTargets() length = %v, want %v", len(got), len(tt.expected))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("determineTargets()[%d] = %v, want %v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestIsOverwriteAllowed(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		metaOut map[string]interface{}
		want    bool
	}{
		{
			name:    "Default false",
			cfg:     &config.Config{},
			metaOut: map[string]interface{}{},
			want:    false,
		},
		{
			name: "Target specific true",
			cfg:  &config.Config{},
			metaOut: map[string]interface{}{
				"overwrite": true,
			},
			want: true,
		},
		{
			name: "Target specific false",
			cfg: &config.Config{
				Generic: map[string]interface{}{"overwrite": true},
			},
			metaOut: map[string]interface{}{
				"overwrite": false,
			},
			// Current logic: checks target first. If target differs?
			// The code says:
			// if target has it, return true if true.
			// if global has it, return true if true.
			// It implies that if target is explicit FALSE, it might still return TRUE if generic is TRUE?
			// Let's check code:
			// if v, ok := metaOut["overwrite"]; ok { if b { return true } }
			// This means if metaOut["overwrite"] is FALSE, it continues to check global.
			// So "overwrite: false" in target DOES NOT overload "overwrite: true" in global.
			// This logic promotes "allow overwrite", i.e. strict OR.
			want: true,
		},
		{
			name: "Global true",
			cfg: &config.Config{
				Generic: map[string]interface{}{"overwrite": true},
			},
			metaOut: map[string]interface{}{},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOverwriteAllowed(tt.cfg, tt.metaOut); got != tt.want {
				t.Errorf("isOverwriteAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}
