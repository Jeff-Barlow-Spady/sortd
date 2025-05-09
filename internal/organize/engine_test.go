package organize_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"sortd/internal/config"
	"sortd/internal/organize"
	"sortd/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicFileMove(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	destDir := filepath.Join(tmpDir, "documents")
	// Create the destination directory to ensure it exists
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err, "Failed to create destination directory")

	// Create a basic config with createDirs set to true
	cfg := &config.Config{}
	cfg.Settings.CreateDirs = true

	t.Run("move single file", func(t *testing.T) {
		// Need a fresh test file for this subtest
		srcFile := filepath.Join(tmpDir, "single_move.txt")
		err := os.WriteFile(srcFile, []byte("single move"), 0644)
		require.NoError(t, err)

		destFile := filepath.Join(destDir, "single_move.txt")
		engine := organize.NewWithConfig(cfg) // Use real engine with config
		err = engine.MoveFile(srcFile, destFile)
		assert.NoError(t, err, "MoveFile should succeed")

		// Verify filesystem state
		_, err = os.Stat(srcFile)
		assert.ErrorIs(t, err, os.ErrNotExist, "Source file should not exist after move")
		_, err = os.Stat(destFile)
		assert.NoError(t, err, "Destination file should exist after move")
	})

	// Refactor duplicate move test
	t.Run("prevent duplicate moves", func(t *testing.T) {
		// Need a fresh test file for this subtest
		srcFile := filepath.Join(tmpDir, "duplicate_move.txt")
		err := os.WriteFile(srcFile, []byte("duplicate move"), 0644)
		require.NoError(t, err)
		destFile := filepath.Join(destDir, "duplicate_move.txt")

		engine := organize.NewWithConfig(cfg) // Use real engine with config

		// First move should succeed
		err = engine.MoveFile(srcFile, destFile)
		assert.NoError(t, err, "First move should succeed")

		// Verify state after first move
		_, err = os.Stat(srcFile)
		assert.ErrorIs(t, err, os.ErrNotExist, "Source file should not exist after first move")
		_, err = os.Stat(destFile)
		assert.NoError(t, err, "Destination file should exist after first move")

		// Create the source file again to attempt the move
		err = os.WriteFile(srcFile, []byte("duplicate move again"), 0644)
		require.NoError(t, err)

		// Configure the engine to use a collision strategy
		cfgWithCollision := &config.Config{}
		cfgWithCollision.Settings.CreateDirs = true
		cfgWithCollision.Settings.Collision = "skip"
		engine = organize.NewWithConfig(cfgWithCollision)

		// Second move should fail because destination exists
		err = engine.MoveFile(srcFile, destFile)
		assert.NoError(t, err, "Second move should be skipped due to collision strategy")

		// Verify state after second attempt (src exists, dest exists)
		_, err = os.Stat(srcFile)
		assert.NoError(t, err, "Source file should still exist after skipped second move")
		_, err = os.Stat(destFile)
		assert.NoError(t, err, "Destination file should still exist after skipped second move")
	})
}

