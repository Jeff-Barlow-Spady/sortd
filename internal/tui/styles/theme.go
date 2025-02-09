package styles

import "github.com/charmbracelet/lipgloss"

// Theme defines the core UI styles
var Theme = struct {
	App        lipgloss.Style
	Title      lipgloss.Style
	Selected   lipgloss.Style
	Unselected lipgloss.Style
	Help       lipgloss.Style
}{
	App: lipgloss.NewStyle().
		Padding(1, 2),
	Title: lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7B61FF")).
		MarginBottom(1),
	Selected: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#73F59F")).
		Bold(true),
	Unselected: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")),
	Help: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#5A9")),
}
