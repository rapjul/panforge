// Package main is the entry point for the panforge application.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/rapjul/panforge/internal/app"
	"github.com/rapjul/panforge/internal/options"
	"github.com/rapjul/panforge/internal/pandoc"
	"github.com/rapjul/panforge/internal/utils"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	var opts options.Options

	versionStr := version
	if version == "dev" {
		versionStr = fmt.Sprintf("%s (commit: %s)", version, commit)
	}

	var rootCmd = &cobra.Command{
		Use:     "panforge [flags] <file>",
		Version: versionStr,
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
		SilenceUsage: true, // Don't show usage on runtime errors
		RunE: func(cmd *cobra.Command, args []string) error {
			// Configure Logging
			logLevel := slog.LevelInfo
			if opts.Verbose {
				logLevel = slog.LevelDebug
			} else if opts.Quiet {
				logLevel = slog.LevelError
			}

			handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: logLevel,
			})
			logger := slog.New(handler)
			opts.Logger = logger

			executor := &app.RealExecutor{
				DryRun:  opts.DryRun,
				Verbose: opts.Verbose,
			}
			return app.Run(cmd.Context(), cmd, args, opts, executor)
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
	rootCmd.Flags().StringSliceVarP(&opts.Targets, "to", "t", []string{}, "Specify output format(s)")
	rootCmd.Flags().BoolVarP(&opts.All, "all", "a", false, "Convert to all formats specified in the YAML header (default: false)")
	rootCmd.Flags().StringVarP(&opts.Output, "output", "o", "", "Specify output filename (default: <filename>.<format>)")
	rootCmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Overwrite existing output file(s) (default: false)")
	rootCmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "n", false, "Print the Pandoc command(s) without executing them (default: false)")
	rootCmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Run Pandoc showing output (default: false)")
	rootCmd.Flags().BoolVarP(&opts.Quiet, "quiet", "q", false, "Suppress program messages (default: false)")
	rootCmd.Flags().StringVarP(&opts.Log, "log", "l", "", "Append program calls to FILE (default: none)")
	rootCmd.Flags().IntVarP(&opts.Concurrency, "concurrency", "c", 0, "Limit number of concurrent pandoc processes (default: number of CPUs)")

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
		Use:   "init",
		Short: "Initialize a new project or file",
		Long:  `Generate a default configuration file or a scaffolded Markdown file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunInit(initOpts)
		},
	}
	initCmd.Flags().BoolVar(&initOpts.Config, "config", false, "Generate a default .panforge.yaml config file (default)")
	initCmd.Flags().BoolVarP(&initOpts.Markdown, "markdown", "m", false, "Generate a sample input.md with frontmatter")
	initCmd.Flags().StringSliceVarP(&initOpts.Formats, "to", "t", []string{}, "Specify output formats for the Markdown template (e.g. pdf,html,epub,docx)")
	initCmd.Flags().BoolVarP(&initOpts.Force, "force", "f", false, "Overwrite existing files")

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
		Run: func(cmd *cobra.Command, args []string) {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			_, _ = fmt.Fprintln(w, "Tool\tStatus\tVersion/Path")
			_, _ = fmt.Fprintln(w, "----\t------\t------------")

			check := func(res utils.CheckResult) {
				status := "FOUND"
				if !res.Found {
					status = "MISSING"
				}
				details := res.Version
				if details == "" {
					details = res.Path
				}
				if !res.Found {
					details = res.Error.Error()
				} else if res.Version != "" {
					// Just take the version number part if possible?
					// Pandoc version output is verbose "pandoc 3.1.2 ...", so it's fine.
					if len(details) > 50 {
						details = details[:47] + "..."
					}
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", res.Name, status, details)
			}

			// Determine what to check
			var toolsToCheck []string
			if len(args) > 0 {
				inputFile := args[0]
				var err error
				toolsToCheck, err = app.GetRequiredTools(inputFile, opts)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error analyzing file %s: %v\n", inputFile, err)
					os.Exit(1)
				}
			} else {
				// Default list of all interesting tools
				toolsToCheck = []string{
					"pandoc",
					"typst",
					"pdflatex",
					"xelatex",
					"lualatex",
					"tectonic",
					"wkhtmltopdf",
					"pandoc-crossref",
					"rsvg-convert",
					// "python",
				}
			}

			// deduplicate just in case
			checked := make(map[string]bool)
			for _, tool := range toolsToCheck {
				if checked[tool] {
					continue
				}
				checked[tool] = true

				// Special handling for known tools if we want to call specific Check functions?
				// Actually CheckTool is generic now.
				// But we might want to keep the "Nice Name" mapping or just use tool name.

				// Re-using utils.CheckTool directly for everything is easiest.
				// However, `utils.CheckPDFEngine` etc were just wrappers.
				// Let's just use CheckTool directly.
				check(utils.CheckTool(tool, ""))
			}

			_ = w.Flush()
		},
	}

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(checkCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
