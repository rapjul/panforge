package options

import "log/slog"

// Options holds CLI flags and runtime configuration.
// It maps command line flags to struct fields.
type Options struct {
	Targets     []string     `flag:"to" shorthand:"t"`
	Output      string       `flag:"output" shorthand:"o"`
	Force       bool         `flag:"force" shorthand:"f"`
	DryRun      bool         `flag:"dry-run" shorthand:"n"`
	Verbose     bool         `flag:"verbose" shorthand:"v"`
	Quiet       bool         `flag:"quiet" shorthand:"q"`
	Log         string       `flag:"log" shorthand:"l"`
	All         bool         `flag:"all" shorthand:"a"`
	Watch       bool         `flag:"watch" shorthand:"w"`
	Concurrency int          `flag:"concurrency" shorthand:"c"`
	Logger      *slog.Logger // Not a flag
}
