package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"sortd/internal/tui/common"
	"sortd/internal/tui/messages"
	"sortd/internal/tui/views"
	"strings"

	"sortd/internal/analysis"
	"sortd/internal/config"
	"sortd/internal/organize"

	"sortd/internal/tui/components"

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

	analysisEngine *analysis.Engine
	organizeEngine *organize.Engine
	config         *config.Config

	// New fields for the new functionality
	menuChoice   int
	configChoice int
	directorySet bool

	fileBrowser  *components.FileBrowser
	configEditor *components.ConfigEditor
	statusBar    *components.StatusBar
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	// Make sure we're not waiting on anything here
	return tea.Batch(
		m.fileBrowser.Init(),
		m.configEditor.Init(),
	)
}

func New() *Model {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	m := &Model{
		selectedFiles:  make(map[string]bool),
		mode:           common.Normal,
		currentDir:     wd,
		currentFile:    "",
		showHelp:       false,
		analysisEngine: analysis.New(),
		organizeEngine: organize.New(),
		config:         config.New(),
		fileBrowser:    components.NewFileBrowser(),
		configEditor:   components.NewConfigEditor(config.New()),
		statusBar:      components.NewStatusBar(),
	}
	return m
}

// View implements tea.Model
func (m *Model) View() string {
	return views.RenderMainView(m)
}

// Update implements tea.Model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	fmt.Println("Model.Update called with message type:", reflect.TypeOf(msg))
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.fileBrowser.SetSize(msg.Width, msg.Height)
	case messages.ConfigUpdateMsg:
		m.config = msg.Config
		m.statusBar.SetText("Configuration saved")
	case messages.ErrorMsg:
		m.statusBar.SetText("Error: " + msg.Err.Error())
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	// Update components
	if cmd := m.fileBrowser.Update(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if cmd := m.configEditor.Update(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if cmd := m.statusBar.Update(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
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
	fmt.Println("Starting Model.copy()")
	newModel := &Model{
		selectedFiles:  make(map[string]bool),
		files:          make([]common.FileEntry, len(m.files)),
		mode:           m.mode,
		cursor:         m.cursor,
		showHelp:       m.showHelp,
		currentDir:     m.currentDir,
		currentFile:    m.currentFile,
		commandBuffer:  m.commandBuffer,
		statusMsg:      m.statusMsg,
		lastKey:        m.lastKey,
		visualMode:     m.visualMode,
		visualStart:    m.visualStart,
		visualEnd:      m.visualEnd,
		analysisEngine: m.analysisEngine,
		organizeEngine: m.organizeEngine,
		config:         m.config,
		menuChoice:     m.menuChoice,
		configChoice:   m.configChoice,
		directorySet:   m.directorySet,
		fileBrowser:    m.fileBrowser,
		configEditor:   m.configEditor,
		statusBar:      m.statusBar,
	}

	copy(newModel.files, m.files)
	for k, v := range m.selectedFiles {
		newModel.selectedFiles[k] = v
	}

	fmt.Println("Finished Model.copy()")
	return newModel
}

func (m *Model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	newModel := m.copy()

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
	case "1", "2", "3", "4", "5", "6":
		if m.currentDir == "" {
			choice := int(msg.String()[0] - '0')
			switch choice {
			case 1: // Browse & Organize
				return m.handleBrowseMode()
			case 2: // Configure Rules
				return m.handleConfigRules()
			case 3: // Configure Patterns
				return m.handleConfigPatterns()
			case 4: // Analysis Report
				return m.handleAnalysis()
			case 5: // Watch Directory
				return m.handleWatchMode()
			}
		}
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

// OrganizeSelected organizes the selected files
func (m *Model) OrganizeSelected() tea.Cmd {
	return func() tea.Msg {
		var filesToOrganize []string
		for path := range m.selectedFiles {
			filesToOrganize = append(filesToOrganize, path)
		}

		if err := m.organizeEngine.OrganizeFiles(filesToOrganize, m.currentDir); err != nil {
			return messages.ErrorMsg{Err: err}
		}

		m.selectedFiles = make(map[string]bool)
		m.scanDirectory()
		return nil
	}
}

func (m *Model) handleBrowseMode() (tea.Model, tea.Cmd) {
	newModel := m.copy()
	// Start in home directory if none selected
	if newModel.currentDir == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			newModel.currentDir = home
		} else {
			newModel.currentDir = "."
		}
	}
	newModel.scanDirectory()
	return newModel, nil
}

func (m *Model) handleConfigRules() (tea.Model, tea.Cmd) {
	newModel := m.copy()
	// Comment out or remove the unused variable
	// Rules := newModel.config.Organize.Patterns
	// ... implement rules editor view
	return newModel, nil
}

func (m *Model) handleConfigPatterns() (tea.Model, tea.Cmd) {
	newModel := m.copy()
	// Comment out or remove the unused variable
	// Patterns := newModel.config.Organize.Patterns
	// ... implement pattern editor view
	return newModel, nil
}

func (m *Model) handleAnalysis() (tea.Model, tea.Cmd) {
	newModel := m.copy()
	return newModel, func() tea.Msg {
		if m.currentDir == "" {
			return messages.ErrorMsg{Err: fmt.Errorf("no directory selected")}
		}
		results, err := m.analysisEngine.ScanDirectory(m.currentDir)
		if err != nil {
			return messages.ErrorMsg{Err: err}
		}
		return messages.AnalysisCompleteMsg{Results: results}
	}
}

func (m *Model) handleWatchMode() (tea.Model, tea.Cmd) {
	newModel := m.copy()
	if newModel.currentDir == "" {
		return newModel, func() tea.Msg {
			return messages.ErrorMsg{Err: fmt.Errorf("please select a directory first")}
		}
	}
	// Enable watch mode on current directory
	return newModel, nil
}
