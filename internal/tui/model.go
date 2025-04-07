package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sortd/internal/analysis"
	"sortd/internal/config"
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

// Message Types for TUI updates
type statusMessage string

// View types for the TUI
type ActiveView int

const (
	ViewList ActiveView = iota
	ViewTree
)

// UI Constants
const (
	StatusBarHeight = 1
	HeaderHeight = 2
)

type filesLoadedMsg struct {
	files []types.FileEntry
	err   error
}
// Define a Results type for analysis package
type Results struct {
	Files []*types.FileInfo
	Stats map[string]int
}

type analysisCompleteMsg struct {
	results *Results
	err     error
}
type organizationCompleteMsg struct {
	err error
}
type directoryCreatedMsg struct {
	name string
	err  error
}
type directoryDeletedMsg struct {
	name string
	err  error
}
type fileDeletedMsg struct {
	name string
	err  error
}
type fileRenamedMsg struct {
	oldName string
	newName string
	err     error
}
type filesSelectedMsg struct {
	selected map[string]bool
}

// errorMsg wraps an error for tea.Cmd processing
type errorMsg struct{ err error }

// Implement the error interface for errorMsg
func (e errorMsg) Error() string { return e.err.Error() }

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
	// Status and command fields
	statusMsg    string
	commandInput string
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

	lastKey string

	// Visual mode state
	visualStart int
	visualEnd   int
	visualMode  bool

	// Engines
	analysisEngine *analysis.Engine
	organizeEngine *organize.Engine

	// Styles
	styles lipgloss.Style

	// Additional fields for refactored Update
	loading         bool
	organizing      bool
	config          *config.Config
	analysisResults *Results
	width           int
	height          int
	activeView      ActiveView
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
	// Determine active panel for border styling
	listActive := m.activeView == ViewList
	treeActive := m.activeView == ViewTree

	// Apply active/inactive styles
	listStyle := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder())
	treeStyle := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder())
	if listActive {
		listStyle = listStyle.BorderStyle(lipgloss.DoubleBorder())
	}
	if treeActive {
		treeStyle = treeStyle.BorderStyle(lipgloss.DoubleBorder())
	}

	// File List View
	listViewContent := m.list.View()

	// File Tree View
	// Assuming FileTree component has a View() method
	fileTreeViewContent := m.fileTree.View()

	// Use appropriate border style based on active view
	mainPanelContent := ""
	if m.activeView == ViewList {
		// Apply the appropriate border style based on active state
		listBorderStyle := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder())
		if listActive {
			listBorderStyle = listBorderStyle.BorderStyle(lipgloss.DoubleBorder())
		}

		mainPanelContent = listBorderStyle.
			Width(m.list.Width()).   // Use the list's width
			Height(m.list.Height()). // Use the list's height
			Render(listViewContent)
	} else {
		// Apply the appropriate border style based on active state
		treeBorderStyle := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder())
		if treeActive {
			treeBorderStyle = treeBorderStyle.BorderStyle(lipgloss.DoubleBorder())
		}

		// We need to implement GetWidth and GetHeight methods for fileTree
		// For now, use reasonable defaults
		treeWidth := m.width / 2
		treeHeight := m.height - StatusBarHeight - HeaderHeight // accounting for borders

		mainPanelContent = treeBorderStyle.
			Width(treeWidth).
			Height(treeHeight).
			Render(fileTreeViewContent)
	}

	// Viewport (File Preview) - potentially overlaps or replaces list view based on state
	// Example: Only show viewport when a file is selected and preview is enabled?
	// For now, let's render it below the list/tree. Adjust layout as needed.
	viewportContent := ""
	if m.viewport.Height > 0 && m.list.SelectedItem() != nil { // Basic condition to show viewport
		viewportStyle := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder())
		viewportContent = viewportStyle.Render(m.viewport.View())
	}

	// Left Panel (Header/Logo/Tips)
	leftPanel := m.headerView() // Assuming headerView provides the styled left panel content

	// Combine Left and Right panels horizontally
	// Adjust widths dynamically based on m.width
	// Ensure total width doesn't exceed m.width
	// Width allocation is handled in the Update method

	// Ensure panels don't overlap and fit within the total width
	// Recalculate rightPanelWidth if leftPanel width is fixed or different
	// rightPanelWidth = m.width - lipgloss.Width(leftPanel) - 1 // Alternative calculation

	// Create the main layout using the calculated panel widths
	mainViewLayout := lipgloss.JoinHorizontal(lipgloss.Top,
		leftPanel,
		mainPanelContent, // Contains either list or tree view with border
	)

	// Combine main layout with viewport (if shown) and status bar
	finalView := lipgloss.JoinVertical(lipgloss.Left,
		mainViewLayout,
		viewportContent,     // Add viewport below
		m.renderStatusBar(), // Status bar/help
	)

	// Ensure the final view respects terminal dimensions
	// This might involve truncation or scrolling handled by components
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, finalView)
}

