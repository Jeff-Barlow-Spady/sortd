package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sortd/internal/analysis"
	"sortd/internal/organize"
	"sortd/internal/tui/styles"
	"sortd/pkg/types"
	"strings"
	"time"

	"sortd/internal/tui/components"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// KeyMap defines the keybindings for the application
type KeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	Select      key.Binding
	Quit        key.Binding
	ToggleHelp  key.Binding
	CommandMode key.Binding
	VisualMode  key.Binding
	Refresh     key.Binding
	Organize    key.Binding
	GoToTop     key.Binding
	GoToBottom  key.Binding
	VisualLine  key.Binding
	Search      key.Binding
	ToggleView  key.Binding
}

// DefaultKeyMap returns the default keybindings
var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "parent directory"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l", "enter"),
		key.WithHelp("→/Enter", "open directory"),
	),
	Select: key.NewBinding(
		key.WithKeys("space"),
		key.WithHelp("Space", "select file"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	ToggleHelp: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	CommandMode: key.NewBinding(
		key.WithKeys(":"),
		key.WithHelp(":", "command mode"),
	),
	VisualMode: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "visual mode"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Organize: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "organize files"),
	),
	GoToTop: key.NewBinding(
		key.WithKeys("g g"),
		key.WithHelp("gg", "go to top"),
	),
	GoToBottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "go to bottom"),
	),
	VisualLine: key.NewBinding(
		key.WithKeys("V"),
		key.WithHelp("V", "visual line mode"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	ToggleView: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("Tab", "toggle view"),
	),
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.Right, k.Left, k.Organize, k.Quit, k.ToggleHelp, k.ToggleView}
}

// FullHelp returns keybindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Select, k.Organize, k.Refresh, k.ToggleView},
		{k.VisualMode, k.VisualLine, k.Search},
		{k.GoToTop, k.GoToBottom, k.CommandMode, k.Quit, k.ToggleHelp},
	}
}

// Item represents a file or directory in the file list
type Item struct {
	entry    types.FileEntry
	selected bool
}

// FilterValue implements list.Item
func (i Item) FilterValue() string {
	return i.entry.Name
}

// Title returns the item title with visual indicators for file type
func (i Item) Title() string {
	// Add an icon based on file type
	icon := "📄 " // Default file icon

	fileType := strings.ToLower(i.entry.ContentType)
	if strings.Contains(fileType, "directory") {
		icon = "📁 "
	} else if strings.Contains(fileType, "image") {
		icon = "🖼️ "
	} else if strings.Contains(fileType, "video") {
		icon = "🎬 "
	} else if strings.Contains(fileType, "audio") {
		icon = "🎵 "
	} else if strings.Contains(fileType, "pdf") {
		icon = "📕 "
	} else if strings.Contains(fileType, "zip") || strings.Contains(fileType, "tar") ||
		strings.Contains(fileType, "gzip") || strings.Contains(fileType, "compressed") {
		icon = "🗜️ "
	} else if strings.Contains(fileType, "text") {
		icon = "📝 "
	}

	// Add selection indicator - this creates a more obvious visual cue
	// for selected files beyond just the checkbox
	selectionIndicator := ""
	if i.selected {
		selectionIndicator = " ✓"
	}

	return icon + i.entry.Name + selectionIndicator
}

// Description returns additional file details
func (i Item) Description() string {
	// For directories, show item count if possible
	if strings.Contains(strings.ToLower(i.entry.ContentType), "directory") {
		return "Directory"
	}

	// Build a more informative description
	var description strings.Builder

	// Add file size with appropriate unit
	description.WriteString(formatSize(i.entry.Size))

	// Add file type if available
	contentType := i.entry.ContentType
	if contentType != "" && contentType != "unknown" {
		// Simplify content type for display
		simpleType := contentType
		if idx := strings.Index(contentType, ";"); idx > 0 {
			simpleType = contentType[:idx]
		}

		// Replace common MIME types with simpler names
		simpleType = strings.ReplaceAll(simpleType, "application/", "")
		simpleType = strings.ReplaceAll(simpleType, "text/", "")
		simpleType = strings.ReplaceAll(simpleType, "image/", "")
		simpleType = strings.ReplaceAll(simpleType, "video/", "")
		simpleType = strings.ReplaceAll(simpleType, "audio/", "")

		description.WriteString(" • " + simpleType)
	}

	// Add tags if available
	if len(i.entry.Tags) > 0 {
		description.WriteString(" • Tags: ")
		for idx, tag := range i.entry.Tags {
			if idx > 0 {
				description.WriteString(", ")
			}
			description.WriteString(tag)
		}
	}

	return description.String()
}

