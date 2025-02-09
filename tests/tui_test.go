package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sortd/internal/tui"
	"sortd/internal/tui/common"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Mode int
type WizardStep int

const (
	Setup Mode = iota
	Normal
	Command
	Visual
)

const (
	WelcomeStep WizardStep = iota
	ConfigStep
	DirectoryStep
	RulesStep
)

type FileEntry struct {
	Name string
	Path string
}

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

type Tui struct {
	selectedFiles map[string]bool
	helpText      string
	cursor        int
	files         []FileEntry
	mode          Mode
	wizardStep    WizardStep
	showHelp      bool
	currentDir    string
	currentFile   string
	wizardChoices map[string]bool
	configPath    string
}

func (t *Tui) Update(msg tea.Msg) (*Tui, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return t.handleKeyMsg(msg)
	}
	return t, nil
}

func (t *Tui) handleKeyMsg(msg tea.KeyMsg) (*Tui, tea.Cmd) {
	newTui := &Tui{
		selectedFiles: make(map[string]bool),
		files:         make([]FileEntry, len(t.files)),
		mode:          t.mode,
		wizardStep:    t.wizardStep,
		helpText:      t.helpText,
		cursor:        t.cursor,
		showHelp:      t.showHelp,
		currentDir:    t.currentDir,
		currentFile:   t.currentFile,
	}

	copy(newTui.files, t.files)
	for k, v := range t.selectedFiles {
		newTui.selectedFiles[k] = v
	}

	switch msg.String() {
	case "j", "down", "↓":
		if newTui.cursor < len(newTui.files)-1 {
			newTui.cursor++
			if len(newTui.files) > 0 {
				newTui.currentFile = newTui.files[newTui.cursor].Name
			}
		}
	case "k", "up", "↑":
		if newTui.cursor > 0 {
			newTui.cursor--
			if len(newTui.files) > 0 {
				newTui.currentFile = newTui.files[newTui.cursor].Name
			}
		}
	case "enter":
		if len(newTui.files) > 0 {
			file := newTui.files[newTui.cursor]
			if info, err := os.Stat(file.Path); err == nil && info.IsDir() {
				newTui.currentDir = file.Path
				newTui.scanDirectory()
			}
		}
	case "?":
		newTui.showHelp = !t.showHelp
	case "q", "esc":
		if newTui.mode == Setup {
			newTui.wizardStep = WelcomeStep
		} else {
			return newTui, tea.Quit
		}
	}

	return newTui, nil
}

func (t *Tui) scanDirectory() error {
	entries, err := os.ReadDir(t.currentDir)
	if err != nil {
		return err
	}

	t.files = make([]FileEntry, 0)
	for _, entry := range entries {
		t.files = append(t.files, FileEntry{
			Name: entry.Name(),
			Path: filepath.Join(t.currentDir, entry.Name()),
		})
	}

	if len(t.files) > 0 {
		t.currentFile = t.files[0].Name
	}

	return nil
}

func NewTui() *Tui {
	tui := &Tui{
		selectedFiles: make(map[string]bool),
		files:         make([]FileEntry, 0),
		wizardChoices: make(map[string]bool),
		helpText: `Navigation:
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
	r: Refresh view`,
		cursor:     0,
		mode:       Normal,
		wizardStep: WelcomeStep,
		showHelp:   true,
	}
	return tui
}

func (t *Tui) ShowHelp() string {
	return Styles.Help.Render(t.helpText)
}

