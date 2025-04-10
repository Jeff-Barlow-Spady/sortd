package organize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sortd/internal/config"
	"sortd/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackupFunctionality tests the backup feature in detail
func TestBackupFunctionality(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := filepath.Join(os.TempDir(), "sortd-test-engine-backup")
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

	// Test basic backup on overwrite
	t.Run("BasicBackupOnOverwrite", func(t *testing.T) {
		// Create test files
		srcFile := filepath.Join(srcDir, "test.txt")
		err := os.WriteFile(srcFile, []byte("new content"), 0644)
		require.NoError(t, err, "Failed to create source file")

		// Create destination directory
		destSubDir := filepath.Join(destDir, "documents")
		err = os.MkdirAll(destSubDir, 0755)
		require.NoError(t, err, "Failed to create destination subdirectory")

		// Create existing file at destination
		destFile := filepath.Join(destSubDir, "test.txt")
		err = os.WriteFile(destFile, []byte("old content"), 0644)
		require.NoError(t, err, "Failed to create destination file")

		// Setup config with backup enabled
		cfg := &config.Config{}
		cfg.Settings.Backup = true
		cfg.Settings.Collision = "overwrite"
		cfg.Organize.Patterns = []types.Pattern{
			{Match: "*.txt", Target: filepath.Join(destDir, "documents")},
		}

		// Create engine with backup enabled
		engine := NewWithConfig(cfg)

		// Move file, which should create a backup
		err = engine.MoveFile(srcFile, destFile)
		assert.NoError(t, err, "MoveFile should succeed")

		// Check destination file has new content
		content, err := os.ReadFile(destFile)
		assert.NoError(t, err, "Should be able to read destination file")
		assert.Equal(t, "new content", string(content), "Destination should have new content")

		// Find backup file
		files, err := os.ReadDir(destSubDir)
		assert.NoError(t, err, "Should be able to read destination directory")

		var backupFile string
		for _, file := range files {
			if strings.HasPrefix(file.Name(), "test.txt.bak.") {
				backupFile = filepath.Join(destSubDir, file.Name())
				break
			}
		}

		// Verify backup file exists and has correct content
		assert.NotEmpty(t, backupFile, "Backup file should exist")
		if backupFile != "" {
			backupContent, err := os.ReadFile(backupFile)
			assert.NoError(t, err, "Should be able to read backup file")
			assert.Equal(t, "old content", string(backupContent), "Backup file should have old content")
		}
	})

	// Test multiple backups
	t.Run("MultipleBackups", func(t *testing.T) {
		// Reset destination directory
		destSubDir := filepath.Join(destDir, "multi-backup")
		os.RemoveAll(destSubDir)
		err := os.MkdirAll(destSubDir, 0755)
		require.NoError(t, err, "Failed to create destination subdirectory")

		// Create destination file
		destFile := filepath.Join(destSubDir, "test.txt")
		err = os.WriteFile(destFile, []byte("original content"), 0644)
		require.NoError(t, err, "Failed to create destination file")

		// Setup config with backup enabled
		cfg := &config.Config{}
		cfg.Settings.Backup = true
		cfg.Settings.Collision = "overwrite"

		// Create engine with backup enabled
		engine := NewWithConfig(cfg)

		// Create and move multiple source files to create multiple backups
		for i := 1; i <= 3; i++ {
			// Create source file with new content
			srcFile := filepath.Join(srcDir, "test.txt")
			err := os.WriteFile(srcFile, []byte("content version "+string(rune('0'+i))), 0644)
			require.NoError(t, err, "Failed to create source file")

			// Move file, which should create a backup
			err = engine.MoveFile(srcFile, destFile)
			assert.NoError(t, err, "MoveFile should succeed")

			// Sleep briefly to ensure unique timestamps
			time.Sleep(10 * time.Millisecond)
		}

		// Should have 3 backup files + current file
		files, err := os.ReadDir(destSubDir)
		assert.NoError(t, err, "Should be able to read destination directory")

		var backupFiles []string
		for _, file := range files {
			if strings.HasPrefix(file.Name(), "test.txt.bak.") {
				backupFiles = append(backupFiles, file.Name())
			}
		}

		assert.Equal(t, 3, len(backupFiles), "Should have 3 backup files")

		// Check current file has latest content
		content, err := os.ReadFile(destFile)
		assert.NoError(t, err, "Should be able to read destination file")
		assert.Equal(t, "content version 3", string(content), "Destination should have latest content")
	})

	// Test backup with createDirs enabled
	t.Run("BackupWithCreateDirs", func(t *testing.T) {
		// Create a deeply nested destination path that doesn't exist yet
		nestedDestDir := filepath.Join(destDir, "nested", "backup", "test")
		destFile := filepath.Join(nestedDestDir, "test.txt")

		// First create the file so we have something to backup
		err := os.MkdirAll(nestedDestDir, 0755)
		require.NoError(t, err, "Failed to create nested destination directory")

		err = os.WriteFile(destFile, []byte("original content"), 0644)
		require.NoError(t, err, "Failed to create destination file")

		// Remove the directories to test createDirs functionality
		os.RemoveAll(filepath.Join(destDir, "nested"))

		// Create source file
		srcFile := filepath.Join(srcDir, "test.txt")
		err = os.WriteFile(srcFile, []byte("new content"), 0644)
		require.NoError(t, err, "Failed to create source file")

		// Setup config with backup and createDirs enabled
		cfg := &config.Config{}
		cfg.Settings.Backup = true
		cfg.Settings.CreateDirs = true
		cfg.Settings.Collision = "overwrite"

		// Create engine
		engine := NewWithConfig(cfg)

		// Move file, which should create directories and backup
		err = engine.MoveFile(srcFile, destFile)
		assert.NoError(t, err, "MoveFile should succeed")

		// Check destination file exists and has new content
		content, err := os.ReadFile(destFile)
		assert.NoError(t, err, "Should be able to read destination file")
		assert.Equal(t, "new content", string(content), "Destination should have new content")

		// Find backup file
		files, err := os.ReadDir(nestedDestDir)
		assert.NoError(t, err, "Should be able to read destination directory")

		var backupFile string
		for _, file := range files {
			if strings.HasPrefix(file.Name(), "test.txt.bak.") {
				backupFile = filepath.Join(nestedDestDir, file.Name())
				break
			}
		}

		// Verify backup file exists
		assert.NotEmpty(t, backupFile, "Backup file should exist")
	})

	// Test that backup is skipped when not needed
	t.Run("SkipBackupWhenNotNeeded", func(t *testing.T) {
		// Create destination directory with no existing file
		cleanDestDir := filepath.Join(destDir, "no-existing")
		err := os.MkdirAll(cleanDestDir, 0755)
		require.NoError(t, err, "Failed to create clean destination directory")

		// Ensure no existing file at destination
		destFile := filepath.Join(cleanDestDir, "new.txt")
		os.Remove(destFile) // Just to be safe

		// Create source file
		srcFile := filepath.Join(srcDir, "new.txt")
		err = os.WriteFile(srcFile, []byte("new content"), 0644)
		require.NoError(t, err, "Failed to create source file")

		// Setup config with backup enabled
		cfg := &config.Config{}
		cfg.Settings.Backup = true

		// Create engine with backup enabled
		engine := NewWithConfig(cfg)

		// Move file - should not create a backup since no file exists at destination
		err = engine.MoveFile(srcFile, destFile)
		assert.NoError(t, err, "MoveFile should succeed")

		// Check destination file has content
		content, err := os.ReadFile(destFile)
		assert.NoError(t, err, "Should be able to read destination file")
		assert.Equal(t, "new content", string(content), "Destination should have content")

		// Verify no backup files were created
		files, err := os.ReadDir(cleanDestDir)
		assert.NoError(t, err, "Should be able to read destination directory")

		backupCount := 0
		for _, file := range files {
			if strings.HasPrefix(file.Name(), "new.txt.bak.") {
				backupCount++
			}
		}

		assert.Equal(t, 0, backupCount, "Should not create backup for non-existent destination file")
	})
}
