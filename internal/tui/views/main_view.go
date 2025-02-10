package views

import (
	"fmt"
	"strings"

	"sortd/internal/tui/styles"
	"sortd/pkg/types"
)

// Add main view rendering logic...

func RenderMainView(m types.ModelReader) string {
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

	// Show current directory
	sb.WriteString(styles.Theme.Subtle.Render(fmt.Sprintf("Directory: %s\n\n", m.CurrentDir())))

	// Main menu
	if m.Mode() == types.Normal && m.CurrentDir() == "" {
		sb.WriteString(styles.Theme.Title.Render("Main Menu\n\n"))
		sb.WriteString("1. Quick Start - Organize Files\n")
		sb.WriteString("2. Setup Configuration\n")
		sb.WriteString("3. Watch Mode (Coming Soon)\n")
		sb.WriteString("4. Show Help\n\n")
	} else {
		// File list view with headers
		sb.WriteString(fmt.Sprintf("%-40s %-15s %10s\n", "Name", "Type", "Size"))
		sb.WriteString(strings.Repeat("-", 70) + "\n")

		for i, file := range m.Files() {
			style := styles.Theme.Unselected
			if m.IsSelected(file.Name) {
				style = styles.Theme.Selected
			}

			cursor := " "
			if i == m.Cursor() {
				cursor = ">"
			}

			// Format size
			var sizeStr string
			if file.Size < 1024 {
				sizeStr = fmt.Sprintf("%dB", file.Size)
			} else if file.Size < 1024*1024 {
				sizeStr = fmt.Sprintf("%.1fKB", float64(file.Size)/1024)
			} else {
				sizeStr = fmt.Sprintf("%.1fMB", float64(file.Size)/(1024*1024))
			}

			// Format content type
			contentType := file.ContentType
			if len(contentType) > 15 {
				contentType = contentType[:12] + "..."
			}

			fileInfo := fmt.Sprintf("%-40s %-15s %10s", file.Name, contentType, sizeStr)
			sb.WriteString(fmt.Sprintf("%s %s\n", cursor, style.Render(fileInfo)))
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
[↑/k] Up  [↓/j] Down  [Space] Select  [Enter] Open  [o] Organize  [Esc] Back  [q] Quit  [?] Help
`)
}

func RenderHelp() string {
	return styles.Theme.Help.Render(`
Navigation:
  ↑/k, ↓/j: Move cursor
  h/←, l/→: Change directory
  gg: Go to top
  G: Go to bottom

Selection:
  Space: Toggle selection
  v: Visual mode
  V: Visual line mode

Organization:
  o: Organize selected files
  r: Refresh view

Commands:
  q, quit: Exit
  :: Command mode
  ?: Toggle help
`)
}
