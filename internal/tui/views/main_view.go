package views

import (
	"strings"

	"sortd/internal/tui/common"
	"sortd/internal/tui/components"
	"sortd/internal/tui/styles"
)

// Add main view rendering logic...

func RenderMainView(m common.ModelReader) string {
	var sb strings.Builder

	// Render banner
	sb.WriteString(renderBanner())

	// Render file list or main menu
	if m.Mode() == common.Normal && m.CurrentDir() != "" {
		fileList := components.NewFileList()
		fileList.SetFiles(m.Files())
		sb.WriteString(fileList.View())
	} else {
		sb.WriteString(renderMainMenu())
	}

	// Help and key commands
	if m.ShowHelp() {
		sb.WriteString("\n" + RenderHelp())
	}
	sb.WriteString("\n" + RenderKeyCommands())

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

func renderBanner() string {
	return styles.Theme.Title.Render(`
	::######:::'#######::'########::'########:'########::
	'##... ##:'##.... ##: ##.... ##:... ##..:: ##.... ##:
	'##:::..:: ##:::: ##: ##:::: ##:::: ##:::: ##:::: ##:
	. ######:: ##:::: ##: ########::::: ##:::: ##:::: ##:
	:..... ##: ##:::: ##: ##.. ##:::::: ##:::: ##:::: ##:
	'##::: ##: ##:::: ##: ##::. ##::::: ##:::: ##:::: ##:
	. ######::. #######:: ##:::. ##:::: ##:::: ########::
	:......::::.......:::..:::::..:::::..:::::........:::
	`)
}

func renderMainMenu() string {
	var s strings.Builder
	s.WriteString(styles.Theme.Title.Render("Main Menu\n\n"))
	s.WriteString("1. Browse & Organize Files\n")
	s.WriteString("2. Configure Organization Rules\n")
	s.WriteString("3. Configure File Patterns\n")
	s.WriteString("4. View Analysis Report\n")
	s.WriteString("5. Watch Directory\n")
	s.WriteString("6. Show Help\n\n")
	s.WriteString("\nCurrent Config:\n")
	s.WriteString("  Directory: " + styles.Theme.Help.Render("(none selected)") + "\n")
	s.WriteString("  Patterns: " + styles.Theme.Help.Render("(default)") + "\n")
	return s.String()
}
