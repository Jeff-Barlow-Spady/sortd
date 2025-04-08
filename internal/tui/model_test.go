package tui

import (
	"os"
	"path/filepath"
	"sortd/pkg/types"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelInitialization(t *testing.T) {
	m := New("test")
	assert.NotNil(t, m)
	assert.Equal(t, types.Normal, m.mode)
	assert.NotEmpty(t, m.currentDir)
	assert.NotNil(t, m.selectedFiles)
	assert.NotNil(t, m.analysisEngine)
	assert.NotNil(t, m.organizeEngine)
}

func TestModelEdgeCases(t *testing.T) {
	t.Run("empty_directory", func(t *testing.T) {
		m := New("test")
		tmpDir := t.TempDir()
		m.SetCurrentDir(tmpDir)

		err := m.ScanDirectory()
		require.NoError(t, err)
		assert.Empty(t, m.Files())

		// Navigation in empty directory should not panic
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		require.NotNil(t, cmd)
		assert.Equal(t, 0, model.(*Model).Cursor())
	})

	t.Run("nonexistent_directory", func(t *testing.T) {
		m := New("test")
		m.SetCurrentDir("/nonexistent/path")

		err := m.ScanDirectory()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	})

	t.Run("permission_denied", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping test when running as root")
		}

		tmpDir := t.TempDir()
		require.NoError(t, os.Chmod(tmpDir, 0000))
		defer os.Chmod(tmpDir, 0755)

		m := New("test")
		m.SetCurrentDir(tmpDir)

		err := m.ScanDirectory()
		assert.Error(t, err)
	})

	t.Run("cursor_bounds", func(t *testing.T) {
		m := New("test")
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644))

		m.SetCurrentDir(tmpDir)
		require.NoError(t, m.ScanDirectory())

		// Try to move cursor out of bounds
		m.SetCursor(-1)
		assert.Equal(t, 0, m.Cursor())

		m.SetCursor(100)
		assert.Equal(t, 0, m.Cursor())
	})

	t.Run("invalid_selection", func(t *testing.T) {
		m := New("test")
		assert.False(t, m.IsSelected("nonexistent.txt"))
	})

	t.Run("directory_navigation_errors", func(t *testing.T) {
		m := New("test")

		// Create test dir and setup the model
		tmpDir := t.TempDir()

		// Create a file that looks like a directory
		fakeDir := filepath.Join(tmpDir, "fake_dir")
		require.NoError(t, os.WriteFile(fakeDir, []byte("not a directory"), 0644))

		m.SetCurrentDir(tmpDir)
		require.NoError(t, m.ScanDirectory())

		// Create a list item for a fake directory
		// We need to manually set up the cursor and items since we won't do
		// real navigation in this test
		m.list.SetItems([]list.Item{
			Item{entry: types.FileEntry{Name: "fake_dir"}}, // Use FileEntry
		})

		// Attempt to enter the fake directory
		// First set cursor to the fake directory
		m.list.Select(0)

		// Simulate pressing Enter to navigate into it
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
		require.NotNil(t, cmd)

		// Directory should not change
		assert.Equal(t, tmpDir, model.(*Model).CurrentDir())
	})
}

func TestModelStateConsistency(t *testing.T) {
	t.Run("selection_persistence", func(t *testing.T) {
		m := New("test")
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644))

		m.SetCurrentDir(tmpDir)
		require.NoError(t, m.ScanDirectory())

		// Select the first file (test.txt) using the correct method
		m.list.Select(0) // Ensure the item is focused
		m.ToggleSelection() // Use the actual method to select

		// Navigate
		model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		assert.True(t, model.(*Model).IsSelected("test.txt"))
	})

	t.Run("cursor_file_sync", func(t *testing.T) {
		m := New("test")
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644))

		m.SetCurrentDir(tmpDir)
		require.NoError(t, m.ScanDirectory())

		// Verify cursor and current file stay in sync
		m.SetCursor(0)
		assert.Equal(t, "test.txt", m.CurrentFile())
	})
}