func (t *Tui) SelectFile(path string) error {
	if path == "" {
		return fmt.Errorf("no file specified")
	}
	// Only allow selecting files that have been added
	found := false
	for _, f := range t.files {
		if f.Path == path {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("file not found: %s", path)
	}
	t.selectedFiles[path] = true
	return nil
}

func (t *Tui) DeselectFile(path string) error {
	if path == "" {
		return fmt.Errorf("no file specified")
	}
	delete(t.selectedFiles, path)
	return nil
}

func (t *Tui) IsSelected(path string) bool {
	_, ok := t.selectedFiles[path]
	return ok
}

func (t *Tui) AddFile(path string) {
	t.files = append(t.files, FileEntry{Name: filepath.Base(path), Path: path})
}

func (t *Tui) MoveCursor(delta int) {
	if len(t.files) == 0 {
		t.cursor = 0
		return
	}

	t.cursor += delta
	if t.cursor < 0 {
		t.cursor = 0
	}
	if t.cursor >= len(t.files) {
		t.cursor = len(t.files) - 1
	}
}

func (t *Tui) View() string {
	if t.mode == Setup {
		var s strings.Builder
		s.WriteString(Styles.Title.Render("Welcome to Sortd"))
		s.WriteString("\n\n")

		if t.wizardStep == WelcomeStep {
			s.WriteString("Quick Start - Organize files now\n")
			s.WriteString("Setup Configuration - Customize rules and patterns\n")
			s.WriteString("\nChoose an option (1-4)")
		}

		return Styles.App.Render(s.String())
	}

	var s strings.Builder
	s.WriteString(Styles.Title.Render("Sortd File Organizer"))
	s.WriteString("\n\n")

	if t.showHelp {
		s.WriteString(Styles.Help.Render(t.helpText))
		s.WriteString("\n\n")
	}

	// Handle empty state first
	if len(t.files) == 0 {
		s.WriteString("No files to display yet.\n")
		return Styles.App.Render(s.String())
	}

	// Only show files and cursor if we have files
	for i, file := range t.files {
		style := Styles.Unselected
		if t.IsSelected(file.Path) {
			style = Styles.Selected
		}

		prefix := "  "
		if i == t.cursor {
			prefix = "> "
		}

		s.WriteString(prefix + style.Render(file.Name) + "\n")
	}

	return Styles.App.Render(s.String())
}

func TestTUIInitialization(t *testing.T) {
	tui := NewTui()
	assert.NotNil(t, tui)
	assert.Contains(t, tui.ShowHelp(), "Navigation:")
	assert.Contains(t, tui.ShowHelp(), "Commands:")
}

func TestTUIInteraction(t *testing.T) {
	tui := NewTui()

	// Test file navigation
	tui.AddFile("test1.txt")
	tui.AddFile("test2.txt")
	tui.AddFile("test3.txt")

	assert.Equal(t, 0, tui.cursor)

	tui.MoveCursor(1)
	assert.Equal(t, 1, tui.cursor)

	tui.MoveCursor(-1)
	assert.Equal(t, 0, tui.cursor)

	// Test bounds
	tui.MoveCursor(-1)
	assert.Equal(t, 0, tui.cursor)

	tui.MoveCursor(10)
	assert.Equal(t, 2, tui.cursor)
}

func TestTuiCommands(t *testing.T) {
	t.Run("help command", func(t *testing.T) {
		tui := NewTui()
		help := tui.ShowHelp()
		assert.Contains(t, help, "Navigation:")
		assert.Contains(t, help, "j/↓, k/↑: Move cursor")
	})

	t.Run("file selection", func(t *testing.T) {
		tui := NewTui()
		tui.AddFile("test.txt")
		err := tui.SelectFile("test.txt")
		assert.NoError(t, err)
		assert.True(t, tui.IsSelected("test.txt"))
	})

	t.Run("file deselection", func(t *testing.T) {
		tui := NewTui()
		tui.SelectFile("test.txt")
		err := tui.DeselectFile("test.txt")
		assert.NoError(t, err)
		assert.False(t, tui.IsSelected("test.txt"))
	})

	t.Run("view rendering", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Normal
		tui.AddFile("test1.txt")
		tui.SelectFile("test1.txt")

		view := tui.View()
		assert.Contains(t, view, "Sortd File Organizer")
		assert.Contains(t, view, "test1.txt")
		assert.Contains(t, view, ">")
	})

	t.Run("empty file selection", func(t *testing.T) {
		tui := NewTui()
		err := tui.SelectFile("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no file specified")
	})

	t.Run("empty file deselection", func(t *testing.T) {
		tui := NewTui()
		err := tui.DeselectFile("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no file specified")
	})

	t.Run("multiple file selection", func(t *testing.T) {
		tui := NewTui()
		files := []string{"test1.txt", "test2.txt", "test3.txt"}

		// Add files first
		for _, file := range files {
			tui.AddFile(file)
		}

		for _, file := range files {
			err := tui.SelectFile(file)
			assert.NoError(t, err)
		}

		for _, file := range files {
			assert.True(t, tui.IsSelected(file))
		}
	})

	t.Run("cursor navigation with empty list", func(t *testing.T) {
		tui := NewTui()

		tui.MoveCursor(1)
		assert.Equal(t, 0, tui.cursor)

		tui.MoveCursor(-1)
		assert.Equal(t, 0, tui.cursor)
	})

	t.Run("view with no files", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Normal
		view := tui.View()

		// Debug output
		t.Logf("View output:\n%q", view)

		assert.Contains(t, view, "Sortd File Organizer")
		assert.NotContains(t, view, ">", "View should not contain cursor when no files")
	})

	t.Run("view with multiple files", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Normal
		files := []string{"test1.txt", "test2.txt", "test3.txt"}

		for _, file := range files {
			tui.AddFile(file)
		}

		tui.SelectFile("test2.txt")
		tui.MoveCursor(1)

		view := tui.View()
		assert.Contains(t, view, "test1.txt")
		assert.Contains(t, view, "test2.txt")
		assert.Contains(t, view, "test3.txt")
		assert.Contains(t, view, ">")

		// Verify styling
		assert.Contains(t, view, Styles.Selected.Render("test2.txt"))
		assert.Contains(t, view, Styles.Unselected.Render("test1.txt"))
	})

	t.Run("style rendering", func(t *testing.T) {
		tui := NewTui()

		// Test help style
		help := tui.ShowHelp()
		assert.Contains(t, help, "Navigation:")

		// Test title style
		view := tui.View()
		assert.Contains(t, view, "Sortd File Organizer")
	})

	t.Run("cursor bounds with files", func(t *testing.T) {
		tui := NewTui()

		// Add some files
		tui.AddFile("test1.txt")
		tui.AddFile("test2.txt")

		// Test upper bound
		tui.MoveCursor(10)
		assert.Equal(t, 1, tui.cursor)

		// Test lower bound
		tui.MoveCursor(-10)
		assert.Equal(t, 0, tui.cursor)

		// Test normal movement
		tui.MoveCursor(1)
		assert.Equal(t, 1, tui.cursor)
	})
}

