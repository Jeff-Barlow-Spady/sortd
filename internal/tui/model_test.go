package tui

import (
	"os"
	"path/filepath"
	"sortd/internal/organize"
	"sortd/pkg/types"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelInitialization(t *testing.T) {
	m := New()
	assert.NotNil(t, m)
	assert.Equal(t, types.Normal, m.mode)
	assert.NotEmpty(t, m.currentDir)
	assert.NotNil(t, m.selectedFiles)
	assert.NotNil(t, m.analysisEngine)
	assert.NotNil(t, m.organizeEngine)
}

func TestModelEdgeCases(t *testing.T) {
	t.Run("empty_directory", func(t *testing.T) {
		m := New()
		tmpDir := t.TempDir()
		m.SetCurrentDir(tmpDir)

		err := m.ScanDirectory()
		require.NoError(t, err)
		assert.Empty(t, m.Files())

		// Navigation in empty directory should not panic
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		require.Nil(t, cmd)
		assert.Equal(t, 0, model.(*Model).Cursor())
	})

	t.Run("nonexistent_directory", func(t *testing.T) {
		m := New()
		m.SetCurrentDir("/nonexistent/path")

		err := m.ScanDirectory()
		assert.Error(t, err)
		assert.Empty(t, m.Files())
	})

	t.Run("permission_denied", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping test when running as root")
		}

		tmpDir := t.TempDir()
		require.NoError(t, os.Chmod(tmpDir, 0000))
		defer os.Chmod(tmpDir, 0755)

		m := New()
		m.SetCurrentDir(tmpDir)

		err := m.ScanDirectory()
		assert.Error(t, err)
	})

	t.Run("cursor_bounds", func(t *testing.T) {
		m := New()
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
		m := New()
		assert.False(t, m.IsSelected("nonexistent.txt"))
	})

	t.Run("directory_navigation_errors", func(t *testing.T) {
		m := New()
		tmpDir := t.TempDir()
		m.SetCurrentDir(tmpDir)

		// Create a file that looks like a directory
		fakeDir := filepath.Join(tmpDir, "fake_dir")
		require.NoError(t, os.WriteFile(fakeDir, []byte("not a directory"), 0644))

		// Attempt to enter the fake directory
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.Nil(t, cmd)
		assert.Equal(t, tmpDir, model.(*Model).CurrentDir()) // Should not change directory
	})
}

