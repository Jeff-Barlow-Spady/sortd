package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"sortd/internal/common"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Use the common FileEntry type
type FileEntry = common.FileEntry

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
	files         []common.FileEntry
	mode          common.Mode
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

func NewTui() *Tui {
	tui := &Tui{
		selectedFiles: make(map[string]bool),
		files:         make([]common.FileEntry, 0),
		mode:          common.Setup,
		cursor:        0,
		showHelp:      true,
	}
	return tui
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
		files:         make([]common.FileEntry, len(t.files)),
		mode:          t.mode,
		cursor:        t.cursor,
		showHelp:      t.showHelp,
		currentDir:    t.currentDir,
		currentFile:   t.currentFile,
		commandBuffer: t.commandBuffer,
		statusMsg:     t.statusMsg,
		lastKey:       t.lastKey,
		visualStart:   t.visualStart,
		visualEnd:     t.visualEnd,
		visualMode:    t.visualMode,
	}

	copy(newTui.files, t.files)
	for k, v := range t.selectedFiles {
		newTui.selectedFiles[k] = v
	}

	switch msg.String() {
	case "1":
		if newTui.mode == common.Setup {
			newTui.mode = common.Normal
			return newTui, nil
		}
	case "2":
		if newTui.mode == common.Setup {
			newTui.mode = common.Command
			return newTui, nil
		}
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
				newTui.cursor = 0 // Reset cursor when entering directory
			} else {
				newTui.selectedFiles[file.Path] = !newTui.selectedFiles[file.Path]
			}
		}
	case " ":
		if len(newTui.files) > 0 {
			file := newTui.files[newTui.cursor]
			newTui.selectedFiles[file.Path] = !newTui.selectedFiles[file.Path]
		}
	case "?":
		newTui.showHelp = !newTui.showHelp
	case "q", "esc":
		if newTui.mode == common.Setup {
			newTui.mode = common.Normal
			return newTui, nil
		}
		return newTui, tea.Quit
	}

	return newTui, nil
}

func (t *Tui) scanDirectory() error {
	entries, err := os.ReadDir(t.currentDir)
	if err != nil {
		return err
	}

	t.files = make([]common.FileEntry, 0)
	for _, entry := range entries {
		fullPath := filepath.Join(t.currentDir, entry.Name())
		t.files = append(t.files, common.FileEntry{
			Name: entry.Name(),
			Path: fullPath,
		})
	}

	if len(t.files) > 0 {
		t.currentFile = t.files[0].Name
	}

	return nil
}