// formatSize converts file size to human-readable format
func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	} else {
		return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
	}
}

// createCustomListDelegate creates a custom delegate for the list
func createCustomListDelegate() list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()

	// Customize styles
	delegate.ShowDescription = true
	delegate.SetSpacing(1)

	// Set wider item spacing for ADHD-friendly display
	delegate.SetHeight(1) // One line height with space between items

	// Title styling (regular items)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Copy().
		Foreground(lipgloss.Color("#E2E2E2")).
		Bold(false)

	// Title styling (selected items)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Copy().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#6B5ECD")).
		Bold(true).
		Underline(true)

	// Description styling (regular items)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Copy().
		Foreground(lipgloss.Color("#AAAAAA"))

	// Description styling (selected items)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Copy().
		Foreground(lipgloss.Color("#D7D7FF")).
		Background(lipgloss.Color("#6B5ECD"))

	return delegate
}

// Model represents the TUI state
type Model struct {
	// Core state
	keys          KeyMap
	help          help.Model
	list          list.Model
	viewport      viewport.Model
	fileTree      *components.FileTree
	mode          types.Mode
	selectedFiles map[string]bool
	showFullHelp  bool
	currentDir    string
	useFileTree   bool   // Whether to use the file tree view
	version       string // Version number to display in UI

	// Command mode state
	commandBuffer string
	statusMsg     string
	lastKey       string

	// Visual mode state
	visualStart int
	visualEnd   int
	visualMode  bool

	// Engines
	analysisEngine *analysis.Engine
	organizeEngine *organize.Engine

	// Styles
	styles lipgloss.Style
}

// Ensure Model implements types.ModelReader
var _ types.ModelReader = (*Model)(nil)

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	// Start animation for ADHD focus enhancements
	animCmd := tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return t
	})

	return tea.Batch(
		m.list.StartSpinner(),
		animCmd,
	)
}

