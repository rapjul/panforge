// Package options defines the available command-line flags and arguments.
package options

import "log/slog"

// Options holds CLI flags and runtime configuration.
// It maps command line flags to struct fields.
type Options struct {
	// Inputs specifies one or more input files explicitly via flags.
	Inputs []string `flag:"input" shorthand:"i"`
	// Targets specifies output format(s) to generate.
	Targets []string `flag:"to" shorthand:"t"`
	// Output specifies the target output filename.
	Output string `flag:"output" shorthand:"o"`
	// Force enables overwriting existing files without confirmation prompts.
	Force bool `flag:"force" shorthand:"F"`
	// DryRun prints Pandoc commands without executing them.
	DryRun bool `flag:"dry-run" shorthand:"n"`
	// Verbose enables verbose logging and Pandoc output.
	Verbose bool `flag:"verbose" shorthand:"v"`
	// Quiet suppresses non-error log messages.
	Quiet bool `flag:"quiet" shorthand:"q"`
	// Log specifies a file to append execution logs to.
	Log string `flag:"log" shorthand:"l"`
	// All forces converting to all formats defined in document configuration.
	All bool `flag:"all" shorthand:"a"`
	// Watch monitors input files and configuration for changes.
	Watch bool `flag:"watch" shorthand:"w"`
	// Concurrency limits the number of concurrent Pandoc executions.
	Concurrency int `flag:"concurrency" shorthand:"c"`
	// RelativeOutput resolves relative output paths against CWD instead of the input file directory.
	RelativeOutput bool `flag:"relative-output" shorthand:"r"`
	// JSON configures log output to be formatted in JSON.
	JSON bool `flag:"json"`
	// Logger is the structured logger instance used throughout execution.
	Logger *slog.Logger // Not a flag
}
