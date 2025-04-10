package organize

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"sortd/internal/config"
	"sortd/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEngineWorkflowIntegration tests how the engine works with workflows
func TestEngineWorkflowIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := filepath.Join(os.TempDir(), "sortd-test-engine-workflow")
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

	// Create test files
	testTxtFile := filepath.Join(srcDir, "test.txt")
	err = os.WriteFile(testTxtFile, []byte("test content"), 0644)
	require.NoError(t, err, "Failed to create test text file")

	testImgFile := filepath.Join(srcDir, "image.jpg")
	err = os.WriteFile(testImgFile, []byte("image content"), 0644)
	require.NoError(t, err, "Failed to create test image file")

	// Create a config with patterns that match workflow rules
	cfg := &config.Config{}
	cfg.Organize.Patterns = []types.Pattern{
		{Match: "*.txt", Target: filepath.Join(destDir, "documents")},
		{Match: "*.jpg", Target: filepath.Join(destDir, "images")},
	}

	// Create workflows that would be in the config
	workflow1 := types.Workflow{
		ID:      "test-txt-workflow",
		Name:    "Text File Workflow",
		Enabled: true,
		Trigger: types.Trigger{
			Type:    types.FilePatternMatch,
			Pattern: "*.txt",
		},
		Actions: []types.Action{
			{
				Type:   types.MoveAction,
				Target: filepath.Join(destDir, "documents"),
			},
		},
	}

	workflow2 := types.Workflow{
		ID:      "test-img-workflow",
		Name:    "Image File Workflow",
		Enabled: true,
		Trigger: types.Trigger{
			Type:    types.FilePatternMatch,
			Pattern: "*.jpg",
		},
		Actions: []types.Action{
			{
				Type:   types.MoveAction,
				Target: filepath.Join(destDir, "images"),
			},
		},
	}

	// Add workflows to config
	cfg.Workflows = []types.Workflow{workflow1, workflow2}

	// Test cases for engine with workflow patterns
	t.Run("OrganizeFiles matching workflow patterns", func(t *testing.T) {
		// Create the engine
		engine := NewWithConfig(cfg)

		// Create destination directories
		err = os.MkdirAll(filepath.Join(destDir, "documents"), 0755)
		require.NoError(t, err, "Failed to create documents directory")

		err = os.MkdirAll(filepath.Join(destDir, "images"), 0755)
		require.NoError(t, err, "Failed to create images directory")

		// Use OrganizeByPatterns to organize the files
		err = engine.OrganizeByPatterns([]string{testTxtFile, testImgFile})
		assert.NoError(t, err, "OrganizeByPatterns should not return an error")

		// Verify files were moved to correct destinations
		_, err = os.Stat(filepath.Join(destDir, "documents", "test.txt"))
		assert.NoError(t, err, "Text file should be moved to documents directory")

		_, err = os.Stat(filepath.Join(destDir, "images", "image.jpg"))
		assert.NoError(t, err, "Image file should be moved to images directory")
	})

	// Test engine with conditional workflow logic
	t.Run("OrganizeFiles with workflow conditions", func(t *testing.T) {
		// Reset test files
		os.RemoveAll(destDir)
		err = os.MkdirAll(destDir, 0755)
		require.NoError(t, err, "Failed to reset destination directory")

		// Create new test files
		testSmallFile := filepath.Join(srcDir, "small.txt")
		err = os.WriteFile(testSmallFile, []byte("small content"), 0644)
		require.NoError(t, err, "Failed to create small test file")

		testLargeFile := filepath.Join(srcDir, "large.txt")
		largeContent := make([]byte, 10*1024) // 10KB
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}
		err = os.WriteFile(testLargeFile, largeContent, 0644)
		require.NoError(t, err, "Failed to create large test file")

		// Create config with size-based conditions
		cfgWithConditions := &config.Config{}
		cfgWithConditions.Organize.Patterns = []types.Pattern{
			{Match: "*.txt", Target: filepath.Join(destDir, "default")},
		}

		// Create workflow with file size condition
		workflowWithCondition := types.Workflow{
			ID:      "size-condition-workflow",
			Name:    "File Size Workflow",
			Enabled: true,
			Trigger: types.Trigger{
				Type:    types.FilePatternMatch,
				Pattern: "*.txt",
			},
			Conditions: []types.Condition{
				{
					Type:     types.FileSizeCondition,
					Field:    "size",
					Operator: types.GreaterThan,
					Value:    "5000", // 5KB
				},
			},
			Actions: []types.Action{
				{
					Type:   types.MoveAction,
					Target: filepath.Join(destDir, "large-files"),
				},
			},
		}

		cfgWithConditions.Workflows = []types.Workflow{workflowWithCondition}

		// Create engine with conditionals
		engine := NewWithConfig(cfgWithConditions)

		// Create destination directories
		err = os.MkdirAll(filepath.Join(destDir, "default"), 0755)
		require.NoError(t, err)

		err = os.MkdirAll(filepath.Join(destDir, "large-files"), 0755)
		require.NoError(t, err)

		// Run engine organization
		// Note: The actual engine implementation doesn't evaluate workflow conditions,
		// that's done by the workflow.Manager. This test is documenting expected behavior
		// if the engine were to consider workflow conditions.
		err = engine.OrganizeByPatterns([]string{testSmallFile, testLargeFile})
		assert.NoError(t, err)

		// Verify files were moved based on patterns, not conditions
		// since the workflow condition evaluation happens in workflow.Manager, not in engine
		_, err = os.Stat(filepath.Join(destDir, "default", "small.txt"))
		assert.NoError(t, err, "Small file should be moved to default directory")

		_, err = os.Stat(filepath.Join(destDir, "default", "large.txt"))
		assert.NoError(t, err, "Large file should be moved to default directory")
	})

	// Test collision behavior with workflow-defined rules
	t.Run("CollisionHandling with workflow options", func(t *testing.T) {
		// Reset test files
		os.RemoveAll(destDir)
		err = os.MkdirAll(destDir, 0755)
		require.NoError(t, err, "Failed to reset destination directory")

		// Create collision test files
		srcFile := filepath.Join(srcDir, "collision.txt")
		err = os.WriteFile(srcFile, []byte("source content"), 0644)
		require.NoError(t, err, "Failed to create source file")

		// Create a destination file with the same name
		destSubDir := filepath.Join(destDir, "documents")
		err = os.MkdirAll(destSubDir, 0755)
		require.NoError(t, err, "Failed to create destination subdirectory")

		destFile := filepath.Join(destSubDir, "collision.txt")
		err = os.WriteFile(destFile, []byte("existing content"), 0644)
		require.NoError(t, err, "Failed to create existing destination file")

		// Create config with overwrite collision strategy
		cfgWithCollision := &config.Config{}
		cfgWithCollision.Settings.Collision = "overwrite"
		cfgWithCollision.Organize.Patterns = []types.Pattern{
			{Match: "*.txt", Target: filepath.Join(destDir, "documents")},
		}

		// Create engine
		engine := NewWithConfig(cfgWithCollision)

		// Test file movement with collision
		err = engine.OrganizeByPatterns([]string{srcFile})
		assert.NoError(t, err, "OrganizeByPatterns should not return an error")

		// Verify file was overwritten
		content, err := os.ReadFile(destFile)
		assert.NoError(t, err, "Should be able to read destination file")
		assert.Equal(t, "source content", string(content), "Destination file should be overwritten")
	})

	// Test backup functionality
	t.Run("BackupOnFileMove", func(t *testing.T) {
		// Reset test files
		os.RemoveAll(destDir)
		err = os.MkdirAll(destDir, 0755)
		require.NoError(t, err, "Failed to reset destination directory")

		// Create backup test files
		srcFile := filepath.Join(srcDir, "backup.txt")
		err = os.WriteFile(srcFile, []byte("new content"), 0644)
		require.NoError(t, err, "Failed to create source file")

		// Create a destination file to be backed up
		destSubDir := filepath.Join(destDir, "documents")
		err = os.MkdirAll(destSubDir, 0755)
		require.NoError(t, err, "Failed to create destination subdirectory")

		destFile := filepath.Join(destSubDir, "backup.txt")
		err = os.WriteFile(destFile, []byte("old content"), 0644)
		require.NoError(t, err, "Failed to create existing destination file")

		// Create config with backup enabled
		cfgWithBackup := &config.Config{}
		cfgWithBackup.Settings.Backup = true
		cfgWithBackup.Settings.Collision = "overwrite"
		cfgWithBackup.Organize.Patterns = []types.Pattern{
			{Match: "*.txt", Target: filepath.Join(destDir, "documents")},
		}

		// Create engine
		engine := NewWithConfig(cfgWithBackup)

		// Test file movement with backup
		err = engine.OrganizeByPatterns([]string{srcFile})
		assert.NoError(t, err, "OrganizeByPatterns should not return an error")

		// Verify file was overwritten
		content, err := os.ReadFile(destFile)
		assert.NoError(t, err, "Should be able to read destination file")
		assert.Equal(t, "new content", string(content), "Destination file should be overwritten")

		// Check for backup file
		backupFiles, err := filepath.Glob(filepath.Join(destSubDir, "backup.txt.bak.*"))
		assert.NoError(t, err, "Should be able to glob backup files")
		assert.NotEmpty(t, backupFiles, "Should find at least one backup file")

		// Verify backup content
		if len(backupFiles) > 0 {
			backupContent, err := os.ReadFile(backupFiles[0])
			assert.NoError(t, err, "Should be able to read backup file")
			assert.Equal(t, "old content", string(backupContent), "Backup file should contain original content")
		}
	})
}

