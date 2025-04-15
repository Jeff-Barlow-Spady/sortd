package watch

import (
	"os"
	"path/filepath"
	"sortd/internal/config"
	"testing"
)

// TestProgressTracking tests the file operation functionality in the watch package
func TestProgressTracking(t *testing.T) {
	// Create a config with dry run for safety
	cfg := &config.Config{
		Settings: config.Settings{
			DryRun: true,
		},
	}

	// Create an adapter with the real logger
	adapter := NewEngineAdapter(cfg)

	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup in-memory learning engine
	err = adapter.SetupLearningEngine(":memory:")
	if err != nil {
		t.Fatalf("Failed to setup learning engine: %v", err)
	}

	// Perform a move operation - this logs the progress via the real logger
	destPath := filepath.Join(tempDir, "dest", "test.txt")
	err = adapter.MoveFile(testFile, destPath)
	if err != nil {
		t.Fatalf("Failed to move file: %v", err)
	}

	// Verify destination directory was created (even in dry run)
	destDir := filepath.Dir(destPath)
	dirInfo, err := os.Stat(destDir)
	if err != nil {
		t.Fatalf("Destination directory was not created: %v", err)
	}
	if !dirInfo.IsDir() {
		t.Error("Created path is not a directory")
	}

	// Test without dry run to verify actual moves
	adapter.SetDryRun(false)

	// Create another test file
	testFile2 := filepath.Join(tempDir, "test2.txt")
	err = os.WriteFile(testFile2, []byte("more test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create second test file: %v", err)
	}

	// Move the file for real this time
	destPath2 := filepath.Join(tempDir, "dest", "test2.txt")
	err = adapter.MoveFile(testFile2, destPath2)
	if err != nil {
		t.Fatalf("Failed to move second file: %v", err)
	}

	// Verify the file was actually moved
	_, err = os.Stat(testFile2)
	if !os.IsNotExist(err) {
		t.Error("Source file still exists after move")
	}

	_, err = os.Stat(destPath2)
	if err != nil {
		t.Errorf("Destination file doesn't exist after move: %v", err)
	}
}
