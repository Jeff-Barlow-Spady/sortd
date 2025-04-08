package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"sortd/internal/analysis"
	"sortd/internal/config"
	"sortd/internal/log"
	"sortd/internal/organize"
	"sortd/internal/tui/components"
	"sortd/internal/tui/styles"
	"sortd/pkg/types"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Message Types reinstated
type statusMessage string

type filesLoadedMsg struct {
	files []types.FileInfo // Use FileInfo
	err   error
}

type analysisCompleteMsg struct {
	results []*types.FileInfo // Changed from analysis.Results
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

// Item represents a list item, embedding the file entry and selection state.
type Item struct {
	entry    types.FileEntry // Changed back to FileEntry
	selected bool
}

// FilterValue implements list.Item
func (i Item) FilterValue() string {
	return i.entry.Name
}

// Title returns the item title with visual indicators for file type
func (i Item) Title() string {
	// Define styles for icons
	fileIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")).Render("📄") // Lighter icon color
	dirIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#5E81AC")).Render("📁")  // Different color for dirs
	dirIndicator := lipgloss.NewStyle().Foreground(lipgloss.Color("#A3BE8C")).Render("/")

	// Determine icon based on whether it's a directory
	icon := fileIcon
	if i.entry.IsDir {
		icon = dirIcon
	}

	// Define base title style
	titleStyle := lipgloss.NewStyle()
	if i.selected {
		// Style for visually selected items (override delegate's selection)
		titleStyle = titleStyle.Foreground(lipgloss.Color("#EBCB8B")).Bold(true)
	}

	// Add a visual indicator for directories (e.g., trailing slash)
	if i.entry.IsDir {
		return lipgloss.JoinHorizontal(lipgloss.Left,
			icon,
			titleStyle.Render(i.entry.Name), // Access field directly
			dirIndicator,
		) // Display with directory indicator
	}

	return lipgloss.JoinHorizontal(lipgloss.Left,
		icon,
		titleStyle.Render(i.entry.Name), // Access field directly
	) // Display file name normally
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
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size < KB:
		return fmt.Sprintf("%dB", size)
	case size < MB:
		return fmt.Sprintf("%.1fKB", float64(size)/KB)
	case size < GB:
		return fmt.Sprintf("%.1fMB", float64(size)/MB)
	default:
		return fmt.Sprintf("%.1fGB", float64(size)/GB)
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
	textInput    textinput.Model
	// Core state
	list          list.Model
	viewport      viewport.Model
	fileTree      *components.FileTree
	mode          types.Mode
	selectedFiles map[string]bool
	showFullHelp  bool
	currentDir    string
	activeView    types.ViewMode // Indicates whether list or tree view is active
	version       string         // Version number to display in UI

	// Command mode state
	commandBuffer string

	// Engines
	analysisEngine *analysis.Engine
	organizeEngine *organize.Engine

	// Styles
	styles lipgloss.Style

	// Additional fields for refactored Update
	loading         bool
	organizing      bool
	width           int
	height          int
	leftPanelWidth  int
	rightPanelWidth int
	mainPanelHeight int
	statusBarHeight int // Usually 1

	// Dimensions (calculated on WindowSizeMsg)
	help help.Model // Help component

	// Visual selection state
	visualMode  bool // Tracks if visual selection is *active*, distinct from main Mode
	visualStart int
	visualEnd   int

	// Concurrency
	mutex sync.Mutex // Mutex for protecting access to shared state

	// Key bindings
	keys *types.KeyMap // Changed to *types.KeyMap
}

// Ensure Model implements types.ModelReader
var _ types.ModelReader = (*Model)(nil)

var _ types.KeyHandlerModel = (*Model)(nil)

// KeyHandlerModel interface implementation

// SetSelectedFiles implements types.KeyHandlerModel.
// TODO: Implement actual logic
func (m *Model) SetSelectedFiles(files map[string]bool) {
	// Placeholder implementation
}

// SetVisualStart implements the KeyHandlerModel interface.
func (m *Model) SetVisualStart(start int) {
	m.visualStart = start
}

// SetVisualEnd implements the KeyHandlerModel interface.
func (m *Model) SetVisualEnd(end int) {
	m.visualEnd = end
}

// SetLoading implements types.KeyHandlerModel
func (m *Model) SetLoading(loading bool) {
	m.loading = loading
}

// SetOrganizing implements types.KeyHandlerModel
func (m *Model) SetOrganizing(organizing bool) {
	m.organizing = organizing
}

// SetVisualMode implements the KeyHandlerModel interface.
func (m *Model) SetVisualMode(visual bool) {
	m.visualMode = visual
}

// Keys returns the key bindings.
func (m *Model) Keys() *types.KeyMap {
	return m.keys
}

// List returns the list component.
func (m *Model) List() list.Model {
	return m.list
}

// Viewport returns the viewport component.
func (m *Model) Viewport() viewport.Model {
	return m.viewport
}

// Mode returns the current TUI mode.
func (m *Model) Mode() types.Mode {
	return m.mode
}

// CurrentDir returns the current directory path.
func (m *Model) CurrentDir() string {
	return m.currentDir
}

// SelectedFiles returns the map of selected file paths.
func (m *Model) SelectedFiles() map[string]bool {
	return m.selectedFiles
}

// SetMode sets the TUI mode.
func (m *Model) SetMode(mode types.Mode) {
	m.mode = mode
	m.textInput.Blur() // Ensure command buffer loses focus when changing mode
}

// SetStatus sets the status message.
func (m *Model) SetStatus(msg string) tea.Cmd {
	m.statusMsg = msg
	return nil
}

// LoadDirectory loads the directory content.
func (m *Model) LoadDirectory(dir string) tea.Cmd {
	m.loading = true
	m.statusMsg = fmt.Sprintf("Loading %s...", dir)
	return func() tea.Msg {
		err := m.ScanDirectory() // ScanDirectory now updates the list internally
		if err != nil {
			return filesLoadedMsg{err: fmt.Errorf("failed to scan %s: %w", dir, err)}
		}
		return filesLoadedMsg{err: nil} // Signal completion (list is already updated)
	}
}

// TriggerOrganizationCmd triggers the organization command.
func (m *Model) TriggerOrganizationCmd(paths []string) tea.Cmd {
	// TODO: Implement organization logic
	return nil
}

// UpdateVisualSelection updates the visual selection.
func (m *Model) UpdateVisualSelection() {
	// TODO: Implement visual selection logic
}

// ToggleSelection toggles the selection state of the currently focused list item.
func (m *Model) ToggleSelection() {
	if len(m.list.Items()) == 0 {
		m.SetStatus("Error: No items in list to select")
		return
	}
	selectedListItem := m.list.SelectedItem()
	if selectedListItem == nil {
		m.SetStatus("Error: No item selected to toggle")
		return
	}

	item, ok := selectedListItem.(Item) // Use the local Item type
	if !ok {
		m.SetStatus("Error: Invalid item type for selection")
		return
	}

	path := item.entry.Path
	if m.selectedFiles[path] {
		delete(m.selectedFiles, path)
		m.SetStatus(fmt.Sprintf("Deselected: %s", item.entry.Name))
	} else {
		m.selectedFiles[path] = true
		m.SetStatus(fmt.Sprintf("Selected: %s", item.entry.Name))
	}
	m.updateStatus() // Ensure status bar reflects change
	m.updateHelp()
}

// ClearSelection deselects all currently selected items.
func (m *Model) ClearSelection() {
	if len(m.selectedFiles) > 0 {
		m.selectedFiles = make(map[string]bool)
		m.SetStatus("Selection cleared")
		m.updateStatus()
		m.updateHelp()
	} else {
		m.SetStatus("No selection to clear")
	}
}

// SetList updates the list component.
func (m *Model) SetList(l list.Model) {
	m.list = l
}

// SetViewport updates the viewport component.
func (m *Model) SetViewport(v viewport.Model) {
	m.viewport = v
}

// SetHelp updates the help component.
func (m *Model) SetHelp(h help.Model) {
	m.help = h
}

// SetShowFullHelp sets the visibility of full help.
func (m *Model) SetShowFullHelp(show bool) {
	m.showFullHelp = show
	m.help.ShowAll = show
	m.updateHelp()
}

// SetCommandBuffer sets the command buffer content.
func (m *Model) SetCommandBuffer(s string) {
	m.textInput.SetValue(s)
	if m.mode == types.Command { // Use local Command mode
		m.textInput.Focus() // Ensure cursor is visible only in command mode
	} else {
		m.textInput.Blur()
	}
}

// SetActiveView sets the active view mode.
func (m *Model) SetActiveView(view types.ViewMode) {
	m.activeView = view
	m.updateStatus()
	m.updateHelp()
	m.UpdateViewportContent() // Update content when view changes
}

// UpdateViewportContent updates the viewport based on the selected list item.
func (m *Model) UpdateViewportContent() {
	selectedListItem := m.list.SelectedItem()
	if selectedListItem == nil {
		m.viewport.SetContent("No item selected.")
		return
	}

	item, ok := selectedListItem.(Item) // Use the local Item type
	if !ok {
		m.viewport.SetContent("Error: Invalid item type.")
		return
	}

	// TODO: Enhance content view (e.g., file preview, analysis details)
	content := fmt.Sprintf("Path: %s\nType: %s\nSize: %d",
		item.entry.Path,
		item.entry.ContentType,
		item.entry.Size,
	)
	m.viewport.SetContent(content)
	m.viewport.GotoTop() // Reset scroll position
}

// Help returns the help component.
func (m *Model) Help() help.Model {
	return m.help
}

// ShowFullHelp returns whether full help is shown.
func (m *Model) ShowFullHelp() bool {
	return m.showFullHelp
}

// CommandBuffer returns the command buffer content.
func (m *Model) CommandBuffer() string {
	return m.textInput.Value()
}

// ActiveView returns the currently active view mode.
func (m *Model) ActiveView() types.ViewMode {
	return m.activeView
}

// StatusMsg returns the current status message.
func (m *Model) StatusMsg() string {
	return m.statusMsg
}

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
	wd, err := filepath.Abs(".") // Use absolute path
	if err != nil {
		wd = "."
	}

	helpModel := help.New()
	helpModel.ShowAll = false

	// Create a new list with custom delegate
	delegate := createCustomListDelegate()

	listModel := list.New(nil, delegate, 0, 0) // Start empty
	listModel.Title = "Loading..."
	listModel.Styles.Title = lipgloss.NewStyle().Copy()
	listModel.SetShowStatusBar(true)     // Show status bar
	listModel.SetFilteringEnabled(false) // Disable filtering initially, can be toggled
	listModel.SetShowHelp(false)         // Use our custom help
	listModel.SetShowTitle(true)         // Show the title
	listModel.StatusMessageLifetime = 3 * time.Second

	// Configure pagination for better scrolling
	listModel.SetShowPagination(true)     // Show pagination
	listModel.Paginator.ActiveDot = "•"   // Active page indicator
	listModel.Paginator.InactiveDot = "○" // Inactive page indicator
	listModel.Paginator.PerPage = 15      // Initial items per page

	// Use dots style pagination which is more compact
	listModel.Styles.PaginationStyle = lipgloss.NewStyle().Copy().Foreground(lipgloss.Color("#5A9"))

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
		textInput:     initializeTextInput(), // Initialize the text input with prompt
		list:          listModel,
		viewport:      viewportModel,
		fileTree:      fileTree,
		selectedFiles: make(map[string]bool),
		mode:          types.Normal,
		currentDir:    wd,
		showFullHelp:  true,
		activeView:    types.ViewList, // Start with list view by default
		version:       version,
		keys: &types.KeyMap{
			// Define initial essential key bindings here
			Quit: key.NewBinding(
				key.WithKeys("q", "ctrl+c"),
				key.WithHelp("q/ctrl+c", "quit"),
			),
			Help: key.NewBinding(
				key.WithKeys("?"),
				key.WithHelp("?", "toggle help"),
			),
			// Navigation
			Up: key.NewBinding(
				key.WithKeys("k", "up"),
				key.WithHelp("k/↑", "up"),
			),
			Down: key.NewBinding(
				key.WithKeys("j", "down"),
				key.WithHelp("j/↓", "down"),
			),
			// Modes
			EnterCmdMode: key.NewBinding(
				key.WithKeys(":"),
				key.WithHelp(":", "enter command mode"),
			),
			// Command Mode Specific
			ExecuteCmd: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "execute command"),
			),
			ExitCmdMode: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "exit command mode"),
			),
			// TODO: Define other key bindings (navigation, selection, modes, etc.)
		},
	}

	// Initialize analysis and organize engines WITH CORRECT CONSTRUCTORS
	m.analysisEngine = analysis.NewWithConfig(config.New()) // Use NewWithConfig
	m.organizeEngine = organize.NewWithConfig(config.New()) // Use NewWithConfig

	// Initial directory scan
	if err := m.ScanDirectory(); err != nil {
		// Handle error gracefully, but don't fail initialization
		m.statusMsg = fmt.Sprintf("Error scanning directory: %v", err)
	}

	return m
}

