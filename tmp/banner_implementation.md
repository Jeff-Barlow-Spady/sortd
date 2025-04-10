# Banner Implementation with Version Number

This guide demonstrates how to implement a standalone banner with version number in the TUI interface.

## 1. Create a Dedicated Banner Component

First, let's extract the banner rendering into its own component to reduce dependencies:

```go
// internal/tui/components/banner.go
package components

import (
    "fmt"
    "github.com/charmbracelet/lipgloss"
)

// Banner represents the application logo and version display
type Banner struct {
    // Configuration
    width       int
    showVersion bool
    version     string

    // Styling
    logoStyle   lipgloss.Style
    versionStyle lipgloss.Style
}

// NewBanner creates a new banner component
func NewBanner(version string) Banner {
    return Banner{
        width:       40,
        showVersion: true,
        version:     version,
        logoStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#7B61FF")),
        versionStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Italic(true),
    }
}

// SetWidth adjusts the banner width
func (b *Banner) SetWidth(width int) {
    b.width = width
}

// View renders the banner
func (b Banner) View() string {
    logo := `
    ███████  ██████  ██████  ████████ ██████
    ██      ██    ██ ██   ██    ██    ██   ██
    ███████ ██    ██ ██████     ██    ██   ██
         ██ ██    ██ ██   ██    ██    ██   ██
    ███████  ██████  ██   ██    ██    ██████
    `

    renderedLogo := b.logoStyle.Render(logo)

    // Add version if enabled
    if b.showVersion {
        versionText := fmt.Sprintf("v%s", b.version)
        renderedVersion := b.versionStyle.Render(versionText)

        // Create a container with the logo and version
        container := lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(lipgloss.Color("#686868")).
            Padding(1).
            Width(b.width)

        return container.Render(
            lipgloss.JoinVertical(
                lipgloss.Left,
                renderedLogo,
                renderedVersion,
            ),
        )
    }

    return renderedLogo
}
```

## 2. Pass Version from Main to TUI

Modify the main entry point to pass the version to the TUI model:

```go
// cmd/sortd/main.go
package main

import (
    "fmt"
    "os"
    "sortd/internal/tui"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/spf13/cobra"
)

var (
    version = "dev"
)

// tuiCmd represents the TUI command
func tuiCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "tui",
        Short: "Start the terminal user interface",
        Long:  `Start the terminal user interface for interactive file organization.`,
        Run: func(cmd *cobra.Command, args []string) {
            // Create and run the TUI with version
            m := tui.New(version)
            p := tea.NewProgram(m, tea.WithAltScreen())
            if _, err := p.Run(); err != nil {
                fmt.Printf("Error running TUI: %v\n", err)
                os.Exit(1)
            }
        },
    }

    return cmd
}
```

## 3. Update the Model to Accept Version

Modify the TUI Model to accept and store the version:

```go
// internal/tui/model.go
package tui

import (
    "sortd/internal/tui/components"
    // Other imports...
)

// Model represents the TUI state
type Model struct {
    // Core state
    keys          KeyMap
    help          help.Model
    list          list.Model
    viewport      viewport.Model
    fileTree      *components.FileTree
    banner        components.Banner // New banner component

    // Other fields...

    // Add version field
    version       string
}

// New creates a new Model
func New(version string) *Model {
    // Other initialization...

    // Create banner with version
    banner := components.NewBanner(version)

    m := &Model{
        // Other fields...
        banner:        banner,
        version:       version,
    }

    return m
}
```

## 4. Update the View Function

Modify the View function to render the banner in its own container:

```go
// internal/tui/view.go (extracted from model.go)
package tui

import (
    "strings"
    "github.com/charmbracelet/lipgloss"
)

// View renders the UI
func (m *Model) View() string {
    // Initialize a string builder for the final output
    var b strings.Builder

    // Calculate available width
    availableWidth := m.viewport.Width
    if availableWidth < 80 {
        availableWidth = 80 // Minimum reasonable width
    }

    // Set the banner width and render it
    m.banner.SetWidth(40) // Fixed width for the banner
    bannerView := m.banner.View()

    // ===== TOP BAR =====
    // Create a top bar with current location and other metadata
    topBarStyle := lipgloss.NewStyle().
        BorderStyle(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("#686868")).
        Padding(0, 1).
        Width(availableWidth).
        Foreground(lipgloss.Color("#4EADFA")).
        Bold(true)

    pathText := "📂 Location: " + m.currentDir

    // Add counts to the top bar
    statsText := fmt.Sprintf("Files: %d • Selected: %d", len(m.list.Items()), len(m.selectedFiles))

    topBar := lipgloss.JoinHorizontal(
        lipgloss.Left,
        lipgloss.NewStyle().Width(availableWidth-len(statsText)-5).Render(pathText),
        lipgloss.NewStyle().Foreground(lipgloss.Color("#5FFFAF")).Render(statsText),
    )

    b.WriteString(topBarStyle.Render(topBar) + "\n")

    // ===== MAIN CONTENT =====

    // Left panel with banner and help
    leftPanelStyle := lipgloss.NewStyle().
        BorderStyle(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("#686868")).
        Padding(1).
        Width(40).
        Height(m.contentHeight)

    // Combine banner with help tips
    var leftContent strings.Builder
    leftContent.WriteString(bannerView)
    leftContent.WriteString("\n\n")

    // Add help tips
    if m.showFullHelp {
        // Add help content...
        helpTipStyle := lipgloss.NewStyle().
            Foreground(lipgloss.Color("#A5D6FF"))

        leftContent.WriteString(helpTipStyle.Render("Quick Tips:") + "\n")
        leftContent.WriteString(helpTipStyle.Render("• ↑/k or ↓/j to navigate") + "\n")
        leftContent.WriteString(helpTipStyle.Render("• Space to select files ✓") + "\n")
        // ... other tips
    }

    // Render left panel with banner and help content
    leftPanelContent := leftContent.String()

    // ... rest of the rendering logic

    // Return final rendered view
    return b.String()
}
```

This approach:

1. Creates a standalone banner component that encapsulates the logo and version rendering
2. Passes the version from main.go to the TUI
3. Renders the banner in its own container with proper styling
4. Combines the banner with help content in the left panel

The result will be a clean, maintainable implementation that clearly shows the version number alongside the logo.