// Update handles all the Bubble Tea messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Store the dimensions
		m.width = msg.Width
		m.height = msg.Height

		// Calculate panel sizes
		// Example: Use a 1/3, 2/3 split for the two main panels
		leftPanelWidth := m.width / 3
		// rightPanelWidth := m.width - leftPanelWidth - 1 // Account for divider/padding

		// Update help
		m.help.Width = m.width

		// Update list dimensions (use right panel width)
		m.list.SetSize(leftPanelWidth, m.height-2-1) // Adjust height for status bar and header

		// Update viewport dimensions (used for preview)
		// This might depend on which view is active, let's assume it shares space with the list for now
		m.viewport.Width = leftPanelWidth
		m.viewport.Height = m.height - 2 - 1

		// Update help
		m.help.Width = m.width
		// If the viewport needs different dimensions based on context, adjust here

	case statusMessage:
		m.statusMsg = string(msg)
		return m, nil

	case filesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error loading directory: %v", msg.err)
			m.list.SetItems(nil) // Clear list on error
			return m, nil
		}

		items := make([]list.Item, len(msg.files))
		for i, entry := range msg.files {
			items[i] = Item{entry: entry}
		}

		// Sort items (directories first, then by name)
		sort.SliceStable(items, func(i, j int) bool {
			itemI := items[i].(Item)
			itemJ := items[j].(Item)
			if itemI.entry.IsDir && !itemJ.entry.IsDir {
				return true
			}
			if !itemI.entry.IsDir && itemJ.entry.IsDir {
				return false
			}
			return itemI.entry.Name < itemJ.entry.Name
		})

		m.list.SetItems(items)
		m.list.Select(0) // Reset cursor to the top
		m.statusMsg = fmt.Sprintf("%d items loaded", len(items))
		// Update viewport content if showing preview
		m.viewport.SetContent(m.getCurrentPreviewContent())
		return m, nil // Return nil cmd after loading

	case analysisCompleteMsg:
		m.loading = false // Should already be false, but just in case
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error analyzing directory: %v", msg.err)
			// Clear analysis view?
			return m, nil
		}
		m.analysisResults = msg.results
		m.statusMsg = fmt.Sprintf("Analysis complete for %s", m.currentDir)
		// Update viewport content if showing preview and analysis is relevant
		m.viewport.SetContent(m.getCurrentPreviewContent())
		return m, nil

	case organizationCompleteMsg:
		m.loading = false
		m.organizing = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Organization failed: %v", msg.err)
			// Potentially return an error command or specific error message type
			return m, m.setStatus(fmt.Sprintf("Error: %v", msg.err)) // Use setStatus for feedback
		}
		// Clear selection after successful organization
		m.selectedFiles = make(map[string]bool)
		m.visualMode = false
		m.visualStart = -1
		// Refresh directory view
		m.statusMsg = "Organization complete."
		cmds = append(cmds, m.loadDirectory(m.currentDir)) // Reload directory content
		return m, tea.Batch(cmds...)

	case directoryCreatedMsg:
		if msg.err != nil {
			cmds = append(cmds, m.setStatus(fmt.Sprintf("Error creating dir: %v", msg.err)))
			return m, tea.Batch(cmds...)

		}
		cmds = append(cmds, m.setStatus(fmt.Sprintf("Directory '%s' created", msg.name)))
		cmds = append(cmds, m.loadDirectory(m.currentDir)) // Refresh list
		return m, tea.Batch(cmds...)

	case directoryDeletedMsg:
		if msg.err != nil {
			cmds = append(cmds, m.setStatus(fmt.Sprintf("Error deleting dir: %v", msg.err)))
			return m, tea.Batch(cmds...)
		}
		cmds = append(cmds, m.setStatus(fmt.Sprintf("Directory '%s' deleted", msg.name)))
		cmds = append(cmds, m.loadDirectory(m.currentDir)) // Refresh list
		return m, tea.Batch(cmds...)

	case fileDeletedMsg:
		if msg.err != nil {
			cmds = append(cmds, m.setStatus(fmt.Sprintf("Error deleting file: %v", msg.err)))
			return m, tea.Batch(cmds...)
		}
		cmds = append(cmds, m.setStatus(fmt.Sprintf("File '%s' deleted", msg.name)))
		cmds = append(cmds, m.loadDirectory(m.currentDir)) // Refresh list
		return m, tea.Batch(cmds...)

	case fileRenamedMsg:
		if msg.err != nil {
			cmds = append(cmds, m.setStatus(fmt.Sprintf("Error renaming: %v", msg.err)))
			return m, tea.Batch(cmds...)
		}
		cmds = append(cmds, m.setStatus(fmt.Sprintf("Renamed '%s' to '%s'", msg.oldName, msg.newName)))
		cmds = append(cmds, m.loadDirectory(m.currentDir)) // Refresh list
		return m, tea.Batch(cmds...)

	case filesSelectedMsg: // Example: external process signals selection change
		m.selectedFiles = msg.selected
		// Visually update the list items based on this external change
		// This might require iterating through m.list.Items() and calling SetItem
		// Or trigger a full list refresh if simpler
		cmds = append(cmds, m.loadDirectory(m.currentDir)) // Refresh to ensure consistency
		return m, tea.Batch(cmds...)

		// Handle generic error messages if specific types aren't used
	case error: // Catch-all for error types if specific msgs aren't used
		m.statusMsg = fmt.Sprintf("Error: %v", msg)
		m.commandInput = ""
		m.mode = types.Visual // Exit command mode on error
		return m, nil

	case tea.KeyMsg:
		// Handle key presses based on mode
		switch m.mode {
		case types.Command:
			return m.handleCommandMode(msg)
		case types.Visual:
			// Visual mode keys might be handled within normal keys or separately
			return m.handleNormalKeys(msg) // Assuming shared handling for now
		case types.Normal:
			return m.handleNormalKeys(msg)
		default:
			return m, nil // Should not happen
		}

	// Default Case: Bubble Tea internal messages or unknown messages
	default:
		// Pass messages to components that might need them
		// Order might matter depending on focus
		// Handle viewport scrolling if it's the active focus
		// Check m.activeView or another focus indicator if implemented
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

		// Handle list updates (navigation, filtering)
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

		// Update file tree if it exists
		if m.fileTree != nil {
			// Since fileTree is not directly implementing tea.Model, we need to handle it differently
			// For now, we'll just update its state without calling Update directly
			// We'll implement proper update logic for fileTree in a future PR
		}

		// Handle help model updates
		m.help, cmd = m.help.Update(msg)
		cmds = append(cmds, cmd)
	}

	// After handling the message, update the viewport content if necessary
	// This ensures previews are updated after list navigation or file changes
	m.updateViewportContent() // Ensure this method exists and is correct

	// Batch all collected commands
	return m, tea.Batch(cmds...)
}

