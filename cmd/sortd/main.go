package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"sortd/internal/analysis"
	"sortd/internal/config"
	"sortd/internal/log"
	"sortd/internal/organize"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
)

// Styles defines the core UI styles
var Styles = struct {
	App        lipgloss.Style
	Title      lipgloss.Style
	Selected   lipgloss.Style
	Unselected lipgloss.Style
	Help       lipgloss.Style
}{
	App: lipgloss.NewStyle().
		Padding(1, 2),
	Title: lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7B61FF")).
		MarginBottom(1),
	Selected: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#73F59F")).
		Bold(true),
	Unselected: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")),
	Help: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#5A9")),
}

type FileEntry struct {
	Name        string
	Path        string
	ContentType string
	Size        int64
	Tags        []string
}

type Mode int

const (
	Setup Mode = iota
	Normal
	Command
	Visual
)

// Add wizard state
type WizardStep int

const (
	WelcomeStep   WizardStep = iota
	ConfigStep    WizardStep = iota
	DirectoryStep WizardStep = iota
	RulesStep
	PatternsStep
	WatchStep
	CompleteStep
)

type model struct {
	files          []FileEntry
	selectedFiles  map[string]bool
	cursor         int
	helpText       string
	showHelp       bool
	currentDir     string
	analysisEngine *analysis.Engine
	organizeEngine *organize.Engine
	mode           Mode
	commandBuffer  string
	statusMsg      string
	wizardStep     WizardStep
	wizardChoices  map[string]bool
	config         *config.Config
}

func initialModel() model {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	m := model{
		currentDir:     wd,
		files:          []FileEntry{},
		selectedFiles:  make(map[string]bool),
		mode:           Setup,
		analysisEngine: analysis.New(),
		organizeEngine: organize.New(),
		helpText: Styles.Help.Render(`
				Navigation:
					j/↓, k/↑: Move cursor
					h/←, l/→: Change directory
					gg: Go to top
					G: Go to bottom

				Selection:
					space: Toggle selection
					v: Visual mode
					V: Visual line mode

				Commands:
					q, quit: Exit
					:: Command mode
					/: Search
					?: Toggle help

				Organization:
					o: Organize selected
					r: Refresh view
	`),
		showHelp:      true,
		wizardStep:    WelcomeStep,
		wizardChoices: make(map[string]bool),
		config:        config.New(),
	}
	m.scanDirectory()
	return m
}

func (m *model) scanDirectory() {
	results, err := m.analysisEngine.ScanDirectory(m.currentDir)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Error scanning directory: %v", err)
		return
	}

	m.files = make([]FileEntry, 0)
	for _, result := range results {
		m.files = append(m.files, FileEntry{
			Name:        filepath.Base(result.Path),
			Path:        result.Path,
			ContentType: result.ContentType,
			Size:        result.Size,
			Tags:        result.Tags,
		})
	}
	sort.Slice(m.files, func(i, j int) bool {
		return m.files[i].Name < m.files[j].Name
	})
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		}

		if m.mode == Normal {
			// Use "tab" or "enter" to trigger directory browsing;
			// otherwise, handle navigation (e.g. "j", "k", etc.)
			switch msg.String() {
			case "tab", "enter":
				return m.handleDirectoryBrowsing(msg)
			default:
				return m.handleNormalMode(msg)
			}
		}

		// Mode-specific handling for other modes.
		var cmd tea.Cmd
		var newModel tea.Model
		switch m.mode {
		case Setup:
			newModel, cmd = m.handleSetupMode(msg)
		case Command:
			newModel, cmd = m.handleCommandMode(msg)
		case Visual:
			newModel, cmd = m.handleVisualMode(msg)
		}
		m = newModel.(model)
		return m, cmd
	}
	return m, nil
}

func (m model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Clear any selections or temporary states
		m.selectedFiles = make(map[string]bool)
		return m, nil
	case "j", "down":
		if m.cursor < len(m.files)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "h", "left":
		if m.currentDir != "/" {
			parent := filepath.Dir(m.currentDir)
			m.currentDir = parent
			m.cursor = 0
			m.scanDirectory()
		}
	case "l", "right":
		if len(m.files) > 0 {
			file := m.files[m.cursor]
			info, err := os.Stat(file.Path)
			if err == nil && info.IsDir() {
				m.currentDir = file.Path
				m.cursor = 0
				m.scanDirectory()
			}
		}
	case ":":
		m.mode = Command
		m.commandBuffer = ":"
		return m, nil
	case "space":
		if len(m.files) > 0 {
			file := m.files[m.cursor]
			m.selectedFiles[file.Name] = !m.selectedFiles[file.Name]
		}
	}
	return m, nil
}

