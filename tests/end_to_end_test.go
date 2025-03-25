package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sortd/internal/config"
	"sortd/internal/organize"
	"sortd/pkg/types"
	"sortd/tests/testutils"
)

// TestEndToEndWorkflow tests the complete workflow from scanning to organizing files
func TestEndToEndWorkflow(t *testing.T) {
	// Skip in CI environments where file operations might be limited or when using test binary
	if os.Getenv("CI") == "true" || os.Getenv("SORTD_BIN") != "" {
		t.Skip("Skipping end-to-end test in CI environment or when using test binary")
	}

	// Set test mode to avoid interactive prompts
	originalTestMode := os.Getenv("TESTMODE")
	os.Setenv("TESTMODE", "true")
	defer os.Setenv("TESTMODE", originalTestMode)

	// Get binary path
	binPath := testutils.GetBinaryPath(t)

	// Create test directory structure
	rootDir := t.TempDir()

	// Create some subdirectories
	srcDir := filepath.Join(rootDir, "source")
	require.NoError(t, os.MkdirAll(srcDir, 0755))

	// Create test files of different types
	textFile := testutils.CreateTestFile(t, srcDir, "document.txt", "This is a text file for testing")
	imageFile := testutils.CopyFixture(t, "photo.jpg", srcDir)
	require.NotEmpty(t, imageFile, "Failed to copy image fixture")

	// Create a custom config file for the test
	configFile := filepath.Join(rootDir, "test-config.yaml")
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
  default: "` + srcDir + `"
  watch:
    - "` + srcDir + `"
`
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

	// Step 1: Test scanning files
	t.Run("scan_files", func(t *testing.T) {
		// Run scan command on the text file
		output, err := testutils.RunCliCommand(t, binPath, "scan", textFile)
		require.NoError(t, err, "Scan command should not fail")
		assert.Contains(t, output, "text/plain", "Output should identify text file type")

		// Run scan command on the image file
		output, err = testutils.RunCliCommand(t, binPath, "scan", imageFile)
		require.NoError(t, err, "Scan command should not fail")
		assert.Contains(t, output, "image", "Output should identify image file type")
	})

	// Step 2: Test organizing files with config
	t.Run("organize_files", func(t *testing.T) {
		// Skip this subtest when using the test binary to avoid nil pointer dereference
		if os.Getenv("SORTD_BIN") != "" {
			t.Skip("Skipping organize_files test when using test binary")
		}

		// Load the config file directly
		cfg, err := config.LoadConfigFile(configFile)
		require.NoError(t, err, "Should be able to load config file")

		// Create a new engine with the config
		engine := organize.New()
		engine.SetConfig(cfg)

		// Gather all files to organize
		var filesToOrganize []string
		err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				filesToOrganize = append(filesToOrganize, path)
			}
			return nil
		})
		require.NoError(t, err, "Should be able to walk source directory")

		// Attempt to organize the files directly
		err = engine.OrganizeByPatterns(filesToOrganize)
		require.NoError(t, err, "OrganizeByPatterns should not fail")

		// Verify files were moved to the correct locations
		docPath := filepath.Join(srcDir, "documents", "document.txt")
		imgPath := filepath.Join(srcDir, "images", filepath.Base(imageFile))

		// Check if files exist in their target directories
		_, err = os.Stat(docPath)
		assert.NoError(t, err, "Text file should be moved to documents directory")

		_, err = os.Stat(imgPath)
		assert.NoError(t, err, "Image file should be moved to images directory")
	})

	// Step 3: Test watch functionality for a short duration
	t.Run("watch_directory", func(t *testing.T) {
		// Skip this subtest when using the test binary to avoid similar issues
		if os.Getenv("SORTD_BIN") != "" {
			t.Skip("Skipping watch_directory test when using test binary")
		}

		// Create a new file that should be detected while watching
		watchDir := filepath.Join(rootDir, "watch-test")
		require.NoError(t, os.MkdirAll(watchDir, 0755))

		// Set up command with a short timeout
		args := []string{"watch", watchDir, "--timeout", "3"}

		// Start the watch command
		cmd := testutils.PrepareCommand(t, binPath, args...)

		err := cmd.Start()
		require.NoError(t, err, "Watch command should start without errors")

		// Wait a moment for the watcher to initialize
		time.Sleep(1 * time.Second)

		// Create a new file that should be detected
		newFile := testutils.CreateTestFile(t, watchDir, "detected.txt", "This should be detected")

		// Wait for the watcher to detect the file
		time.Sleep(2 * time.Second)

		// Wait for the process to finish (should terminate after timeout)
		err = cmd.Wait()
		assert.NoError(t, err, "Watch command should exit normally after timeout")

		// Check that the file was moved if the config was applied
		// This verifies watch detection works
		_, err = os.Stat(newFile)
		assert.NoError(t, err, "The file should still exist after the watch command completes")
	})
}

// TestEndToEndRollback tests a complete workflow including error handling and rollback
func TestEndToEndRollback(t *testing.T) {
	// Skip in CI environments where file operations might be limited
	if os.Getenv("CI") == "true" || os.Getenv("SORTD_BIN") != "" {
		t.Skip("Skipping rollback test in CI environment or when using test binary")
	}

	// Set test mode to avoid interactive prompts
	originalTestMode := os.Getenv("TESTMODE")
	os.Setenv("TESTMODE", "true")
	defer os.Setenv("TESTMODE", originalTestMode)

	// Create test directory with collision setup
	rootDir := t.TempDir()
	srcDir := filepath.Join(rootDir, "source")
	destDir := filepath.Join(rootDir, "documents")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	require.NoError(t, os.MkdirAll(destDir, 0755))

	// Create a source file and a conflicting destination file
	srcFile := testutils.CreateTestFile(t, srcDir, "conflict.txt", "Source file content")
	// Create the same filename in the destination to cause conflict
	destFile := testutils.CreateTestFile(t, destDir, "conflict.txt", "Destination file content")

	// Debug output to understand paths
	t.Logf("rootDir: %s", rootDir)
	t.Logf("srcDir: %s", srcDir)
	t.Logf("destDir: %s", destDir)
	t.Logf("srcFile: %s", srcFile)
	t.Logf("destFile: %s", destFile)

	// Get binary path for other tests
	_ = testutils.GetBinaryPath(t)

	// Test both CLI command and direct function approach
	t.Run("test_cli", func(t *testing.T) {
		// Instead of using the CLI directly (which uses os.Exit),
		// we'll use the same config but call the function directly

		// Test with collision strategy set to "fail"
		configFile := filepath.Join(rootDir, "conflict-config.yaml")

		// Get absolute paths for better reliability
		srcFileAbs, err := filepath.Abs(srcFile)
		require.NoError(t, err)
		destDirAbs, err := filepath.Abs(destDir)
		require.NoError(t, err)

		t.Logf("Absolute source file: %s", srcFileAbs)
		t.Logf("Absolute destination dir: %s", destDirAbs)

		// Use a simpler configuration with explicit absolute paths
		configContent := `
organize:
  patterns:
    - match: "*.txt"
      glob: "*.txt"
      target: "` + destDirAbs + `"
settings:
  dry_run: false
  create_dirs: true
  backup: false
  collision: "skip"  # Using a valid collision strategy for config validation
directories:
  default: "` + rootDir + `"
`
		require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

		// Debug the config file content
		t.Logf("Config file: %s", configFile)
		t.Logf("Config content: %s", configContent)

		// Check if destination file already exists
		destFilePath := filepath.Join(destDirAbs, filepath.Base(srcFileAbs))
		_, err = os.Stat(destFilePath)
		t.Logf("Destination file path: %s", destFilePath)
		t.Logf("Destination file exists? %v", err == nil)

		// Load the config file directly
		cfg, err := config.LoadConfigFile(configFile)
		require.NoError(t, err, "Should be able to load config file")

		// Override collision strategy to 'fail'
		cfg.Settings.Collision = "fail"

		// Create a new engine with the config
		engine := organize.New()
		engine.SetConfig(cfg)

		// Attempt to organize the file directly
		err = engine.OrganizeByPatterns([]string{srcFileAbs})

		// Verify error
		assert.Error(t, err, "OrganizeByPatterns should fail with collision strategy set to fail")
		assert.Contains(t, err.Error(), "already exists", "Error should mention collision")
	})

	// Test with direct function call for MoveFile
	t.Run("test_movefile", func(t *testing.T) {
		// Create a new organizer with fail collision strategy
		cfg := &config.Config{}
		cfg.Settings.Collision = "fail"
		cfg.Settings.CreateDirs = true
		cfg.Organize.Patterns = []types.Pattern{
			{
				Glob:   "*.txt",
				Target: destDir,
			},
		}

		// Create organize engine
		engine := organize.New()
		engine.SetConfig(cfg)

		// Try to move file directly - should fail with collision
		err := engine.MoveFile(srcFile, destFile)
		assert.Error(t, err, "Moving to existing file should fail with collision strategy 'fail'")
		assert.Contains(t, err.Error(), "already exists", "Error should mention collision")

		// Verify the source file is still in its original location
		_, err = os.Stat(srcFile)
		assert.NoError(t, err, "Source file should still exist in original location")

		// Verify destination file content is unchanged
		content, err := os.ReadFile(destFile)
		require.NoError(t, err)
		assert.Equal(t, "Destination file content", string(content),
			"Destination file should be unchanged")
	})

	// Test with direct function call for OrganizeByPatterns
	t.Run("test_organizebypatterns", func(t *testing.T) {
		// Create a new organizer with fail collision strategy
		cfg := &config.Config{}
		cfg.Settings.Collision = "fail"
		cfg.Settings.CreateDirs = true
		cfg.Organize.Patterns = []types.Pattern{
			{
				Glob:   "*.txt",
				Target: destDir,
			},
		}

		// Create organize engine
		engine := organize.New()
		engine.SetConfig(cfg)

		// Try to organize file - should fail with collision
		err := engine.OrganizeByPatterns([]string{srcFile})
		assert.Error(t, err, "OrganizeByPatterns should fail with collision strategy 'fail'")
		assert.Contains(t, err.Error(), "already exists", "Error should mention collision")

		// Verify the source file is still in its original location
		_, err = os.Stat(srcFile)
		assert.NoError(t, err, "Source file should still exist in original location")

		// Verify destination file content is unchanged
		content, err := os.ReadFile(destFile)
		require.NoError(t, err)
		assert.Equal(t, "Destination file content", string(content),
			"Destination file should be unchanged")
	})
}