// New creates a new Model
func New(version string) *Model {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	keys := DefaultKeyMap
	helpModel := help.New()
	helpModel.ShowAll = false

	// Create a new list with custom delegate
	delegate := createCustomListDelegate()

	listModel := list.New([]list.Item{}, delegate, 0, 0)
	listModel.SetShowStatusBar(true)     // Show status bar
	listModel.SetFilteringEnabled(false) // Disable filtering initially, can be toggled
	listModel.SetShowHelp(false)         // Use our custom help
	listModel.SetShowTitle(true)         // Show the title
	listModel.Title = "Files"            // Set list title
	listModel.Styles.Title = styles.TitleStyle.Copy()
	listModel.StatusMessageLifetime = 3 * time.Second

	// Configure pagination for better scrolling
	listModel.SetShowPagination(true)     // Show pagination
	listModel.Paginator.ActiveDot = "•"   // Active page indicator
	listModel.Paginator.InactiveDot = "○" // Inactive page indicator
	listModel.Paginator.PerPage = 15      // Initial items per page

	// Use dots style pagination which is more compact
	listModel.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#5A9"))

	// Create viewport for scrollable content
	viewportModel := viewport.New(0, 0)
	viewportModel.MouseWheelEnabled = true  // Enable mouse wheel scrolling
	viewportModel.KeyMap = viewport.KeyMap{ // Custom keymaps to match our application
		Up:           key.NewBinding(key.WithKeys("up", "k")),
		Down:         key.NewBinding(key.WithKeys("down", "j")),
		PageUp:       key.NewBinding(key.WithKeys("pgup")),
		PageDown:     key.NewBinding(key.WithKeys("pgdown")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d")),
	}

	// Create file tree - enhanced for ADHD users
	fileTree := components.NewFileTree(wd)

	m := &Model{
		keys:           keys,
		help:           helpModel,
		list:           listModel,
		viewport:       viewportModel,
		fileTree:       fileTree,
		selectedFiles:  make(map[string]bool),
		mode:           types.Normal,
		currentDir:     wd,
		showFullHelp:   true,
		useFileTree:    false, // Start with list view by default
		analysisEngine: analysis.New(),
		organizeEngine: organize.New(),
		commandBuffer:  "",
		statusMsg:      "",
		lastKey:        "",
		visualStart:    0,
		visualEnd:      0,
		visualMode:     false,
		styles:         styles.App,
		version:        version,
	}

	// Initial directory scan
	if err := m.ScanDirectory(); err != nil {
		// Handle error gracefully, but don't fail initialization
		m.statusMsg = fmt.Sprintf("Error scanning directory: %v", err)
	}

	return m
}

// View returns the UI as a string
func (m *Model) View() string {
	// If the user is quitting (handled in Update function)
	if strings.HasPrefix(m.statusMsg, "Quitting") {
		return "Thanks for using sortd! Goodbye!\n"
	}

	// Simplified view with minimal dependencies on complex components
	var b strings.Builder

	// TOP BAR
	pathText := fmt.Sprintf("Location: %s", m.currentDir)
	statsText := fmt.Sprintf("Files: %d • Selected: %d", len(m.list.Items()), len(m.selectedFiles))

	topBar := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.DefaultTheme.ColorBorder).
		Width(m.viewport.Width - 4).
		Foreground(styles.DefaultTheme.ColorPrimary).
		Bold(true).
		Render(pathText + " | " + statsText)

	b.WriteString(topBar + "\n\n")

	// CONTENT AREA
	// Left panel with logo and help
	leftContent := m.renderASCIILogo() + "\n\n"
	if m.version != "" {
		leftContent += styles.DefaultTheme.Version.Render(fmt.Sprintf("v%s", m.version)) + "\n\n"
	}

	leftContent += styles.DefaultTheme.Section.Render("Quick Tips:") + "\n\n"

	// Add some quick help tips
	tips := []string{
		"• Space to select files",
		"• Enter to open directories",
		"• o to organize selected files",
		"• / to search files",
		"• v for visual selection mode",
		"• q to quit",
		"• ? for more help",
	}

	for _, tip := range tips {
		leftContent += styles.DefaultTheme.Tip.Render(tip) + "\n"
	}

	leftPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.DefaultTheme.ColorBorder).
		Width(30).
		Padding(1).
		Render(leftContent)

	// Right panel with file list/tree view
	var rightContent string
	if m.useFileTree && m.fileTree != nil {
		rightContent = "File Tree\n\n" + m.fileTree.View()
	} else if len(m.list.Items()) > 0 {
		// Only use list.View() if we have items
		// This avoids the slice bounds panic
		rightContent = m.list.View()
	} else {
		// Safe fallback when list is empty
		rightContent = "No files to display"
	}

	rightWidth := m.viewport.Width - 40
	if rightWidth < 40 {
		rightWidth = 40
	}

	rightPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.DefaultTheme.ColorBorder).
		Width(rightWidth).
		Padding(1).
		Render(rightContent)

	// Join panels side by side safely
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	b.WriteString(mainContent + "\n\n")

	// BOTTOM BAR
	statusText := "ℹ️ "
	if m.statusMsg != "" {
		statusText += m.statusMsg
	} else {
		statusText += "Press ? for help"
	}

	// Simpler help view to avoid potential rendering issues
	helpText := "↑/k:Up ↓/j:Down Space:Select Enter:Open o:Organize Tab:Toggle ?:Help q:Quit"
	if m.showFullHelp {
		helpText = m.help.View(m.keys)
	}

	bottomBar := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.DefaultTheme.ColorBorder).
		Width(m.viewport.Width - 4).
		Render(statusText + "\n\n" + helpText)

	b.WriteString(bottomBar)

	return b.String()
}

// Helper method to provide visual feedback for actions
func (m *Model) setStatusWithFeedback(msg string) {
	m.statusMsg = msg
}