func TestTuiFileManagement(t *testing.T) {
	t.Run("add and remove files", func(t *testing.T) {
		tui := NewTui()

		// Add files
		tui.AddFile("test1.txt")
		tui.AddFile("test2.txt")

		assert.Equal(t, 2, len(tui.files))
		assert.Contains(t, tui.files, FileEntry{Name: "test1.txt", Path: "test1.txt"})
		assert.Contains(t, tui.files, FileEntry{Name: "test2.txt", Path: "test2.txt"})
	})

	t.Run("selection state persistence", func(t *testing.T) {
		tui := NewTui()

		// Add and select files
		tui.AddFile("test1.txt")
		tui.AddFile("test2.txt")

		err := tui.SelectFile("test1.txt")
		assert.NoError(t, err)

		// Verify selection persists after cursor movement
		tui.MoveCursor(1)
		assert.True(t, tui.IsSelected("test1.txt"))

		// Verify selection state is independent
		assert.False(t, tui.IsSelected("test2.txt"))
	})
}

// Add this test to verify the View() function works with empty state
func TestTuiViewEmpty(t *testing.T) {
	tui := NewTui()
	view := tui.View()

	// Should still render title even with no files
	assert.Contains(t, view, "Sortd File Organizer")
}

func TestTuiWizard(t *testing.T) {
	t.Run("welcome_step", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Setup
		tui.wizardStep = WelcomeStep

		// Simulate selecting option 1
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")}
		newTui, _ := tui.Update(msg)
		assert.Equal(t, Normal, newTui.mode)
	})

	t.Run("config_step", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Setup
		tui.wizardStep = WelcomeStep

		// Simulate selecting option 2
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")}
		newTui, _ := tui.Update(msg)
		assert.Equal(t, ConfigStep, newTui.wizardStep)
	})
}

