package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sortd/internal/analysis"
	"sortd/internal/organize"
	"sortd/internal/tui/styles"
	"sortd/pkg/types"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the TUI state
type Model struct {
	// Core state
	mode          types.Mode
	selectedFiles map[string]bool
	files         []types.FileEntry
	cursor        int
	currentDir    string
	currentFile   string
	showHelp      bool

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
}

// Ensure Model implements types.ModelReader
var _ types.ModelReader = (*Model)(nil)

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	return nil
}

func New() *Model {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	m := &Model{
		selectedFiles:  make(map[string]bool),
		files:          make([]types.FileEntry, 0),
		mode:           types.Normal,
		cursor:         0,
		currentDir:     wd,
		currentFile:    "",
		showHelp:       true,
		analysisEngine: analysis.New(),
		organizeEngine: organize.New(),
		commandBuffer:  "",
		statusMsg:      "",
		lastKey:        "",
		visualStart:    0,
		visualEnd:      0,
		visualMode:     false,
	}

	// Initial directory scan
	if err := m.ScanDirectory(); err != nil {
		// Handle error gracefully, but don't fail initialization
		m.statusMsg = fmt.Sprintf("Error scanning directory: %v", err)
	}

	return m
}

// View implements tea.Model
func (m *Model) View() string {
	var s strings.Builder
	s.WriteString(styles.Title.Render("Sortd File Organizer"))
	s.WriteString("\n\n")

	if m.mode == types.Setup {
		s.WriteString("Welcome to Sortd\n\n")
		s.WriteString("Choose an option (1-4)\n\n")
		s.WriteString("1. Quick Start - Organize Files\n")
		s.WriteString("2. Setup Configuration\n")
		s.WriteString("3. Watch Mode (Coming Soon)\n")
		s.WriteString("4. Show Help\n\n")
		s.WriteString("Quick Start Guide\n")
		return styles.App.Render(s.String())
	}

	if m.showHelp {
		s.WriteString(m.getHelp())
		s.WriteString("\n\n")
	}

	// Handle empty state first
	if len(m.files) == 0 {
		s.WriteString("No files to display yet.\n")
		return styles.App.Render(s.String())
	}

	// Only show files and cursor if we have files
	for i, file := range m.files {
		style := styles.Unselected
		if m.selectedFiles[file.Path] {
			style = styles.Selected
		}

		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}

		s.WriteString(prefix + style.Render(file.Name) + "\n")
	}

	return styles.App.Render(s.String())
}

func (m *Model) getHelp() string {
	return styles.Help.Render(`Navigation:
	j/↓: Move down
	k/↑: Move up
	h/←, l/→: Change directory
	enter: Open directory
	gg: Go to top
	G: Go to bottom

Selection:
	space: Select file
	v: Visual mode
	V: Visual line mode

Commands:
	q: Quit
	:: Command mode
	/: Search
	?: Toggle help

Organization:
	o: Organize selected files
	r: Refresh view`)
}

// Update implements tea.Model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	newModel := m.copy()

	switch newModel.mode {
	case types.Command:
		return newModel.handleCommandMode(msg)
	default:
		return newModel.handleNormalKeys(msg)
	}
}

func (m *Model) copy() *Model {
	newModel := &Model{
		selectedFiles:  make(map[string]bool),
		files:          make([]types.FileEntry, len(m.files)),
		mode:           m.mode,
		cursor:         m.cursor,
		currentDir:     m.currentDir,
		currentFile:    m.currentFile,
		showHelp:       m.showHelp,
		analysisEngine: m.analysisEngine,
		organizeEngine: m.organizeEngine,
		commandBuffer:  m.commandBuffer,
		statusMsg:      m.statusMsg,
		lastKey:        m.lastKey,
		visualStart:    m.visualStart,
		visualEnd:      m.visualEnd,
		visualMode:     m.visualMode,
	}

	copy(newModel.files, m.files)
	for k, v := range m.selectedFiles {
		newModel.selectedFiles[k] = v
	}

	return newModel
}