func (m model) handleCommandMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = Normal
		m.commandBuffer = ""
		return m, nil
	case "enter":
		cmd := strings.TrimPrefix(m.commandBuffer, ":")
		m.mode = Normal
		m.commandBuffer = ""
		return m, m.executeCommand(cmd)
	case "backspace":
		if len(m.commandBuffer) > 1 {
			m.commandBuffer = m.commandBuffer[:len(m.commandBuffer)-1]
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.commandBuffer += msg.String()
		}
	}
	return m, nil
}

func (m model) handleVisualMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "V":
		for i := m.cursor; i < len(m.files); i++ {
			m.selectedFiles[m.files[i].Path] = true
		}
		m.mode = Normal
	case "v":
		if len(m.files) > 0 {
			m.selectedFiles[m.files[m.cursor].Path] = true
		}
		m.mode = Normal
	case "esc":
		m.mode = Normal
	}
	return m, nil
}

func (m model) executeCommand(cmd string) tea.Cmd {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "q", "quit":
		return tea.Quit
	case "w", "write":
		if err := config.SaveConfig(m.config); err != nil {
			m.statusMsg = fmt.Sprintf("Error saving config: %v", err)
		}
	case "e":
		if len(parts) > 1 {
			if err := os.Chdir(parts[1]); err != nil {
				m.statusMsg = fmt.Sprintf("Error changing directory: %v", err)
				return nil
			}
			m.currentDir = parts[1]
			m.scanDirectory()
		}
	case "help":
		m.showHelp = true
	}
	return nil
}

func (m model) organizeSelected() tea.Msg {
	var filesToOrganize []string
	for path := range m.selectedFiles {
		filesToOrganize = append(filesToOrganize, path)
	}

	if err := m.organizeEngine.OrganizeFiles(filesToOrganize, m.currentDir); err != nil {
		return errMsg{err}
	}

	m.selectedFiles = make(map[string]bool)
	m.scanDirectory()
	return nil
}

func (m model) handleSetupMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "1" && m.wizardStep == WelcomeStep {
		m.mode = Normal
		m.wizardStep = CompleteStep
		m.scanDirectory()
		m.cursor = 0
		return m, nil
	}
	return m, nil
}

func (m model) initWatchMode() tea.Cmd {
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return errMsg{err}
		}

		go func() {
			for {
				select {
				case event := <-watcher.Events:
					log.Debug("watch event", "event", event)
					if event.Op&fsnotify.Write == fsnotify.Write {
						log.Debug("file modified", "path", event.Name)
					}
				case err := <-watcher.Errors:
					log.Error("watch error", "error", err)
					return
				}
			}
		}()

		err = watcher.Add(m.currentDir)
		if err != nil {
			return errMsg{err}
		}

		return watchStartedMsg{watcher}
	}
}

type watchStartedMsg struct {
	watcher *fsnotify.Watcher
}

type errMsg struct {
	err error
}

func (m model) wizardView() string {
	var s strings.Builder
	s.WriteString(Styles.Title.Render())
	s.WriteString("\n\n")

	switch m.wizardStep {
	case WelcomeStep:
		s.WriteString("Welcome to Sortd!\n\n")
		s.WriteString("1. Quick Start - Organize files now\n")
		s.WriteString("2. Setup Configuration - Customize rules and patterns\n")
		s.WriteString("3. Watch Mode - Monitor directory for changes\n")
		s.WriteString("4. Show Help - Learn about commands\n\n")
		s.WriteString("Choose an option (1-4)")

	case ConfigStep:
		s.WriteString("Configuration Setup\n\n")
		s.WriteString(renderConfigOptions(m))

	case DirectoryStep:
		s.WriteString(fmt.Sprintf("Current directory: %s\n\n", m.currentDir))
		if m.wizardChoices["watch"] {
			s.WriteString("Press ENTER to start watching\n")
		} else {
			s.WriteString("Press ENTER to confirm, TAB to browse")
		}
	}

	s.WriteString("\n\n")
	s.WriteString(Styles.Help.Render("Commands: "))
	s.WriteString(getCommandsForStep(m.wizardStep))

	return Styles.App.Render(s.String())
}

func renderConfigOptions(m model) string {
	var s strings.Builder
	options := []string{
		"Default directories",
		"Organization rules",
		"File patterns",
		"Watch mode settings",
	}

	for i, opt := range options {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		s.WriteString(prefix + opt + "\n")
	}
	return s.String()
}

