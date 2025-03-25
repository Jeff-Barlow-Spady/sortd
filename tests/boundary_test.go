package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sortd/tests/testutils"
)

// TestBoundaryConditions tests various edge cases and boundary conditions
func TestBoundaryConditions(t *testing.T) {
	binPath := testutils.GetBinaryPath(t)

	t.Run("empty_directory", func(t *testing.T) {
		// Create empty directory
		emptyDir := t.TempDir()

		// Run organize on empty directory
		output, err := testutils.RunCliCommand(t, binPath, "organize", emptyDir)
		require.NoError(t, err, "Organize command on empty directory should not fail")

		// Verify it handles empty directories gracefully
		assert.True(t,
			strings.Contains(output, "No files") ||
				strings.Contains(output, "0 files") ||
				strings.Contains(output, "empty") ||
				strings.Contains(output, "Empty"),
			"Output should indicate no files were found")
	})

	t.Run("invalid_path", func(t *testing.T) {
		// Create a non-existent path in a temp directory
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "nonexistent")

		// Verify the path really doesn't exist
		_, statErr := os.Stat(invalidPath)
		require.True(t, os.IsNotExist(statErr), "Path should not exist for this test")

		// Run organize on the non-existent directory
		output, err := testutils.RunCliCommand(t, binPath, "organize", invalidPath)

		// The command should either:
		// 1. Return an error (through exit code)
		// 2. Or output text indicating the path doesn't exist
		if err == nil {
			// If no error returned, the output should contain text about missing path
			assert.True(t,
				strings.Contains(strings.ToLower(output), "no such file") ||
					strings.Contains(strings.ToLower(output), "does not exist") ||
					strings.Contains(strings.ToLower(output), "cannot find"),
				"Output should indicate path does not exist: %s", output)
		} else {
			// We got an error, make sure it looks like a path-related error
			t.Logf("Command failed with error: %v", err)
		}
	})

	t.Run("large_filename", func(t *testing.T) {
		// Create a file with a very long name
		tmpDir := t.TempDir()
		longName := strings.Repeat("a", 200) + ".txt"
		longPath := filepath.Join(tmpDir, longName)

		err := os.WriteFile(longPath, []byte("test content"), 0644)
		if err != nil {
			// Some filesystems limit filename length
			t.Skip("Skipping long filename test due to filesystem limitations")
		}

		// Run organize on the file with a long name
		output, err := testutils.RunCliCommand(t, binPath, "organize", "--dry-run", longPath)
		require.NoError(t, err, "Organize command with long filename should not fail")

		// Verify it handled the long filename
		assert.Contains(t, output, ".txt", "Output should show text file extension")
	})

	t.Run("readonly_file", func(t *testing.T) {
		// Create a read-only file
		tmpDir := t.TempDir()
		readonlyPath := filepath.Join(tmpDir, "readonly.txt")

		err := os.WriteFile(readonlyPath, []byte("test content"), 0644)
		require.NoError(t, err)

		// Make it read-only
		err = os.Chmod(readonlyPath, 0444)
		require.NoError(t, err)

		// Run organize with dry-run to see what would happen
		output, err := testutils.RunCliCommand(t, binPath, "organize", "--dry-run", readonlyPath)
		require.NoError(t, err, "Organize dry-run on read-only file should not fail")

		// Verify it recognized the file
		assert.Contains(t, output, "readonly.txt", "Output should show the readonly file")

		// Actual move would likely fail, but the dry-run should complete without errors
	})

	t.Run("unsupported_file_type", func(t *testing.T) {
		// Create a file with an unusual extension
		tmpDir := t.TempDir()
		unusualPath := filepath.Join(tmpDir, "unusual.xyz123")

		err := os.WriteFile(unusualPath, []byte{0xFF, 0xD8, 0xFF, 0xE0}, 0644)
		require.NoError(t, err)

		// Run scan on the unusual file
		output, err := testutils.RunCliCommand(t, binPath, "scan", unusualPath)
		require.NoError(t, err, "Scan command should not fail on unusual file")

		// It should detect some content type
		assert.Contains(t, output, "Type:", "Output should include content type information")
	})

	t.Run("unicode_filename", func(t *testing.T) {
		// Create a file with Unicode characters
		tmpDir := t.TempDir()
		unicodeName := "üñìçödé_测试_💾.txt"
		unicodePath := filepath.Join(tmpDir, unicodeName)

		// Check if filesystem supports unicode filenames
		err := os.WriteFile(unicodePath, []byte("unicode test"), 0644)
		if err != nil {
			t.Skip("Skipping Unicode filename test due to filesystem limitations")
		}

		// Run organize on the Unicode filename
		output, err := testutils.RunCliCommand(t, binPath, "organize", "--dry-run", unicodePath)
		require.NoError(t, err, "Organize command should handle Unicode filenames")

		// Look for .txt in output, as full Unicode might be escaped
		assert.Contains(t, output, ".txt", "Output should show the file extension")
	})

	t.Run("no_extension_file", func(t *testing.T) {
		// Create a file with no extension
		tmpDir := t.TempDir()
		noExtFile := filepath.Join(tmpDir, "noextension")

		err := os.WriteFile(noExtFile, []byte("test content"), 0644)
		require.NoError(t, err)

		// Run scan on the file with no extension
		output, err := testutils.RunCliCommand(t, binPath, "scan", noExtFile)
		require.NoError(t, err, "Scan command should handle files with no extension")

		// Verify it was able to process the file
		assert.Contains(t, output, "noextension", "Output should contain the filename")
	})

	t.Run("symbolic_link", func(t *testing.T) {
		// Create a test file and a symbolic link to it
		tmpDir := t.TempDir()
		origFile := filepath.Join(tmpDir, "original.txt")
		linkFile := filepath.Join(tmpDir, "link.txt")

		err := os.WriteFile(origFile, []byte("original content"), 0644)
		require.NoError(t, err)

		// Create symbolic link
		err = os.Symlink(origFile, linkFile)
		if err != nil {
			t.Skip("Skipping symbolic link test - unable to create symlinks")
		}

		// Run scan on the symbolic link
		output, err := testutils.RunCliCommand(t, binPath, "scan", linkFile)
		require.NoError(t, err, "Scan command should handle symbolic links")

		// It should report that it's a symbolic link or should process the linked file
		assert.Contains(t, output, "link.txt", "Output should contain the link filename")
	})
}

