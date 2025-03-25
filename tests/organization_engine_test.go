package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sortd/internal/config"
	"sortd/internal/organize"
	"sortd/pkg/types"
	"sortd/tests/testutils"
)

type FileInfo struct {
	Path        string
	ContentType string
	Size        int64
}

type OrganizationEngine struct {
	files map[string]FileInfo
	mu    sync.RWMutex
}

func NewOrganizationEngine() *OrganizationEngine {
	return &OrganizationEngine{
		files: make(map[string]FileInfo),
	}
}

func (e *OrganizationEngine) MoveFile(src, dest string) error {
	// Clean paths for comparison
	cleanSrc := filepath.Clean(src)
	cleanDest := filepath.Clean(dest)

	// Check for same file
	if cleanSrc == cleanDest {
		return fmt.Errorf("source and destination are the same file: %s", src)
	}

	// Verify source exists and get info
	srcInfo, err := os.Stat(cleanSrc)
	if err != nil {
		return fmt.Errorf("source file error: %w", err)
	}

	if srcInfo.IsDir() {
		return fmt.Errorf("cannot move directory as file: %s", src)
	}

	// Check if destination exists on filesystem
	if _, err := os.Stat(cleanDest); err == nil {
		return fmt.Errorf("destination file already exists on filesystem: %s", dest)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking destination: %w", err)
	}

	// Lock for thread safety
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if destination is tracked in our files map
	if _, exists := e.files[cleanDest]; exists {
		return fmt.Errorf("destination file already exists in tracking: %s", dest)
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(cleanDest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Record the move
	e.files[cleanDest] = FileInfo{
		Path: cleanDest,
	}

	return nil
}

func (e *OrganizationEngine) OrganizeFiles(files []string, destDir string) error {
	for _, file := range files {
		dest := filepath.Join(destDir, filepath.Base(file))
		if err := e.MoveFile(file, dest); err != nil {
			return err
		}
	}
	return nil
}

func TestBasicFileMove(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	destDir := filepath.Join(tmpDir, "documents")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	t.Run("move single file", func(t *testing.T) {
		engine := NewOrganizationEngine()
		err := engine.MoveFile(testFile, filepath.Join(destDir, "test.txt"))
		assert.NoError(t, err, "Should complete organization")
	})

	t.Run("prevent duplicate moves", func(t *testing.T) {
		engine := NewOrganizationEngine()
		err := engine.MoveFile(testFile, filepath.Join(destDir, "test.txt"))
		assert.NoError(t, err)
		err = engine.MoveFile(testFile, filepath.Join(destDir, "test.txt"))
		assert.Error(t, err)
	})
}

func TestFileOrganization(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	destDir := filepath.Join(tmpDir, "organized")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	t.Run("basic file move", func(t *testing.T) {
		engine := NewOrganizationEngine()
		err := engine.MoveFile(testFile, filepath.Join(destDir, "test.txt"))
		assert.NoError(t, err)
	})

	t.Run("prevent duplicate moves", func(t *testing.T) {
		engine := NewOrganizationEngine()
		err := engine.MoveFile(testFile, filepath.Join(destDir, "test.txt"))
		assert.NoError(t, err)
		err = engine.MoveFile(testFile, filepath.Join(destDir, "test.txt"))
		assert.Error(t, err, "Should prevent moving to existing destination")
	})

	t.Run("batch file organization", func(t *testing.T) {
		engine := NewOrganizationEngine()
		files := []string{
			filepath.Join(tmpDir, "test1.txt"),
			filepath.Join(tmpDir, "test2.txt"),
			filepath.Join(tmpDir, "test3.txt"),
		}
		for _, file := range files {
			err := os.WriteFile(file, []byte("test content"), 0644)
			require.NoError(t, err)
		}
		err := engine.OrganizeFiles(files, destDir)
		assert.NoError(t, err)
	})
}

func TestOrganizationEdgeCases(t *testing.T) {
	t.Run("move to existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcFile := filepath.Join(tmpDir, "source.txt")
		destFile := filepath.Join(tmpDir, "dest.txt")

		err := os.WriteFile(srcFile, []byte("source"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(destFile, []byte("dest"), 0644)
		require.NoError(t, err)

		engine := NewOrganizationEngine()
		err = engine.MoveFile(srcFile, destFile)
		assert.Error(t, err, "Should prevent overwriting existing files")
	})

	t.Run("move non-existent file", func(t *testing.T) {
		engine := NewOrganizationEngine()
		err := engine.MoveFile("nonexistent.txt", "dest.txt")
		assert.Error(t, err, "Should handle non-existent source files")
	})

	t.Run("move to invalid path", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcFile := filepath.Join(tmpDir, "source.txt")
		err := os.WriteFile(srcFile, []byte("test"), 0644)
		require.NoError(t, err)

		engine := NewOrganizationEngine()
		err = engine.MoveFile(srcFile, "/nonexistent/path/dest.txt")
		assert.Error(t, err, "Should handle invalid destination paths")
	})

	t.Run("concurrent moves", func(t *testing.T) {
		tmpDir := t.TempDir()
		engine := NewOrganizationEngine()

		// Create test files
		files := make([]string, 10)
		for i := 0; i < 10; i++ {
			path := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
			err := os.WriteFile(path, []byte("test"), 0644)
			require.NoError(t, err)
			files[i] = path
		}

		// Move files concurrently
		destDir := filepath.Join(tmpDir, "dest")
		err := os.MkdirAll(destDir, 0755)
		require.NoError(t, err)

		var wg sync.WaitGroup
		errChan := make(chan error, len(files))

		for _, file := range files {
			wg.Add(1)
			go func(f string) {
				defer wg.Done()
				dest := filepath.Join(destDir, filepath.Base(f))
				if err := engine.MoveFile(f, dest); err != nil {
					errChan <- err
				}
			}(file)
		}

		// Wait for all goroutines to finish
		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			assert.NoError(t, err, "Should handle concurrent moves without errors")
		}

		// Verify no duplicate moves occurred
		engine.mu.RLock()
		movedFiles := make(map[string]bool)
		for dest := range engine.files {
			assert.False(t, movedFiles[dest], "Should not have duplicate moves")
			movedFiles[dest] = true
		}
		engine.mu.RUnlock()
	})

	t.Run("move empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		emptyFile := filepath.Join(tmpDir, "empty.txt")
		err := os.WriteFile(emptyFile, []byte{}, 0644)
		require.NoError(t, err)

		engine := NewOrganizationEngine()
		dest := filepath.Join(tmpDir, "dest", "empty.txt")
		err = engine.MoveFile(emptyFile, dest)
		assert.NoError(t, err, "Should handle empty files")
	})

	t.Run("move file with special characters", func(t *testing.T) {
		tmpDir := t.TempDir()
		specialFile := filepath.Join(tmpDir, "special!@#$%^&*.txt")
		err := os.WriteFile(specialFile, []byte("test"), 0644)
		require.NoError(t, err)

		engine := NewOrganizationEngine()
		dest := filepath.Join(tmpDir, "dest", "special!@#$%^&*.txt")
		err = engine.MoveFile(specialFile, dest)
		assert.NoError(t, err, "Should handle special characters in filenames")
	})

	t.Run("move to same location", func(t *testing.T) {
		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "file.txt")
		err := os.WriteFile(file, []byte("test"), 0644)
		require.NoError(t, err)

		engine := NewOrganizationEngine()
		err = engine.MoveFile(file, file)
		assert.Error(t, err, "Should prevent moving file to same location")
	})
}