func getCommandsForStep(step WizardStep) string {
	switch step {
	case WelcomeStep:
		return "1-4: Select option • q: Quit"
	case ConfigStep:
		return "↑/↓: Navigate • enter: Select • esc: Back"
	case DirectoryStep:
		return "enter: Confirm • tab: Browse • esc: Back"
	default:
		return "enter: Continue • esc: Back • q: Quit"
	}
}

type WatchMode struct {
	dir    string
	paused bool
	files  map[string]bool
}

func NewWatchMode(dir string) (*WatchMode, error) {
	if dir == "" {
		return nil, fmt.Errorf("directory not specified")
	}
	return &WatchMode{
		dir:    dir,
		files:  make(map[string]bool),
		paused: false,
	}, nil
}

func (m model) View() string {
	var s strings.Builder

	// Handle setup/wizard mode separately
	if m.mode == Setup {
		return m.wizardView()
	}

	// Always show banner in Normal mode
	s.WriteString(Styles.Title.Render(`
                               $$\  $$\      $$\ 
                               $$ | $  |     $$ |
 $$$$$$$\  $$$$$$\   $$$$$$\ $$$$$$\\_/ $$$$$$$ |
$$  _____|$$  __$$\ $$  __$$\\_$$  _|  $$  __$$ |
\$$$$$$\  $$ /  $$ |$$ |  \__| $$ |    $$ /  $$ |
 \____$$\ $$ |  $$ |$$ |       $$ |$$\ $$ |  $$ |
$$$$$$$  |\$$$$$$  |$$ |       \$$$$  |\$$$$$$$ |
\_______/  \______/ \__|        \____/  \_______|
                                                 
                                                 
                                                 
`))
	s.WriteString("\n")
	s.WriteString(Styles.Title.Render("Sortd File Organizer"))
	s.WriteString("\n")

	// Show current directory
	s.WriteString(fmt.Sprintf("Directory: %s\n\n", m.currentDir))

	// Command buffer if in command mode
	if m.mode == Command {
		s.WriteString(m.commandBuffer)
		s.WriteString("\n\n")
	}

	// File listing
	if len(m.files) == 0 {
		s.WriteString("No files in current directory\n")
	} else {
		for i, file := range m.files {
			style := Styles.Unselected
			prefix := "  "

			if m.selectedFiles[file.Name] {
				style = Styles.Selected
			}
			if i == m.cursor {
				prefix = "> "
			}

			s.WriteString(fmt.Sprintf("%s%s\n", prefix, style.Render(file.Name)))
		}
	}

	// Status message
	if m.statusMsg != "" {
		s.WriteString("\n" + m.statusMsg + "\n")
	}

	// Help text
	if m.showHelp {
		s.WriteString("\n" + Styles.Help.Render(m.helpText))
	}

	return Styles.App.Render(s.String())
}

func (m model) handleDirectoryBrowsing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		entries, err := os.ReadDir(m.currentDir)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error reading directory: %v", err)
			return m, nil
		}
		m.files = make([]FileEntry, 0)
		// If any subdirectory exists, display only directories.
		hasDir := false
		for _, entry := range entries {
			if entry.IsDir() {
				hasDir = true
				break
			}
		}
		for _, entry := range entries {
			if hasDir {
				if entry.IsDir() {
					m.files = append(m.files, FileEntry{
						Name: entry.Name(),
						Path: filepath.Join(m.currentDir, entry.Name()),
					})
				}
			} else {
				m.files = append(m.files, FileEntry{
					Name: entry.Name(),
					Path: filepath.Join(m.currentDir, entry.Name()),
				})
			}
		}
		sort.Slice(m.files, func(i, j int) bool {
			return m.files[i].Name < m.files[j].Name
		})
		m.cursor = 0
	case "enter":
		if len(m.files) > 0 {
			selected := m.files[m.cursor]
			if info, err := os.Stat(selected.Path); err == nil && info.IsDir() {
				m.currentDir = selected.Path
				m.cursor = 0
				m.scanDirectory()
			}
		}
	}
	return m, nil
}

func main() {
	// Determine if we're running in test mode.
	testMode := os.Getenv("TESTMODE") == "true"
	var p *tea.Program
	if testMode {
		// Run in test mode without alt screen by not specifying tea.WithAltScreen.
		p = tea.NewProgram(initialModel(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	} else {
		p = tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	}

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

func handleScan(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("no file specified")
	}

	path := args[0]
	engine := analysis.New()

	result, err := engine.Scan(path)
	if err != nil {
		if err == analysis.ErrFileNotFound {
			return fmt.Errorf("file not found")
		}
		return err
	}

	fmt.Print(result.String())
	return nil
}