func (t *Tui) View() string {
	var s strings.Builder
	s.WriteString(Styles.Title.Render("Sortd File Organizer"))
	s.WriteString("\n\n")

	if t.mode == common.Setup {
		s.WriteString("Welcome to Sortd\n\n")
		s.WriteString("Choose an option (1-4)\n\n")
		s.WriteString("1. Quick Start - Organize Files\n")
		s.WriteString("2. Setup Configuration\n")
		s.WriteString("3. Watch Mode (Coming Soon)\n")
		s.WriteString("4. Show Help\n\n")
		s.WriteString("Quick Start Guide\n")
		return Styles.App.Render(s.String())
	}

	if t.showHelp {
		s.WriteString(t.GetHelp())
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
		if t.selectedFiles[file.Path] {
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

func (t *Tui) GetHelp() string {
	return Styles.Help.Render(`Navigation:
	j/↓, k/↑: Move cursor
	h/←, l/→: Change directory
	enter: Open directory
	gg: Go to top
	G: Go to bottom

Selection:
	space: Toggle selection
	v: Visual mode
	V: Visual line mode

Commands:
	q: Quit
	:: Command mode
	/: Search
	?: Toggle help

Organization:
	o: Organize selected
	r: Refresh view`)
}

func (t *Tui) SelectFile(path string) error {
	if path == "" {
		return fmt.Errorf("no file specified")
	}

	// Convert to absolute path if not already
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(t.currentDir, path)
	}

	// Check if file exists in current directory
	found := false
	for _, f := range t.files {
		if f.Path == absPath || f.Name == path {
			found = true
			t.selectedFiles[f.Path] = true
			break
		}
	}
	if !found {
		return fmt.Errorf("file not found: %s", path)
	}
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

func (t *Tui) AddFile(file common.FileEntry) {
	t.files = append(t.files, file)
	if len(t.files) == 1 {
		t.currentFile = t.files[0].Name
	}
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

func (t *Tui) SetCurrentDir(dir string) {
	t.currentDir = dir
	t.cursor = 0 // Reset cursor when changing directories
	t.scanDirectory()
}

func (t *Tui) CurrentDir() string {
	return t.currentDir
}

func (t *Tui) CurrentFile() string {
	return t.currentFile
}

func (t *Tui) SetCursor(pos int) {
	if pos >= 0 && pos < len(t.files) {
		t.cursor = pos
		t.currentFile = t.files[pos].Name
	}
}

func (t *Tui) ScanDirectory() error {
	return t.scanDirectory()
}

func (t *Tui) VisualMode() bool {
	return t.visualMode
}

func (t *Tui) SetShowHelp(show bool) {
	t.showHelp = show
}

func (t *Tui) ShowHelp() bool {
	return t.showHelp
}

func TestTUIInitialization(t *testing.T) {
	tui := NewTui()
	assert.NotNil(t, tui)
	assert.Equal(t, common.Setup, tui.mode)
	assert.True(t, tui.showHelp)
	assert.Contains(t, tui.GetHelp(), "Navigation:")
	assert.Contains(t, tui.GetHelp(), "Commands:")
}

func TestTUIInteraction(t *testing.T) {
	tui := NewTui()

	// Test file navigation
	tui.AddFile(common.FileEntry{Name: "test1.txt", Path: "test1.txt"})
	tui.AddFile(common.FileEntry{Name: "test2.txt", Path: "test2.txt"})
	tui.AddFile(common.FileEntry{Name: "test3.txt", Path: "test3.txt"})

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
		help := tui.GetHelp()
		assert.Contains(t, help, "Navigation:")
		assert.Contains(t, help, "j/↓, k/↑: Move cursor")
	})

	t.Run("file selection", func(t *testing.T) {
		tui := NewTui()
		file := common.FileEntry{Name: "test.txt", Path: "test.txt"}
		tui.AddFile(file)
		err := tui.SelectFile("test.txt")
		assert.NoError(t, err)
		assert.True(t, tui.IsSelected("test.txt"))
	})

	t.Run("view rendering", func(t *testing.T) {
		tui := NewTui()
		tui.mode = common.Normal
		file := common.FileEntry{Name: "test1.txt", Path: "test1.txt"}
		tui.AddFile(file)
		tui.SelectFile("test1.txt")

		view := tui.View()
		assert.Contains(t, view, "Sortd File Organizer")
		assert.Contains(t, view, "test1.txt")
		assert.Contains(t, view, ">")
	})

	t.Run("multiple file selection", func(t *testing.T) {
		tui := NewTui()
		files := []common.FileEntry{
			{Name: "test1.txt", Path: "test1.txt"},
			{Name: "test2.txt", Path: "test2.txt"},
			{Name: "test3.txt", Path: "test3.txt"},
		}

		// Add files first
		for _, file := range files {
			tui.AddFile(file)
		}

		for _, file := range files {
			err := tui.SelectFile(file.Name)
			assert.NoError(t, err)
		}

		for _, file := range files {
			assert.True(t, tui.IsSelected(file.Path))
		}
	})
}

func TestTuiFileManagement(t *testing.T) {
	t.Run("add and remove files", func(t *testing.T) {
		tui := NewTui()

		// Add files
		tui.AddFile(common.FileEntry{Name: "test1.txt", Path: "test1.txt"})
		tui.AddFile(common.FileEntry{Name: "test2.txt", Path: "test2.txt"})

		assert.Equal(t, 2, len(tui.files))
		assert.Contains(t, tui.files, common.FileEntry{Name: "test1.txt", Path: "test1.txt"})
		assert.Contains(t, tui.files, common.FileEntry{Name: "test2.txt", Path: "test2.txt"})
	})

	t.Run("selection state persistence", func(t *testing.T) {
		tui := NewTui()

		// Add and select files
		tui.AddFile(common.FileEntry{Name: "test1.txt", Path: "test1.txt"})
		tui.AddFile(common.FileEntry{Name: "test2.txt", Path: "test2.txt"})

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
		tui.mode = common.Setup

		// Simulate selecting option 1
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")}
		newTui, _ := tui.Update(msg)
		assert.Equal(t, common.Normal, newTui.mode)
	})

	t.Run("config_step", func(t *testing.T) {
		tui := NewTui()
		tui.mode = common.Setup

		// Simulate selecting option 2
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")}
		newTui, _ := tui.Update(msg)
		assert.Equal(t, common.Command, newTui.mode)
	})
}

func TestTuiNavigation(t *testing.T) {
	t.Run("command help visibility", func(t *testing.T) {
		tui := NewTui()
		tui.mode = common.Normal
		tui.showHelp = true

		view := tui.View()
		assert.Contains(t, view, "Navigation:")
		assert.Contains(t, view, "Commands:")

		// Test help toggle
		newTui, _ := tui.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		assert.False(t, newTui.showHelp)
	})
}

func TestTUIKeyboardNavigation(t *testing.T) {
	t.Run("basic cursor movement", func(t *testing.T) {
		m := NewTui()
		m.mode = common.Normal
		m.AddFile(common.FileEntry{Name: "test1.txt", Path: "test1.txt"})
		m.AddFile(common.FileEntry{Name: "test2.txt", Path: "test2.txt"})

		// Test down
		newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		assert.Nil(t, cmd)
		assert.Equal(t, 1, newModel.cursor)

		// Test up
		newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		assert.Nil(t, cmd)
		assert.Equal(t, 0, newModel.cursor)
	})

	t.Run("quit command", func(t *testing.T) {
		m := NewTui()
		m.mode = common.Normal

		// Test quit with 'q'
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		assert.NotNil(t, cmd)

		// Test quit with 'esc'
		_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("esc")})
		assert.NotNil(t, cmd)
	})
}