// View returns the UI as a string
func (m *Model) View() string {
	// Ensure dimensions are calculated (safety check, should happen on init/resize)
	if m.width == 0 || m.height == 0 {
		return "Initializing or resizing..."
	}

	// --- Left Panel (Header/Tips) ---
	// Style the container for the left panel
	leftPanel := lipgloss.NewStyle().
		Width(m.leftPanelWidth).
		Height(m.mainPanelHeight).
		// BorderStyle(lipgloss.NormalBorder()). // Optional border
		Align(lipgloss.Left, lipgloss.Top).
		Render(m.headerView()) // Render header content inside

	// --- Right Panel (List/Tree) ---
	var rightPanelContentView string // Content string from the list/tree view itself

	if m.activeView == types.ViewList {
		// Get the view string from the list component.
		// It should already be sized correctly via m.list.SetSize()
		rightPanelContentView = m.list.View()
	} else { // Tree View
		// Assuming tree view also respects its height or handles scrolling
		rightPanelContentView = m.fileTree.View()
	}

	// --- Apply Border to Right Panel Content (Optional but good for structure) ---
	// Create a style ONLY for the border and width, NOT height.
	rightPanelBorderStyle := lipgloss.NewStyle().
		Width(m.rightPanelWidth). // Ensure border takes the correct width
		// Height(m.mainPanelHeight). // DO NOT SET HEIGHT HERE
		BorderStyle(lipgloss.NormalBorder()) // Default border

	// Set focused border if needed
	if m.activeView == types.ViewList { // Assuming focus follows activeView
		rightPanelBorderStyle = rightPanelBorderStyle.BorderStyle(lipgloss.DoubleBorder())
	}
	// Render the content INSIDE the border style
	rightPanelWithBorder := rightPanelBorderStyle.Render(rightPanelContentView)

	// --- Assemble Panels ---
	// Join Left panel and the (bordered) Right panel content horizontally
	mainLayout := lipgloss.JoinHorizontal(lipgloss.Top,
		leftPanel,
		rightPanelWithBorder, // Use the content wrapped in the border style
	)

	// --- Status Bar ---
	statusBar := m.renderStatusBar() // Already sized correctly in renderStatusBar

	// --- Final Assembly ---
	// Combine main layout and status bar vertically
	finalView := lipgloss.JoinVertical(lipgloss.Left,
		mainLayout,
		statusBar,
	)

	// Return the final view string
	return finalView
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

		// Calculate status bar height (usually 1 line)
		m.statusBarHeight = 1 // Or calculate based on renderStatusBar content?

		// Calculate available height for main panels
		// Subtract status bar height and maybe 1 line for top padding/border
		m.mainPanelHeight = m.height - m.statusBarHeight - 1
		if m.mainPanelHeight < 1 { // Ensure minimum height
			m.mainPanelHeight = 1
		}

		// Calculate panel widths (approx 1/3 left, 2/3 right)
		m.leftPanelWidth = m.width / 3
		// Subtract 1 for potential border/divider between panels
		m.rightPanelWidth = m.width - m.leftPanelWidth - 1
		// Ensure minimum widths
		if m.rightPanelWidth < 10 {
			m.rightPanelWidth = 10
		}
		if m.leftPanelWidth < 10 { // Adjust left if right hit minimum
			m.leftPanelWidth = m.width - m.rightPanelWidth - 1
			if m.leftPanelWidth < 10 {
				m.leftPanelWidth = 10
			}
		}

		// Update help width
		m.help.Width = m.width

		// Update list dimensions (uses right panel size)
		m.list.SetSize(m.rightPanelWidth, m.mainPanelHeight)

		// Update viewport dimensions (uses right panel size, for now)
		m.viewport.Width = m.rightPanelWidth
		m.viewport.Height = m.mainPanelHeight

		// Update FileTree dimensions (if applicable)
		// m.fileTree.SetSize(m.rightPanelWidth, m.mainPanelHeight) // Uncomment if tree has SetSize

		// If the viewport needs different dimensions based on context, adjust here

	case statusMessage:
		m.statusMsg = string(msg)
		return m, nil

	case filesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error loading directory: %v", msg.err)
			return m, nil
		}

		m.statusMsg = fmt.Sprintf("Loaded %s", filepath.Base(m.currentDir))

		// Update viewport content if showing preview
		m.UpdateViewportContent()
		return m, nil // Return nil cmd after successful load and internal update

	case analysisCompleteMsg:
		m.loading = false // Should already be false, but just in case
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error analyzing directory: %v", msg.err)
			// Clear analysis view?
			return m, nil
		}
		// TODO: Process msg.results []*types.FileInfo - update list items, etc.
		m.statusMsg = fmt.Sprintf("Analysis complete for %s", m.currentDir)
		// Update viewport content if showing preview and analysis is relevant
		m.UpdateViewportContent()
		cmds = append(cmds, m.SetStatus(m.statusMsg)) // Set status with potential timer
		return m, tea.Batch(cmds...)

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
		m.visualMode = false // Deactivate visual selection mode
		m.visualStart = -1
		m.SetMode(types.Normal) // Ensure main mode is Normal
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
		m.textInput.SetValue("") // Fix: Use SetValue instead of assignment
		m.mode = types.Normal       // Exit command mode on error
		// Potentially add a timer to clear the error message
		// Return a command to set the status message which includes a clear timer
		return m, nil

	case tea.KeyMsg:
		// Handle key presses based on the current mode and key bindings

		// Always check for quit first, regardless of mode.
		if key.Matches(msg, m.keys.Quit) { // Correctly access Quit field
			return m, tea.Quit
		}

		// Handle global keys (like help) before mode-specific keys.
		if key.Matches(msg, m.keys.Help) { // Correct field name is Help
			m.help.ShowAll = !m.help.ShowAll
			m.showFullHelp = m.help.ShowAll
			return m, nil // Consumed the key, no further updates needed for this msg
		}

		// Mode-specific key handling
		switch m.mode {
		case types.Normal, types.Visual: // Handle keys for Normal and Visual modes
			if key.Matches(msg, m.keys.EnterCmdMode) {
				m.mode = types.Command
				m.textInput.Reset()
				m.statusMsg = m.textInput.Prompt // Set status to prompt immediately
				m.textInput.Focus()
				cmd := m.textInput.Focus()
				return m, cmd
			}

			// Handle list navigation (pass msg to list)
			if key.Matches(msg, m.keys.Up) || key.Matches(msg, m.keys.Down) {
				m.list, cmd = m.list.Update(msg)
				cmds = append(cmds, cmd)
				// Update viewport if list selection changes
				cmds = append(cmds, m.updateViewportContentCmd())
				return m, tea.Batch(cmds...)
			}

			// TODO: Handle other Normal/Visual mode keys (selection, view toggle, etc.)

		case types.Command: // Handle keys for Command mode
			if key.Matches(msg, m.keys.ExecuteCmd) {
				// Execute the command
				cmdStr := m.textInput.Value()
				m.textInput.Reset() // Clear input after execution
				m.textInput.Blur()
				m.mode = types.Normal // Return to normal mode
				m.statusMsg = ""      // Clear command status

				switch cmdStr {
				case ":q", ":quit":
					return m, tea.Quit
				case ":help":
					m.help.ShowAll = !m.help.ShowAll
					m.showFullHelp = m.help.ShowAll
					return m, nil // Command handled
				default:
					// Handle unknown command or pass to future command handler
					m.statusMsg = fmt.Sprintf("Unknown command: %s", cmdStr)
					// Potentially return a command to clear the status after a delay
					return m, nil
				}
			} else if key.Matches(msg, m.keys.ExitCmdMode) { // Fix typo: ExitCmd -> ExitCmdMode
				// Exit command mode without executing
				m.textInput.Reset()
				m.textInput.Blur()
				m.mode = types.Normal
				m.statusMsg = ""
				return m, nil // Consume the key
			}

			// Pass keys to the text input model
			m.textInput, cmd = m.textInput.Update(msg)
			cmds = append(cmds, cmd)
			// Update status message to reflect command input
			m.statusMsg = m.textInput.View()
			return m, tea.Batch(cmds...)
		}

		// If no key was matched and handled above, break out of the KeyMsg case
		// This allows the default block to potentially handle other msg types if needed,
		// but KeyMsgs should ideally be fully handled within this case.

	// Default Case: Bubble Tea internal messages or unknown messages (No changes needed here)
	default:
		// This section should generally handle only non-KeyMsg types now.
		// However, let's keep component updates here for other msg types (like WindowSizeMsg)
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		m.list, cmd = m.list.Update(msg) // List might still need other msg types
		cmds = append(cmds, cmd)
		m.help, cmd = m.help.Update(msg) // Help processes KeyMsgs too
		cmds = append(cmds, cmd)
		// Sync help state regardless of message type, as component might change state internally
		m.showFullHelp = m.help.ShowAll
	}

	// After handling the message, update the viewport content if necessary
	// cmds = append(cmds, m.updateViewportContentCmd()) // Moved viewport update after list navigation

	return m, tea.Batch(cmds...)
}