func TestModelKeyHandling(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		setupModel    func() *Model
		expectedState func(*testing.T, *Model)
	}{
		{
			name: "quit on q",
			key:  "q",
			setupModel: func() *Model {
				m := New("test")
				m.mode = types.Normal
				return m
			},
			expectedState: func(t *testing.T, m *Model) {
				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
				assert.NotNil(t, cmd)
				// Check if it's a quit command by type
				_, isQuit := cmd().(tea.QuitMsg)
				assert.True(t, isQuit)
			},
		},
		{
			name: "toggle help on ?",
			key:  "?",
			setupModel: func() *Model {
				m := New("test")
				m.mode = types.Normal
				m.showFullHelp = false
				return m
			},
			expectedState: func(t *testing.T, m *Model) {
				newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
				assert.True(t, newModel.(*Model).ShowHelp())
			},
		},
		{
			name: "enter command mode on :",
			key:  ":",
			setupModel: func() *Model {
				m := New("test")
				m.mode = types.Normal
				return m
			},
			expectedState: func(t *testing.T, m *Model) {
				newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
				require.NotNil(t, cmd)
				assert.Equal(t, types.Command, newModel.(*Model).mode)
				assert.Equal(t, ": ", newModel.(*Model).statusMsg) // Check status msg shows prompt
			},
		},
		{
			name: "cursor movement down",
			key:  "j",
			setupModel: func() *Model {
				m := New("test")
				m.mode = types.Normal

				// Create mock items
				items := []list.Item{
					Item{entry: types.FileEntry{Name: "1"}},
					Item{entry: types.FileEntry{Name: "2"}},
				}
				m.list.SetItems(items)
				return m
			},
			expectedState: func(t *testing.T, m *Model) {
				newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
				assert.Equal(t, 1, newModel.(*Model).Cursor())
			},
		},
		{
			name: "cursor movement up",
			key:  "k",
			setupModel: func() *Model {
				m := New("test")
				m.mode = types.Normal

				// Create mock items and set cursor position
				items := []list.Item{
					Item{entry: types.FileEntry{Name: "1"}},
					Item{entry: types.FileEntry{Name: "2"}},
				}
				m.list.SetItems(items)
				m.list.Select(1) // Select item at index 1
				return m
			},
			expectedState: func(t *testing.T, m *Model) {
				newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
				assert.Equal(t, 0, newModel.(*Model).Cursor())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := tt.setupModel()
			tt.expectedState(t, model)
		})
	}
}

func TestModelFileOperations(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "sortd-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create some test files
	testFiles := []string{
		"test1.txt",
		"test2.txt",
		"test3.jpg",
	}

	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		err := os.WriteFile(path, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	// Initialize model with test directory
	m := New("test")
	m.SetCurrentDir(tmpDir)
	require.NoError(t, m.ScanDirectory())

	// Test file scanning
	assert.Equal(t, len(testFiles), len(m.Files()))
	for _, f := range testFiles {
		found := false
		for _, file := range m.Files() {
			if file.Name() == f {
				found = true
				break
			}
		}
		assert.True(t, found, "File %s should be found", f)
	}

	// Test file selection
	// Find the index of the first test file
	selectIndex := -1
	for i, item := range m.list.Items() {
		if item.(Item).entry.Name == testFiles[0] {
			selectIndex = i
			break
		}
	}
	require.NotEqual(t, -1, selectIndex, "Test file '%s' not found in list", testFiles[0])
	m.list.Select(selectIndex) // Focus the item
	m.ToggleSelection()         // Select using the actual method
	assert.True(t, m.IsSelected(testFiles[0]))
	assert.False(t, m.IsSelected(testFiles[1]))

	// Test cursor movement
	m.SetCursor(1)
	assert.Equal(t, 1, m.Cursor())
	assert.Equal(t, testFiles[1], m.CurrentFile())
}

func TestModelCommandExecution(t *testing.T) {
	t.Run("enter_command_mode", func(t *testing.T) {
		m := New("test")
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
		require.NotNil(t, cmd)
		newModel := model.(*Model)
		assert.Equal(t, types.Command, newModel.mode)
		assert.Equal(t, ": ", newModel.statusMsg) // Check status msg shows prompt
	})

	t.Run("execute_quit_command", func(t *testing.T) {
		m := New("test")
		m.mode = types.Command
		m.textInput.SetValue(":quit") // Set value in text input
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, cmd)
		_, isQuit := cmd().(tea.QuitMsg)
		assert.True(t, isQuit)
		assert.Equal(t, types.Normal, model.(*Model).mode)
	})

	t.Run("execute_help_command", func(t *testing.T) {
		m := New("test")
		m.mode = types.Command
		m.textInput.SetValue(":help") // Set value in text input
		m.showFullHelp = false
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.Nil(t, cmd)
		assert.True(t, model.(*Model).ShowHelp())
	})
}

func TestModelVisualMode(t *testing.T) {
	m := New("test")

	// Create test items
	items := []list.Item{
		Item{entry: types.FileEntry{Name: "1.txt", Path: "1.txt"}},
		Item{entry: types.FileEntry{Name: "2.txt", Path: "2.txt"}},
		Item{entry: types.FileEntry{Name: "3.txt", Path: "3.txt"}},
	}
	m.list.SetItems(items)

	// Enter visual mode
	m.visualMode = true
	m.visualStart = 0
	m.visualEnd = 2
	m.UpdateVisualSelection()

	// Verify selection
	assert.True(t, m.IsSelected("1.txt"))
	assert.True(t, m.IsSelected("2.txt"))
	assert.True(t, m.IsSelected("3.txt"))
}