// TestOrganizationEngine tests the actual organization engine implementation
func TestOrganizationEngine(t *testing.T) {
	// Create temp test directory
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err, "Failed to create test file")

	// Create destination directory
	destDir := filepath.Join(tmpDir, "documents")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err, "Failed to create destination directory")

	// Create an instance of the actual organization engine
	engine := organize.New()

	// Configure the engine
	cfg := &config.Config{}
	cfg.Settings.CreateDirs = true
	cfg.Settings.Backup = false
	cfg.Settings.Collision = "rename"
	cfg.Settings.DryRun = false
	cfg.Organize.Patterns = []types.Pattern{
		{
			Glob:    "*.txt",
			DestDir: "documents",
		},
	}
	engine.SetConfig(cfg)

	t.Run("move single file", func(t *testing.T) {
		destPath := filepath.Join(destDir, "test.txt")
		err := engine.MoveFile(testFile, destPath)
		assert.NoError(t, err, "Should successfully move file")

		// Verify file was moved
		_, err = os.Stat(destPath)
		assert.NoError(t, err, "File should exist at destination")
		_, err = os.Stat(testFile)
		assert.Error(t, err, "File should no longer exist at source")

		// Create the test file again for subsequent tests
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err, "Failed to recreate test file")
	})

	t.Run("organize file by pattern", func(t *testing.T) {
		// Test using pattern-based organization
		err := engine.OrganizeFile(testFile)
		assert.NoError(t, err, "Should successfully organize file")

		// Verify file was organized according to pattern
		destPath := filepath.Join(destDir, "test.txt")
		_, err = os.Stat(destPath)
		assert.NoError(t, err, "File should exist at destination")
		_, err = os.Stat(testFile)
		assert.Error(t, err, "File should no longer exist at source")

		// Create the test file again for subsequent tests
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err, "Failed to recreate test file")
	})

	t.Run("dry run should not move files", func(t *testing.T) {
		// Enable dry run
		engine.SetDryRun(true)

		err := engine.OrganizeFile(testFile)
		assert.NoError(t, err, "Dry run should complete without errors")

		// Verify file was NOT moved
		_, err = os.Stat(testFile)
		assert.NoError(t, err, "File should still exist at source after dry run")

		// Disable dry run for subsequent tests
		engine.SetDryRun(false)
	})

	t.Run("organize directory", func(t *testing.T) {
		// Create multiple test files
		testDir := filepath.Join(tmpDir, "source")
		require.NoError(t, os.MkdirAll(testDir, 0755), "Failed to create test directory")

		// Create test files
		testutils.CreateTestFiles(t, testDir)

		// Organize the directory
		organized, err := engine.OrganizeDir(testDir)
		assert.NoError(t, err, "Should organize directory without errors")
		assert.NotEmpty(t, organized, "Should return list of organized files")

		// Verify text files were moved to documents
		textFilePath := filepath.Join(destDir, "test1.txt")
		_, err = os.Stat(textFilePath)
		assert.NoError(t, err, "Text file should be moved to documents directory")
	})

	t.Run("handle collision", func(t *testing.T) {
		// Create a file that would cause a collision
		collisionSource := filepath.Join(tmpDir, "collision.txt")
		err := os.WriteFile(collisionSource, []byte("collision test"), 0644)
		require.NoError(t, err, "Failed to create collision source file")

		// Create the same file at destination
		collisionDest := filepath.Join(destDir, "collision.txt")
		err = os.WriteFile(collisionDest, []byte("existing file"), 0644)
		require.NoError(t, err, "Failed to create collision destination file")

		// Organize with rename collision strategy
		err = engine.OrganizeFile(collisionSource)
		assert.NoError(t, err, "Should handle collision according to strategy")

		// Original destination file should still exist
		_, err = os.Stat(collisionDest)
		assert.NoError(t, err, "Original destination file should still exist")

		// Source file should be moved to a renamed path
		_, err = os.Stat(filepath.Join(destDir, "collision_1.txt"))
		assert.NoError(t, err, "Renamed file should exist")
	})
}

