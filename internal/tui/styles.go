package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Main application styles
	App = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#626262")).
		Padding(0, 1)

	// Title style for components
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#4F4FB7")).
			Padding(0, 1)

	// Status style for info messages
	StatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#959595"))

	// Error style for error messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	// Success style for success messages
	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00"))

	// Selected item highlight
	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#4F4FB7"))

	// File picker styles
	FilePickerSelected = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#4F4FB7"))

	FilePickerFile = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC"))

	FilePickerDirectory = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#81A1C1")).
				Bold(true)

	FilePickerCursor = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#4F4FB7"))

	FilePickerSymlink = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D08770")).
				Italic(true)
)