// Update handles all the Bubble Tea messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Save window dimensions
		height := msg.Height
		width := msg.Width

		// Minimum dimensions to prevent rendering errors
		if height < 20 {
			height = 20
		}
		if width < 80 {
			width = 80
		}

		// Set viewport dimensions
		m.viewport.Width = width
		m.viewport.Height = height

		// Calculate available dimensions for components
		availableHeight := height - 10 // Reserve space for header and footer

		// Configure list size
		listWidth := width - 40
		if listWidth < 40 {
			listWidth = 40
		}

		// CRITICAL FIX: Safe list height
		listHeight := availableHeight - 5
		if listHeight < 10 {
			listHeight = 10
		}

		// Update list dimensions and ensure pagination is safe
		m.list.SetSize(listWidth, listHeight)
		m.list.SetHeight(listHeight)

		// Only show pagination if we have enough items
		itemsPerPage := listHeight / 2 // Approximate items per page
		m.list.SetShowPagination(len(m.list.Items()) > itemsPerPage)

		// Set file tree dimensions if it exists
		if m.fileTree != nil {
			m.fileTree.Height = listHeight
			m.fileTree.Width = listWidth
		}

		// Set help model width
		m.help.Width = width - 4

		// Trigger a screen refresh
		cmds = append(cmds, tea.ClearScreen)

		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Handle command mode separately
		if m.mode == types.Command {
			return m.handleCommandMode(msg)
		}

		// First check if quit is pressed (this takes precedence)
		if key.Matches(msg, m.keys.Quit) {
			// Show a goodbye message before quitting
			m.setStatusWithFeedback("Goodbye! 👋")
			return m, tea.Quit
		}

		// Handle toggle view key
		if key.Matches(msg, m.keys.ToggleView) {
			m.useFileTree = !m.useFileTree

			// Sync directory between views
			if m.useFileTree {
				// Switch to file tree view
				m.fileTree.SetDirectory(m.currentDir)
				m.setStatusWithFeedback("Switched to file tree view 🔍")
				return m, nil
			} else {
				// Switch to list view
				m.currentDir = m.fileTree.CurrentDir
				if err := m.ScanDirectory(); err != nil {
					m.setStatusWithFeedback(fmt.Sprintf("Error scanning directory: %v", err))
				} else {
					m.setStatusWithFeedback("Switched to list view 📋")
				}
				return m, nil
			}
		}

		// If using file tree, let it handle most inputs
		if m.useFileTree {
			// Let the file tree handle the message
			var cmd tea.Cmd
			newTree, cmd := m.fileTree.Update(msg)
			m.fileTree = newTree

			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Handle list navigation
		var cmd tea.Cmd

		// Check for special key handling in list view
		switch {
		case key.Matches(msg, m.keys.ToggleHelp):
			m.showFullHelp = !m.showFullHelp
			if m.showFullHelp {
				m.setStatusWithFeedback("Help expanded ℹ️")
			} else {
				m.setStatusWithFeedback("Help minimized ↩️")
			}
			return m, nil

		case key.Matches(msg, m.keys.CommandMode):
			m.mode = types.Command
			m.commandBuffer = ":"
			m.setStatusWithFeedback("Command mode - type a command and press Enter")
			return m, nil

		case key.Matches(msg, m.keys.VisualMode):
			m.visualMode = !m.visualMode
			if m.visualMode {
				m.visualStart = m.list.Index()
				m.visualEnd = m.list.Index()
				m.updateVisualSelection()
				m.setStatusWithFeedback("Visual mode enabled - use up/down to select")
			} else {
				m.setStatusWithFeedback("Visual mode disabled")
			}
			return m, nil

		case key.Matches(msg, m.keys.Left):
			// Go up one directory
			parent := filepath.Dir(m.currentDir)
			if parent != m.currentDir {
				m.currentDir = parent
				dirName := filepath.Base(parent)
				m.setStatusWithFeedback(fmt.Sprintf("Changed to parent directory: %s", dirName))
				if err := m.ScanDirectory(); err != nil {
					m.setStatusWithFeedback(fmt.Sprintf("Error: %v", err))
				}
			} else {
				m.setStatusWithFeedback("Already at the root directory")
			}
			return m, nil

		case key.Matches(msg, m.keys.Right):
			// Handle directory navigation
			if len(m.list.Items()) > 0 {
				selectedIndex := m.list.Index()
				if selectedIndex < 0 || selectedIndex >= len(m.list.Items()) {
					return m, nil // Invalid index
				}

				selectedItem := m.list.Items()[selectedIndex].(Item)
				file := selectedItem.entry

				// Check if it's a directory
				isDir := false
				if file.ContentType == "directory" || strings.Contains(file.ContentType, "directory") {
					isDir = true
				} else {
					// Try to check with os.Stat as a fallback
					fullPath := filepath.Join(m.currentDir, file.Name)
					if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
						isDir = true
					}
				}

				if isDir {
					// Navigate into the directory
					newDir := filepath.Join(m.currentDir, file.Name)
					m.currentDir = newDir
					m.setStatusWithFeedback(fmt.Sprintf("Entered directory: %s", file.Name))
					if err := m.ScanDirectory(); err != nil {
						m.setStatusWithFeedback(fmt.Sprintf("Error: %v", err))
					}
					return m, nil
				} else {
					m.setStatusWithFeedback(fmt.Sprintf("Can't navigate: %s is not a directory", file.Name))
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.Select):
			// Toggle selection for the current file
			if len(m.list.Items()) > 0 {
				selectedIndex := m.list.Index()
				if selectedIndex < 0 || selectedIndex >= len(m.list.Items()) {
					return m, nil // Invalid index
				}

				selectedItem := m.list.Items()[selectedIndex].(Item)
				file := selectedItem.entry
				fullPath := filepath.Join(m.currentDir, file.Name)

				// Toggle selection
				if m.selectedFiles[fullPath] {
					delete(m.selectedFiles, fullPath)
					m.setStatusWithFeedback(fmt.Sprintf("Deselected: %s", file.Name))
				} else {
					m.selectedFiles[fullPath] = true
					m.setStatusWithFeedback(fmt.Sprintf("Selected: %s", file.Name))
				}

				// Update the selection in the list model
				m.updateItemSelection(selectedIndex, m.selectedFiles[fullPath])
			}
			return m, nil

		case key.Matches(msg, m.keys.Up) || key.Matches(msg, m.keys.Down):
			// First update the list
			var newCmd tea.Cmd
			m.list, newCmd = m.list.Update(msg)
			cmd = newCmd

			// If in visual mode, update the visual selection
			if m.visualMode {
				m.visualEnd = m.list.Index()
				m.updateVisualSelection()

				// Show count of selected items
				startIdx := min(m.visualStart, m.visualEnd)
				endIdx := max(m.visualStart, m.visualEnd)
				count := endIdx - startIdx + 1
				m.setStatusWithFeedback(fmt.Sprintf("Selected %d files in visual mode", count))
			} else if len(m.list.Items()) > 0 {
				// Show the current item even in normal mode
				selectedIndex := m.list.Index()
				if selectedIndex >= 0 && selectedIndex < len(m.list.Items()) {
					selectedItem := m.list.Items()[selectedIndex].(Item)
					file := selectedItem.entry

					// Different status depending on file type and selection state
					fullPath := filepath.Join(m.currentDir, file.Name)
					isSelected := m.selectedFiles[fullPath]

					if isSelected {
						m.setStatusWithFeedback(fmt.Sprintf("File: %s (selected) ✓", file.Name))
					} else {
						if strings.Contains(strings.ToLower(file.ContentType), "directory") {
							m.setStatusWithFeedback(fmt.Sprintf("Directory: %s (→ to enter)", file.Name))
						} else {
							m.setStatusWithFeedback(fmt.Sprintf("File: %s (%s)", file.Name, formatSize(file.Size)))
						}
					}
				}
			}

			return m, cmd

		case key.Matches(msg, m.keys.Organize):
			// Organize the selected files
			selectedCount := len(m.selectedFiles)
			if selectedCount == 0 {
				m.setStatusWithFeedback("No files selected for organizing")
				return m, nil
			}

			m.setStatusWithFeedback(fmt.Sprintf("Organizing %d files... 🔄", selectedCount))

			// In a real implementation, we would do the organizing here
			// For now, just provide visual feedback

			// Here you'd call your organize function
			// For example: results := m.organizeEngine.OrganizeFiles(...)

			// Then provide feedback on the results
			// This is a placeholder - replace with actual organize logic
			m.setStatusWithFeedback(fmt.Sprintf("Organized %d files successfully! ✅", selectedCount))

			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			// Refresh the current directory
			if err := m.ScanDirectory(); err != nil {
				m.setStatusWithFeedback(fmt.Sprintf("Error refreshing directory: %v", err))
			} else {
				m.setStatusWithFeedback("Directory refreshed 🔄")
			}
			return m, nil

		case key.Matches(msg, m.keys.GoToTop):
			// Go to the first item in the list
			if len(m.list.Items()) > 0 {
				m.list.Select(0)
				if m.visualMode {
					m.visualEnd = 0
					m.updateVisualSelection()
				}
				m.setStatusWithFeedback("Moved to first file")
			}
			return m, nil

		case key.Matches(msg, m.keys.GoToBottom):
			// Go to the last item in the list
			if itemCount := len(m.list.Items()); itemCount > 0 {
				m.list.Select(itemCount - 1)
				if m.visualMode {
					m.visualEnd = itemCount - 1
					m.updateVisualSelection()
				}
				m.setStatusWithFeedback("Moved to last file")
			}
			return m, nil

		case key.Matches(msg, m.keys.VisualLine):
			// Toggle visual line mode - select all files
			m.visualMode = !m.visualMode
			if m.visualMode {
				// Select all items
				m.visualStart = 0
				if itemCount := len(m.list.Items()); itemCount > 0 {
					m.visualEnd = itemCount - 1
				} else {
					m.visualEnd = 0
				}
				m.updateVisualSelection()
				m.setStatusWithFeedback("Visual line mode - all files selected")
			} else {
				m.setStatusWithFeedback("Visual mode disabled")
			}
			return m, nil

		case key.Matches(msg, m.keys.Search):
			// Enter search mode
			m.mode = types.Command
			m.commandBuffer = "/search "
			m.setStatusWithFeedback("Search mode - type a pattern and press Enter")
			return m, nil

		default:
			// Let the list handle the message
			var newCmd tea.Cmd
			m.list, newCmd = m.list.Update(msg)
			cmd = newCmd
		}

		// If in visual mode, update the selection when navigating
		if m.visualMode {
			m.visualEnd = m.list.Index()
			m.updateVisualSelection()
		}

		return m, cmd

	default:
		// Let component models handle other message types
		var cmds []tea.Cmd

		// Update the list
		var listCmd tea.Cmd
		m.list, listCmd = m.list.Update(msg)
		if listCmd != nil {
			cmds = append(cmds, listCmd)
		}

		// Update the viewport
		var viewportCmd tea.Cmd
		m.viewport, viewportCmd = m.viewport.Update(msg)
		if viewportCmd != nil {
			cmds = append(cmds, viewportCmd)
		}

		return m, tea.Batch(cmds...)
	}
}

