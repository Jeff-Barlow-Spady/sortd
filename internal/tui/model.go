package tui

import (
	"os"
	"path/filepath"
	"sort"
	"sortd/internal/tui/common"
	"sortd/internal/tui/views"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	// Core state
	mode          common.Mode
	selectedFiles map[string]bool
	files         []common.FileEntry
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
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	return nil
}

func New() *Model {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	return &Model{
		selectedFiles: make(map[string]bool),
		mode:          common.Normal,
		currentDir:    wd,
		currentFile:   "",
		showHelp:      false,
	}
}

// View implements tea.Model
func (m *Model) View() string {
	return views.RenderMainView(m)
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
	case common.Command:
		return newModel.handleCommandMode(msg)
	default:
		return newModel.handleNormalKeys(msg)
	}
}

func (m *Model) copy() *Model {
	newModel := &Model{
		selectedFiles: make(map[string]bool),
		files:         make([]common.FileEntry, len(m.files)),
		mode:          m.mode,
		cursor:        m.cursor,
		showHelp:      m.showHelp,
		currentDir:    m.currentDir,
		currentFile:   m.currentFile,
		commandBuffer: m.commandBuffer,
		statusMsg:     m.statusMsg,
		lastKey:       m.lastKey,
		visualMode:    m.visualMode,
		visualStart:   m.visualStart,
		visualEnd:     m.visualEnd,
	}

	copy(newModel.files, m.files)
	for k, v := range m.selectedFiles {
		newModel.selectedFiles[k] = v
	}

	return newModel
}

func (m *Model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	newModel := New()

	switch msg.String() {
	case "esc":
		// Clear selections
		newModel.selectedFiles = make(map[string]bool)
		newModel.visualMode = false
		return newModel, tea.Quit
	case "j", "down", "↓":
		if newModel.cursor < len(newModel.files)-1 {
			newModel.cursor++
			if len(newModel.files) > 0 {
				newModel.currentFile = newModel.files[newModel.cursor].Name
			}
		}
	case "k", "up", "↑":
		if newModel.cursor > 0 {
			newModel.cursor--
			if len(newModel.files) > 0 {
				newModel.currentFile = newModel.files[newModel.cursor].Name
			}
		}
	case "h", "left":
		if newModel.currentDir != "/" {
			parent := filepath.Dir(newModel.currentDir)
			newModel.currentDir = parent
			newModel.cursor = 0
			newModel.scanDirectory()
		}
	case "l", "right":
		if len(newModel.files) > 0 {
			file := newModel.files[newModel.cursor]
			if info, err := os.Stat(file.Path); err == nil && info.IsDir() {
				newModel.currentDir = file.Path
				newModel.cursor = 0
				newModel.scanDirectory()
			}
		}
	case ":":
		newModel.mode = common.Command
		newModel.commandBuffer = ":"
		return newModel, nil
	case " ":
		if len(newModel.files) > 0 {
			file := newModel.files[newModel.cursor].Name
			if newModel.visualMode {
				// Select range in visual mode
				start := min(newModel.visualStart, newModel.cursor)
				end := max(newModel.visualStart, newModel.cursor)
				for i := start; i <= end; i++ {
					newModel.selectedFiles[newModel.files[i].Name] = true
				}
			} else {
				newModel.selectedFiles[file] = !newModel.selectedFiles[file]
			}
		}
	case "v":
		newModel.visualMode = !newModel.visualMode
		if newModel.visualMode {
			newModel.visualStart = newModel.cursor
		}
	case "G":
		newModel.cursor = len(newModel.files) - 1
		newModel.currentFile = newModel.files[newModel.cursor].Name
	case "g":
		if newModel.lastKey == "g" {
			newModel.cursor = 0
			newModel.currentFile = newModel.files[0].Name
		}
	case "?":
		newModel.showHelp = !newModel.showHelp
	case "q":
		return newModel, tea.Quit
	}

	newModel.lastKey = msg.String()
	return newModel, nil
}

func (m *Model) handleCommandMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	newModel := m.copy()

	switch msg.String() {
	case "esc":
		newModel.mode = common.Normal
		newModel.commandBuffer = ""
		return newModel, nil
	case "enter":
		cmd := strings.TrimPrefix(newModel.commandBuffer, ":")
		newModel.mode = common.Normal
		newModel.commandBuffer = ""
		return newModel, newModel.executeCommand(cmd)
	case "backspace":
		if len(newModel.commandBuffer) > 1 {
			newModel.commandBuffer = newModel.commandBuffer[:len(newModel.commandBuffer)-1]
		}
	default:
		if len(msg.String()) == 1 {
			newModel.commandBuffer += msg.String()
		}
	}
	return newModel, nil
}

func (m *Model) executeCommand(cmd string) tea.Cmd {
	// Implement command execution
	switch cmd {
	case "q", "quit":
		return tea.Quit
		// Add more commands as needed
	}
	return nil
}

// File operations
func (m *Model) scanDirectory() error {
	entries, err := os.ReadDir(m.currentDir)
	if err != nil {
		return err
	}

	m.files = make([]common.FileEntry, 0)
	for _, entry := range entries {
		m.files = append(m.files, common.FileEntry{
			Name: entry.Name(),
			Path: filepath.Join(m.currentDir, entry.Name()),
		})
	}

	// Sort files for consistent ordering
	sort.Slice(m.files, func(i, j int) bool {
		return m.files[i].Name < m.files[j].Name
	})

	return nil
}

// Getters
func (m *Model) Files() []common.FileEntry {
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

func (m *Model) Mode() common.Mode {
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

// ScanDirectory scans the current directory
func (m *Model) ScanDirectory() error {
	return m.scanDirectory()
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
