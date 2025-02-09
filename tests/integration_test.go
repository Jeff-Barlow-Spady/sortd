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
	testDir := t.TempDir()
	createTestFiles(t, testDir)

	t.Run("basic navigation", func(t *testing.T) {
		cmd := exec.Command(binPath)
		cmd.Dir = testDir
		cmd.Env = append(os.Environ(),
			"TESTMODE=true",
			"HEADLESS=true",
			"CI=true",
		)

		// Use stdbuf to disable output buffering
		cmd = exec.Command("stdbuf", "-o0", binPath)
		cmd.Dir = testDir
		stdin, err := cmd.StdinPipe()
		require.NoError(t, err)
		stdout, err := cmd.StdoutPipe()
		require.NoError(t, err)

		err = cmd.Start()
		require.NoError(t, err)

		// Increase timeouts
		time.Sleep(2 * time.Second)
		fmt.Fprintln(stdin, "j")
		time.Sleep(1 * time.Second)
		fmt.Fprintln(stdin, "q")

		output, err := io.ReadAll(stdout)
		require.NoError(t, err)

		cleanOutput := stripANSI(string(output))
		assert.Contains(t, cleanOutput, "file1.txt")
		assert.Contains(t, cleanOutput, "file2.txt")
	})
}

func TestBinaryFileSelection(t *testing.T) {
	binPath := buildBinary(t)
	testDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "test1.txt"), []byte("test1"), 0644))

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
		cmd.Env = append(os.Environ(), "TESTMODE=true")
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

		assert.Contains(t, string(output), "Quick Start")
	})
}