func TestModel_Update(t *testing.T) {
	t.Run("quit_key_should_trigger_quit_command", func(t *testing.T) {
		// Initialize a model in normal mode
		m := New("test")
		tmpDir := t.TempDir()
		m.SetCurrentDir(tmpDir)

		// Add at least one item to prevent nil pointer dereference
		m.list.SetItems([]list.Item{
			Item{entry: types.FileEntry{Name: "test.txt", Path: filepath.Join(tmpDir, "test.txt")}},
		})

		// Send 'q' key message
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

		// Should generate a quit command
		require.NotNil(t, cmd, "Pressing 'q' should generate a command")
		result := cmd()
		_, isQuit := result.(tea.QuitMsg)
		assert.True(t, isQuit, "Pressing 'q' should generate a tea.QuitMsg")
	})
}

func TestModel_View(t *testing.T) {
	tests := []struct {
		name        string
		setupModel  func() *Model
		contains    string
		notContains string
	}{
		{
			name: "empty directory",
			setupModel: func() *Model {
				m := New("test")
				m.mode = types.Normal
				m.showFullHelp = false

				// Create a temp directory and set it
				tmpDir := t.TempDir()
				m.SetCurrentDir(tmpDir)
				_ = m.ScanDirectory() // Scan the empty directory

				return m
			},
			contains:    "No files or directories found.",
			notContains: "file1.txt",
		},
		{
			name: "with files",
			setupModel: func() *Model {
				m := New("test")
				m.mode = types.Normal
				m.showFullHelp = false

				// Create a temp directory with files
				tmpDir := t.TempDir()
				testFile := filepath.Join(tmpDir, "file1.txt")
				require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

				m.SetCurrentDir(tmpDir)
				require.NoError(t, m.ScanDirectory())

				return m
			},
			contains:    "Location",
			notContains: "No files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := tt.setupModel()
			got := model.View()

			if tt.contains != "" {
				assert.Contains(t, got, tt.contains)
			}

			if tt.notContains != "" {
				assert.NotContains(t, got, tt.notContains)
			}
		})
	}
}

func TestModel_DirectoryNavigation(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "test.txt"), []byte("test"), 0644))

	m := New("test")
	m.SetCurrentDir(tmpDir)
	require.NoError(t, m.ScanDirectory())

	// Verify we found the subdir
	found := false
	for _, file := range m.Files() {
		if file.Name() == "subdir" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find the subdirectory")

	// Test entering directory
	m.SetCurrentDir(subDir)
	require.NoError(t, m.ScanDirectory())

	// Should find the test.txt file
	found = false
	for _, file := range m.Files() {
		if file.Name() == "test.txt" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find test.txt in subdir")
}

func TestToggleViewKey(t *testing.T) {
	m := New("test")

	// Check that the ToggleView key binding is properly set up
	// TODO: Verify how ToggleView key is accessed (m.keyMap.ToggleView?)
	// assert.Equal(t, "tab", m.keys.ToggleView.Keys()[0])

	// Verify it's included in help
	shortHelp := m.list.ShortHelp() // Try accessing ShortHelp via the list component
	found := false
	for _, binding := range shortHelp {
		if len(binding.Keys()) > 0 && binding.Keys()[0] == "tab" { // Check first key rune
			found = true
			break
		}
	}
	assert.True(t, found, "ToggleView should be included in short help")
}

func TestFilePickerSetup(t *testing.T) {
	t.Skip("filePicker not implemented")
}

func TestToggleViewBetweenListAndTree(t *testing.T) {
	m := New("test")

	// Initialize with a real directory
	tmpDir := t.TempDir()
	m.SetCurrentDir(tmpDir)

	// Initially should be in list view
	assert.Equal(t, types.ViewList, m.activeView, "Initial view should be ViewList")

	// Send tab key to toggle
	keyMsg := tea.KeyMsg{Type: tea.KeyTab}
	newModel, _ := m.Update(keyMsg)

	// Should be in file tree view
	updatedModel := newModel.(*Model)
	assert.Equal(t, types.ViewTree, updatedModel.activeView, "View should be toggled to ViewTree")
}

func TestFilePickerRelatedFunctions(t *testing.T) {
	t.Skip("Tests that reference filePicker are not applicable")
}

func TestWindowSizeMsgHandling(t *testing.T) {
	t.Skip("This test references filePicker which isn't implemented")
}

func TestModelKeyBindings(t *testing.T) {
	m := New("test")

	// Test normal mode keys
	assert.NotEmpty(t, m.Keys(), "Keys should not be empty in Normal mode")

	m.mode = types.Visual
	assert.NotEmpty(t, m.Keys(), "Keys should not be empty in Visual mode")

	m.mode = types.Command
	assert.NotEmpty(t, m.Keys(), "Keys should not be empty in Command mode")

	// Add more specific key binding checks if needed
}