// TestEngineWithComplexWorkflows tests more complex workflows with multiple conditions and actions
func TestEngineWithComplexWorkflows(t *testing.T) {
	// This test demonstrates how to test more complex workflow scenarios
	// In a real application, these would be handled by the workflow.Manager
	// but we're demonstrating the expected organization behavior

	// Create a temporary directory for testing
	tmpDir := filepath.Join(os.TempDir(), "sortd-test-complex-workflow")
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

	// Create test files with different timestamps
	oldFile := filepath.Join(srcDir, "old.txt")
	err = os.WriteFile(oldFile, []byte("old content"), 0644)
	require.NoError(t, err, "Failed to create old file")

	// Set the modified time to 7 days ago
	oldTime := time.Now().Add(-7 * 24 * time.Hour)
	err = os.Chtimes(oldFile, oldTime, oldTime)
	require.NoError(t, err, "Failed to set old file time")

	newFile := filepath.Join(srcDir, "new.txt")
	err = os.WriteFile(newFile, []byte("new content"), 0644)
	require.NoError(t, err, "Failed to create new file")

	// Create destination directories
	err = os.MkdirAll(filepath.Join(destDir, "documents"), 0755)
	require.NoError(t, err, "Failed to create documents directory")

	err = os.MkdirAll(filepath.Join(destDir, "archive"), 0755)
	require.NoError(t, err, "Failed to create archive directory")

	// Create config with absolute paths for destination directories
	cfg := &config.Config{}
	cfg.Settings.CreateDirs = true

	// Use absolute paths for the patterns
	archiveDir := filepath.Join(destDir, "archive")
	documentsDir := filepath.Join(destDir, "documents")

	cfg.Organize.Patterns = []types.Pattern{
		// Special pattern for old files (matched first)
		{Match: "old.txt", Target: archiveDir},
		// Base pattern to catch all text files
		{Match: "*.txt", Target: documentsDir},
	}

	// Create the engine
	engine := NewWithConfig(cfg)

	// Test organization with multiple patterns
	err = engine.OrganizeByPatterns([]string{oldFile, newFile})
	assert.NoError(t, err, "OrganizeByPatterns should not return an error")

	// Verify files were moved to correct destinations
	_, err = os.Stat(filepath.Join(destDir, "archive", "old.txt"))
	assert.NoError(t, err, "Old file should be moved to archive directory")

	_, err = os.Stat(filepath.Join(destDir, "documents", "new.txt"))
	assert.NoError(t, err, "New file should be moved to documents directory")
}