// handleNormalKeys handles key presses in Normal, Visual, and VisualLine modes
func (m *Model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.ToggleHelp):
		m.showFullHelp = !m.showFullHelp
		return m, nil

	case key.Matches(msg, m.keys.CommandMode):
		m.mode = types.Normal
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
		m.mode = types.Visual
		m.commandBuffer = ""
		return m, nil

	case tea.KeyEnter:
		cmd := m.commandBuffer[1:] // Remove the leading ':'
		m.mode = types.Visual
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

func (m *Model) loadDirectory(dir string) tea.Cmd {
	m.loading = true
	m.statusMsg = fmt.Sprintf("Loading %s...", dir)
	// In a real app, this would return a tea.Cmd that fetches data
	// and sends back a filesLoadedMsg or errorMsg
	return func() tea.Msg {
		// Simulate loading time and success/error
		time.Sleep(50 * time.Millisecond)
		// Example success:
		return filesLoadedMsg{files: []types.FileEntry{{Name: "file1.txt"}, {Name: "subdir", IsDir: true}}, err: nil}
		// Example error:
		// return errorMsg{err: fmt.Errorf("failed to read %s", dir)}
	}
}

func (m *Model) updateViewportContent() {
	selectedItem := m.list.SelectedItem()
	if selectedItem == nil {
		m.viewport.SetContent("No file selected.")
		return
	}

	item := selectedItem.(Item)
	if item.entry.IsDir {
		// Maybe show directory stats or analysis results?
		m.viewport.SetContent(fmt.Sprintf("Directory: %s", item.entry.Name))
	} else {
		// In a real app, load file content here async
		// For now, just show basic info
		m.viewport.SetContent(fmt.Sprintf("File: %s\nSize: %s\nType: %s",
			item.entry.Name,
			formatSize(item.entry.Size),
			item.entry.ContentType))
	}
	// Ensure viewport scrolls to top after content change
	m.viewport.GotoTop()
}

