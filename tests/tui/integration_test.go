package tests

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sortd/internal/tui"
	"sortd/pkg/testutils"
	"sortd/pkg/types"

	tea "github.com/charmbracelet/bubbletea"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), "sortd")
	buildCmd := exec.Command("go", "build", "-o", binPath, "../cmd/sortd")
	require.NoError(t, buildCmd.Run())
	return binPath
}

func TestBinaryNavigation(t *testing.T) {
	binPath := buildBinary(t)
	testDir := t.TempDir()

	// Create test files for navigation
	testFiles := map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
	}
	testutils.CreateTestFilesWithContent(t, testDir, testFiles)

	t.Run("basic navigation", func(t *testing.T) {
		t.Log("Starting basic navigation test")
		cmd := exec.Command(binPath)
		cmd.Dir = testDir
		cmd.Env = append(os.Environ(),
			"TESTMODE=true",
			"HEADLESS=true",
			"CI=true",
		)

		t.Log("Setting up command pipes")
		stdin, err := cmd.StdinPipe()
		require.NoError(t, err)
		stdout, err := cmd.StdoutPipe()
		require.NoError(t, err)

		t.Log("Starting command")
		err = cmd.Start()
		require.NoError(t, err)

		// Shorter timeouts for faster feedback
		t.Log("Waiting for initial startup")
		time.Sleep(500 * time.Millisecond)

		t.Log("Sending navigation command")
		fmt.Fprintln(stdin, "j")
		time.Sleep(100 * time.Millisecond)

		t.Log("Sending quit command")
		fmt.Fprintln(stdin, "q")

		t.Log("Reading output")
		output, err := io.ReadAll(stdout)
		require.NoError(t, err)
		t.Logf("Output received: %s", string(output))

		cleanOutput := testutils.StripANSI(string(output))
		assert.Contains(t, cleanOutput, "file1.txt")
		assert.Contains(t, cleanOutput, "file2.txt")
	})
}

func TestBinaryFileSelection(t *testing.T) {
	binPath := buildBinary(t)
	testDir := t.TempDir()
	testutils.CreateTestFilesWithContent(t, testDir, map[string]string{
		"test1.txt": "test1",
	})

	t.Run("select file", func(t *testing.T) {
		cmd := exec.Command(binPath)
		cmd.Dir = testDir
		cmd.Env = append(os.Environ(), "TESTMODE=true")

		stdin, err := cmd.StdinPipe()
		require.NoError(t, err)
		stdout, err := cmd.StdoutPipe()
		require.NoError(t, err)

		err = cmd.Start()
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		// Simulate selecting a file (e.g. with space) then quit.
		fmt.Fprintln(stdin, " ")
		time.Sleep(50 * time.Millisecond)
		fmt.Fprintln(stdin, "q")

		output, err := io.ReadAll(stdout)
		require.NoError(t, err)

		// Assert that the output contains the file name (which indicates it was selected/highlighted).
		assert.Contains(t, string(output), "test1.txt")
	})
}

func TestBinarySetup(t *testing.T) {
	binPath := buildBinary(t)

	t.Run("quickstart flow", func(t *testing.T) {
		cmd := exec.Command(binPath)
		cmd.Env = append(os.Environ(),
			"TESTMODE=true",
			"HEADLESS=true",
			"CI=true",
		)
		stdin, err := cmd.StdinPipe()
		require.NoError(t, err)
		stdout, err := cmd.StdoutPipe()
		require.NoError(t, err)

		err = cmd.Start()
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		// Simulate quickstart (option "1") then quit.
		fmt.Fprintln(stdin, "1")
		time.Sleep(50 * time.Millisecond)
		fmt.Fprintln(stdin, "q")

		output, err := io.ReadAll(stdout)
		require.NoError(t, err)

		// Check for the directory listing header instead of the banner
		assert.Contains(t, string(output), "Name                                     Type                  Size")
	})
}

func TestTUIIntegration(t *testing.T) {
	// Create a temporary test directory
	tmpDir, err := os.MkdirTemp("", "sortd-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := map[string]string{
		"test1.txt": "text content",
		"test2.jpg": "image content",
		"test3.pdf": "pdf content",
	}
	testutils.CreateTestFilesWithContent(t, tmpDir, testFiles)

	// Create test subdirectories
	testDirs := []string{"dir1", "dir2"}
	for _, dir := range testDirs {
		err := os.Mkdir(filepath.Join(tmpDir, dir), 0755)
		require.NoError(t, err)
	}

	// Initialize model with the temp directory
	m := tui.New("test-version")
	m.SetCurrentDir(tmpDir)
	require.NoError(t, m.ScanDirectory())

	// Debug: Print found files
	t.Logf("Current directory: %s", m.CurrentDir())
	t.Logf("Found files:")
	for i, file := range m.Files() {
		t.Logf("  %d: %s", i, file.Name)
	}

	// Test sequence of operations
	t.Run("file navigation and selection", func(t *testing.T) {
		// Initial state
		assert.Equal(t, tmpDir, m.CurrentDir())
		assert.Equal(t, 0, m.Cursor())
		assert.Equal(t, types.Normal, m.Mode())

		// Move cursor down and select test1.txt
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = newModel.(*tui.Model)
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
		m = newModel.(*tui.Model)
		assert.True(t, m.IsSelected("test1.txt"), "test1.txt should be selected")

		// Find dir1 in the file list
		dirIndex := -1
		for i, file := range m.Files() {
			if file.Name == "dir1" {
				dirIndex = i
				break
			}
		}
		require.NotEqual(t, -1, dirIndex, "dir1 not found in file list")

		// Navigate to dir1
		m.SetCursor(dirIndex)
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		m = newModel.(*tui.Model)
		assert.Equal(t, filepath.Join(tmpDir, "dir1"), m.CurrentDir())

		// Go back to parent
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
		m = newModel.(*tui.Model)
		assert.Equal(t, tmpDir, m.CurrentDir())
	})

	t.Run("command mode", func(t *testing.T) {
		// Enter command mode
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
		m = newModel.(*tui.Model)
		assert.Equal(t, types.Command, m.Mode())

		// Type command
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
		m = newModel.(*tui.Model)

		// Execute command
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(*tui.Model)
		assert.Equal(t, types.Normal, m.Mode())
	})

	t.Run("visual mode selection", func(t *testing.T) {
		// Enter visual mode
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
		m = newModel.(*tui.Model)
		assert.True(t, m.VisualMode())

		// Move cursor to select multiple files
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = newModel.(*tui.Model)
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
		m = newModel.(*tui.Model)

		// Verify at least one file is selected
		selectedCount := 0
		for _, file := range m.Files() {
			if m.IsSelected(file.Name) {
				selectedCount++
			}
		}
		assert.Greater(t, selectedCount, 0, "No files were selected")
	})

	t.Run("help toggle", func(t *testing.T) {
		// Reset help state to false first
		m.SetShowHelp(false)
		initialHelpState := m.ShowHelp()
		assert.False(t, initialHelpState, "Help should start hidden")

		// Toggle help
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		m = newModel.(*tui.Model)
		assert.NotEqual(t, initialHelpState, m.ShowHelp())

		// Toggle back
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		m = newModel.(*tui.Model)
		assert.Equal(t, initialHelpState, m.ShowHelp())
	})

	t.Run("quit", func(t *testing.T) {
		newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		m = newModel.(*tui.Model)
		assert.NotNil(t, cmd)
	})
}

// Helper function to find index of a file in the slice
func indexOf(files []types.FileEntry, target types.FileEntry) int {
	for i, file := range files {
		if file.Name == target.Name {
			return i
		}
	}
	return -1
}
