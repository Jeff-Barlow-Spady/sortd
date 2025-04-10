package organize

import (
	"os"
	"path/filepath"
	"testing"

	"sortd/internal/config"
	"sortd/internal/errors"
	"sortd/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorHandling tests how the engine handles various error conditions
func TestErrorHandling(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := filepath.Join(os.TempDir(), "sortd-test-engine-errors")
	err := os.MkdirAll(tmpDir, 0755)
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	// Create source and destination directories
	srcDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "destination")

	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err, "Failed to create source directory")

	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err, "Failed to create destination directory")

	// Test handling of non-existent source file
	t.Run("NonExistentSourceFile", func(t *testing.T) {
		// Create an engine
		engine := New()

		// Try to move a non-existent file
		nonExistentFile := filepath.Join(srcDir, "does-not-exist.txt")
		destFile := filepath.Join(destDir, "target.txt")

		err := engine.MoveFile(nonExistentFile, destFile)
		assert.Error(t, err, "MoveFile should return an error for non-existent source")

		// Verify it's a FileError with the expected code
		fileErr, ok := err.(*errors.FileError)
		if assert.True(t, ok, "Error should be a *errors.FileError") {
			assert.Equal(t, errors.FileAccessDenied, fileErr.Kind(), "Error should have FileAccessDenied code")
		}
	})

	// Test handling of invalid source (directory as source)
	t.Run("DirectoryAsSourceFile", func(t *testing.T) {
		// Create an engine
		engine := New()

		// Try to move a directory as a file
		destFile := filepath.Join(destDir, "target-dir.txt")

		err := engine.MoveFile(srcDir, destFile)
		assert.Error(t, err, "MoveFile should return an error when source is a directory")

		// Verify it's a FileError with the expected code
		fileErr, ok := err.(*errors.FileError)
		if assert.True(t, ok, "Error should be a *errors.FileError") {
			assert.Equal(t, errors.InvalidOperation, fileErr.Kind(), "Error should have InvalidOperation code")
		}
	})

	// Test permission denied scenario
	t.Run("PermissionDenied", func(t *testing.T) {
		// Skip test on Windows as permission handling works differently
		if os.Getenv("GOOS") == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		// Create a read-only directory
		readOnlyDir := filepath.Join(tmpDir, "read-only")
		err := os.MkdirAll(readOnlyDir, 0555) // read-only permissions
		require.NoError(t, err, "Failed to create read-only directory")

		// Create source file
		srcFile := filepath.Join(srcDir, "permission-test.txt")
		err = os.WriteFile(srcFile, []byte("test content"), 0644)
		require.NoError(t, err, "Failed to create source file")

		// Destination in read-only directory
		destFile := filepath.Join(readOnlyDir, "cannot-write.txt")

		// Create an engine
		engine := New()

		// Try to move file to read-only directory
		err = engine.MoveFile(srcFile, destFile)
		assert.Error(t, err, "MoveFile should return an error for permission denied")
	})

	// Test collision handling errors
	t.Run("InvalidCollisionStrategy", func(t *testing.T) {
		// Create a file
		srcFile := filepath.Join(srcDir, "collision-test.txt")
		err := os.WriteFile(srcFile, []byte("test content"), 0644)
		require.NoError(t, err, "Failed to create source file")

		// Create destination file
		destFile := filepath.Join(destDir, "collision-test.txt")
		err = os.WriteFile(destFile, []byte("existing content"), 0644)
		require.NoError(t, err, "Failed to create destination file")

		// Create config with invalid collision strategy
		cfg := &config.Config{}
		cfg.Settings.Collision = "invalid-strategy"

		// Create engine with invalid collision strategy
		engine := NewWithConfig(cfg)

		// Try to move file with invalid collision strategy
		err = engine.MoveFile(srcFile, destFile)
		assert.Error(t, err, "MoveFile should return an error for invalid collision strategy")

		// Verify it's a ConfigError with the expected code
		configErr, ok := err.(*errors.ConfigError)
		if assert.True(t, ok, "Error should be a *errors.ConfigError") {
			assert.Equal(t, errors.InvalidConfig, configErr.Kind(), "Error should have InvalidConfig code")
		}
	})

	// Test destination directory creation failure
	t.Run("DirectoryCreationFailed", func(t *testing.T) {
		// Only run this test on non-Windows platforms
		if os.Getenv("GOOS") == "windows" {
			t.Skip("Skipping directory creation test on Windows")
		}

		// Create a file
		srcFile := filepath.Join(srcDir, "dir-test.txt")
		err := os.WriteFile(srcFile, []byte("test content"), 0644)
		require.NoError(t, err, "Failed to create source file")

		// Create a file where a directory should be
		fileInsteadOfDir := filepath.Join(destDir, "not-a-dir")
		err = os.WriteFile(fileInsteadOfDir, []byte("blocking file"), 0644)
		require.NoError(t, err, "Failed to create blocking file")

		// Target would be inside what should be a directory
		destFile := filepath.Join(fileInsteadOfDir, "dir-test.txt")

		// Create engine with createDirs enabled
		cfg := &config.Config{}
		cfg.Settings.CreateDirs = true
		engine := NewWithConfig(cfg)

		// Try to move file
		err = engine.MoveFile(srcFile, destFile)
		assert.Error(t, err, "MoveFile should return an error when directory creation fails")
	})

	// Test error propagation from OrganizeByPatterns
	t.Run("OrganizeByPatternsErrorPropagation", func(t *testing.T) {
		// Create config with patterns
		cfg := &config.Config{}
		cfg.Organize.Patterns = []types.Pattern{
			{Match: "*.txt", Target: filepath.Join(destDir, "documents")},
		}

		// Create engine
		engine := NewWithConfig(cfg)

		// Try to organize non-existent files
		nonExistentFiles := []string{
			filepath.Join(srcDir, "does-not-exist1.txt"),
			filepath.Join(srcDir, "does-not-exist2.txt"),
		}

		// Should return the first error encountered
		err := engine.OrganizeByPatterns(nonExistentFiles)
		assert.Error(t, err, "OrganizeByPatterns should return an error for non-existent files")

		// The error should be wrapped with additional context
		assert.Contains(t, err.Error(), "failed to move", "Error should indicate move failure")
	})

	// Test error handling with destination directory creation
	t.Run("CreateDestDirsFailure", func(t *testing.T) {
		// Create source file
		srcFile := filepath.Join(srcDir, "create-dirs-test.txt")
		err := os.WriteFile(srcFile, []byte("test content"), 0644)
		require.NoError(t, err, "Failed to create source file")

		// Setup config with createDirs disabled
		cfg := &config.Config{}
		cfg.Settings.CreateDirs = false

		// Create engine
		engine := NewWithConfig(cfg)

		// Non-existent destination directory
		nonExistentDir := filepath.Join(destDir, "does", "not", "exist")
		destFile := filepath.Join(nonExistentDir, "test.txt")

		// Try to move file to non-existent directory without creation
		err = engine.MoveFile(srcFile, destFile)
		assert.Error(t, err, "MoveFile should return an error when destination directory doesn't exist")
	})

	// Test OrganizeFile with no config
	t.Run("OrganizeFileNoConfig", func(t *testing.T) {
		// Create engine with no config
		engine := New()

		// Create source file
		srcFile := filepath.Join(srcDir, "no-config-test.txt")
		err := os.WriteFile(srcFile, []byte("test content"), 0644)
		require.NoError(t, err, "Failed to create source file")

		// Try to organize file with no config
		err = engine.OrganizeFile(srcFile)
		assert.Error(t, err, "OrganizeFile should return an error when no config is set")

		// Verify it's a ConfigError with the expected code
		configErr, ok := err.(*errors.ConfigError)
		if assert.True(t, ok, "Error should be a *errors.ConfigError") {
			assert.Equal(t, errors.ConfigNotSet, configErr.Kind(), "Error should have ConfigNotSet code")
		}
	})
}
