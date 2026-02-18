package ui

import "github.com/charmbracelet/lipgloss"

// Styles for various CLI elements.
var (
	// CommandStyle is the style for the main command executable.
	CommandStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("57")) // Purple
	// FlagStyle is the style for command-line flags.
	FlagStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")) // Cyan
	// ValueStyle is the style for general argument values.
	ValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // Gold/Yellow
	// PathStyle is the style for file paths.
	PathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // Orange (Distinct!)
	// ErrorStyle is the style for error messages.
	ErrorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")) // Red
	// KeyStyle is the style for configuration keys.
	KeyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244")) // Light Gray
	// StringValStyle is the style for configuration string values.
	StringValStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("35")) // Green
	// DryRunStyle is the style for the dry run indicator.
	DryRunStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("202")) // Orange/Red
)