// Helper function to set status message and return command
func (m *Model) setStatus(msg string) tea.Cmd {
	m.statusMsg = msg
	// Optional: Add a delay here if you want the message to disappear
	// time.Sleep(2 * time.Second)
	// return clearStatusMessage{} // Define a new message type to clear status
	// For now, just return the message itself for immediate display
	return func() tea.Msg {
		return statusMessage(msg)
	}
}

// ScanDirectory performs the directory scanning and updates the model's list
func (m *Model) ScanDirectory() error {
	m.SetLoading(true)
	// m.analysisEngine.SetConfig(m.config) // Config should be set on init
	results, err := m.analysisEngine.ScanDirectory(m.currentDir)
	if err != nil {
		m.handleError(fmt.Errorf("ScanDirectory failed: %w", err)) // Log error
		m.SetLoading(false)                                        // Reset loading state on error
		return err                                                 // Return the error
	} else {
		// Convert []*types.FileInfo to []list.Item
		items := make([]list.Item, len(results))
		for i, fileInfo := range results {
			// Ensure fileInfo is not nil before dereferencing
			if fileInfo != nil {
				// Construct FileEntry from FileInfo
				fileStat, err := os.Stat(fileInfo.Path)
				isDir := false
				if err == nil {
					isDir = fileStat.IsDir()
				} else {
					log.Warnf("Could not stat file %s to determine IsDir: %v", fileInfo.Path, err)
				}
				entry := types.FileEntry{
					Name:        filepath.Base(fileInfo.Path),
					Path:        fileInfo.Path,
					ContentType: fileInfo.ContentType,
					Size:        fileInfo.Size,
					Tags:        fileInfo.Tags,
					IsDir:       isDir,
				}
				items[i] = Item{entry: entry} // Use the constructed entry
			}
			// Consider logging or handling nil fileInfo case
		}

		// Restore selection state before updating the list
		for i := range items {
			item := items[i].(Item) // Assume Item type, should be safe here
			if _, ok := m.selectedFiles[item.entry.Path]; ok {
				// Need to update the item in the slice. Since Item is a struct,
				// we modify a copy, so we need to reassign it.
				updatedItem := item
				updatedItem.selected = true
				items[i] = updatedItem
			}
		}

		m.list.SetItems(items)                                                        // Update the list component
		m.statusMsg = fmt.Sprintf("Scanned %d items in %s", len(items), m.currentDir) // Update status
		m.SetLoading(false)
		return nil // Return nil on success
	}
}