// TestOrganizeWithConfig tests the organization engine with various configurations
func TestOrganizeWithConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some test files
	testutils.CreateTestFiles(t, tmpDir)

	// Test with different pattern configurations
	t.Run("organize with multiple patterns", func(t *testing.T) {
		engine := organize.New()
		cfg := &config.Config{}
		cfg.Settings.CreateDirs = true
		cfg.Settings.DryRun = false
		cfg.Organize.Patterns = []types.Pattern{
			{
				Glob:    "*.txt",
				DestDir: "text",
			},
			{
				Glob:    "*.jpg",
				DestDir: "images",
			},
			{
				Glob:    "*.pdf",
				DestDir: "documents",
			},
			{
				Glob:    "*.mp3",
				DestDir: "audio",
			},
		}
		engine.SetConfig(cfg)

		// Organize all files in the temp directory
		organized, err := engine.OrganizeDir(tmpDir)
		assert.NoError(t, err, "Should organize without errors")
		assert.Len(t, organized, 4, "Should organize 4 files")

		// Verify files were moved to appropriate directories
		_, err = os.Stat(filepath.Join(tmpDir, "text", "test1.txt"))
		assert.NoError(t, err, "Text file should be in text directory")

		_, err = os.Stat(filepath.Join(tmpDir, "images", "test2.jpg"))
		assert.NoError(t, err, "JPG file should be in images directory")

		_, err = os.Stat(filepath.Join(tmpDir, "documents", "test3.pdf"))
		assert.NoError(t, err, "PDF file should be in documents directory")

		_, err = os.Stat(filepath.Join(tmpDir, "audio", "test4.mp3"))
		assert.NoError(t, err, "MP3 file should be in audio directory")
	})
}