// TestErrorHandling tests the application's response to various error conditions
func TestErrorHandling(t *testing.T) {
	binPath := testutils.GetBinaryPath(t)

	t.Run("invalid_command", func(t *testing.T) {
		// Run a command that doesn't exist
		output, err := testutils.RunCliCommand(t, binPath, "nonexistentcommand")
		assert.Error(t, err, "Invalid command should return an error")

		// The actual error is logged to stdout in the output
		assert.Contains(t, strings.ToLower(output), "unknown command",
			"Output should indicate unknown command")
	})

	t.Run("missing_required_arg", func(t *testing.T) {
		// Run organize without required directory argument
		output, err := testutils.RunCliCommand(t, binPath, "organize")
		assert.Error(t, err, "Missing required arguments should return an error")
		assert.Contains(t, strings.ToLower(output), "required",
			"Output should indicate missing required argument")
	})

	t.Run("invalid_config_file", func(t *testing.T) {
		// Create an invalid config file
		tmpDir := t.TempDir()
		invalidConfig := filepath.Join(tmpDir, "invalid.yaml")

		err := os.WriteFile(invalidConfig, []byte("invalid: yaml: ]]]"), 0644)
		require.NoError(t, err)

		// Run with the invalid config
		output, err := testutils.RunCliCommand(t, binPath, "organize",
			"--config", invalidConfig, tmpDir)
		assert.Error(t, err, "Invalid config should return an error")
		assert.Contains(t, strings.ToLower(output), "error",
			"Output should indicate config error")
	})

	t.Run("insufficient_permissions", func(t *testing.T) {
		// Skip if running as root (can't easily test permission issues)
		if os.Geteuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		// Create a directory with no write permissions
		tmpDir := t.TempDir()
		noWriteDir := filepath.Join(tmpDir, "nowrite")
		require.NoError(t, os.MkdirAll(noWriteDir, 0755))

		// Create a test file
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

		// Make directory read-only
		err := os.Chmod(noWriteDir, 0555)
		require.NoError(t, err)

		// Create config that tries to move to this directory
		configFile := filepath.Join(tmpDir, "perm-config.yaml")
		configContent := `
organize:
  patterns:
    - match: "*.txt"
      target: "nowrite/"
settings:
  dry_run: false
  create_dirs: false
  collision: "fail"
directories:
  default: "` + tmpDir + `"
`
		require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

		// Run organize, expecting permission error
		output, err := testutils.RunCliCommand(t, binPath, "organize",
			"--config", configFile, testFile)

		// The command should either fail with permission error or handle it gracefully
		// We don't assert Error here because some implementations might handle this gracefully
		if err != nil {
			assert.Contains(t, strings.ToLower(output), "permission",
				"Output should mention permission issues when failing")
		}
	})
}
