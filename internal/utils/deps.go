// Package utils contains common helper functions used across the application.
package utils

import (
	"os/exec"
	"strings"
)

// CheckResult holds the status of a checked tool
type CheckResult struct {
	Name    string
	Found   bool
	Path    string
	Version string
	Error   error
}

// CheckTool checks if a tool is available in PATH and optionally returns its version.
//
// Parameters:
//   - `name`: the command name of the tool
//   - `versionFlag`: the flag to use to check the version (if empty, tries common flags)
func CheckTool(name string, versionFlag string) CheckResult {
	path, err := exec.LookPath(name)
	if err != nil {
		return CheckResult{Name: name, Found: false, Error: err}
	}

	res := CheckResult{Name: name, Found: true, Path: path}

	// If explicit flag provided, try only that
	if versionFlag != "" {
		cmd := exec.Command(name, versionFlag)
		out, err := cmd.Output()
		if err == nil {
			output := strings.TrimSpace(string(out))
			if len(output) > 0 {
				lines := strings.Split(output, "\n")
				if len(lines) > 0 {
					res.Version = strings.TrimSpace(lines[0])
				}
			}
		}
		return res
	}

	// Otherwise, try getting version using common flags
	// Priority: --version, -version, version
	flags := []string{"--version", "-version", "version"}

	for _, flag := range flags {
		//nolint:gosec // G204: Subprocess launched with variable is intended for tool checking
		cmd := exec.Command(name, flag)
		out, err := cmd.Output()
		if err == nil {
			output := strings.TrimSpace(string(out))
			if len(output) > 0 {
				lines := strings.Split(output, "\n")
				if len(lines) > 0 {
					res.Version = strings.TrimSpace(lines[0])
					return res
				}
			}
		}
	}

	return res
}

// CheckPandoc verifies pandoc availability and returns check result.
func CheckPandoc() CheckResult {
	return CheckTool("pandoc", "")
}

// CheckPDFEngine verifies a PDF engine availability.
//
// Parameters:
//   - `engine`: the name of the PDF engine (e.g., "pdflatex")
func CheckPDFEngine(engine string) CheckResult {
	// Most latex engines support --version
	return CheckTool(engine, "")
}

// CheckTypst verifies typst availability.
func CheckTypst() CheckResult {
	return CheckTool("typst", "")
}

// CheckOptionalTool check for optional dependencies.
//
// Parameters:
//   - `name`: the command name of the tool
func CheckOptionalTool(name string) CheckResult {
	return CheckTool(name, "")
}
