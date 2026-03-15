// Package ui provides user interface utilities and logging handlers.
package ui

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// PrettyHandler is a slog.Handler that prints human-readable output.
type PrettyHandler struct {
	w        io.Writer
	mu       sync.Mutex
	minLevel slog.Level
}

// NewPrettyHandler creates a new PrettyHandler.
func NewPrettyHandler(minLevel slog.Level) *PrettyHandler {
	return &PrettyHandler{
		w:        os.Stderr,
		minLevel: minLevel,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.minLevel
}

// Handle handles the Record.
func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Special handling for Errors
	if r.Level >= slog.LevelError {
		msg := r.Message
		// If we have an "error" attribute, append it
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "error" {
				msg = fmt.Sprintf("%s: %v", msg, a.Value)
			}
			return true
		})
		_, _ = fmt.Fprintf(h.w, "%s %s\n", ErrorStyle.Render("ERROR"), msg)
		return nil
	}

	// Scan attributes for command/args or config
	var cmdStr string
	var cmdArgs []string
	var configFound bool
	var isDryRun bool

	r.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "args":
			valAny := a.Value.Any()
			if val, ok := valAny.([]string); ok {
				cmdArgs = val
			} else if valSlice, ok := valAny.([]any); ok {
				// slog might convert []string to []any
				for _, v := range valSlice {
					cmdArgs = append(cmdArgs, fmt.Sprintf("%v", v))
				}
			}
		case "command":
			cmdStr = a.Value.String()
		case "config":
			configFound = true
			if r.Level == slog.LevelDebug {
				_, _ = fmt.Fprintln(h.w, KeyStyle.Render(r.Message))
				// Assume value is a map
				if v, ok := a.Value.Any().(map[string]any); ok {
					renderMap(h.w, v, 1)
				}
			}
		case "dry_run":
			if a.Value.Kind() == slog.KindBool && a.Value.Bool() {
				isDryRun = true
			}
		}
		return true
	})

	// Determine what to render
	var cmdToRender any
	if len(cmdArgs) > 0 {
		cmdToRender = cmdArgs
	} else if cmdStr != "" {
		cmdToRender = cmdStr
	}

	if cmdToRender != nil {
		if isDryRun {
			_, _ = fmt.Fprintln(h.w, DryRunStyle.Render("DRY RUN: The following command will not be executed:"))
		}
		rendered := renderCommand(cmdToRender)
		_, _ = fmt.Fprintln(h.w, rendered)
		return nil
	}

	if configFound {
		return nil
	}

	// Default fallback: just print message
	_, _ = fmt.Fprintln(h.w, r.Message)
	return nil
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // Not implementing attrs state for this simple CLI
}

// WithGroup returns a new Handler with the given group name appended to
// the receiver's existing groups.
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return h // Not implementing groups
}

// renderCommand parses a command string (or slice) and styles it with wrapping.
func renderCommand(cmd any) string {
	var parts []string
	switch v := cmd.(type) {
	case []string:
		parts = v
	case string:
		// Fallback for string input (legacy or other logs)
		parts = strings.Fields(v)
	default:
		return fmt.Sprintf("%v", cmd)
	}

	if len(parts) == 0 {
		return ""
	}

	// Get terminal width
	//nolint:gosec,nolintlint // Fd() result is always a valid int
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		width = 80 // Default fallback
	}

	var sb strings.Builder
	// Command always first
	cmdStr := CommandStyle.Render(parts[0])
	sb.WriteString(cmdStr)

	currentLineLen := lipgloss.Width(cmdStr)
	indent := "  "

	// Iterate through remaining parts
	for i := 1; i < len(parts); i++ {
		part := parts[i]

		// Sticky Flag Logic: Look ahead
		var nextPart string
		var useNext bool

		if strings.HasPrefix(part, "-") && i+1 < len(parts) {
			nextPart = parts[i+1]
			// heuristic: if next part is NOT a flag, keep them together
			if !strings.HasPrefix(nextPart, "-") {
				useNext = true
			}
		}

		// Function to style a single part
		stylePart := func(p string) string {
			switch {
			case strings.HasPrefix(p, "-"):
				return FlagStyle.Render(p)
			case isPath(p):
				return PathStyle.Render(p)
			default:
				return ValueStyle.Render(p)
			}
		}

		styledPart := stylePart(part)
		partLen := lipgloss.Width(styledPart) + 1 // + space

		if useNext {
			styledNext := stylePart(nextPart)
			totalLen := partLen + lipgloss.Width(styledNext) + 1 // + space for next

			// Check wrap for the chunk
			if currentLineLen+totalLen > width {
				sb.WriteString(" \\") // Shell continuation
				sb.WriteString("\n")
				sb.WriteString(indent)
				currentLineLen = len(indent)
			} else {
				sb.WriteString(" ")
				currentLineLen++
			}

			sb.WriteString(styledPart)
			sb.WriteString(" ")
			sb.WriteString(styledNext)
			currentLineLen += lipgloss.Width(styledPart) + 1 + lipgloss.Width(styledNext)
			i++ // Skip next iteration
		} else {
			// Normal wrapping
			if currentLineLen+partLen > width {
				sb.WriteString(" \\") // Shell continuation
				sb.WriteString("\n")
				sb.WriteString(indent)
				currentLineLen = len(indent)
			} else {
				sb.WriteString(" ")
				currentLineLen++
			}
			sb.WriteString(styledPart)
			currentLineLen += lipgloss.Width(styledPart)
		}
	}

	return sb.String()
}

func isPath(s string) bool {
	// Clean quotes
	clean := strings.Trim(s, "\"'")

	// Simple heuristics
	if strings.Contains(clean, "/") || strings.Contains(clean, "\\") {
		return true
	}

	ext := strings.ToLower(clean)
	// Check for common extensions in this context
	commonExts := []string{".md", ".html", ".pdf", ".docx", ".epub", ".tex", ".yaml", ".yml", ".json"}
	for _, e := range commonExts {
		if strings.HasSuffix(ext, e) {
			return true
		}
	}
	return false
}

func renderMap(w io.Writer, m map[string]any, indentLevel int) {
	indent := strings.Repeat("  ", indentLevel)
	for k, v := range m {
		_, _ = fmt.Fprintf(w, "%s%s: ", indent, KeyStyle.Render(k))
		switch val := v.(type) {
		case map[string]any:
			_, _ = fmt.Fprintln(w)
			renderMap(w, val, indentLevel+1)
		case []any:
			_, _ = fmt.Fprintln(w)
			for _, item := range val {
				_, _ = fmt.Fprintf(w, "%s  - %v\n", indent, item)
			}
		default:
			_, _ = fmt.Fprintf(w, "%v\n", StringValStyle.Render(fmt.Sprintf("%v", val)))
		}
	}
}