func (m *Model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	newModel := m.copy()

	switch msg.String() {
	case "q", "ctrl+c":
		return newModel, tea.Quit
	case "?":
		newModel.showHelp = !newModel.showHelp
		return newModel, nil
	case ":":
		newModel.mode = types.Command
		newModel.commandBuffer = ":"
		return newModel, nil
	case "esc":
		if newModel.visualMode {
			newModel.visualMode = false
			newModel.selectedFiles = make(map[string]bool)
			return newModel, nil
		}
		return newModel, nil
	case "j", "down":
		if newModel.cursor < len(newModel.files)-1 {
			newModel.cursor++
			if len(newModel.files) > 0 {
				newModel.currentFile = newModel.files[newModel.cursor].Name
			}
			if newModel.visualMode {
				newModel.visualEnd = newModel.cursor
				newModel.updateVisualSelection()
			}
		}
	case "k", "up":
		if newModel.cursor > 0 {
			newModel.cursor--
			if len(newModel.files) > 0 {
				newModel.currentFile = newModel.files[newModel.cursor].Name
			}
			if newModel.visualMode {
				newModel.visualEnd = newModel.cursor
				newModel.updateVisualSelection()
			}
		}
	case "v":
		if !newModel.visualMode {
			newModel.visualMode = true
			newModel.visualStart = newModel.cursor
			newModel.visualEnd = newModel.cursor
			newModel.updateVisualSelection()
		} else {
			newModel.visualMode = false
			// Keep selections
		}
	case " ": // Space key for selection
		if len(newModel.files) > 0 {
			file := newModel.files[newModel.cursor]
			if newModel.selectedFiles[file.Path] {
				delete(newModel.selectedFiles, file.Path)
			} else {
				newModel.selectedFiles[file.Path] = true
			}
		}
	case "enter":
		if len(newModel.files) > 0 {
			file := newModel.files[newModel.cursor]
			info, err := os.Stat(file.Path)
			if err == nil && info.IsDir() {
				newModel.currentDir = file.Path
				if err := newModel.ScanDirectory(); err != nil {
					newModel.statusMsg = fmt.Sprintf("Error: %v", err)
				}
			}
		}
	}
	return newModel, nil
}

func (m *Model) updateVisualSelection() {
	start := min(m.visualStart, m.visualEnd)
	end := max(m.visualStart, m.visualEnd)

	// Clear previous selections first
	m.selectedFiles = make(map[string]bool)

	// Select all files in range
	for i := start; i <= end && i < len(m.files); i++ {
		m.selectedFiles[m.files[i].Path] = true
	}
}

func (m *Model) handleCommandMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	newModel := m.copy()

	switch msg.Type {
	case tea.KeyEsc:
		newModel.mode = types.Normal
		newModel.commandBuffer = ""
		return newModel, nil
	case tea.KeyEnter:
		cmd := newModel.commandBuffer[1:] // Remove the leading ':'
		newModel.mode = types.Normal
		newModel.commandBuffer = ""
		return newModel.executeCommand(cmd)
	case tea.KeyBackspace:
		if len(newModel.commandBuffer) > 1 { // Keep the ':'
			newModel.commandBuffer = newModel.commandBuffer[:len(newModel.commandBuffer)-1]
		}
		return newModel, nil
	default:
		if msg.Type == tea.KeyRunes {
			if newModel.commandBuffer == "" {
				newModel.commandBuffer = ":"
			}
			newModel.commandBuffer += string(msg.Runes)
		}
		return newModel, nil
	}
}

func (m *Model) executeCommand(cmd string) (tea.Model, tea.Cmd) {
	newModel := m.copy()
	// Handle command execution
	switch cmd {
	case "q", "quit":
		return newModel, tea.Quit
	case "help":
		newModel.showHelp = !newModel.showHelp
	}
	return newModel, nil
}

