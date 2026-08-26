package app

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/rapjul/panforge/internal/options"
	"github.com/rapjul/panforge/internal/utils"
)

// DefaultCheckTools defines the standard list of tools checked when no input file is specified.
var DefaultCheckTools = []string{
	ToolPandoc,
	"typst",
	DefaultPDFEngine,
	"xelatex",
	"lualatex",
	"tectonic",
	"wkhtmltopdf",
	"pandoc-crossref",
	"rsvg-convert",
}

// PrintToolCheckRow formats and prints an individual tool dependency check row to the writer.
//
// Parameters:
//   - w: writer where the formatted row is printed (typically a tabwriter)
//   - res: the result of checking an individual tool dependency
//
// Returns:
//   - bool: true if the tool was found, false otherwise
func PrintToolCheckRow(w io.Writer, res utils.CheckResult) bool {
	status := "FOUND"
	if !res.Found {
		status = "MISSING"
	}
	details := res.Version
	if details == "" {
		details = res.Path
	}
	if !res.Found {
		if res.Error != nil {
			details = res.Error.Error()
		} else {
			details = "not found"
		}
	} else if res.Version != "" {
		// Truncate long version strings (e.g. verbose TeX engine banners) to preserve table alignment.
		if len(details) > 50 {
			details = details[:47] + "..."
		}
	}
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", res.Name, status, details)
	return res.Found
}

// RunCheck executes tool dependency checks and prints the formatted results table.
//
// Parameters:
//   - ctx: context for cancellation (unused currently, reserved for async checks)
//   - stdout: destination writer for the tool status table
//   - stderr: destination writer for diagnostic error output
//   - args: optional CLI arguments; if provided, args[0] is inspected for file-specific tool requirements
//   - opts: runtime configuration options passed to the check command
//
// Returns:
//   - error: an error if file analysis fails or if required tools for a specific input file are missing
func RunCheck(ctx context.Context, stdout io.Writer, stderr io.Writer, args []string, opts options.Options) error {
	w := tabwriter.NewWriter(stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "Tool\tStatus\tVersion/Path")
	_, _ = fmt.Fprintln(w, "----\t------\t------------")

	allFound := true

	// Determine what to check
	var toolsToCheck []string
	if len(args) > 0 {
		inputFile := args[0]
		var err error
		toolsToCheck, err = GetRequiredTools(inputFile, opts)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "Error analyzing file %s: %v\n", inputFile, err)
			return err
		}
	} else {
		toolsToCheck = DefaultCheckTools
	}

	// Deduplicate tools to check
	checked := make(map[string]bool)
	for _, tool := range toolsToCheck {
		if checked[tool] {
			continue
		}
		checked[tool] = true
		if !PrintToolCheckRow(w, utils.CheckTool(tool, "")) {
			allFound = false
		}
	}

	_ = w.Flush()

	if len(args) > 0 && !allFound {
		return fmt.Errorf("missing required dependencies for %s", args[0])
	}

	return nil
}