func TestTuiNavigation(t *testing.T) {
	t.Run("command help visibility", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Normal

		view := tui.View()
		assert.Contains(t, view, "Navigation:")
		assert.Contains(t, view, "Commands:")

		// Test help toggle
		newTui, _ := tui.Update("?")
		assert.NotEqual(t, tui.showHelp, newTui.showHelp)
	})
}

// Simplified test for core navigation
func TestTUIKeyboardNavigation(t *testing.T) {
	t.Run("basic cursor movement", func(t *testing.T) {
		m := NewTui()
		m.mode = Normal
		m.AddFile("test1.txt")
		m.AddFile("test2.txt")

		// Test down
		newModel, cmd := m.Update("j")
		assert.Nil(t, cmd)
		assert.Equal(t, 1, newModel.cursor)

		// Test up
		newModel, cmd = newModel.Update("k")
		assert.Nil(t, cmd)
		assert.Equal(t, 0, newModel.cursor)
	})

	t.Run("quit command", func(t *testing.T) {
		m := NewTui()
		m.mode = Normal

		_, cmd := m.Update("q")
		assert.Equal(t, tea.Quit, cmd)
	})
}

// Simplified help test
func TestHelpToggle(t *testing.T) {
	t.Run("toggle help visibility", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Normal
		tui.showHelp = true

		// Toggle help off
		newTui, _ := tui.Update("?")
		assert.False(t, newTui.showHelp)
	})
}

func TestTuiStyleRendering(t *testing.T) {
	t.Run("normal mode", func(t *testing.T) {
		tui := NewTui()
		view := tui.View()
		assert.Contains(t, view, "Sortd File Organizer")
	})

	t.Run("setup mode", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Setup
		view := tui.View()
		assert.Contains(t, view, "Welcome to Sortd")
	})
}

func TestTuiCursorNavigation(t *testing.T) {
	t.Run("cursor navigation", func(t *testing.T) {
		tui := NewTui()
		tui.AddFile("file1.txt")
		tui.AddFile("file2.txt")

		tui.MoveCursor(1)
		assert.Equal(t, 1, tui.cursor, "Should move down")

		tui.MoveCursor(-1)
		assert.Equal(t, 0, tui.cursor, "Should move up")

		tui.MoveCursor(10)
		assert.Equal(t, 1, tui.cursor, "Should clamp to max index")
	})
}

func TestWatchModeBrowsing(t *testing.T) {
	t.Run("directory browsing with tab", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Setup
		tui.wizardStep = DirectoryStep
		tui.wizardChoices["watch"] = true

		// Create test directory structure
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		tui.currentDir = tmpDir

		// Test tab navigation
		newTui, _ := tui.Update("tab")
		assert.Contains(t, newTui.files, FileEntry{Name: "subdir", Path: subDir})

		// Test directory selection
		newTui.cursor = 0 // Select subdir
		newTui, _ = newTui.Update("enter")
		assert.Equal(t, subDir, newTui.currentDir)
	})

	t.Run("watch mode initialization", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Setup
		tui.wizardStep = DirectoryStep
		tui.wizardChoices["watch"] = true

		// Test enter to start watching
		newTui, _ := tui.Update("enter")
		assert.Equal(t, Normal, newTui.mode)
	})
}