func TestFileOrganization(t *testing.T) {
	tmpDir := t.TempDir()

	// Files to create in tmpDir
	filesToCreate := map[string]string{
		"image1.jpg":    "jpg content",
		"documentA.pdf": "pdf content",
		"image2.png":    "png content",
		"archive.zip":   "zip content",
		"readme.txt":    "text content",
	}

	for name, content := range filesToCreate {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Define destination directories relative to tmpDir
	imgDest := filepath.Join(tmpDir, "Images")
	pdfDest := filepath.Join(tmpDir, "Documents")
	otherDest := filepath.Join(tmpDir, "Other")

	// Setup config with patterns
	cfg := &config.Config{
		Organize: struct {
			Patterns []types.Pattern `yaml:"patterns"`
		}{
			Patterns: []types.Pattern{
				{Match: "*.jpg", Target: imgDest},
				{Match: "*.png", Target: imgDest},
				{Match: "*.pdf", Target: pdfDest},
				{Match: "*", Target: otherDest}, // Catch-all for others
			},
		},
		Settings: config.Settings{
			DryRun:         false,
			CreateDirs:     true,
			Confirm:        false,
			MaxDepth:       10,
			FollowSymlinks: false,
			IgnoreHidden:   true,
			LogLevel:       "info",
			Backup:         false,
			Collision:      "skip", // Provide a default collision strategy for the test
		},
	}

	// Test organizing the directory
	t.Run("organize directory by patterns", func(t *testing.T) {
		engine := organize.NewWithConfig(cfg)

		results, err := engine.OrganizeDirectory(tmpDir)
		assert.NoError(t, err, "OrganizeDirectory should succeed")
		require.Len(t, results, len(filesToCreate), "Should have results for all initial files")

		// Verify results and filesystem state
		expectedMoves := map[string]string{
			filepath.Join(tmpDir, "image1.jpg"):    filepath.Join(imgDest, "image1.jpg"),
			filepath.Join(tmpDir, "documentA.pdf"): filepath.Join(pdfDest, "documentA.pdf"),
			filepath.Join(tmpDir, "image2.png"):    filepath.Join(imgDest, "image2.png"),
			filepath.Join(tmpDir, "archive.zip"):   filepath.Join(otherDest, "archive.zip"),
			filepath.Join(tmpDir, "readme.txt"):    filepath.Join(otherDest, "readme.txt"),
		}

		processedSources := make(map[string]bool)
		for _, res := range results {
			assert.NoError(t, res.Error, "Result for %s should not have an error", res.SourcePath)
			assert.True(t, res.Moved, "Result for %s should indicate file was moved", res.SourcePath)

			expectedDest, ok := expectedMoves[res.SourcePath]
			assert.True(t, ok, "Source path %s not found in expected moves", res.SourcePath)
			assert.Equal(t, expectedDest, res.DestinationPath, "Incorrect destination for %s", res.SourcePath)

			// Check filesystem
			_, err := os.Stat(res.SourcePath)
			assert.ErrorIs(t, err, os.ErrNotExist, "Source file %s should not exist after move", res.SourcePath)
			_, err = os.Stat(res.DestinationPath)
			assert.NoError(t, err, "Destination file %s should exist after move", res.DestinationPath)

			processedSources[res.SourcePath] = true
		}

		assert.Equal(t, len(expectedMoves), len(processedSources), "Number of processed results should match expected moves")
	})
}

func TestOrganizationEdgeCases(t *testing.T) {
	// Basic config for tests that don't need patterns
	basicCfg := &config.Config{}

	t.Run("move to existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcFile := filepath.Join(tmpDir, "source.txt")
		destFile := filepath.Join(tmpDir, "dest.txt")

		err := os.WriteFile(srcFile, []byte("source"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(destFile, []byte("dest"), 0644)
		require.NoError(t, err)

		// Create a config that doesn't have a collision handling strategy
		// this should cause the engine to error when the destination exists
		errCfg := &config.Config{}
		errCfg.Settings.Collision = "error" // Use an invalid strategy to force an error

		engine := organize.NewWithConfig(errCfg) // Use engine with invalid collision strategy
		err = engine.MoveFile(srcFile, destFile)
		assert.Error(t, err, "MoveFile should error when destination exists and using 'error' collision strategy")

		// Verify files are untouched
		_, err = os.Stat(srcFile)
		assert.NoError(t, err, "Source should still exist")
		_, err = os.Stat(destFile)
		assert.NoError(t, err, "Destination should still exist")
	})

	// Refactored non-existent file test
	t.Run("move non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		engine := organize.NewWithConfig(basicCfg) // Use real engine
		err := engine.MoveFile(filepath.Join(tmpDir, "nonexistent.txt"), filepath.Join(tmpDir, "dest.txt"))
		assert.Error(t, err, "MoveFile should error for non-existent source")
		// Check that the specific error is os.ErrNotExist or wraps it
		assert.ErrorIs(t, err, os.ErrNotExist, "Error should indicate file not found")
	})

	// Refactored invalid path test - MoveFile attempts MkdirAll, so this might succeed depending on permissions
	// Let's test moving *to* a file path instead, which should fail.
	t.Run("move to path occupied by a file", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcFile := filepath.Join(tmpDir, "source.txt")
		destDirAsFile := filepath.Join(tmpDir, "dest_dir_file.txt") // Treat this as the intended directory path

		err := os.WriteFile(srcFile, []byte("test"), 0644)
		require.NoError(t, err)
		// Create a file where the destination *directory* should be
		err = os.WriteFile(destDirAsFile, []byte("test"), 0644)
		require.NoError(t, err)

		engine := organize.NewWithConfig(basicCfg) // Use real engine
		// Attempt to move into the 'directory' which is actually a file
		destFile := filepath.Join(destDirAsFile, "source.txt")
		err = engine.MoveFile(srcFile, destFile)
		// os.Rename should fail because parent path (destDirAsFile) is not a directory
		assert.Error(t, err, "MoveFile should error when dest parent path is not a directory")
	})

	// Test concurrent file moves
	t.Run("concurrent moves", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "001") // Create a subdirectory
		err := os.MkdirAll(srcDir, 0755)
		require.NoError(t, err)

		// Create destination directory
		destDir := filepath.Join(srcDir, "dest_concurrent")
		err = os.MkdirAll(destDir, 0755)
		require.NoError(t, err, "Failed to create destination directory")

		// Create a number of files to move concurrently
		numFiles := 10
		for i := 0; i < numFiles; i++ {
			srcFile := filepath.Join(srcDir, fmt.Sprintf("file%d.txt", i))
			err := os.WriteFile(srcFile, []byte(fmt.Sprintf("content %d", i)), 0644)
			require.NoError(t, err)
		}

		// Create a config with createDirs enabled
		cfg := &config.Config{}
		cfg.Settings.CreateDirs = true

		// Create a WaitGroup to sync goroutines
		var wg sync.WaitGroup
		wg.Add(numFiles)

		// Create shared engine to test concurrent access
		engine := organize.NewWithConfig(cfg)

		// Launch concurrent move operations
		for i := 0; i < numFiles; i++ {
			go func(fileIdx int) {
				defer wg.Done()
				srcFile := filepath.Join(srcDir, fmt.Sprintf("file%d.txt", fileIdx))
				destFile := filepath.Join(destDir, fmt.Sprintf("file%d.txt", fileIdx))
				err := engine.MoveFile(srcFile, destFile)
				if err != nil {
					t.Errorf("error moving %s: %v", srcFile, err)
				}
			}(i)
		}

		// Wait for all operations to complete
		wg.Wait()

		// Verify results
		for i := 0; i < numFiles; i++ {
			srcFile := filepath.Join(srcDir, fmt.Sprintf("file%d.txt", i))
			destFile := filepath.Join(destDir, fmt.Sprintf("file%d.txt", i))

			// Source should no longer exist
			_, err = os.Stat(srcFile)
			assert.ErrorIs(t, err, os.ErrNotExist, "Source file %s should not exist after concurrent move", srcFile)

			// Destination should exist
			_, err = os.Stat(destFile)
			assert.NoError(t, err, "Destination file %s should exist after concurrent move", destFile)
		}
	})

	// Test moving empty files
	t.Run("move empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "001")
		err := os.MkdirAll(srcDir, 0755)
		require.NoError(t, err)

		// Create destination directory
		destDir := filepath.Join(srcDir, "dest_empty")
		err = os.MkdirAll(destDir, 0755)
		require.NoError(t, err, "Failed to create destination directory")

		// Create empty source file
		emptyFile := filepath.Join(srcDir, "empty.txt")
		err = os.WriteFile(emptyFile, []byte{}, 0644)
		require.NoError(t, err)

		cfg := &config.Config{}
		cfg.Settings.CreateDirs = true

		engine := organize.NewWithConfig(cfg)
		err = engine.MoveFile(emptyFile, filepath.Join(destDir, "empty.txt"))
		assert.NoError(t, err, "MoveFile should handle empty files")

		// Verify file moved correctly
		_, err = os.Stat(emptyFile)
		assert.ErrorIs(t, err, os.ErrNotExist, "Source empty file should not exist")
		_, err = os.Stat(filepath.Join(destDir, "empty.txt"))
		assert.NoError(t, err, "Destination empty file should exist")
	})

	// Test moving files with special characters in name
	t.Run("move file with special characters", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "001")
		err := os.MkdirAll(srcDir, 0755)
		require.NoError(t, err)

		// Create destination directory
		destDir := filepath.Join(srcDir, "dest_special")
		err = os.MkdirAll(destDir, 0755)
		require.NoError(t, err, "Failed to create destination directory")

		// Create file with special chars
		specialFile := filepath.Join(srcDir, "special!@#$%^&*.txt")
		err = os.WriteFile(specialFile, []byte("special content"), 0644)
		require.NoError(t, err)

		cfg := &config.Config{}
		cfg.Settings.CreateDirs = true

		engine := organize.NewWithConfig(cfg)
		err = engine.MoveFile(specialFile, filepath.Join(destDir, "special!@#$%^&*.txt"))
		assert.NoError(t, err, "MoveFile should handle special characters in filenames")

		// Verify file moved correctly
		_, err = os.Stat(specialFile)
		assert.ErrorIs(t, err, os.ErrNotExist, "Source special file should not exist")
		_, err = os.Stat(filepath.Join(destDir, "special!@#$%^&*.txt"))
		assert.NoError(t, err, "Destination special file should exist")
	})

	// Refactored same location test
	t.Run("move to same location", func(t *testing.T) {
		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "file.txt")
		err := os.WriteFile(file, []byte("test"), 0644)
		require.NoError(t, err)

		engine := organize.NewWithConfig(basicCfg) // Use real engine
		err = engine.MoveFile(file, file)
		assert.NoError(t, err, "MoveFile should not error when source and destination are the same")

		// Verify file still exists
		_, err = os.Stat(file)
		assert.NoError(t, err, "File should still exist after failed same-location move")
	})
}