// File operations
func (m *Model) scanDirectory() error {
	results, err := m.analysisEngine.ScanDirectory(m.currentDir)
	if err != nil {
		return err
	}

	m.files = make([]types.FileEntry, 0)
	for _, result := range results {
		m.files = append(m.files, types.FileEntry{
			Name:        filepath.Base(result.Path),
			Path:        result.Path,
			ContentType: result.ContentType,
			Size:        result.Size,
			Tags:        result.Tags,
		})
	}

	// Sort files for consistent ordering
	sort.Slice(m.files, func(i, j int) bool {
		return m.files[i].Name < m.files[j].Name
	})

	return nil
}

// Getters
func (m *Model) Files() []types.FileEntry {
	return m.files
}

func (m *Model) Cursor() int {
	return m.cursor
}

func (m *Model) IsSelected(name string) bool {
	return m.selectedFiles[name]
}

func (m *Model) ShowHelp() bool {
	return m.showHelp
}

func (m *Model) Mode() types.Mode {
	return m.mode
}

// SetCurrentDir sets the current directory
func (m *Model) SetCurrentDir(dir string) {
	m.currentDir = dir
}

// CurrentDir returns the current directory
func (m *Model) CurrentDir() string {
	return m.currentDir
}

// CurrentFile returns the current file
func (m *Model) CurrentFile() string {
	return m.currentFile
}

// SetCursor sets the cursor position
func (m *Model) SetCursor(pos int) {
	if pos >= 0 && pos < len(m.files) {
		m.cursor = pos
		m.currentFile = m.files[pos].Name
	}
}

// ScanDirectory scans the current directory and updates the file list
func (m *Model) ScanDirectory() error {
	// Check if directory exists first
	if _, err := os.Stat(m.currentDir); os.IsNotExist(err) {
		m.files = []types.FileEntry{} // Clear files on nonexistent directory
		return fmt.Errorf("directory does not exist: %s", m.currentDir)
	}

	// Get directory contents
	entries, err := os.ReadDir(m.currentDir)
	if err != nil {
		m.files = []types.FileEntry{} // Clear files on error
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Process entries
	files := make([]types.FileEntry, 0)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't get info for
		}

		fullPath := filepath.Join(m.currentDir, entry.Name())
		contentType := "application/octet-stream"
		if !info.IsDir() {
			if result, err := m.analysisEngine.Scan(fullPath); err == nil {
				contentType = result.ContentType
			}
		}

		files = append(files, types.FileEntry{
			Name:        entry.Name(),
			Path:        fullPath,
			ContentType: contentType,
			Size:        info.Size(),
		})
	}

	m.files = files
	if len(files) > 0 {
		m.currentFile = files[0].Name
	} else {
		m.currentFile = ""
	}
	m.cursor = 0 // Reset cursor when scanning new directory
	return nil
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

// VisualMode returns whether visual mode is active
func (m *Model) VisualMode() bool {
	return m.visualMode
}

// SetShowHelp sets the help visibility state
func (m *Model) SetShowHelp(show bool) {
	m.showHelp = show
}

// AddFile adds a file to the model's file list
func (m *Model) AddFile(file types.FileEntry) {
	m.files = append(m.files, file)
	// Sort files to maintain order
	sort.Slice(m.files, func(i, j int) bool {
		return m.files[i].Name < m.files[j].Name
	})
}

// SelectFile selects a file by name
func (m *Model) SelectFile(name string) error {
	for _, file := range m.files {
		if file.Name == name {
			m.selectedFiles[name] = true
			return nil
		}
	}
	return fmt.Errorf("file not found: %s", name)
}

// MoveCursor moves the cursor by delta positions
func (m *Model) MoveCursor(delta int) {
	newPos := m.cursor + delta
	if newPos >= 0 && newPos < len(m.files) {
		m.cursor = newPos
		if len(m.files) > 0 {
			m.currentFile = m.files[m.cursor].Name
		}
	}
}