func TestConfigurationSetup(t *testing.T) {
	t.Run("config option selection", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Setup
		tui.wizardStep = ConfigStep

		// Test cursor movement
		newTui, _ := tui.Update("j")
		assert.Equal(t, 1, newTui.cursor)

		// Test option selection
		newTui, _ = newTui.Update("enter")
		// Verify the selected option was handled
		assert.Equal(t, RulesStep, newTui.wizardStep)
	})

	t.Run("config navigation bounds", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Setup
		tui.wizardStep = ConfigStep

		// Test upper bound
		newTui, _ := tui.Update("k")
		assert.Equal(t, 0, newTui.cursor)

		// Test lower bound
		for i := 0; i < 5; i++ {
			newTui, _ = newTui.Update("j")
		}
		assert.Equal(t, 3, newTui.cursor) // Should stop at last option
	})

	t.Run("escape from config", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Setup
		tui.wizardStep = ConfigStep

		newTui, _ := tui.Update("esc")
		assert.Equal(t, WelcomeStep, newTui.wizardStep)
		assert.Equal(t, 0, newTui.cursor)
	})
}

func TestHelpSection(t *testing.T) {
	t.Run("help toggle", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Normal

		// Help should be visible by default
		assert.True(t, tui.showHelp)
		view := tui.View()
		assert.Contains(t, view, "Navigation:")
		assert.Contains(t, view, "Commands:")

		// Toggle help off
		newTui, _ := tui.Update("?")
		assert.False(t, newTui.showHelp)
		view = newTui.View()
		assert.NotContains(t, view, "Navigation:")

		// Toggle help back on
		newTui, _ = newTui.Update("?")
		assert.True(t, newTui.showHelp)
		view = newTui.View()
		assert.Contains(t, view, "Navigation:")
	})

	t.Run("help content", func(t *testing.T) {
		tui := NewTui()
		view := tui.View()

		// Check all help sections are present
		assert.Contains(t, view, "Navigation:")
		assert.Contains(t, view, "j/↓, k/↑: Move cursor")
		assert.Contains(t, view, "h/←, l/→: Change directory")

		assert.Contains(t, view, "Selection:")
		assert.Contains(t, view, "space: Toggle selection")
		assert.Contains(t, view, "v: Visual mode")

		assert.Contains(t, view, "Commands:")
		assert.Contains(t, view, "q, quit: Exit")
		assert.Contains(t, view, "?: Toggle help")

		assert.Contains(t, view, "Organization:")
		assert.Contains(t, view, "o: Organize selected")
	})

	t.Run("help in setup mode", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Setup
		view := tui.View()

		// Setup mode should show different help
		assert.Contains(t, view, "Choose an option (1-4)")
		assert.Contains(t, view, "Quick Start")
		assert.Contains(t, view, "Setup Configuration")
	})

	t.Run("help persistence", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Normal
		tui.showHelp = false

		// Help state should persist through navigation
		newTui, _ := tui.Update("j")
		assert.False(t, newTui.showHelp)

		// Help state should persist through mode changes
		newTui.mode = Setup
		view := newTui.View()
		assert.NotContains(t, view, "Navigation:")
	})
}

func TestTui_SelectOption(t *testing.T) {
	type args struct {
		option string
	}
	tests := []struct {
		name    string
		tui     *Tui
		step    WizardStep
		args    args
		wantErr bool
	}{
		{
			name: "valid option",
			tui: &Tui{
				mode:       Setup,
				wizardStep: WelcomeStep,
			},
			args: args{
				option: "1",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newTui, _ := tt.tui.Update(tt.args.option)
			if (newTui == nil) != tt.wantErr {
				t.Errorf("Tui.Update() error = %v, wantErr %v", newTui == nil, tt.wantErr)
			}
			if !tt.wantErr {
				assert.NotNil(t, newTui)
			}
		})
	}
}