func (m *Model) getCurrentPreviewContent() string {
	// This is a simplified version of what updateViewportContent does
	// It should return the string content directly
	selectedItem := m.list.SelectedItem()
	if selectedItem == nil {
		return "No file selected."
	}
	item := selectedItem.(Item)
	if item.entry.IsDir {
		return fmt.Sprintf("Directory: %s", item.entry.Name)
	}
	// Simulate getting file preview content
	return fmt.Sprintf("Preview for: %s\nSize: %s", item.entry.Name, formatSize(item.entry.Size))
}

func (m *Model) handleError(err error) (tea.Model, tea.Cmd) {
	m.statusMsg = fmt.Sprintf("Error: %v", err)
	m.commandInput = ""
	m.mode = types.Visual // Exit command mode on error
	// Potentially add a timer to clear the error message
	// Return a command to set the status message which includes a clear timer
	return m, m.setStatus(m.statusMsg)
}

// renderStatusBar renders the status bar at the bottom of the screen
// setStatus is a helper to set the status message and potentially trigger a clear timer
func (m *Model) setStatus(msg string) tea.Cmd {
	m.statusMsg = msg
	// Optional: Add a timer to clear the message
	return tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
		if m.statusMsg == msg { // Check if message is still the same
			return statusMessage("")
		}
		return nil
	})
}

// renderStatusBar renders the status bar at the bottom of the screen
func (m *Model) renderStatusBar() string {
	// Create a styled status bar with the current status message
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#333333")).Padding(0, 1)

	// Show help hint or status message
	statusText := m.statusMsg
	if statusText == "" {
		statusText = "Press ? for help"
	}

	// Create the status bar with full width
	statusBar := statusStyle.Width(m.width).Render(statusText)

	return statusBar
}

// End of file