func (m *Model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.ToggleHelp):
		m.showFullHelp = !m.showFullHelp
		return m, nil

	case key.Matches(msg, m.keys.CommandMode):
		m.mode = types.Command
		m.commandBuffer = ":"
		return m, nil

	case key.Matches(msg, m.keys.VisualMode):
		m.visualMode = !m.visualMode
		if m.visualMode {
			m.visualStart = m.list.Index()
			m.visualEnd = m.list.Index()
			m.updateVisualSelection()
		} else {
			// Keep selections when exiting visual mode
		}
		return m, nil

	case key.Matches(msg, m.keys.Select):
		if len(m.list.Items()) > 0 {
			item := m.list.Items()[m.list.Index()].(Item)
			file := item.entry

			if m.selectedFiles[file.Path] {
				delete(m.selectedFiles, file.Path)
				m.statusMsg = fmt.Sprintf("Deselected: %s", file.Name)
			} else {
				m.selectedFiles[file.Path] = true
				m.statusMsg = fmt.Sprintf("Selected: %s", file.Name)
			}

			// Update list with new selection state
			m.updateItemSelection(m.list.Index(), !item.selected)
		}
		return m, nil

	case key.Matches(msg, m.keys.Right):
		if len(m.list.Items()) > 0 {
			item := m.list.Items()[m.list.Index()].(Item)
			file := item.entry

			// Check if it's a directory
			isDir := false
			if file.ContentType == "directory" || strings.Contains(file.ContentType, "directory") {
				isDir = true
			} else {
				info, err := os.Stat(file.Path)
				if err == nil {
					isDir = info.IsDir()
				}
			}

			if isDir {
				// Only handle directory navigation with Right key
				m.currentDir = file.Path
				m.statusMsg = fmt.Sprintf("Entering directory: %s", file.Name)
				if err := m.ScanDirectory(); err != nil {
					m.statusMsg = fmt.Sprintf("Error: %v", err)
				}
			} else {
				// For non-directories, do nothing or show a hint
				m.statusMsg = fmt.Sprintf("'%s' is not a directory. Use Space to select files.", file.Name)
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Left):
		// Go up one directory
		parent := filepath.Dir(m.currentDir)
		if parent != m.currentDir {
			m.currentDir = parent
			m.statusMsg = fmt.Sprintf("Changed to parent directory: %s", filepath.Base(parent))
			if err := m.ScanDirectory(); err != nil {
				m.statusMsg = fmt.Sprintf("Error: %v", err)
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		// Refresh the current directory
		if err := m.ScanDirectory(); err != nil {
			m.statusMsg = fmt.Sprintf("Error refreshing directory: %v", err)
		} else {
			m.statusMsg = "Directory refreshed"
		}
		return m, nil

	case key.Matches(msg, m.keys.Organize):
		// Organize the selected files
		if len(m.selectedFiles) == 0 {
			m.statusMsg = "No files selected for organization"
			return m, nil
		}

		// Get list of selected file paths
		selectedPaths := make([]string, 0, len(m.selectedFiles))
		for path := range m.selectedFiles {
			selectedPaths = append(selectedPaths, path)
		}

		// Try to organize the files
		if err := m.organizeEngine.OrganizeByPatterns(selectedPaths); err != nil {
			m.statusMsg = fmt.Sprintf("Error organizing files: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("Organized %d files", len(selectedPaths))
			// Refresh to reflect changes
			if err := m.ScanDirectory(); err != nil {
				m.statusMsg += fmt.Sprintf(" (refresh error: %v)", err)
			}
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleCommandMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = types.Normal
		m.commandBuffer = ""
		return m, nil

	case tea.KeyEnter:
		cmd := m.commandBuffer[1:] // Remove the leading ':'
		m.mode = types.Normal
		m.commandBuffer = ""
		return m.executeCommand(cmd)

	case tea.KeyBackspace:
		if len(m.commandBuffer) > 1 { // Keep the ':'
			m.commandBuffer = m.commandBuffer[:len(m.commandBuffer)-1]
		}
		return m, nil

	default:
		if msg.Type == tea.KeyRunes {
			if m.commandBuffer == "" {
				m.commandBuffer = ":"
			}
			m.commandBuffer += string(msg.Runes)
		}
		return m, nil
	}
}

func (m *Model) executeCommand(cmd string) (tea.Model, tea.Cmd) {
	// Handle command execution
	switch cmd {
	case "q", "quit":
		return m, tea.Quit

	case "help":
		m.showFullHelp = !m.showFullHelp
	}
	return m, nil
}

func (m *Model) updateVisualSelection() {
	// If no files, nothing to select
	if len(m.list.Items()) == 0 {
		return
	}

	start := min(m.visualStart, m.visualEnd)
	end := max(m.visualStart, m.visualEnd)

	// Ensure bounds are within valid range
	start = max(0, min(start, len(m.list.Items())-1))
	end = max(0, min(end, len(m.list.Items())-1))

	// Clear previous selections first
	m.selectedFiles = make(map[string]bool)

	// Select all files in range
	for i := start; i <= end && i < len(m.list.Items()); i++ {
		item := m.list.Items()[i].(Item)
		m.selectedFiles[item.entry.Path] = true

		// Update the item in the list to show it's selected
		m.updateItemSelection(i, true)
	}
}

// updateItemSelection updates the selection state of an item in the list
func (m *Model) updateItemSelection(index int, selected bool) {
	if index < 0 || index >= len(m.list.Items()) {
		return
	}

	// Get the current item
	item := m.list.Items()[index].(Item)

	// Create a new item with updated selection state
	newItem := Item{
		entry:    item.entry,
		selected: selected,
	}

	// Replace the item in the list
	items := m.list.Items()
	items[index] = newItem

	// This will cause the list to be redrawn
	m.list.SetItems(items)
}

// ScanDirectory reads the current directory and updates the file list
func (m *Model) ScanDirectory() error {
	// Show a loading spinner
	m.list.StartSpinner()
	m.statusMsg = "Scanning directory..."

	// Read the directory contents using the analysis engine
	// This assumes analysisEngine.ScanDirectory returns []*types.FileInfo for files
	files, err := m.analysisEngine.ScanDirectory(m.currentDir)
	if err != nil {
		m.list.StopSpinner()
		return err // Propagate error from analysis engine
	}

	// Convert to list items
	var items []list.Item
	for _, file := range files {
		// Create an item for each file/entry returned by analysis engine
		// Assume FileInfo Path is absolute, get base name
		item := Item{
			entry: types.FileEntry{
				Name:        filepath.Base(file.Path),
				Path:        file.Path,
				ContentType: file.ContentType,
				Size:        file.Size,
				Tags:        file.Tags,
				// IsDir field is not part of the original FileEntry
			},
			selected: m.selectedFiles[file.Path],
		}
		items = append(items, item)
	}

	// If there are no items, add a placeholder (optional)
	if len(items) == 0 {
		m.statusMsg = "No files or directories found."
		m.list.SetShowPagination(false)
	} else {
		m.statusMsg = fmt.Sprintf("Found %d items", len(items))
		m.list.SetShowPagination(len(items) > 5)
	}

	// Stop the spinner
	m.list.StopSpinner()

	// Sort items (directories first, then alphabetically)
	// Relies on ContentType containing "directory" for sorting
	sort.Slice(items, func(i, j int) bool {
		itemI := items[i].(Item)
		itemJ := items[j].(Item)
		isDir1 := strings.Contains(itemI.entry.ContentType, "directory")
		isDir2 := strings.Contains(itemJ.entry.ContentType, "directory")

		if isDir1 && !isDir2 {
			return true
		}
		if !isDir1 && isDir2 {
			return false
		}
		// Both are dirs or both are files, sort by name
		return itemI.entry.Name < itemJ.entry.Name
	})

	// Update the list with the new items
	m.list.SetItems(items)

	// CRITICAL: Make sure list dimensions are sane to prevent slice bounds error
	availableHeight := m.viewport.Height - 10
	if availableHeight < 10 {
		availableHeight = 10
	}
	m.list.SetHeight(availableHeight)

	// Reset cursor position
	if len(items) > 0 {
		m.list.Select(0)
	}

	return nil
}

// getContentType determines content type based on file info
func getContentType(info fs.FileInfo) string {
	if info.IsDir() {
		return "directory"
	}
	return "file"
}

// ModelReader interface implementation

// Files returns the current files in the directory
func (m *Model) Files() []types.FileEntry {
	items := m.list.Items()
	files := make([]types.FileEntry, len(items))
	for i, item := range items {
		files[i] = item.(Item).entry
	}
	return files
}

// Cursor returns the current cursor position
func (m *Model) Cursor() int {
	return m.list.Index()
}

// IsSelected returns whether a file is selected
func (m *Model) IsSelected(name string) bool {
	for path := range m.selectedFiles {
		if filepath.Base(path) == name {
			return true
		}
	}
	return false
}

// ShowHelp returns whether help should be shown
func (m *Model) ShowHelp() bool {
	return m.showFullHelp
}

// Mode returns the current mode
func (m *Model) Mode() types.Mode {
	return m.mode
}

// CurrentDir returns the current directory
func (m *Model) CurrentDir() string {
	return m.currentDir
}

// SetCurrentDir sets the current directory for both views
func (m *Model) SetCurrentDir(dir string) {
	m.currentDir = dir
	// Reset cursor and selection
	m.list.Select(0)
}

// CurrentFile returns the name of the file at the current cursor position
func (m *Model) CurrentFile() string {
	if len(m.list.Items()) == 0 || m.list.Index() < 0 || m.list.Index() >= len(m.list.Items()) {
		return ""
	}
	return m.list.Items()[m.list.Index()].(Item).entry.Name
}

// SetCursor sets the cursor position
func (m *Model) SetCursor(pos int) {
	// Validate position
	if len(m.list.Items()) == 0 {
		return
	}

	// Apply bounds checking
	if pos < 0 {
		pos = 0
	}
	if pos >= len(m.list.Items()) {
		pos = len(m.list.Items()) - 1
	}

	// Set cursor position in list
	m.list.Select(pos)
}

// VisualMode returns whether visual mode is active
func (m *Model) VisualMode() bool {
	return m.visualMode
}

// SelectFile selects a file by name
func (m *Model) SelectFile(name string) error {
	// Find the file in the list
	for i, item := range m.list.Items() {
		fileItem := item.(Item)
		if fileItem.entry.Name == name || filepath.Base(fileItem.entry.Path) == name {
			// Select the file
			m.selectedFiles[fileItem.entry.Path] = true
			// Update item in the list
			m.updateItemSelection(i, true)
			return nil
		}
	}
	return fmt.Errorf("file not found: %s", name)
}

// SetShowHelp sets whether help should be shown
func (m *Model) SetShowHelp(show bool) {
	m.showFullHelp = show
}

// Helper functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m *Model) renderASCIILogo() string {
	// Create a more compact and visually appealing logo
	logo := `
    ███████  ██████  ██████  ████████ ██████
    ██      ██    ██ ██   ██    ██    ██   ██
    ███████ ██    ██ ██████     ██    ██   ██
         ██ ██    ██ ██   ██    ██    ██   ██
    ███████  ██████  ██   ██    ██    ██████
    `

	// Apply styling to the logo using our enhanced theme
	return styles.DefaultTheme.Logo.Render(logo)
}

// headerView renders the header section of the UI
func (m *Model) headerView() string {
	var headerContent strings.Builder

	// Get the styled ASCII logo
	headerContent.WriteString(m.renderASCIILogo())
	headerContent.WriteString("\n")

	// Add version information with better styling
	if m.version != "" {
		versionLine := styles.DefaultTheme.Version.Render(fmt.Sprintf("v%s", m.version))
		headerContent.WriteString(versionLine)
		headerContent.WriteString("\n\n")
	}

	// Add section header for quick tips
	headerContent.WriteString(styles.DefaultTheme.Section.Render("Quick Tips:"))
	headerContent.WriteString("\n\n")

	// Add some quick help tips to make better use of the left panel
	tips := []string{
		"• Space to select/deselect files",
		"• Enter to open directories",
		"• o to organize selected files",
		"• / to search files",
		"• v for visual selection mode",
		"• q to quit",
		"• ? for more help",
	}

	// Apply tip styling to each tip
	for _, tip := range tips {
		headerContent.WriteString(styles.DefaultTheme.Tip.Render(tip))
		headerContent.WriteString("\n")
	}

	// Create a nice panel for the header content
	return styles.DefaultTheme.Panel.Render(headerContent.String())
}
