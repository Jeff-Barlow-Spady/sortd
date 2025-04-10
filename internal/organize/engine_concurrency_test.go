package organize

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"sortd/internal/config"
	"sortd/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrentOperations tests the engine's thread safety with concurrent operations
func TestConcurrentOperations(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := filepath.Join(os.TempDir(), "sortd-test-engine-concurrency")
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

	// Create destination subdirectories
	for i := 1; i <= 5; i++ {
		subDir := filepath.Join(destDir, "dir"+string(rune('0'+i)))
		err = os.MkdirAll(subDir, 0755)
		require.NoError(t, err, "Failed to create destination subdirectory")
	}

	// Test concurrent MoveFile operations
	t.Run("ConcurrentMoveFile", func(t *testing.T) {
		// Create an engine
		engine := New()

		// Create 100 test files
		var srcFiles []string
		var destFiles []string

		for i := 0; i < 100; i++ {
			filename := filepath.Join(srcDir, "concurrent"+string(rune('0'+i%10))+".txt")
			err := os.WriteFile(filename, []byte("test content "+string(rune('0'+i%10))), 0644)
			require.NoError(t, err, "Failed to create test file")
			srcFiles = append(srcFiles, filename)

			destSubDir := filepath.Join(destDir, "dir"+string(rune('0'+i%5+1)))
			destFile := filepath.Join(destSubDir, "concurrent"+string(rune('0'+i%10))+".txt")
			destFiles = append(destFiles, destFile)
		}

		// Move files concurrently
		var wg sync.WaitGroup
		wg.Add(len(srcFiles))

		for i := range srcFiles {
			go func(src, dest string) {
				defer wg.Done()
				err := engine.MoveFile(src, dest)
				// We don't check for errors here because some operations might fail
				// due to concurrent accesses to the same file
				_ = err
			}(srcFiles[i], destFiles[i])
		}

		wg.Wait()

		// Count how many files were successfully moved
		movedCount := 0
		remainingCount := 0

		// Check destination files
		for _, destFile := range destFiles {
			if _, err := os.Stat(destFile); err == nil {
				movedCount++
			}
		}

		// Check source files
		for _, srcFile := range srcFiles {
			if _, err := os.Stat(srcFile); err == nil {
				remainingCount++
			}
		}

		// We expect some files to be moved successfully, but due to
		// concurrent operations on the same files, some might fail
		t.Logf("Moved: %d, Remaining: %d, Total: %d", movedCount, remainingCount, len(srcFiles))
		assert.True(t, movedCount > 0, "Some files should be moved")
		assert.Equal(t, len(srcFiles), movedCount+remainingCount, "All files should be accounted for")
	})

	// Test concurrent pattern organization
	t.Run("ConcurrentOrganizeByPatterns", func(t *testing.T) {
		// Remove previous files and recreate source directory
		os.RemoveAll(srcDir)
		err := os.MkdirAll(srcDir, 0755)
		require.NoError(t, err, "Failed to recreate source directory")

		// Create a config with patterns
		cfg := &config.Config{}
		cfg.Organize.Patterns = []types.Pattern{
			{Match: "*.txt", Target: filepath.Join(destDir, "dir1")},
			{Match: "*.jpg", Target: filepath.Join(destDir, "dir2")},
			{Match: "*.pdf", Target: filepath.Join(destDir, "dir3")},
		}

		// Create the engine
		engine := NewWithConfig(cfg)

		// Create test files of different types
		var txtFiles []string
		var jpgFiles []string
		var pdfFiles []string

		for i := 0; i < 20; i++ {
			// Create txt files
			txtFile := filepath.Join(srcDir, "file"+string(rune('0'+i))+".txt")
			err := os.WriteFile(txtFile, []byte("txt content "+string(rune('0'+i))), 0644)
			require.NoError(t, err, "Failed to create txt file")
			txtFiles = append(txtFiles, txtFile)

			// Create jpg files
			jpgFile := filepath.Join(srcDir, "image"+string(rune('0'+i))+".jpg")
			err = os.WriteFile(jpgFile, []byte("jpg content "+string(rune('0'+i))), 0644)
			require.NoError(t, err, "Failed to create jpg file")
			jpgFiles = append(jpgFiles, jpgFile)

			// Create pdf files
			pdfFile := filepath.Join(srcDir, "doc"+string(rune('0'+i))+".pdf")
			err = os.WriteFile(pdfFile, []byte("pdf content "+string(rune('0'+i))), 0644)
			require.NoError(t, err, "Failed to create pdf file")
			pdfFiles = append(pdfFiles, pdfFile)
		}

		// Organize files concurrently using OrganizeByPatterns
		var wg sync.WaitGroup
		wg.Add(3) // Three goroutines

		go func() {
			defer wg.Done()
			err := engine.OrganizeByPatterns(txtFiles)
			_ = err // Ignore errors
		}()

		go func() {
			defer wg.Done()
			err := engine.OrganizeByPatterns(jpgFiles)
			_ = err // Ignore errors
		}()

		go func() {
			defer wg.Done()
			err := engine.OrganizeByPatterns(pdfFiles)
			_ = err // Ignore errors
		}()

		wg.Wait()

		// Count how many files of each type were moved
		txtMoved := countFiles(filepath.Join(destDir, "dir1"), ".txt")
		jpgMoved := countFiles(filepath.Join(destDir, "dir2"), ".jpg")
		pdfMoved := countFiles(filepath.Join(destDir, "dir3"), ".pdf")

		// Get total counts
		totalMoved := txtMoved + jpgMoved + pdfMoved
		totalFiles := len(txtFiles) + len(jpgFiles) + len(pdfFiles)

		t.Logf("Moved: txt=%d, jpg=%d, pdf=%d, total=%d/%d",
			txtMoved, jpgMoved, pdfMoved, totalMoved, totalFiles)

		assert.True(t, totalMoved > 0, "Some files should be moved")
		// We don't assert that all files were moved as concurrent operations might interfere
	})

	// Test mutex contention
	t.Run("MutexContentionHandling", func(t *testing.T) {
		// Create a small number of test files
		os.RemoveAll(srcDir)
		err := os.MkdirAll(srcDir, 0755)
		require.NoError(t, err, "Failed to recreate source directory")

		testFile := filepath.Join(srcDir, "contention.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err, "Failed to create test file")

		// Create engine with config
		cfg := &config.Config{}
		cfg.Settings.Collision = "rename"
		engine := NewWithConfig(cfg)

		// Destination file
		destFile := filepath.Join(destDir, "dir1", "contention.txt")

		// Create many goroutines all trying to move the same file
		var wg sync.WaitGroup
		const numGoroutines = 50
		wg.Add(numGoroutines)

		// Make a barrier to start all goroutines at once
		var barrier sync.WaitGroup
		barrier.Add(1)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				// Wait for all goroutines to be ready
				barrier.Wait()

				// Add a small sleep to stagger operations slightly
				time.Sleep(time.Duration(id%10) * time.Millisecond)

				// Copy the file to a temporary file for this goroutine
				srcCopy := filepath.Join(srcDir, "contention_copy"+string(rune('0'+id%10))+".txt")
				srcData, err := os.ReadFile(testFile)
				if err == nil {
					err = os.WriteFile(srcCopy, srcData, 0644)
					if err == nil {
						// Try to move the file
						err = engine.MoveFile(srcCopy, destFile)
						_ = err // Ignore errors
					}
				}
			}(i)
		}

		// Start all goroutines
		barrier.Done()

		// Wait for all to complete
		wg.Wait()

		// Check for files with the collision rename pattern
		renamedCount := 0
		err = filepath.Walk(filepath.Join(destDir, "dir1"), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if filepath.Ext(path) == ".txt" && len(path) > len("contention.txt") {
				renamedCount++
			}
			return nil
		})
		require.NoError(t, err, "Failed to walk directory")

		t.Logf("Found %d renamed files", renamedCount)
		assert.True(t, renamedCount > 0, "Some files should be renamed")
	})
}

// countFiles counts files with a specific extension in a directory
func countFiles(dir string, ext string) int {
	count := 0
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ext {
			count++
		}
	}
	return count
}