// Helper to create a tea.Cmd for scanning the directory
func (m *Model) loadDirectory(dir string) tea.Cmd {
	m.loading = true
	m.statusMsg = fmt.Sprintf("Loading %s...", dir)
	return func() tea.Msg {
		err := m.ScanDirectory() // ScanDirectory now updates the list internally
		if err != nil {
			return filesLoadedMsg{err: fmt.Errorf("failed to scan %s: %w", dir, err)}
		}
		return filesLoadedMsg{err: nil} // Signal completion (list is already updated)
	}
}

// ModelReader interface implementation

// Files returns the current files in the directory
func (m *Model) Files() []*types.FileInfo {
	items := m.list.Items()
	files := make([]*types.FileInfo, len(items))
	for i, item := range items {
		if listItem, ok := item.(Item); ok { // Type assert first
			// Construct FileInfo from FileEntry for the return type
			fileInfo := types.FileInfo{
				Path:        listItem.entry.Path,
				ContentType: listItem.entry.ContentType,
				Size:        listItem.entry.Size,
				Tags:        listItem.entry.Tags,
			}
			files[i] = &fileInfo // Assign the address of the constructed FileInfo
		} else {
			// Handle case where item is not of type Item (should ideally not happen)
			// Log an error or skip? For now, set to nil or handle appropriately.
			log.Warnf("Item at index %d is not of expected type Item", i)
			files[i] = nil // Or some default/error indicator
		}
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

// CurrentFile returns the name of the file at the current cursor position
func (m *Model) CurrentFile() string {
	currentItem := m.list.SelectedItem()
	if currentItem == nil {
		return ""
	}
	if listItem, ok := currentItem.(Item); ok {
		return filepath.Base(listItem.entry.Path) // Extract filename from path
	}
	return "" // Should ideally not happen if items are always Item
}

// GetFocusedFileName returns the name of the file at the current cursor position
func (m *Model) GetFocusedFileName() string {
	if len(m.list.Items()) == 0 || m.list.Index() < 0 || m.list.Index() >= len(m.list.Items()) {
		return ""
	}
	// Fix: Get item first, then call Name() to help with pointer receiver
	item, ok := m.list.Items()[m.list.Index()].(Item)
	if !ok {
		log.Warnf("Failed to assert list item to Item type")
		return "<error>"
	}
	return item.entry.Name
}

// GetCursorIndex returns the current cursor position
func (m *Model) GetCursorIndex() int {
	return m.list.Index()
}

// GetSelectedFiles returns the current selected files
func (m *Model) GetSelectedFiles() map[string]bool {
	selectedCopy := make(map[string]bool)
	for path, isSelected := range m.selectedFiles {
		if isSelected { // Although GetSelectedFiles currently returns only true values
			selectedCopy[path] = true
		}
	}
	return selectedCopy
}

// GetOrganizePaths returns the paths selected for organization.
func (m *Model) GetOrganizePaths() []string {
	selected := m.GetSelectedFiles() // Get the map[string]bool
	paths := make([]string, 0, len(selected))
	for path, isSelected := range selected {
		if isSelected { // Although GetSelectedFiles currently returns only true values
			paths = append(paths, path)
		}
	}
	sort.Strings(paths) // Ensure consistent order
	return paths
}

// SetCursor sets the list cursor index with bounds checking.
func (m *Model) SetCursor(index int) {
	itemsLen := len(m.list.Items())
	if itemsLen == 0 {
		return // No items to select
	}
	// Clamp index to valid bounds
	if index < 0 {
		index = 0
	} else if index >= itemsLen {
		index = itemsLen - 1
	}
	m.list.Select(index)
}

// SelectFile finds an item by its full path and sets the cursor to it.
func (m *Model) SelectFile(path string) {
	items := m.list.Items()
	for i, item := range items {
		if listItem, ok := item.(Item); ok {
			if listItem.entry.Path == path {
				m.SetCursor(i)
				return
			}
		}
	}
	// Optional: Log if file not found?
}

// GetListItems returns the raw items from the list component.
func (m *Model) GetListItems() []list.Item {
	return m.list.Items()
}

// SetShowHelp sets whether the full help menu is displayed.
func (m *Model) SetShowHelp(show bool) {
	m.showFullHelp = show
}

// VisualMode returns the current visual mode state.
func (m *Model) VisualMode() bool {
	return m.visualMode
}

func (m *Model) handleError(err error) (tea.Model, tea.Cmd) {
	m.statusMsg = fmt.Sprintf("Error: %v", err) // Log the actual error
	m.textInput.SetValue("")                 // Clear potentially problematic command
	m.mode = types.Normal                       // Revert to Normal mode on error
	// Potentially add a timer to clear the error message
	// Return a command to set the status message which includes a clear timer
	return m, m.SetStatus(m.statusMsg) // Use SetStatus which might add a timer
}

// renderStatusBar renders the status bar at the bottom of the screen
func (m *Model) renderStatusBar() string {
	// Base styles
	statusBarStyle := lipgloss.NewStyle().
		// Ensure the bar itself takes the full width
		Width(m.width).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333"))
	statusTextSyle := statusBarStyle.Copy().Padding(0, 1)
	locationStyle := statusBarStyle.Copy().Padding(0, 1)

	// Status message (left side)
	statusText := m.statusMsg
	if statusText == "" {
		statusText = "Press ? for help"
	}
	renderedStatus := statusTextSyle.Render(statusText) // Render first

	// Location info (right side)
	// Use AbbreviatePath if you want to shorten it, otherwise use m.currentDir directly
	locationText := fmt.Sprintf("Location: %s", filepath.Base(m.currentDir))
	renderedLocation := locationStyle.Render(locationText) // Render first

	// Calculate remaining space for the flexible spacer
	spaceWidth := m.width - lipgloss.Width(renderedStatus) - lipgloss.Width(renderedLocation)
	if spaceWidth < 0 {
		spaceWidth = 0 // Prevent negative width
	}
	// Use a style for the spacer to ensure it takes up the calculated width
	flexSpace := lipgloss.NewStyle().Width(spaceWidth).Render("")

	// Combine parts horizontally within the full-width status bar style
	// The outer style ensures background color spans the whole width.
	statusBarContent := lipgloss.JoinHorizontal(lipgloss.Top,
		renderedStatus,
		flexSpace,
		renderedLocation,
	)

	// Apply the main status bar style to the joined content
	// This ensures the background covers the entire line.
	return statusBarStyle.Render(statusBarContent)
}

func (m *Model) getCurrentPreviewContent() string {
	// This is a simplified version of what updateViewportContent does
	// It should return the string content directly
	selectedItem := m.list.SelectedItem()
	if selectedItem == nil {
		return "No file selected."
	}
	// Fix: Get item first, then call Name() to help with pointer receiver
	item, ok := selectedItem.(Item)
	if !ok {
		log.Warnf("Failed to assert list item to Item type")
		return "<error>"
	}
	return fmt.Sprintf("Preview for: %s\nSize: %s", item.entry.Name, formatSize(item.entry.Size))
}

func (m *Model) updateViewportContent() {
	selectedItem := m.list.SelectedItem()
	if selectedItem == nil {
		m.viewport.SetContent("No item selected.")
		return
	}

	item, ok := selectedItem.(Item)
	if !ok {
		m.viewport.SetContent("Error: Invalid item type.")
		return
	}

	if strings.Contains(strings.ToLower(item.entry.ContentType), "directory") {
		// Maybe show directory stats or analysis results?
		m.viewport.SetContent(fmt.Sprintf("Directory: %s", item.entry.Name))
	} else {
		// If file, show size and potentially type/preview
		m.viewport.SetContent(fmt.Sprintf("File: %s\nSize: %s\nType: %s",
			item.entry.Name,
			formatSize(item.entry.Size),
			item.entry.ContentType))
		// TODO: Add more sophisticated preview (e.g., head of text file, image dimensions)
	}
	// Ensure viewport scrolls to top after content change
	m.viewport.GotoTop()
}

// updateStatus is a placeholder for updating the status bar display.
func (m *Model) updateStatus() {
	// TODO: Implement logic to update status based on current state
	// e.g., m.statusMsg = fmt.Sprintf(...)
	// Maybe update list/viewport titles here too?
}

// updateHelp is a placeholder for updating the help view display.
func (m *Model) updateHelp() {
	// TODO: Implement logic to update help keys based on current mode/state
	// m.help.SetKeyMap(m.mode, m.keys)
}

func (m *Model) analyzeItem(item Item) tea.Cmd {
	// Extract necessary info (assuming item.entry implements necessary methods)
	entryInfo := item.entry
	fullPath := filepath.Join(m.currentDir, entryInfo.Name)

	// Call the analysis engine with the path
	return func() tea.Msg {
		_, err := m.analysisEngine.Analyze(fullPath) // Call correct method with path
		if err != nil {
			// Send an error message back to the main update loop
			return error(err)
		}
		// Analysis succeeded (or no error returned), potentially send a success/update msg
		// For now, just return nil if no error
		return nil
	}
}

func (m *Model) HandleFileUpdate(fileInfo *types.FileInfo) tea.Cmd {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	log.Debugf("HandleFileUpdate received for: %s", fileInfo.Path)

	// Find the item in the list and update its underlying FileEntry
	items := m.list.Items()
	for i, item := range items {
		listItem, ok := item.(Item)
		if !ok {
			log.Warnf("List item is not of type Item: %T", item)
			continue
		}

		if listItem.entry.Path == fileInfo.Path {
			log.Debugf("Updating item in list: %s", fileInfo.Path)

			// Need to get IsDir status, might involve a stat call if not readily available
			// For simplicity, let's try to fetch it. Consider optimizing later.
			fileStat, err := os.Stat(fileInfo.Path)
			isDir := false
			if err == nil {
				isDir = fileStat.IsDir()
			} else {
				log.Warnf("Could not stat file %s to determine IsDir: %v", fileInfo.Path, err)
			}

			// Create the updated FileEntry
			updatedEntry := types.FileEntry{
				Name:        filepath.Base(fileInfo.Path),
				Path:        fileInfo.Path,
				ContentType: fileInfo.ContentType,
				Size:        fileInfo.Size,
				Tags:        fileInfo.Tags,
				IsDir:       isDir,
			}

			// Update the item in the list
			updatedListItem := Item{entry: updatedEntry, selected: listItem.selected}
			cmd := m.list.SetItem(i, updatedListItem)
			log.Debugf("Item updated: %+v", updatedListItem)
			return cmd // Return command from SetItem
		}
	}

	log.Debugf("File not found in current list view, ignoring update: %s", fileInfo.Path)
	return nil // No update command needed if file not found
}

func (m *Model) HandleFileDeletion(fileInfo *types.FileInfo) tea.Cmd {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	log.Debugf("HandleFileDeletion received for: %s", fileInfo.Path)

	// Find the index of the item to delete
	deleteIndex := -1
	items := m.list.Items()
	for i, item := range items {
		listItem, ok := item.(Item)
		if !ok {
			log.Warnf("List item is not of type Item: %T", item)
			continue
		}
		if listItem.entry.Path == fileInfo.Path {
			deleteIndex = i
			break
		}
	}

	// If found, remove the item from the list
	if deleteIndex != -1 {
		log.Debugf("Removing item from list at index %d: %s", deleteIndex, fileInfo.Path)
		m.list.RemoveItem(deleteIndex)
		// If the list becomes empty or the cursor was on the deleted item,
		// reset cursor appropriately. list.RemoveItem might handle some cases.
		if m.list.Index() >= len(m.list.Items()) {
			m.list.Select(max(0, len(m.list.Items())-1))
		}
		return m.updateStatusCmd("File removed: " + filepath.Base(fileInfo.Path))
	}

	log.Debugf("File not found in current list view, ignoring deletion: %s", fileInfo.Path)
	return nil // No command needed if file not found
}

func (m *Model) SetCurrentDir(dir string) {
	m.currentDir = dir
	// Potentially trigger a directory scan or update the file tree
	// m.ScanDirectory() // Example: Rescan after changing directory
}

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

	// Only render logo if there's enough space
	const minWidthForLogo = 50 // Adjust as needed
	if m.leftPanelWidth >= minWidthForLogo {
		// Get the styled ASCII logo
		headerContent.WriteString(m.renderASCIILogo())
		headerContent.WriteString("\n")
	}

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

	// Return the raw content string, the View method will handle paneling
	return headerContent.String()
	// Create a nice panel for the header content
	// return styles.DefaultTheme.Panel.Render(headerContent.String())
}

func (m *Model) updateStatusCmd(msg string) tea.Cmd {
	return func() tea.Msg {
		m.updateStatus()
		return nil
	}
}

// updateViewportContentCmd returns a command that updates the viewport content
// based on the currently selected item in the list.
func (m *Model) updateViewportContentCmd() tea.Cmd {
	return func() tea.Msg {
		i := m.list.SelectedItem()
		if i == nil {
			m.viewport.SetContent("No item selected.")
			return nil // Or a specific message if needed
		}

		listItem, ok := i.(Item) // Use the Item struct defined in this package
		if !ok {
			m.viewport.SetContent(fmt.Sprintf("Error: Unexpected item type %T", i))
			return nil
		}

		if listItem.entry.IsDir {
			m.viewport.SetContent(fmt.Sprintf("Directory: %s", listItem.entry.Name))
			return nil
		}

		// Read file content
		filePath := filepath.Join(m.currentDir, listItem.entry.Name)
		content, err := os.ReadFile(filePath)
		if err != nil {
			m.viewport.SetContent(fmt.Sprintf("Error reading file '%s': %v", listItem.entry.Name, err))
			return nil
		}

		m.viewport.SetContent(string(content))
		m.viewport.GotoTop() // Reset viewport scroll on new content
		return nil // Indicate completion, maybe a specific msg if redraw is complex
	}
}

// initializeTextInput creates and configures the text input component.
func initializeTextInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = ": "
	ti.Placeholder = "Enter command (e.g., :q to quit)"
	ti.CharLimit = 256
	ti.Width = 80 // Adjust width as needed, maybe based on terminal size
	ti.Focus()    // Start focused for immediate command entry if needed initially (or blur)
	ti.Blur()     // But typically start blurred until ':' is pressed
	return ti
}
