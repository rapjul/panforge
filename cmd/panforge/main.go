// Package main is the entry point for the panforge application.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rapjul/panforge/internal/app"
	"github.com/rapjul/panforge/internal/options"
	"github.com/rapjul/panforge/internal/pandoc"
	"github.com/rapjul/panforge/internal/ui"
)

var (
	version = "dev"
	commit  = "none"
)

// formatVersion formats the version identifier with optional commit hash for development builds.
//
// Parameters:
//   - v: base version string
//   - commit: commit hash or identifier
//
// Returns:
//   - string: formatted version string
func formatVersion(v, commit string) string {
	if v == "dev" {
		return fmt.Sprintf("%s (commit: %s)", v, commit)
	}
	return v
}

// newRootCmd builds the root cobra.Command along with its options and subcommands.
func newRootCmd() (*cobra.Command, *options.Options) {
	opts := &options.Options{}

	var rootCmd = &cobra.Command{
		Use:     "panforge [flags] <file>",
		Version: formatVersion(version, commit),
		Short:   "A wrapper for pandoc with complex configurations",
		Long: `panforge enables complex Pandoc conversions using a YAML configuration
  and metadata in the Markdown document's frontmatter.

To generate shell completion scripts, run:
  panforge completion [bash|zsh|fish|powershell]`,
		Example: `  # Normal usage
  panforge input.md

  # Pass flags directly to Pandoc (using --)
  # This serves to distinguish between flags for panforge and flags for pandoc itself.
  panforge input.md -- --from markdown --to html5

  # Dry run to see the generated command
  panforge input.md --dry-run`,
		SilenceUsage:  true, // Don't show usage on runtime errors
		SilenceErrors: true, // Don't print errors automatically, we handle them
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Logger = ui.NewLogger(opts.Verbose, opts.Quiet, opts.JSON)

			executor := &app.RealExecutor{
				DryRun:  opts.DryRun,
				Verbose: opts.Verbose,
			}
			return app.Run(cmd.Context(), cmd, args, *opts, executor)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return []string{"md", "markdown"}, cobra.ShellCompDirectiveFilterFileExt
		},
		Args: cobra.ArbitraryArgs,
	}

	// Define flags
	rootCmd.Flags().StringSliceVarP(&opts.Inputs, "input", "i", []string{}, "Specify input file(s)")
	rootCmd.Flags().StringSliceVarP(&opts.Targets, "to", "t", []string{}, "Specify output format(s)")
	rootCmd.Flags().StringSliceVar(&opts.Targets, "target", []string{}, "Alias for --to")
	rootCmd.Flags().BoolVarP(&opts.All, "all", "a", false, "Convert to all formats specified in the YAML header (default: false)")
	rootCmd.Flags().StringVarP(&opts.Output, "output", "o", "", "Specify output filename (default: <filename>.<format>)")
	rootCmd.Flags().BoolVarP(&opts.Force, "force", "F", false, "Overwrite existing output file(s) (default: false)")
	rootCmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "n", false, "Print the Pandoc command(s) without executing them (default: false)")
	rootCmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Run Pandoc showing output (default: false)")
	rootCmd.Flags().BoolVarP(&opts.Quiet, "quiet", "q", false, "Suppress program messages (default: false)")
	rootCmd.Flags().StringVarP(&opts.Log, "log", "l", "", "Append program calls to FILE (default: none)")
	rootCmd.Flags().BoolVar(&opts.JSON, "json", false, "Output logs in JSON format")
	rootCmd.Flags().IntVarP(&opts.Concurrency, "concurrency", "c", 0, "Limit number of concurrent pandoc processes (default: number of CPUs)")
	rootCmd.Flags().BoolVarP(&opts.RelativeOutput, "relative-output", "r", false, "Resolve relative output paths against CWD instead of input file directory (alias: --relative-to-cwd)")
	rootCmd.Flags().BoolVar(&opts.RelativeOutput, "relative-to-cwd", false, "Alias for --relative-output")
	_ = rootCmd.Flags().MarkHidden("relative-to-cwd")

	rootCmd.Flags().BoolVarP(&opts.Watch, "watch", "w", false, "Watch input file for changes and re-run (implies --force for overwriting existing output file(s))")

	// Disable auto-sorting of flags to preserve order of post-args if mixed
	rootCmd.Flags().SortFlags = false

	// Register completion for --watch/-w flag
	_ = rootCmd.RegisterFlagCompletionFunc("watch", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})

	// Register completion for --to/-t flag
	_ = rootCmd.RegisterFlagCompletionFunc("to", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		formats, err := pandoc.GetSupportedFormats()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return formats, cobra.ShellCompDirectiveNoFileComp
	})

	// Init Command
	var initOpts app.InitOptions
	var initCmd = &cobra.Command{
		Use:   "init [filename]",
		Short: "Initialize a new project or file",
		Long:  `Generate a default configuration file or a scaffolded Markdown file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				initOpts.Filename = args[0]
			}
			return app.RunInit(initOpts)
		},
	}
	initCmd.Flags().BoolVar(&initOpts.Config, "config", false, "Generate a default .panforge.yaml config file (default)")
	initCmd.Flags().BoolVarP(&initOpts.Markdown, "markdown", "m", false, "Generate a sample input.md with frontmatter")
	initCmd.Flags().StringSliceVarP(&initOpts.Formats, "to", "t", []string{}, "Specify output formats for the Markdown template (e.g. pdf,html,epub,docx)")
	initCmd.Flags().BoolVarP(&initOpts.Force, "force", "F", false, "Overwrite existing files")

	_ = initCmd.RegisterFlagCompletionFunc("to", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return app.KnownFormats, cobra.ShellCompDirectiveNoFileComp
	})

	// Check Command
	var checkCmd = &cobra.Command{
		Use:   "check [file]",
		Short: "Check for installed dependencies",
		Long: `Check for installed dependencies.
If a file is provided, it checks only for the tools required by that file's configuration.
If no file is provided, it checks for all known tools.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunCheck(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), args, *opts)
		},
	}

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(checkCmd)

	return rootCmd, opts
}

// main is the entry point for the panforge application.
func main() {
	rootCmd, opts := newRootCmd()

	if err := rootCmd.Execute(); err != nil {
		if opts.Logger != nil {
			opts.Logger.Error(err.Error())
		} else {
			// Fallback if logger initialization failed (unlikely)
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
