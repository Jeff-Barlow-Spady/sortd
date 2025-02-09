package tests

import (
	"os"
	"path/filepath"
	"testing"

	"sortd/internal/tui"
	"sortd/internal/tui/common"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelEdgeCases(t *testing.T) {
	t.Run("empty_directory", func(t *testing.T) {
		m := tui.New()
		tmpDir := t.TempDir()
		m.SetCurrentDir(tmpDir)

		err := m.ScanDirectory()
		require.NoError(t, err)
		assert.Empty(t, m.Files())

		// Navigation in empty directory should not panic
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		require.Nil(t, cmd)
		assert.Equal(t, 0, model.(*tui.Model).Cursor())
	})

	t.Run("nonexistent_directory", func(t *testing.T) {
		m := tui.New()
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

		m := tui.New()
		m.SetCurrentDir(tmpDir)

		err := m.ScanDirectory()
		assert.Error(t, err)
	})

	t.Run("cursor_bounds", func(t *testing.T) {
		m := tui.New()
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
		m := tui.New()
		assert.False(t, m.IsSelected("nonexistent.txt"))
	})

	t.Run("directory_navigation_errors", func(t *testing.T) {
		m := tui.New()
		tmpDir := t.TempDir()
		m.SetCurrentDir(tmpDir)

		// Create a file that looks like a directory
		fakeDir := filepath.Join(tmpDir, "fake_dir")
		require.NoError(t, os.WriteFile(fakeDir, []byte("not a directory"), 0644))

		files := m.Files()
		files = append(files, common.FileEntry{Name: "fake_dir", Path: fakeDir})
		m.IsSelected("fake_dir")

		// Attempt to enter the fake directory
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.Nil(t, cmd)
		assert.Equal(t, tmpDir, model.(*tui.Model).CurrentDir()) // Should not change directory
	})
}

func TestModelStateConsistency(t *testing.T) {
	t.Run("selection_persistence", func(t *testing.T) {
		m := tui.New()
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644))

		m.SetCurrentDir(tmpDir)
		require.NoError(t, m.ScanDirectory())

		// Select a file
		m.IsSelected("test.txt")

		// Navigate
		model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		assert.True(t, model.(*tui.Model).IsSelected("test.txt"))
	})

	t.Run("cursor_file_sync", func(t *testing.T) {
		m := tui.New()
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644))

		m.SetCurrentDir(tmpDir)
		require.NoError(t, m.ScanDirectory())

		// Verify cursor and current file stay in sync
		m.SetCursor(0)
		assert.Equal(t, "test.txt", m.CurrentFile())
	})
}
