package views

import (
	"fmt"
	"strings"

	"sortd/internal/tui/common"
	"sortd/internal/tui/styles"
)

// Add main view rendering logic...

func RenderMainView(m common.ModelReader) string {
	var sb strings.Builder

	// Render title banner
	sb.WriteString(styles.Theme.Title.Render(`
	::######:::'#######::'########::'########:'########::
	'##... ##:'##.... ##: ##.... ##:... ##..:: ##.... ##:
	'##:::..:: ##:::: ##: ##:::: ##:::: ##:::: ##:::: ##:
	. ######:: ##:::: ##: ########::::: ##:::: ##:::: ##:
	:..... ##: ##:::: ##: ##.. ##:::::: ##:::: ##:::: ##:
	'##::: ##: ##:::: ##: ##::. ##::::: ##:::: ##:::: ##:
	. ######::. #######:: ##:::. ##:::: ##:::: ########::
	:......::::.......:::..:::::..:::::..:::::........:::
	`))
	sb.WriteString("\n")

	// Main menu
	if m.Mode() == common.Normal && m.CurrentDir() == "" {
		sb.WriteString(styles.Theme.Title.Render("Main Menu\n\n"))
		sb.WriteString("1. Quick Start - Organize Files\n")
		sb.WriteString("2. Setup Configuration\n")
		sb.WriteString("3. Watch Mode (Coming Soon)\n")
		sb.WriteString("4. Show Help\n\n")
	} else {
		// File list view
		for i, file := range m.Files() {
			style := styles.Theme.Unselected
			if m.IsSelected(file.Name) {
				style = styles.Theme.Selected
			}

			cursor := " "
			if i == m.Cursor() {
				cursor = ">"
			}

			sb.WriteString(fmt.Sprintf("%s %s\n", cursor, style.Render(file.Name)))
		}
	}

	// Show detailed help if enabled
	if m.ShowHelp() {
		sb.WriteString("\n" + RenderHelp())
	}

	// Always show key commands at bottom
	sb.WriteString("\n" + styles.Theme.Help.Render(RenderKeyCommands()))

	return styles.Theme.App.Render(sb.String())
}

func RenderKeyCommands() string {
	return styles.Theme.Help.Render(`
[↑/k] Up  [↓/j] Down  [Space] Select  [Enter] Open  [Esc] Back  [q] Quit  [?] Help
`)
}

func RenderHelp() string {
	return styles.Theme.Help.Render(`
Quick Start Guide:
...
`)
}
