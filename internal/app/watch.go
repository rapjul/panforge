package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rapjul/panforge/internal/options"
)

// Watch monitors the input file (and optional config file) for changes and re-runs the conversion.
//
// Watch monitors the input file (and optional config file) for changes and re-runs the conversion.
//
// Parameters:
//   - `ctx`: context for cancellation
//   - `inputFile`: path to the file being watched
//   - `configFile`: path to the optional config file
//   - `postArgs`: arguments to pass to the pandoc command
//   - `opts`: configuration options
//   - `executor`: used to run the command
func Watch(ctx context.Context, inputFile string, configFile string, postArgs []string, opts options.Options, executor CommandExecutor) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Add input file to watcher
	if err := watcher.Add(inputFile); err != nil {
		return fmt.Errorf("failed to watch file %s: %w", inputFile, err)
	}
	if configFile != "" {
		if err := watcher.Add(configFile); err != nil {
			if opts.Logger != nil {
				opts.Logger.Warn("failed to watch config file", "file", configFile, "error", err)
			} else {
				fmt.Printf("Warning: failed to watch config file %s: %v\n", configFile, err)
			}
		} else {
			if opts.Logger != nil {
				opts.Logger.Info("watching config file", "file", configFile)
			} else {
				fmt.Printf("Watching config file: %s\n", configFile)
			}
		}
	}

	if opts.Logger != nil {
		opts.Logger.Info("watching for changes (Press Ctrl+C to stop)", "file", inputFile)
	} else {
		fmt.Printf("Watching %s for changes... (Press Ctrl+C to stop)\n", inputFile)
	}

	// Run initially
	if err := Process(ctx, inputFile, postArgs, opts, executor); err != nil {
		if opts.Logger != nil {
			opts.Logger.Error("processing failed", "error", err)
		} else {
			log.Printf("Error processing file: %v", err)
		}
	}

	var debounceTimer *time.Timer
	const debounceDuration = 100 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// We care about Write, Rename, Create (if recreated)
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Chmod) || event.Has(fsnotify.Create) {
				// Debounce logic
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDuration, func() {
					if opts.Logger != nil {
						opts.Logger.Info("file changed, re-running...")
					} else {
						fmt.Println("\nFile changed, re-running...")
					}

					// Re-add watches if they were removed (atomic save)
					watcher.Add(inputFile)
					if configFile != "" {
						watcher.Add(configFile)
					}

					if err := Process(ctx, inputFile, postArgs, opts, executor); err != nil {
						if opts.Logger != nil {
							opts.Logger.Error("processing failed", "error", err)
						} else {
							log.Printf("Error processing file: %v", err)
						}
					} else {
						if opts.Logger != nil {
							opts.Logger.Info("done")
						} else {
							fmt.Println("Done.")
						}
					}
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			if opts.Logger != nil {
				opts.Logger.Error("watcher error", "error", err)
			} else {
				log.Printf("Watcher error: %v", err)
			}
		}
	}
}