func TestTuiOptionSelection(t *testing.T) {
	t.Run("select with space", func(t *testing.T) {
		tui := NewTui()
		tui.mode = common.Normal
		file := common.FileEntry{Name: "test1.txt", Path: "test1.txt"}
		tui.AddFile(file)

		// Test selection with space
		newTui, _ := tui.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
		assert.True(t, newTui.IsSelected(file.Path))
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
		tui.mode = common.Setup
		view := tui.View()
		assert.Contains(t, view, "Welcome to Sortd")
	})
}

func TestTuiCursorNavigation(t *testing.T) {
	t.Run("cursor navigation", func(t *testing.T) {
		tui := NewTui()
		tui.AddFile(common.FileEntry{Name: "file1.txt", Path: "file1.txt"})
		tui.AddFile(common.FileEntry{Name: "file2.txt", Path: "file2.txt"})

		tui.MoveCursor(1)
		assert.Equal(t, 1, tui.cursor, "Should move down")

		tui.MoveCursor(-1)
		assert.Equal(t, 0, tui.cursor, "Should move up")

		tui.MoveCursor(10)
		assert.Equal(t, 1, tui.cursor, "Should clamp to max index")
	})
}

func TestWatchModeBrowsing(t *testing.T) {
	t.Run("directory_browsing_with_tab", func(t *testing.T) {
		tui := NewTui()
		tui.mode = common.Normal
		tui.selectedFiles["watch"] = true

		// Create test directory structure
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		tui.SetCurrentDir(tmpDir)
		require.NoError(t, tui.scanDirectory())

		// Find subdir index
		subdirIndex := -1
		for i, f := range tui.files {
			if filepath.Base(f.Path) == "subdir" {
				subdirIndex = i
				break
			}
		}
		require.NotEqual(t, -1, subdirIndex, "subdir not found")

		// Move cursor to subdir
		tui.SetCursor(subdirIndex)

		// Test directory selection
		newTui, _ := tui.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.Equal(t, subDir, newTui.currentDir)
	})

	t.Run("watch_mode_initialization", func(t *testing.T) {
		tui := NewTui()
		tui.mode = common.Normal
		tui.selectedFiles["watch"] = true

		// Test enter to start watching
		newTui, _ := tui.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.Equal(t, common.Normal, newTui.mode)
	})
}

func TestInputValidation(t *testing.T) {
	t.Run("handle invalid command", func(t *testing.T) {
		tui := NewTui()
		_, cmd := tui.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("invalid")})
		assert.Nil(t, cmd)
	})
}

