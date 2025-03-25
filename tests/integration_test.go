package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sortd/tests/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBinaryCommands(t *testing.T) {
	binPath := testutils.GetBinaryPath(t)

	t.Run("help command", func(t *testing.T) {
		output, err := testutils.RunCliCommand(t, binPath, "--help")
		require.NoError(t, err, "Help command should not fail")
		assert.Contains(t, output, "Usage:", "Help output should contain usage information")
		assert.Contains(t, output, "Available Commands:", "Help output should list commands")
	})

	t.Run("version command", func(t *testing.T) {
		output, err := testutils.RunCliCommand(t, binPath, "version")
		require.NoError(t, err, "Version command should not fail")
		assert.Contains(t, output, "sortd version", "Version output should contain version information")
	})
}

func TestOrganizeCommand(t *testing.T) {
	binPath := testutils.GetBinaryPath(t)
	testDir := t.TempDir()
	testutils.CreateTestFiles(t, testDir)

	t.Run("organize with dry-run", func(t *testing.T) {
		// Run organize with dry-run flag
		output, err := testutils.RunCliCommand(t, binPath, "organize", testDir, "--dry-run")
		require.NoError(t, err, "Organize with dry-run should not fail")

		// Check that the output mentions the test files
		assert.Contains(t, output, "test1.txt", "Output should mention text file")

		// Verify files haven't been moved (dry-run)
		_, err = os.Stat(filepath.Join(testDir, "test1.txt"))
		assert.NoError(t, err, "File should still exist in original location after dry-run")
	})

	t.Run("organize with fixture files", func(t *testing.T) {
		// Create a directory with copied fixture files
		fixturesDir := filepath.Join(t.TempDir(), "fixtures")
		require.NoError(t, os.MkdirAll(fixturesDir, 0755))
		testutils.CopyFixtures(t, fixturesDir)

		// Run organize with dry-run on fixture files
		output, err := testutils.RunCliCommand(t, binPath, "organize", fixturesDir, "--dry-run")
		require.NoError(t, err, "Organize command with fixtures should not fail")

		// Check for some expected filenames in output
		if _, err := os.Stat(filepath.Join(fixturesDir, "sample.txt")); err == nil {
			assert.Contains(t, output, "sample.txt", "Output should mention sample.txt fixture")
		}
	})
}

func TestSetupCommand(t *testing.T) {
	binPath := testutils.GetBinaryPath(t)

	t.Run("setup dry-run", func(t *testing.T) {
		// Run setup with dry-run flag
		output, err := testutils.RunCliCommand(t, binPath, "setup", "--dry-run")
		require.NoError(t, err, "Setup with dry-run should not fail")

		// Check output for setup-related information
		assert.True(t,
			strings.Contains(output, "configuration") ||
				strings.Contains(output, "setup") ||
				strings.Contains(output, "config"),
			"Output should contain setup-related information")
	})
}

func TestRulesCommand(t *testing.T) {
	binPath := testutils.GetBinaryPath(t)

	t.Run("rules list", func(t *testing.T) {
		// Run rules list command
		output, err := testutils.RunCliCommand(t, binPath, "rules", "list")
		require.NoError(t, err, "Rules list command should not fail")

		// Output should mention rules or indicate there are none
		assert.True(t,
			strings.Contains(output, "rules") ||
				strings.Contains(output, "Rules") ||
				strings.Contains(output, "No rules"),
			"Output should mention rules")
	})
}

func TestCliConfigIntegration(t *testing.T) {
	binPath := testutils.GetBinaryPath(t)
	tmpDir := t.TempDir()

	// Create test config
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
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Create test files
	testutils.CreateTestFiles(t, tmpDir)

	// Create target directories
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "documents"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "images"), 0755))

	t.Run("organize with config", func(t *testing.T) {
		// Run organize with config
		_, err := testutils.RunCliCommand(t, binPath, "organize",
			"--config", configPath,
			filepath.Join(tmpDir, "test1.txt"))

		require.NoError(t, err, "Organize command should not fail")

		// Verify file was moved according to rule
		movedFile := filepath.Join(tmpDir, "documents", "test1.txt")
		_, err = os.Stat(movedFile)
		assert.NoError(t, err, "File should be moved to target directory")

		// Verify original file no longer exists
		_, err = os.Stat(filepath.Join(tmpDir, "test1.txt"))
		assert.True(t, os.IsNotExist(err), "Original file should no longer exist")
	})
}