func TestModelStateConsistency(t *testing.T) {
	t.Run("selection_persistence", func(t *testing.T) {
		m := New()
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644))

		m.SetCurrentDir(tmpDir)
		require.NoError(t, m.ScanDirectory())

		// Select a file
		m.SelectFile("test.txt")

		// Navigate
		model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		assert.True(t, model.(*Model).IsSelected("test.txt"))
	})

	t.Run("cursor_file_sync", func(t *testing.T) {
		m := New()
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
		initialState  *Model
		expectedState func(*testing.T, *Model)
	}{
		{
			name: "quit on q",
			key:  "q",
			initialState: &Model{
				mode: types.Normal,
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
			initialState: &Model{
				mode:     types.Normal,
				showHelp: false,
			},
			expectedState: func(t *testing.T, m *Model) {
				newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
				assert.True(t, newModel.(*Model).showHelp)
			},
		},
		{
			name: "enter command mode on :",
			key:  ":",
			initialState: &Model{
				mode: types.Normal,
			},
			expectedState: func(t *testing.T, m *Model) {
				newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
				assert.Equal(t, types.Command, newModel.(*Model).mode)
				assert.Equal(t, ":", newModel.(*Model).commandBuffer)
			},
		},
		{
			name: "cursor movement down",
			key:  "j",
			initialState: &Model{
				mode:   types.Normal,
				cursor: 0,
				files:  []types.FileEntry{{Name: "1"}, {Name: "2"}},
			},
			expectedState: func(t *testing.T, m *Model) {
				newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
				assert.Equal(t, 1, newModel.(*Model).cursor)
			},
		},
		{
			name: "cursor movement up",
			key:  "k",
			initialState: &Model{
				mode:   types.Normal,
				cursor: 1,
				files:  []types.FileEntry{{Name: "1"}, {Name: "2"}},
			},
			expectedState: func(t *testing.T, m *Model) {
				newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
				assert.Equal(t, 0, newModel.(*Model).cursor)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.expectedState(t, tt.initialState)
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
	m := New()
	m.SetCurrentDir(tmpDir)
	require.NoError(t, m.ScanDirectory())

	// Test file scanning
	assert.Equal(t, len(testFiles), len(m.files))
	for _, f := range testFiles {
		found := false
		for _, file := range m.files {
			if file.Name == f {
				found = true
				break
			}
		}
		assert.True(t, found, "File %s should be found", f)
	}

	// Test file selection
	m.selectedFiles[testFiles[0]] = true
	assert.True(t, m.IsSelected(testFiles[0]))
	assert.False(t, m.IsSelected(testFiles[1]))

	// Test cursor movement
	m.SetCursor(1)
	assert.Equal(t, 1, m.cursor)
	assert.Equal(t, testFiles[1], m.currentFile)
}

// Mock organize engine for testing
type mockOrganizeEngine struct {
	*organize.Engine
	organizedFiles []string
	targetDir      string
}

func newMockOrganizeEngine() *organize.Engine {
	mock := &mockOrganizeEngine{
		Engine: organize.New(),
	}
	return mock.Engine
}

func TestModelCommandExecution(t *testing.T) {
	t.Run("enter_command_mode", func(t *testing.T) {
		m := New()
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
		require.Nil(t, cmd)
		newModel := model.(*Model)
		assert.Equal(t, types.Command, newModel.mode)
		assert.Equal(t, ":", newModel.commandBuffer)
	})

	t.Run("execute_quit_command", func(t *testing.T) {
		m := New()
		m.mode = types.Command
		m.commandBuffer = ":quit"
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, cmd)
		_, isQuit := cmd().(tea.QuitMsg)
		assert.True(t, isQuit)
		assert.Equal(t, types.Normal, model.(*Model).mode)
	})

	t.Run("execute_help_command", func(t *testing.T) {
		m := New()
		m.mode = types.Command
		m.commandBuffer = ":help"
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.Nil(t, cmd)
		assert.True(t, model.(*Model).showHelp)
	})
}

func TestModelVisualMode(t *testing.T) {
	m := New()
	m.files = []types.FileEntry{
		{Name: "1.txt"},
		{Name: "2.txt"},
		{Name: "3.txt"},
	}

	// Enter visual mode
	m.visualMode = true
	m.visualStart = 0

	// Select range
	m.cursor = 2
	m.selectedFiles = make(map[string]bool)
	for i := m.visualStart; i <= m.cursor; i++ {
		m.selectedFiles[m.files[i].Name] = true
	}

	// Verify selection
	assert.True(t, m.IsSelected("1.txt"))
	assert.True(t, m.IsSelected("2.txt"))
	assert.True(t, m.IsSelected("3.txt"))
}

func TestModelCopy(t *testing.T) {
	original := New()
	original.files = []types.FileEntry{{Name: "test.txt"}}
	original.selectedFiles["test.txt"] = true
	original.cursor = 1
	original.mode = types.Visual
	original.showHelp = true

	copied := original.copy()

	assert.Equal(t, original.files, copied.files)
	assert.Equal(t, original.selectedFiles, copied.selectedFiles)
	assert.Equal(t, original.cursor, copied.cursor)
	assert.Equal(t, original.mode, copied.mode)
	assert.Equal(t, original.showHelp, copied.showHelp)

	// Verify deep copy by modifying original
	original.files[0].Name = "modified.txt"
	assert.NotEqual(t, original.files[0].Name, copied.files[0].Name)
}

func TestModel_Update(t *testing.T) {
	tests := []struct {
		name     string
		model    *Model
		msg      tea.Msg
		wantMode types.Mode
	}{
		{
			name: "quit from normal mode",
			model: &Model{
				mode: types.Normal,
			},
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")},
			wantMode: types.Normal,
		},
		{
			name: "quit from setup mode",
			model: &Model{
				mode: types.Setup,
			},
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")},
			wantMode: types.Normal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newModel, _ := tt.model.Update(tt.msg)
			m, ok := newModel.(*Model)
			require.True(t, ok)
			assert.Equal(t, tt.wantMode, m.mode)
		})
	}
}

// TestModel_View tests the View method
func TestModel_View(t *testing.T) {
	tests := []struct {
		name    string
		model   *Model
		want    string
		wantErr bool
	}{
		{
			name: "empty directory",
			model: &Model{
				mode:     types.Normal,
				showHelp: false,
				files:    []types.FileEntry{},
			},
			want: "No files to display yet",
		},
		{
			name: "with files",
			model: &Model{
				mode:     types.Normal,
				showHelp: false,
				files: []types.FileEntry{
					{Name: "file1.txt", Path: "file1.txt"},
					{Name: "file2.txt", Path: "file2.txt"},
				},
			},
			want: "file1.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.model.View()
			assert.Contains(t, got, tt.want)
		})
	}
}

func TestModel_FileOperations(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644))

	m := &Model{
		mode:          types.Normal,
		currentDir:    tmpDir,
		selectedFiles: make(map[string]bool),
	}

	require.NoError(t, m.scanDirectory())
	assert.Equal(t, 1, len(m.files))

	// Test file selection
	require.NoError(t, m.SelectFile("test.txt"))
	assert.True(t, m.IsSelected(filepath.Join(tmpDir, "test.txt")))

	// Test cursor movement
	m.MoveCursor(1)
	assert.Equal(t, 0, m.cursor) // Should clamp to max index

	m.MoveCursor(-1)
	assert.Equal(t, 0, m.cursor) // Should clamp to min index
}

func TestModel_DirectoryNavigation(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "test.txt"), []byte("test"), 0644))

	m := &Model{
		mode:       types.Normal,
		currentDir: tmpDir,
	}

	require.NoError(t, m.scanDirectory())
	assert.Equal(t, 1, len(m.files))

	// Test entering directory
	m.SetCurrentDir(subDir)
	require.NoError(t, m.scanDirectory())
	assert.Equal(t, 1, len(m.files))
	assert.Equal(t, "test.txt", m.files[0].Name)
}

func TestModel_VisualMode(t *testing.T) {
	m := &Model{
		mode: types.Normal,
		files: []types.FileEntry{
			{Name: "file1.txt", Path: "file1.txt"},
			{Name: "file2.txt", Path: "file2.txt"},
			{Name: "file3.txt", Path: "file3.txt"},
		},
	}

	// Test entering visual mode
	m.mode = types.Visual
	m.visualStart = 0
	m.visualEnd = 2

	// Test visual selection
	assert.True(t, m.visualMode)
	assert.Equal(t, 0, m.visualStart)
	assert.Equal(t, 2, m.visualEnd)
}
