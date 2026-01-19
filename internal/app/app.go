// Package app implements the core application logic for panforge.
package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/rapjul/panforge/internal/config"
	"github.com/rapjul/panforge/internal/options"
	"github.com/rapjul/panforge/internal/pandoc"
	"github.com/rapjul/panforge/internal/utils"
)

// CommandExecutor abstracts command execution for testing purposes.
// It allows mocking the actual os/exec calls in unit tests.
type CommandExecutor interface {
	Run(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error
}

// RealExecutor implements CommandExecutor using os/exec.
// It handles actual system command execution.
type RealExecutor struct {
	// DryRun indicates if the command should be printed instead of executed.
	DryRun bool
	// Verbose indicates if the command should be logged behavior details.
	Verbose bool
}

// Run executes a system command using os/exec.
//
// Parameters:
//   - `ctx`: context for cancellation
//   - `name`: command name
//   - `args`: command arguments
//   - `stdout`: writer for standard output
//   - `stderr`: writer for standard error
func (e *RealExecutor) Run(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	if e.DryRun {
		return nil
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// Options holds CLI flags
// Moved to internal/options

// Run is the main execution logic for the panforge application.
//
// Parameters:
//   - `ctx`: context for cancellation
//   - `cmd`: the cobra command being executed
//   - `args`: command line arguments
//   - `opts`: parsed command line flags
//   - `executor`: interface for running system commands
func Run(ctx context.Context, cmd *cobra.Command, args []string, opts options.Options, executor CommandExecutor) error {
	// 1. Parse Input File
	inputFile, postArgs := parseArgs(args)
	if inputFile == "" {
		if len(opts.Targets) > 0 || opts.Output != "" {
			return fmt.Errorf("no input file found")
		}
		return cmd.Help()
	}

	// Resolve input file path (if not stdin)
	if inputFile != "-" {
		resolvedInput, err := utils.ResolvePath(inputFile)
		if err != nil {
			return fmt.Errorf("failed to resolve input file path: %w", err)
		}
		inputFile = resolvedInput
	}

	// Handle stdin input
	if inputFile == "-" {
		tmpFile, err := os.CreateTemp("", "panforge-stdin-*.md")
		if err != nil {
			return fmt.Errorf("failed to create temp file for stdin: %w", err)
		}
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		if _, err := io.Copy(tmpFile, cmd.InOrStdin()); err != nil {
			_ = tmpFile.Close()
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			return fmt.Errorf("failed to close temp file: %w", err)
		}
		inputFile = tmpFile.Name()
	}

	// 2. Initial Config Loading & Execution
	// If watch mode is enabled, we'll hand off to the Watcher (implemented elsewhere).
	// For now, let's just call Process once if Watch is false.

	// Determine default config path for watching
	defaultConfigPath, _, _ := config.LoadDefaultConfig("default")

	if opts.Watch {
		return Watch(ctx, inputFile, defaultConfigPath, postArgs, opts, executor)
	}

	return Process(ctx, inputFile, postArgs, opts, executor)
}

// Process handles a single run of the conversion logic.
//
// Parameters:
//   - `ctx`: context for cancellation
//   - `inputFile`: path to the markdown file to convert
//   - `postArgs`: additional arguments to pass to pandoc
//   - `opts`: configuration options
//   - `executor`: used to run the pandoc command
//
//nolint:gocyclo // Code is complex but manageable; refactoring deferred
func Process(ctx context.Context, inputFile string, postArgs []string, opts options.Options, executor CommandExecutor) error {
	// 2. Initial Config Loading
	formats, err := pandoc.GetSupportedFormats()
	if err != nil {
		return fmt.Errorf("failed to get supported formats: %w", err)
	}
	if len(formats) == 0 {
		return fmt.Errorf("pandoc not found. Please install it from https://pandoc.org/installing.html")
	}

	_, cfg, err := config.LoadConfig(inputFile)
	if err != nil {
		// If config loading fails (e.g. no YAML header), we only proceed if
		// the user explicitly provided targets via CLI args.
		if len(opts.Targets) == 0 {
			return fmt.Errorf("input file has no valid YAML header and no target format specified: %w", err)
		}
		// Proceed with empty config if interactive/CLI targets are present
		cfg = &config.Config{}
	}

	_, defaultCfg, _ := config.LoadDefaultConfig("default")
	if defaultCfg != nil {
		if cfg.Title == "" {
			cfg.Title = defaultCfg.Title
		}
		if cfg.FilenameTemplate == "" {
			cfg.FilenameTemplate = defaultCfg.FilenameTemplate
		}
		if cfg.SlugifyFilename == nil {
			cfg.SlugifyFilename = defaultCfg.SlugifyFilename
		}
		if cfg.OutputMap == nil {
			cfg.OutputMap = defaultCfg.OutputMap
		} else {
			for k, v := range defaultCfg.OutputMap {
				if _, exists := cfg.OutputMap[k]; !exists {
					cfg.OutputMap[k] = v
				}
			}
		}
		if cfg.Generic == nil {
			cfg.Generic = make(map[string]interface{})
		}
		if defaultCfg.Generic != nil {
			for k, v := range defaultCfg.Generic {
				if _, exists := cfg.Generic[k]; !exists {
					cfg.Generic[k] = v
				}
			}
		}
	}

	// 3. Determine Targets
	targets := DetermineTargets(opts, cfg)

	// 4. Process Each Target
	g, ctx := errgroup.WithContext(ctx)

	// Semaphore to limit concurrency
	limit := int64(opts.Concurrency)
	if limit <= 0 {
		limit = int64(runtime.NumCPU())
	}
	sem := semaphore.NewWeighted(limit)

	var logMu sync.Mutex
	var promptMu sync.Mutex
	var logFile *os.File
	if opts.Log != "" {
		var err error
		logFile, err = os.OpenFile(opts.Log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) //nolint:gosec // 0644 is standard for logs
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		defer func() { _ = logFile.Close() }()
	}

	for _, t := range targets {
		t := t // capture loop variable
		g.Go(func() error {
			if err := sem.Acquire(ctx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			// Resolve Format
			fmtStr := pandoc.NormalizeFormat(t)
			// Check if t maps to an output entry in YAML
			var metaOut map[string]interface{}

			// Logic to find specific output config in YAML:
			// logic similar to ruby resolve_target_format
			if val, ok := cfg.OutputMap[t]; ok {
				if m, ok := val.(map[string]interface{}); ok {
					metaOut = m
					if to, ok := m["to"].(string); ok && to != "" {
						fmtStr = to
					}
				}
			} else if val, ok := cfg.Generic[t]; ok {
				if m, ok := val.(map[string]interface{}); ok {
					metaOut = m
				}
			}

			if metaOut == nil {
				metaOut = make(map[string]interface{})
			}

			// Generate Output Filename
			outputFile := opts.Output
			if outputFile == "" {
				outputFile = pandoc.GenerateOutputFilename(inputFile, cfg, metaOut, fmtStr)
			}

			// Resolve output file path
			resolvedOutput, err := utils.ResolvePath(outputFile)
			if err != nil {
				return fmt.Errorf("failed to resolve output file path: %w", err)
			}
			outputFile = resolvedOutput

			// Check overwrite
			if _, err := os.Stat(outputFile); err == nil {
				// If watch mode is on, we implicitly allow overwrite (otherwise it would block execution)
				if !opts.Force && !opts.Watch && !isOverwriteAllowed(cfg, metaOut) {
					// Ask for confirmation
					promptMu.Lock()
					overwrite := askForConfirmation(outputFile, os.Stdin, os.Stderr)
					promptMu.Unlock()

					if !overwrite {
						// Log that we are skipping to avoid aborting other targets in the errgroup
						if opts.Logger != nil {
							opts.Logger.Warn("skipping target", "file", outputFile, "reason", "already exists and overwrite declined")
						} else {
							fmt.Fprintf(os.Stderr, "Skipping %s: file already exists and overwrite was declined\n", outputFile)
						}
						return nil
					}
				}
			}

			// Build Command
			pandocArgs := []string{inputFile}
			pandocArgs = append(pandocArgs, "--to", fmtStr)
			pandocArgs = append(pandocArgs, "--output", outputFile)

			// Add YAML args
			pandocArgs = append(pandocArgs, pandoc.GetArgs(metaOut)...)

			// Add CLI args that were passed after inputs or generically
			// (Note: this logic is simplified compared to Ruby's careful flag stripping)
			for i := 0; i < len(postArgs); i++ {
				arg := postArgs[i]
				if arg == "-t" {
					postArgs[i] = "--to"
				}
			}
			pandocArgs = append(pandocArgs, postArgs...)

			// Execute
			// Improve logging to show quoted arguments
			var quotedArgs []string
			for _, arg := range pandocArgs {
				if strings.Contains(arg, " ") || strings.Contains(arg, "\"") {
					quotedArgs = append(quotedArgs, fmt.Sprintf("%q", arg))
				} else {
					quotedArgs = append(quotedArgs, arg)
				}
			}
			cmdStr := "pandoc " + strings.Join(quotedArgs, " ")

			// Log execution
			// We use Info level. If --quiet is set, logger should be configured to Error level only.
			if opts.Logger != nil {
				opts.Logger.Info("executing command", "command", cmdStr)
			} else if !opts.Quiet {
				// Fallback if no logger validation
				fmt.Printf("panforge calling: %s\n", cmdStr)
			}

			if logFile != nil {
				logMu.Lock()
				_, _ = fmt.Fprintf(logFile, "panforge calling: %s\n", cmdStr)
				logMu.Unlock()
			}

			// Use executor
			// Note: Writing to os.Stdout/Stderr concurrently might interleave output
			if err := executor.Run(ctx, "pandoc", pandocArgs, os.Stdout, os.Stderr); err != nil {
				return fmt.Errorf("pandoc failed: %w", err)
			}
			return nil
		})
	}

	return g.Wait()
}

// parseArgs determines the input file from the command line arguments.
//
// Parameters:
//   - `args`: command line arguments
//
// Returns:
//   - string: input filename
//   - []string: remaining arguments
func parseArgs(args []string) (string, []string) {
	// heuristics to find input file (first non-flag arg?)
	// Cobra strips flags defined on it, so args here are non-flag args.
	for i, arg := range args {
		// Allow "-" as input file (stdin)
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			inputFile := arg
			postArgs := args[i+1:]
			return inputFile, postArgs
		}
	}
	return "", []string{}
}

// DetermineTargets figures out which output formats to build.
//
// Parameters:
//   - `opts`: CLI targets
//   - `cfg`: YAML configuration from the file
//
// It prioritizes CLI targets > 'outputs' list in YAML > 'output' map in YAML > Default "html".
func DetermineTargets(opts options.Options, cfg *config.Config) []string {
	if len(opts.Targets) > 0 {
		return opts.Targets
	}
	// User clarification: "It is all formats in the YAML header metadata block at the top of the input Markdown file."
	// This means if --all is passed (or default behavior), we should look at 'outputs' and 'output' in the YAML.

	// Check if 'outputs' list is defined
	if len(cfg.Outputs) > 0 {
		var targets []string
		for _, v := range cfg.Outputs {
			if s, ok := v.(string); ok {
				targets = append(targets, s)
			}
		}
		return targets
	}

	// Check if 'output' map is defined
	if len(cfg.OutputMap) > 0 {
		var targets []string
		for k := range cfg.OutputMap {
			targets = append(targets, k)
		}
		// Sort for deterministic order
		sort.Strings(targets)
		return targets
	}

	// Fallback to auto detection or default
	return []string{"html"}
}

// isOverwriteAllowed checks if overwrite is explicitly allowed in configuration.
//
// Parameters:
//   - `cfg`: the global config
//   - `metaOut`: the format-specific config
func isOverwriteAllowed(cfg *config.Config, metaOut map[string]interface{}) bool {
	// Check specific target config
	if v, ok := metaOut["overwrite"]; ok {
		if b, ok := v.(bool); ok && b {
			return true
		}
	}
	// Check global config
	if v, ok := cfg.Generic["overwrite"]; ok {
		if b, ok := v.(bool); ok && b {
			return true
		}
	}
	return false
}

// askForConfirmation prompts the user for yes/no confirmation.
//
// Parameters:
//   - `filename`: the file being overwritten
//   - `r`: the input reader (usually stdin)
//   - `w`: the output writer (usually stderr)
func askForConfirmation(filename string, r io.Reader, w io.Writer) bool {
	_, _ = fmt.Fprintf(w, "File '%s' already exists. Overwrite? [y/N]: ", filename)

	reader := bufio.NewReader(r)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// GetRequiredTools determines which tools are needed for the given input file.
//
// Parameters:
//   - `inputFile`: path to the input markdown file
//   - `opts`: runtime options
//
// It returns a list of tool names that should be checked (e.g. "pandoc", "pdflatex").
//
//nolint:gocyclo // Code is complex but manageable; refactoring deferred
func GetRequiredTools(inputFile string, opts options.Options) ([]string, error) {
	required := []string{"pandoc"}

	// If no input file, return basic set + all known engines?
	// The requirement was: if no file, check all.
	if inputFile == "" {
		// Callers responsibility to handle empty input file case for "check all" behavior
		return required, nil
	}

	// Load config
	// We might fail to resolve path here if it doesn't exist, but let's try
	resolvedInput, err := utils.ResolvePath(inputFile)
	if err == nil {
		inputFile = resolvedInput
	}

	// We use LoadConfig logic. Capture error but proceed if possible (logic similar to Process)
	_, cfg, err := config.LoadConfig(inputFile)
	if err != nil {
		// If we can't load config, we can't determine specific tools, just return base
		return required, nil
	}

	// Load default config to fill in gaps if possible, mostly for output map
	_, defaultCfg, _ := config.LoadDefaultConfig("default")
	if defaultCfg != nil {
		if cfg.OutputMap == nil {
			cfg.OutputMap = defaultCfg.OutputMap
		} else {
			for k, v := range defaultCfg.OutputMap {
				if _, exists := cfg.OutputMap[k]; !exists {
					cfg.OutputMap[k] = v
				}
			}
		}
		if cfg.Generic == nil {
			cfg.Generic = make(map[string]interface{})
		}
		if defaultCfg.Generic != nil {
			for k, v := range defaultCfg.Generic {
				if _, exists := cfg.Generic[k]; !exists {
					cfg.Generic[k] = v
				}
			}
		}
	}

	targets := DetermineTargets(opts, cfg)

	hasTypst := false

	for _, t := range targets {
		// Normalize format
		fmtStr := pandoc.NormalizeFormat(t)

		// Check for overrides in config to fully resolve format (e.g. target "paper" might be "latex" or "typst")
		var metaOut map[string]interface{}
		if val, ok := cfg.OutputMap[t]; ok {
			if m, ok := val.(map[string]interface{}); ok {
				metaOut = m
				if to, ok := m["to"].(string); ok && to != "" {
					fmtStr = to
				}
			}
		} else if val, ok := cfg.Generic[t]; ok {
			if m, ok := val.(map[string]interface{}); ok {
				metaOut = m
			}
		}

		if fmtStr == "typst" {
			hasTypst = true
		}
		if fmtStr == "pdf" || fmtStr == "latex" || fmtStr == "beamer" || fmtStr == "context" {
			// It's a PDF-generative format (via Latex/ConTeXt usually, or via pdf-engine)
			// Actually pandoc supports outputting pdf from many things directly via engine.
			// Ideally we check if 'pdf-engine' is set for this target or globally.

			engine := "pdflatex" // default
			if metaOut != nil {
				if e, ok := metaOut["pdf-engine"].(string); ok && e != "" {
					engine = e
				}
			}
			if engine == "pdflatex" {
				// Check global defaults if not set in target
				if e, ok := cfg.Generic["pdf-engine"].(string); ok && e != "" {
					engine = e
				}
			}

			// Only set pdfEngine if not already found (or maybe collect all unique engines?)
			// For simplicity let's assume one engine for now or collect them.
			// Let's rely on checking logic.

			// We effectively want to check this engine.
			// Since we might have multiple targets with different engines, let's just append to required directly?
			// But for "hasPDF" logic let's keep it simple.
			// Actually, let's just add it to a set.
			if !contains(required, engine) {
				required = append(required, engine)
			}
		}
	}

	// "pdf" format in pandoc implies using a pdf-engine.
	// If one of the targets effectively results in a PDF via latex/etc, we have added the engine above.
	// Logic above:
	// 1. Normalize format (e.g. "pdf" remains "pdf").
	// 2. If format is "pdf", pandoc uses default engine (pdflatex) unless specified.

	if hasTypst && !contains(required, "typst") {
		required = append(required, "typst")
	}

	return required, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