func TestEngine_OrganizeByPatterns(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"test.txt",
		"image.jpg",
		"archive.zip",
	}
	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0644))
	}

	// Create config with patterns
	cfg := &config.Config{
		Organize: struct {
			Patterns []types.Pattern `yaml:"patterns"`
		}{
			Patterns: []types.Pattern{
				{Match: "*.txt", Target: "documents/"},
				{Match: "*.jpg", Target: "images/"},
				{Match: "*.zip", Target: "archives/"},
			},
		},
		Settings: config.Settings{
			DryRun:         false,
			CreateDirs:     true,
			Confirm:        false,
			MaxDepth:       10,
			FollowSymlinks: false,
			IgnoreHidden:   true,
			LogLevel:       "info",
			Backup:         false,
			Collision:      "rename",
		},
	}

	// Create engine
	engine := organize.NewWithConfig(cfg)

	// Test organizing all files
	filesToOrganize := make([]string, len(testFiles))
	for i, file := range testFiles {
		filesToOrganize[i] = filepath.Join(tempDir, file)
	}

	// Organize files
	require.NoError(t, engine.OrganizeByPatterns(filesToOrganize))

	// Verify files were moved
	for _, file := range testFiles {
		originalPath := filepath.Join(tempDir, file)
		_, err := os.Stat(originalPath)
		require.Error(t, err)
		require.True(t, os.IsNotExist(err), "Source file %s should not exist after organize", file)

		// Determine expected target directory and potential renamed file
		var targetDir string
		switch filepath.Ext(file) {
		case ".txt":
			targetDir = filepath.Join(tempDir, "documents")
		case ".jpg":
			targetDir = filepath.Join(tempDir, "images")
		case ".zip":
			targetDir = filepath.Join(tempDir, "archives")
		default:
			t.Fatalf("Unexpected file extension in test setup: %s", file)
		}

		// Check if the file exists with original name or renamed name
		originalTargetPath := filepath.Join(targetDir, file)
		renamedTargetPath := filepath.Join(targetDir, strings.TrimSuffix(file, filepath.Ext(file))+"_(1)"+filepath.Ext(file))

		_, errOriginal := os.Stat(originalTargetPath)
		_, errRenamed := os.Stat(renamedTargetPath)

		// Assert that *either* the original or the renamed file exists
		require.True(t, errOriginal == nil || errRenamed == nil,
			"Expected file %s to exist in %s as either original or renamed, but neither found. Original err: %v, Renamed err: %v",
			file, targetDir, errOriginal, errRenamed)
	}
}
