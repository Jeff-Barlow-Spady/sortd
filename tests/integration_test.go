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

	// Create test directory with files
	testDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "test1.txt"), []byte("test1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "test2.txt"), []byte("test2"), 0644))

	t.Run("basic navigation", func(t *testing.T) {
		cmd := exec.Command(binPath)
		cmd.Dir = testDir

		stdin, err := cmd.StdinPipe()
		require.NoError(t, err)

		stdout, err := cmd.StdoutPipe()
		require.NoError(t, err)

		err = cmd.Start()
		require.NoError(t, err)

		// Wait for UI to initialize
		time.Sleep(100 * time.Millisecond)

		// Test navigation
		fmt.Fprintln(stdin, "j") // down
		time.Sleep(50 * time.Millisecond)
		fmt.Fprintln(stdin, "k") // up
		time.Sleep(50 * time.Millisecond)
		fmt.Fprintln(stdin, "q") // quit

		output, err := io.ReadAll(stdout)
		require.NoError(t, err)

		outStr := string(output)
		assert.Contains(t, outStr, "test1.txt")
		assert.Contains(t, outStr, "test2.txt")
	})
}

func TestBinaryFileSelection(t *testing.T) {
	binPath := buildBinary(t)
	testDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "test1.txt"), []byte("test1"), 0644))

	t.Run("select file", func(t *testing.T) {
		cmd := exec.Command(binPath)
		cmd.Dir = testDir

		stdin, err := cmd.StdinPipe()
		require.NoError(t, err)

		stdout, err := cmd.StdoutPipe()
		require.NoError(t, err)

		err = cmd.Start()
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		// Select file
		fmt.Fprintln(stdin, " ") // space to select
		time.Sleep(50 * time.Millisecond)
		fmt.Fprintln(stdin, "q") // quit

		output, err := io.ReadAll(stdout)
		require.NoError(t, err)

		// Selected files should be highlighted
		assert.Contains(t, string(output), Styles.Selected.Render("test1.txt"))
	})
}

func TestBinarySetup(t *testing.T) {
	binPath := buildBinary(t)

	t.Run("quickstart flow", func(t *testing.T) {
		cmd := exec.Command(binPath)
		stdin, err := cmd.StdinPipe()
		require.NoError(t, err)

		stdout, err := cmd.StdoutPipe()
		require.NoError(t, err)

		err = cmd.Start()
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		// Test quickstart
		fmt.Fprintln(stdin, "1") // quickstart option
		time.Sleep(50 * time.Millisecond)
		fmt.Fprintln(stdin, "q") // quit

		output, err := io.ReadAll(stdout)
		require.NoError(t, err)

		assert.Contains(t, string(output), "Quick Start")
	})
}