func TestTui_SelectOption(t *testing.T) {
	type args struct {
		option string
	}
	tests := []struct {
		name    string
		tui     *Tui
		step    Mode
		args    args
		wantErr bool
	}{
		{
			name: "valid option",
			tui: &Tui{
				mode: common.Setup,
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
	files := createTestFiles(t, tmpDir)

	m := NewTui()
	m.mode = common.Normal // Switch to normal mode for navigation
	m.SetCurrentDir(tmpDir)
	require.NoError(t, m.scanDirectory())

	// Add test files to the model
	for _, file := range files {
		m.AddFile(file)
	}

	tests := []struct {
		name     string
		keys     []string
		wantFile string
		wantDir  string
	}{
		{
			name:     "navigate_down",
			keys:     []string{"j"},
			wantFile: "file2.txt",
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
			keys:     []string{"j", "j", "enter"},
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

func TestDirectoryNavigation(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	tui := NewTui()
	tui.mode = common.Normal
	tui.SetCurrentDir(tmpDir)
	require.NoError(t, tui.ScanDirectory())

	// Find subdir index
	subdirIndex := -1
	for i, f := range tui.files {
		if filepath.Base(f.Path) == "subdir" {
			subdirIndex = i
			break
		}
	}
	require.NotEqual(t, -1, subdirIndex, "subdir not found")

	// Move cursor to subdir
	tui.SetCursor(subdirIndex)

	// Test directory selection
	newTui, _ := tui.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, subDir, newTui.currentDir)
}

func TestFileSelection(t *testing.T) {
	t.Run("single_selection", func(t *testing.T) {
		tmpDir := t.TempDir()
		createTestFiles(t, tmpDir)

		tui := NewTui()
		tui.mode = common.Normal
		tui.SetCurrentDir(tmpDir)
		require.NoError(t, tui.scanDirectory())

		// Select file1.txt
		err := tui.SelectFile("file1.txt")
		require.NoError(t, err)
		assert.True(t, tui.IsSelected(filepath.Join(tmpDir, "file1.txt")))
	})
}

func TestTabNavigation(t *testing.T) {
	t.Run("tab_enter_changes_directory", func(t *testing.T) {
		// Setup
		tmpDir := t.TempDir()
		createTestFiles(t, tmpDir)

		tui := NewTui()
		tui.mode = common.Normal
		tui.SetCurrentDir(tmpDir)
		require.NoError(t, tui.scanDirectory())

		// Find subdir index
		subdirIndex := -1
		for i, f := range tui.files {
			if filepath.Base(f.Path) == "subdir" {
				subdirIndex = i
				break
			}
		}
		require.NotEqual(t, -1, subdirIndex, "subdir not found")

		// Move cursor to subdir
		tui.SetCursor(subdirIndex)

		// Simulate entering the directory
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

// Update createTestFiles to ensure consistent file creation
func createTestFiles(t *testing.T, tmpDir string) []common.FileEntry {
	files := []struct {
		path    string
		content string
	}{
		{filepath.Join(tmpDir, "file1.txt"), "test content 1"},
		{filepath.Join(tmpDir, "file2.txt"), "test content 2"},
		{filepath.Join(tmpDir, "subdir", "file3.txt"), "test content 3"},
	}

	var entries []common.FileEntry
	for _, f := range files {
		require.NoError(t, os.MkdirAll(filepath.Dir(f.path), 0755))
		require.NoError(t, os.WriteFile(f.path, []byte(f.content), 0644))
		entries = append(entries, common.FileEntry{
			Name: filepath.Base(f.path),
			Path: f.path,
		})
	}
	return entries
}

func TestModel(t *testing.T) {
	t.Run("initialization", func(t *testing.T) {
		// Create a clean temp directory for the test
		tmpDir := t.TempDir()

		m := NewTui()
		m.SetCurrentDir(tmpDir)

		assert.Equal(t, common.Setup, m.mode)
		assert.Empty(t, m.files)
		assert.True(t, m.showHelp)
	})

	t.Run("file_navigation", func(t *testing.T) {
		// Setup test directory
		tmpDir := t.TempDir()
		createTestFiles(t, tmpDir)

		m := NewTui()
		m.mode = common.Normal // Switch to normal mode for navigation
		m.SetCurrentDir(tmpDir)
		require.NoError(t, m.ScanDirectory())

		// Test initial state
		assert.Equal(t, 3, len(m.files)) // file1.txt, file2.txt, subdir
		assert.Equal(t, 0, m.cursor)

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
				newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
				m = newModel
				assert.Equal(t, tt.wantFile, m.currentFile)
			})
		}
	})

	t.Run("directory_navigation", func(t *testing.T) {
		tmpDir := t.TempDir()
		createTestFiles(t, tmpDir)

		m := NewTui()
		m.mode = common.Normal // Switch to normal mode for navigation
		m.SetCurrentDir(tmpDir)
		require.NoError(t, m.ScanDirectory())

		// Find subdir index
		subdirIndex := -1
		for i, f := range m.files {
			if filepath.Base(f.Path) == "subdir" {
				subdirIndex = i
				break
			}
		}
		require.NotEqual(t, -1, subdirIndex, "subdir not found")

		// Navigate to subdir
		m.SetCursor(subdirIndex)
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel

		expectedDir := filepath.Join(tmpDir, "subdir")
		assert.Equal(t, expectedDir, m.currentDir)
		assert.Equal(t, 1, len(m.files)) // file3.txt
	})
}
