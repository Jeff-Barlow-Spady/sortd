package testutils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// FixturesDir is the path to the test fixtures directory
const FixturesDir = "fixtures"

// GetBinaryPath returns the path to the sortd binary.
// It checks for the binary in these locations:
// 1. SORTD_BIN environment variable
// 2. In the root directory
// 3. In the PATH
// 4. Fallback to a default location (../cmd/sortd/sortd) for backwards compatibility
// Only builds from source if no binary is found
func GetBinaryPath(t *testing.T) string {
	t.Helper()

	// Use a static variable to cache the binary path once it's found/built
	// This ensures we only build the binary once per test run
	if cachedBinaryPath != "" {
		return cachedBinaryPath
	}

	// Check for environment variable override (useful for CI/CD)
	if binPath := os.Getenv("SORTD_BIN"); binPath != "" {
		if _, err := os.Stat(binPath); err == nil {
			cachedBinaryPath = binPath
			return binPath
		}
		t.Logf("Warning: SORTD_BIN environment variable set to %s but file not found", binPath)
	}

	// Get the absolute path to the project root
	_, thisFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // Go up three levels from this file

	// Check project root (when built with go build -o sortd ./cmd/sortd)
	rootPath := filepath.Join(projectRoot, "sortd")
	if _, err := os.Stat(rootPath); err == nil {
		// Found an existing binary in the root directory
		cachedBinaryPath = rootPath
		return rootPath
	}

	// Check if it's in the PATH
	if path, err := exec.LookPath("sortd"); err == nil {
		cachedBinaryPath = path
		return path
	}

	// No binary found, need to build one
	t.Log("No existing binary found, building from source")
	binPath := BuildBinary(t)
	cachedBinaryPath = binPath
	return binPath
}

// Cache for the binary path to avoid rebuilding
var cachedBinaryPath string

// CreateTestFile creates a simple test file with some content
func CreateTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err, "Failed to create test file")
	return path
}

// CopyFixture copies a fixture file to the target directory
func CopyFixture(t *testing.T, fixtureName, targetDir string) string {
	t.Helper()

	// Read source file
	sourceFile := filepath.Join(FixturesDir, fixtureName)
	content, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Logf("Warning: Could not read fixture file %s: %v", sourceFile, err)
		return ""
	}

	// Write to target directory
	targetFile := filepath.Join(targetDir, fixtureName)
	err = os.WriteFile(targetFile, content, 0644)
	require.NoError(t, err, "Failed to copy fixture file")

	return targetFile
}

// CopyFixtures copies all fixture files to the target directory
func CopyFixtures(t *testing.T, targetDir string) {
	t.Helper()

	// Get list of files in fixtures directory
	entries, err := os.ReadDir(FixturesDir)
	if err != nil {
		t.Logf("Warning: Could not read fixtures directory: %v", err)
		return
	}

	// Copy each file (skip directories for simplicity)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Read source file
		sourceFile := filepath.Join(FixturesDir, entry.Name())
		content, err := os.ReadFile(sourceFile)
		if err != nil {
			t.Logf("Warning: Could not read fixture file %s: %v", sourceFile, err)
			continue
		}

		// Write to target directory
		targetFile := filepath.Join(targetDir, entry.Name())
		err = os.WriteFile(targetFile, content, 0644)
		require.NoError(t, err, "Failed to copy fixture file")
	}
}

// CreateTestFiles creates a set of test files in the specified directory
func CreateTestFiles(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"test1.txt": "This is a text file",
		"test2.jpg": "Pretend this is an image",
		"test3.pdf": "Pretend this is a PDF",
		"test4.mp3": "Pretend this is audio",
	}

	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))
	}
}

// RunCliCommand runs the sortd binary with the given arguments.
// This is a helper for tests to ensure consistent binary path resolution.
// If binPath is provided, that specific binary is used.
// If binPath is empty, GetBinaryPath() is used to find the binary.
func RunCliCommand(t *testing.T, binPath string, args ...string) (string, error) {
	t.Helper()

	// If no binPath provided, use the helper to find it
	if binPath == "" {
		binPath = GetBinaryPath(t)
	}

	// Create a context with timeout to prevent hanging tests
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()

	// Check if the error was due to context timeout
	if ctx.Err() == context.DeadlineExceeded {
		t.Logf("Command timed out after 30 seconds: %s %v", binPath, args)
		return "Command timed out after 30 seconds", ctx.Err()
	}

	// When there's an error, combine stdout and stderr for easier testing
	if err != nil {
		stderrStr := stderr.String()
		t.Logf("Command stderr: %s", stderrStr)

		// If stdout is empty but stderr has content, return stderr as the output
		if output == "" && stderrStr != "" {
			output = stderrStr
		} else if stderrStr != "" {
			// If both have content, combine them
			output = output + "\n" + stderrStr
		}

		// Handle exit error specifically (in a safe way to avoid panics)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr != nil {
			// Create a more detailed error that includes the exit code and stderr
			return output, fmt.Errorf("command failed with exit code %d", exitErr.ExitCode())
		}
	}

	return output, err
}

// PrepareCommand creates an exec.Cmd for the sortd binary with the given arguments.
// This is useful when you need to control the command execution flow (like pipes or background execution).
// If binPath is provided, that specific binary is used.
// If binPath is empty, GetBinaryPath() is used to find the binary.
func PrepareCommand(t *testing.T, binPath string, args ...string) *exec.Cmd {
	t.Helper()

	// If no binPath provided, use the helper to find it
	if binPath == "" {
		binPath = GetBinaryPath(t)
	}

	// Create a context with timeout to prevent hanging tests
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel) // Ensure cancellation when test completes

	return exec.CommandContext(ctx, binPath, args...)
}

// BuildBinary builds the sortd binary for testing and returns its path.
// It builds the binary in a temporary directory to ensure isolation.
func BuildBinary(t *testing.T) string {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), "sortd")

	// Get the absolute path to the project root
	_, thisFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // Go up three levels from this file
	cmdPath := filepath.Join(projectRoot, "cmd", "sortd")

	t.Logf("Building from source at: %s", cmdPath)
	buildCmd := exec.Command("go", "build", "-o", binPath, cmdPath)

	var stderr bytes.Buffer
	buildCmd.Stderr = &stderr

	err := buildCmd.Run()
	if err != nil {
		t.Logf("Build error: %v", err)
		t.Logf("Build stderr: %s", stderr.String())
		require.NoError(t, err, "Failed to build binary")
	}

	// Ensure the binary is executable
	err = os.Chmod(binPath, 0755)
	if err != nil {
		t.Logf("Failed to set executable permissions: %v", err)
	}

	return binPath
}
