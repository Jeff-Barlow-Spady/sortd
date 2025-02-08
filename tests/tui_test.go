package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
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
	helpText      string
	cursor        int
	files         []string
}

func NewTui() *Tui {
	return &Tui{
		selectedFiles: make(map[string]bool),
		helpText: `Available Commands:
- h, help: Show this help message
- q, quit: Exit the application
- s <file>: Select a file
- d <file>: Deselect a file
- o: Organize selected files`,
		cursor: 0,
		files:  make([]string, 0),
	}
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
		if f == path {
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
	t.files = append(t.files, path)
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
	var s strings.Builder

	s.WriteString(Styles.Title.Render("Sortd File Organizer"))
	s.WriteString("\n\n")

	for i, file := range t.files {
		style := Styles.Unselected
		if t.IsSelected(file) {
			style = Styles.Selected
		}

		prefix := "  "
		if i == t.cursor {
			prefix = "> "
		}

		s.WriteString(prefix + style.Render(file) + "\n")
	}

	return Styles.App.Render(s.String())
}

func TestTUIInitialization(t *testing.T) {
	tui := NewTui()
	assert.NotNil(t, tui)
	assert.Empty(t, tui.selectedFiles)
	assert.Contains(t, tui.ShowHelp(), "Available Commands")
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
		assert.Contains(t, help, "Available Commands")
		assert.Contains(t, help, "help: Show this help message")
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
		tui.AddFile("test1.txt")
		tui.SelectFile("test1.txt")

		view := tui.View()
		assert.Contains(t, view, "Sortd File Organizer")
		assert.Contains(t, view, "test1.txt")
		assert.Contains(t, view, ">") // Cursor indicator
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
		view := tui.View()

		assert.Contains(t, view, "Sortd File Organizer")
		assert.NotContains(t, view, ">")
	})

	t.Run("view with multiple files", func(t *testing.T) {
		tui := NewTui()
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
		assert.Contains(t, help, "Available Commands")

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
		assert.Contains(t, tui.files, "test1.txt")
		assert.Contains(t, tui.files, "test2.txt")
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