func TestTuiOptionSelection(t *testing.T) {
	t.Run("select with enter", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Normal
		tui.AddFile("test1.txt")

		// Test selection with enter
		newTui, _ := tui.Update("enter")
		assert.True(t, newTui.IsSelected("test1.txt"))

		// Test deselection with enter
		newTui, _ = newTui.Update("enter")
		assert.False(t, newTui.IsSelected("test1.txt"))
	})

	t.Run("select with space", func(t *testing.T) {
		tui := NewTui()
		tui.mode = Normal
		tui.AddFile("test1.txt")

		// Test selection with space
		newTui, _ := tui.Update(" ")
		assert.True(t, newTui.IsSelected("test1.txt"))
	})
}

// Add this function to create a test config
func createTestConfig(t *testing.T, tmpDir string) string {
	configContent := `
organize:
  patterns:
    - match: "*.txt"
      target: "documents/"
    - match: "*.jpg"
      target: "images/"
settings:
  dry_run: false
  create_dirs: true
  backup: false
  collision: "rename"
directories:
  default: "` + tmpDir + `"
  watch:
    - "` + tmpDir + `"
watch_mode:
  enabled: true
  interval: 5
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))
	return configPath
}

// Update TestFileNavigation to use config
func TestFileNavigation(t *testing.T) {
	// Setup test directory structure
	tmpDir := t.TempDir()
	copyTestData(t, tmpDir)
	configPath := createTestConfig(t, tmpDir)

	// Initialize TUI with the correct test data directory
	m := NewTui()
	m.configPath = configPath
	m.currentDir = tmpDir
	require.NoError(t, m.scanDirectory())

	tests := []struct {
		name     string
		keys     []string
		wantFile string
		wantDir  string
	}{
		{
			name:     "navigate_down",
			keys:     []string{"j"},
			wantFile: "file1.txt",
			wantDir:  tmpDir,
		},
		{
			name:     "navigate_up",
			keys:     []string{"j", "k"},
			wantFile: "file1.txt",
			wantDir:  tmpDir,
		},
		{
			name:     "enter_directory",
			keys:     []string{"enter"},
			wantFile: "file3.txt",
			wantDir:  filepath.Join(tmpDir, "subdir"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := m
			var cmd tea.Cmd

			// Apply test keys
			for _, key := range tt.keys {
				model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
				require.Nil(t, cmd)
			}

			assert.Equal(t, tt.wantFile, model.currentFile)
			assert.Equal(t, tt.wantDir, model.currentDir)
		})
	}
}

func TestMainMenu(t *testing.T) {
	m := NewTui()

	tests := []struct {
		name        string
		key         string
		wantMode    Mode
		wantContent string
	}{
		{
			name:        "quick_start",
			key:         "1",
			wantMode:    Normal,
			wantContent: "file1.txt", // First file should be visible
		},
		{
			name:        "show_help",
			key:         "4",
			wantMode:    Normal,
			wantContent: "Quick Start Guide",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := m
			var cmd tea.Cmd

			model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			require.Nil(t, cmd)

			assert.Equal(t, tt.wantMode, model.mode)
			view := model.View()
			assert.Contains(t, view, tt.wantContent)
		})
	}
}

func TestFileSelection(t *testing.T) {
	t.Run("single_selection", func(t *testing.T) {
		tmpDir := t.TempDir()
		createTestFiles(t, tmpDir)

		tui := NewTui()
		tui.currentDir = tmpDir
		require.NoError(t, tui.scanDirectory())

		// Select existing file
		err := tui.SelectFile("file1.txt")
		require.NoError(t, err)
		assert.True(t, tui.IsSelected("file1.txt"))
	})
}

func TestTabNavigation(t *testing.T) {
	t.Run("tab_enter_changes_directory", func(t *testing.T) {
		// Setup
		tmpDir := t.TempDir()
		createTestFiles(t, tmpDir)

		tui := NewTui()
		tui.currentDir = tmpDir
		require.NoError(t, tui.scanDirectory())

		// Simulate entering the empty directory
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newTui, _ := tui.Update(msg)

		expectedDir := filepath.Join(tmpDir, "subdir")
		assert.Equal(t, expectedDir, newTui.currentDir)
	})
}

func TestFileOperations(t *testing.T) {
	tmpDir := t.TempDir() // This will be automatically cleaned up

	t.Cleanup(func() {
		// Any additional cleanup needed
		os.RemoveAll(tmpDir)
	})

	// Test implementation...
	createTestFiles(t, tmpDir)
}

// Add this helper function
func createTestFiles(t *testing.T, tmpDir string) {
	files := []string{"file1.txt", "file2.txt", "subdir/file3.txt"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte("test"), 0644))
	}
}

func TestModel(t *testing.T) {
	t.Run("initialization", func(t *testing.T) {
		m := tui.New()
		assert.Equal(t, common.Normal, m.Mode())
		assert.Empty(t, m.Files())
		assert.False(t, m.ShowHelp())
	})

	t.Run("file_navigation", func(t *testing.T) {
		// Setup test directory
		tmpDir := t.TempDir()
		createTestFiles(t, tmpDir)

		m := tui.New()
		m.SetCurrentDir(tmpDir)
		require.NoError(t, m.ScanDirectory())

		// Test initial state
		assert.Equal(t, 3, len(m.Files())) // file1.txt, file2.txt, subdir
		assert.Equal(t, 0, m.Cursor())

		// Test navigation
		tests := []struct {
			name     string
			key      string
			wantFile string
		}{
			{"move_down", "j", "file2.txt"},
			{"move_up", "k", "file1.txt"},
			{"down_arrow", "↓", "file2.txt"},
			{"up_arrow", "↑", "file1.txt"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
				require.Nil(t, cmd)
				m = model.(*tui.Model)
				assert.Equal(t, tt.wantFile, m.CurrentFile())
			})
		}
	})

	t.Run("file_selection", func(t *testing.T) {
		tmpDir := t.TempDir()
		createTestFiles(t, tmpDir)

		m := tui.New()
		m.SetCurrentDir(tmpDir)
		require.NoError(t, m.ScanDirectory())

		// Test selection
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
		require.Nil(t, cmd)
		m = model.(*tui.Model)
		assert.True(t, m.IsSelected("file1.txt"))

		// Test deselection
		model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
		require.Nil(t, cmd)
		m = model.(*tui.Model)
		assert.False(t, m.IsSelected("file1.txt"))
	})

	t.Run("directory_navigation", func(t *testing.T) {
		tmpDir := t.TempDir()
		createTestFiles(t, tmpDir)

		m := tui.New()
		m.SetCurrentDir(tmpDir)
		require.NoError(t, m.ScanDirectory())

		// Navigate to subdir
		m.SetCursor(2) // Move to subdir
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.Nil(t, cmd)
		m = model.(*tui.Model)

		assert.Equal(t, filepath.Join(tmpDir, "subdir"), m.CurrentDir())
		assert.Equal(t, 1, len(m.Files())) // file3.txt
	})

	t.Run("help_toggle", func(t *testing.T) {
		m := tui.New()

		// Toggle help on
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		require.Nil(t, cmd)
		m = model.(*tui.Model)
		assert.True(t, m.ShowHelp())

		// Toggle help off
		model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		require.Nil(t, cmd)
		m = model.(*tui.Model)
		assert.False(t, m.ShowHelp())
	})

	t.Run("main_menu", func(t *testing.T) {
		m := tui.New()

		tests := []struct {
			name     string
			key      string
			wantMode common.Mode
		}{
			{"quick_start", "1", common.Normal},
			{"show_help", "4", common.Normal},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
				require.Nil(t, cmd)
				m = model.(*tui.Model)
				assert.Equal(t, tt.wantMode, m.Mode())
			})
		}
	})

	t.Run("quit", func(t *testing.T) {
		m := tui.New()

		// Test quit with 'q'
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		assert.Equal(t, tea.Quit, cmd)

		// Test quit with 'esc'
		_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("esc")})
		assert.Equal(t, tea.Quit, cmd)
	})
}
