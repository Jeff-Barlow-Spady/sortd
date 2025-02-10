package styles

import "github.com/charmbracelet/lipgloss"

// Styles defines the core UI styles
var (
	App = lipgloss.NewStyle().
		Padding(1, 2)

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7B61FF")).
		MarginBottom(1)

	Selected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#73F59F")).
			Bold(true)

	Unselected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	Help = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#5A9"))
)

// FileListStyle defines the style for the file list
var FileListStyle = lipgloss.NewStyle().
	Padding(1, 2).
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#7B61FF"))
