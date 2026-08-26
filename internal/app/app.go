// Package app implements the core application logic for panforge.
package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

const (
	// DefaultFormat defines the fallback format when no targets are specified.
	DefaultFormat = "html"
	// DefaultPDFEngine defines the default PDF rendering engine.
	DefaultPDFEngine = "pdflatex"
	// ToolPandoc defines the pandoc executable name.
	ToolPandoc = "pandoc"
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
//   - ctx: context for cancellation
//   - name: command name
//   - args: command arguments
//   - stdout: writer for standard output
//   - stderr: writer for standard error
func (e *RealExecutor) Run(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	if e.DryRun {
		return nil
	}
	//nolint:gosec,nolintlint // Subprocess launched here is the intended behavior
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
//   - ctx: context for cancellation
//   - cmd: the cobra command being executed
//   - args: command line arguments
//   - opts: parsed command line flags
//   - executor: interface for running system commands
//
// Returns:
//   - error: an error if execution fails
func Run(ctx context.Context, cmd *cobra.Command, args []string, opts options.Options, executor CommandExecutor) error {
	// 1. Parse Input Files
	inputFiles, postArgs := parseArgs(cmd, args, opts.Inputs)
	if len(inputFiles) == 0 {
		if len(opts.Targets) > 0 || opts.Output != "" {
			return fmt.Errorf("no input file found")
		}
		if cmd != nil {
			return cmd.Help()
		}
		return nil
	}

	// Prevent fixed output collision when multiple files are given
	if len(inputFiles) > 1 && opts.Output != "" {
		if !strings.Contains(opts.Output, "{") {
			return fmt.Errorf("cannot specify a fixed output filename (--output) when processing multiple input files; use a filename template (e.g., '{title}.{ext}') or process files individually")
		}
	}

	// 2. Initial Config Loading & Execution
	defaultConfigPath, _, _ := config.LoadDefaultConfig("default")

	if opts.Watch {
		if len(inputFiles) > 1 {
			if opts.Logger != nil {
				opts.Logger.Warn("watch mode only monitors the first input file", "file", inputFiles[0])
			} else if !opts.Quiet {
				fmt.Printf("Warning: watch mode only monitors the first input file (%s)\n", inputFiles[0])
			}
		}
		inputFile := inputFiles[0]
		if inputFile != "-" {
			resolvedInput, err := utils.ResolvePath(inputFile)
			if err != nil {
				return fmt.Errorf("failed to resolve input file path: %w", err)
			}
			inputFile = resolvedInput
		}
		return Watch(ctx, inputFile, defaultConfigPath, postArgs, opts, executor)
	}

	// Process each input file
	for _, inputFile := range inputFiles {
		isStdin := inputFile == "-"
		var cleanupTemp func()
		if !isStdin {
			resolvedInput, err := utils.ResolvePath(inputFile)
			if err != nil {
				return fmt.Errorf("failed to resolve input file path: %w", err)
			}
			inputFile = resolvedInput
		} else {
			var in io.Reader = os.Stdin
			if cmd != nil {
				in = cmd.InOrStdin()
			}
			var err error
			inputFile, cleanupTemp, err = createStdinTempFile(in)
			if err != nil {
				return err
			}
		}

		err := Process(ctx, inputFile, postArgs, opts, executor)
		if cleanupTemp != nil {
			cleanupTemp()
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// Process handles a single run of the conversion logic.
//
// Parameters:
//   - ctx: context for cancellation
//   - inputFile: path to the Markdown file to convert
//   - postArgs: additional arguments to pass to pandoc
//   - opts: configuration options
//   - executor: used to run the pandoc command
//
// Returns:
//   - error: an error if processing fails
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
	mergeDefaultConfig(cfg, defaultCfg)

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
		g.Go(func() error {
			if err := sem.Acquire(ctx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			fmtStr, metaOut := resolveTargetConfig(t, cfg)
			baseDir := filepath.Dir(inputFile)

			if opts.Verbose && opts.Logger != nil {
				opts.Logger.Debug("Effective configuration", "config", metaOut)
			}

			if err := pandoc.ValidateMetadata(metaOut, baseDir); err != nil {
				return err
			}

			outputFile, err := resolveOutputPath(inputFile, opts.Output, fmtStr, cfg, metaOut, opts.RelativeOutput)
			if err != nil {
				return err
			}

			allowed, err := confirmOverwrite(outputFile, cfg, metaOut, opts, &promptMu)
			if err != nil {
				return err
			}
			if !allowed {
				if opts.Logger != nil {
					opts.Logger.Warn("skipping target", "file", outputFile, "reason", "already exists and overwrite declined")
				} else {
					fmt.Fprintf(os.Stderr, "Skipping %s: file already exists and overwrite was declined\n", outputFile)
				}
				return nil
			}

			pandocArgs, cmdStr := buildPandocCommand(inputFile, outputFile, fmtStr, metaOut, baseDir, postArgs)

			if opts.Logger != nil {
				quotedArgs := make([]string, 0, len(pandocArgs))
				for _, arg := range pandocArgs {
					if strings.Contains(arg, " ") || strings.Contains(arg, "\"") {
						quotedArgs = append(quotedArgs, fmt.Sprintf("%q", arg))
					} else {
						quotedArgs = append(quotedArgs, arg)
					}
				}
				fullArgs := append([]string{ToolPandoc}, quotedArgs...)
				logArgs := []any{"command", cmdStr, "args", fullArgs}
				msg := "executing the command"
				if opts.DryRun {
					msg = "dry run: displaying the command"
					logArgs = append(logArgs, "dry_run", true)
				}
				opts.Logger.Info(msg, logArgs...)
			} else if !opts.Quiet {
				// Fallback if no logger validation
				fmt.Printf("panforge calling: %s\n", cmdStr)
			}

			if logFile != nil {
				logMu.Lock()
				_, _ = fmt.Fprintf(logFile, "panforge calling: %s\n", cmdStr)
				logMu.Unlock()
			}

			return executePandoc(ctx, executor, pandocArgs)
		})
	}

	return g.Wait()
}

// createStdinTempFile reads standard input into a temporary file and returns its path and a cleanup callback.
//
// Parameters:
//   - r: reader to consume standard input from
//
// Returns:
//   - string: path to the created temporary file
//   - func(): cleanup function to remove the temporary file
//   - error: an error if file creation or copying fails
func createStdinTempFile(r io.Reader) (string, func(), error) {
	tmpFile, err := os.CreateTemp("", "panforge-stdin-*.md")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file for stdin: %w", err)
	}
	tmpName := tmpFile.Name()
	cleanup := func() {
		_ = os.Remove(tmpName)
	}

	if _, err := io.Copy(tmpFile, r); err != nil {
		_ = tmpFile.Close()
		cleanup()
		return "", nil, fmt.Errorf("failed to read stdin: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to close temp file: %w", err)
	}
	return tmpName, cleanup, nil
}

// mergeDefaultConfig merges missing configuration values from defaultCfg into cfg.
//
// Parameters:
//   - cfg: destination configuration to receive default values
//   - defaultCfg: source default configuration
func mergeDefaultConfig(cfg, defaultCfg *config.Config) {
	if defaultCfg == nil || cfg == nil {
		return
	}
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
		cfg.Generic = make(map[string]any)
	}
	if defaultCfg.Generic != nil {
		for k, v := range defaultCfg.Generic {
			if _, exists := cfg.Generic[k]; !exists {
				cfg.Generic[k] = v
			}
		}
	}
}

// resolveTargetConfig resolves the output format string and format-specific metadata map.
//
// Parameters:
//   - target: the target format name (e.g. "html", "pdf", "custom_target")
//   - cfg: document configuration containing OutputMap and Generic settings
//
// Returns:
//   - string: normalized pandoc format identifier
//   - map[string]any: merged metadata options for the format
func resolveTargetConfig(target string, cfg *config.Config) (string, map[string]any) {
	fmtStr := pandoc.NormalizeFormat(target)
	var metaOut map[string]any

	if cfg != nil {
		if val, ok := cfg.OutputMap[target]; ok {
			if m, ok := val.(map[string]any); ok {
				metaOut = m
				if to, ok := m["to"].(string); ok && to != "" {
					fmtStr = to
				}
			}
		} else if val, ok := cfg.Generic[target]; ok {
			if m, ok := val.(map[string]any); ok {
				metaOut = m
			}
		}

		if metaOut == nil {
			metaOut = make(map[string]any)
		}

		if cfg.Generic != nil {
			for k, v := range cfg.Generic {
				if pandoc.IgnoredMetadataKeys[k] {
					continue
				}
				if _, exists := metaOut[k]; !exists {
					metaOut[k] = v
				}
			}
		}
	} else {
		metaOut = make(map[string]any)
	}

	return fmtStr, metaOut
}

// resolveOutputPath calculates the canonical destination path for a target conversion.
//
// Parameters:
//   - inputFile: path to the source Markdown document
//   - explicitOutput: user-specified output flag value (if any)
//   - fmtStr: target output format string
//   - cfg: document configuration
//   - metaOut: format-specific metadata map
//   - relativeOutput: whether --relative-output was specified
//
// Returns:
//   - string: absolute or canonical resolved output path
//   - error: an error if path resolution fails
func resolveOutputPath(inputFile, explicitOutput, fmtStr string, cfg *config.Config, metaOut map[string]any, relativeOutput bool) (string, error) {
	outputFile := explicitOutput
	if outputFile == "" {
		outputFile = pandoc.GenerateOutputFilename(inputFile, cfg, metaOut, fmtStr)
	}

	// If output file is relative, resolve relative to input directory
	// UNLESS --relative-output is set or input file is from stdin.
	isStdinTemp := strings.Contains(filepath.Base(inputFile), "panforge-stdin-")
	if !filepath.IsAbs(outputFile) && !relativeOutput {
		if isStdinTemp {
			// When input is piped from stdin, the temp file lives in the OS temp directory.
			// Relative output paths must resolve against the working directory, not the temp directory.
			outputFile = filepath.Join(".", outputFile)
		} else {
			dir := filepath.Dir(inputFile)
			if inputFile != "-" && dir != "." {
				outputFile = filepath.Join(dir, outputFile)
			}
		}
	}

	resolvedOutput, err := utils.ResolvePath(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to resolve output file path: %w", err)
	}
	return resolvedOutput, nil
}

// confirmOverwrite checks if an existing file may be overwritten.
//
// Parameters:
//   - outputFile: path of the output file to check
//   - cfg: document configuration
//   - metaOut: format-specific metadata map
//   - opts: CLI options (Force, Watch, DryRun)
//   - promptMu: mutex to synchronize terminal prompts during concurrent execution
//
// Returns:
//   - bool: true if overwrite is permitted or confirmed, false if skipped
//   - error: an error if prompt interaction fails
func confirmOverwrite(outputFile string, cfg *config.Config, metaOut map[string]any, opts options.Options, promptMu *sync.Mutex) (bool, error) {
	if opts.DryRun {
		return true, nil
	}
	if _, err := os.Stat(outputFile); err != nil {
		return true, nil
	}

	// In watch mode, overwrite is implicitly permitted so file changes trigger live rebuilds without blocking.
	if opts.Force || opts.Watch || isOverwriteAllowed(cfg, metaOut) {
		if opts.Verbose && opts.Logger != nil {
			reason := "force flag set"
			if opts.Watch {
				reason = "watch mode active"
			} else if isOverwriteAllowed(cfg, metaOut) {
				reason = "overwrite allowed in configuration"
			}
			opts.Logger.Debug("overwriting existing output file", "file", outputFile, "reason", reason)
		}
		return true, nil
	}

	if promptMu != nil {
		promptMu.Lock()
		defer promptMu.Unlock()
	}
	return askForConfirmation(outputFile, os.Stdin, os.Stderr), nil
}

// buildPandocCommand assembles the argument list and display string for pandoc invocation.
//
// Parameters:
//   - inputFile: input document path
//   - outputFile: output destination path
//   - fmtStr: output format identifier
//   - metaOut: metadata key-value map for the format
//   - baseDir: base directory used for resolving relative paths in metadata
//   - postArgs: additional passthrough arguments
//
// Returns:
//   - []string: arguments to pass to pandoc executable
//   - string: human-readable quoted command string
func buildPandocCommand(inputFile, outputFile, fmtStr string, metaOut map[string]any, baseDir string, postArgs []string) ([]string, string) {
	resolvedMetaOut := pandoc.ResolveMetadataPaths(metaOut, baseDir)

	pandocArgs := []string{inputFile}
	pandocArgs = append(pandocArgs, "--to", fmtStr)
	pandocArgs = append(pandocArgs, "--output", outputFile)
	pandocArgs = append(pandocArgs, pandoc.GetArgs(resolvedMetaOut)...)

	// Normalize shorthand '-t' to '--to' to avoid ambiguity when passed alongside target definitions.
	normalizedPostArgs := make([]string, len(postArgs))
	copy(normalizedPostArgs, postArgs)
	for i := 0; i < len(normalizedPostArgs); i++ {
		if normalizedPostArgs[i] == "-t" {
			normalizedPostArgs[i] = "--to"
		}
	}
	pandocArgs = append(pandocArgs, normalizedPostArgs...)

	// Quote arguments containing spaces or quotes for unambiguous logging and display.
	var quotedArgs []string
	for _, arg := range pandocArgs {
		if strings.Contains(arg, " ") || strings.Contains(arg, "\"") {
			quotedArgs = append(quotedArgs, fmt.Sprintf("%q", arg))
		} else {
			quotedArgs = append(quotedArgs, arg)
		}
	}
	cmdStr := ToolPandoc + " " + strings.Join(quotedArgs, " ")
	return pandocArgs, cmdStr
}

// executePandoc runs the pandoc command using the executor and formats error output on failure.
//
// Parameters:
//   - ctx: context for cancellation
//   - executor: command runner implementation
//   - pandocArgs: arguments passed to pandoc
//
// Returns:
//   - error: an error with formatted stderr diagnostics if execution fails
func executePandoc(ctx context.Context, executor CommandExecutor, pandocArgs []string) error {
	var stderrBuf strings.Builder
	stderrWriter := io.MultiWriter(os.Stderr, &stderrBuf)

	if err := executor.Run(ctx, ToolPandoc, pandocArgs, os.Stdout, stderrWriter); err != nil {
		// Extract up to the last 5 lines of stderr because TeX engines often produce extensive output where the actual error is at the end.
		errMsg := err.Error()
		if stderrStr := strings.TrimSpace(stderrBuf.String()); stderrStr != "" {
			lines := strings.Split(stderrStr, "\n")
			if len(lines) > 5 {
				errMsg = strings.Join(lines[len(lines)-5:], "\n")
			} else {
				errMsg = stderrStr
			}
		}
		_, _ = fmt.Fprintln(os.Stderr) // Separator
		return fmt.Errorf("an error occurred running the pandoc command: %s", errMsg)
	}
	return nil
}

// parseArgs determines the input files and trailing Pandoc arguments from CLI args.
//
// Parameters:
//   - cmd: the cobra command being executed (optional)
//   - args: command line arguments
//   - explicitInputs: input files specified via --input / -i flags
//
// Returns:
//   - []string: input filenames
//   - []string: remaining passthrough arguments
func parseArgs(cmd *cobra.Command, args []string, explicitInputs []string) ([]string, []string) {
	var inputFiles []string
	var postArgs []string

	dashIdx := -1
	if cmd != nil {
		dashIdx = cmd.ArgsLenAtDash()
	}

	if dashIdx >= 0 && dashIdx <= len(args) {
		for _, arg := range args[:dashIdx] {
			if arg == "-" || !strings.HasPrefix(arg, "-") {
				inputFiles = append(inputFiles, arg)
			}
		}
		postArgs = append(postArgs, args[dashIdx:]...)
	} else {
		for i, arg := range args {
			if arg == "-" || !strings.HasPrefix(arg, "-") {
				inputFiles = append(inputFiles, arg)
			} else {
				postArgs = append(postArgs, args[i:]...)
				break
			}
		}
	}

	if len(explicitInputs) > 0 {
		inputFiles = append(inputFiles, explicitInputs...)
	}

	return inputFiles, postArgs
}

// DetermineTargets figures out which output formats to build.
//
// Parameters:
//   - opts: CLI targets
//   - cfg: YAML configuration from the file
//
// It prioritizes CLI targets > 'outputs' list in YAML > 'output' map in YAML > Default "html".
func DetermineTargets(opts options.Options, cfg *config.Config) []string {
	if len(opts.Targets) > 0 {
		return opts.Targets
	}

	if cfg == nil {
		return []string{DefaultFormat}
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
	return []string{DefaultFormat}
}

// isOverwriteAllowed checks if overwrite is explicitly allowed in configuration.
//
// Parameters:
//   - cfg: the global config
//   - metaOut: the format-specific config
//
// Returns:
//   - bool: true if overwrite is allowed, false otherwise
func isOverwriteAllowed(cfg *config.Config, metaOut map[string]any) bool {
	if metaOut != nil {
		if v, ok := metaOut["overwrite"]; ok {
			if b, ok := v.(bool); ok && b {
				return true
			}
		}
	}
	if cfg != nil && cfg.Generic != nil {
		if v, ok := cfg.Generic["overwrite"]; ok {
			if b, ok := v.(bool); ok && b {
				return true
			}
		}
	}
	return false
}

// askForConfirmation prompts the user for yes/no confirmation.
//
// Parameters:
//   - filename: the file being overwritten
//   - r: the input reader (usually stdin)
//   - w: the output writer (usually stderr)
//
// Returns:
//   - bool: true if the user confirmed, false otherwise
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

// resolvePDFEngine returns the configured or default PDF engine for a target metadata map.
//
// Parameters:
//   - metaOut: target-specific metadata map (optional)
//   - generic: generic metadata map (optional)
//
// Returns:
//   - string: the name of the PDF engine (e.g. "pdflatex", "tectonic", "xelatex")
func resolvePDFEngine(metaOut map[string]any, generic map[string]any) string {
	engine := DefaultPDFEngine
	if metaOut != nil {
		if e, ok := metaOut["pdf-engine"].(string); ok && e != "" {
			engine = e
		}
	}
	if engine == DefaultPDFEngine && generic != nil {
		if e, ok := generic["pdf-engine"].(string); ok && e != "" {
			engine = e
		}
	}
	return engine
}

// GetRequiredTools determines which tools are needed for the given input file.
//
// Parameters:
//   - inputFile: path to the input Markdown file
//   - opts: runtime options
//
// Returns:
//   - []string: list of tool names that should be checked (e.g. "pandoc", "pdflatex")
//   - error: an error if the input file cannot be read
func GetRequiredTools(inputFile string, opts options.Options) ([]string, error) {
	required := []string{ToolPandoc}

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
		//nolint:nilerr // Intentional fallback to base tools if config load fails
		return required, nil
	}

	// Load default config to fill in gaps if possible, mostly for output map
	_, defaultCfg, _ := config.LoadDefaultConfig("default")
	mergeDefaultConfig(cfg, defaultCfg)

	targets := DetermineTargets(opts, cfg)
	hasTypst := false

	for _, t := range targets {
		fmtStr, metaOut := resolveTargetConfig(t, cfg)

		if fmtStr == "typst" {
			hasTypst = true
		}
		if fmtStr == "pdf" || fmtStr == "latex" || fmtStr == "beamer" || fmtStr == "context" {
			engine := resolvePDFEngine(metaOut, cfg.Generic)
			if !contains(required, engine) {
				required = append(required, engine)
			}
		}
	}

	if hasTypst && !contains(required, "typst") {
		required = append(required, "typst")
	}

	return required, nil
}

// contains checks if a string slice contains a specific item.
//
// Parameters:
//   - slice: the slice to check
//   - item: the item to look for
//
// Returns:
//   - bool: true if the item is found, false otherwise
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
