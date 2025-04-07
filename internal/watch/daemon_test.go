package watch_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"sortd/internal/config"
	"sortd/internal/watch"
	"sortd/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDaemon_BasicFileMove(t *testing.T) {
	// 1. Setup temporary directories
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "watchdir")
	destDir := filepath.Join(tmpDir, "destdir")
	require.NoError(t, os.Mkdir(watchDir, 0755))
	// destDir will be created by the engine if CreateDirs is true

	// 2. Create config
	cfg := &config.Config{}
	cfg.WatchDirectories = []string{watchDir}
	cfg.Organize.Patterns = []types.Pattern{
		{Match: "*.txt", Target: destDir},
	}
	cfg.Settings.CreateDirs = true // Important for the test
	cfg.Settings.DryRun = false     // Ensure files are actually moved

	// 3. Initialize Daemon
	daemon := watch.NewDaemon(cfg)

	// Optional: Setup a callback for debugging or finer-grained assertions
	// movedChan := make(chan string, 1)
	// daemon.SetCallback(func(src, dest string, err error) {
	//  if err == nil && dest != "" {
	//   movedChan <- dest
	//  }
	// })

	// 4. Start Daemon
	err := daemon.Start()
	require.NoError(t, err, "Daemon should start without error")
	defer daemon.Stop() // Ensure daemon is stopped even on test failure

	// 5. Create a test file
	testFileName := "testfile.txt"
	testFilePath := filepath.Join(watchDir, testFileName)
	err = os.WriteFile(testFilePath, []byte("test content"), 0644)
	require.NoError(t, err, "Failed to create test file")

	// 6. Wait for processing
	// Give fsnotify and the daemon goroutine time to process.
	// This is a common challenge in testing async file watchers.
	// A more robust approach might use the callback and channel (commented out above).
	time.Sleep(200 * time.Millisecond) // Adjust timing as needed

	// 7. Check if file was moved
	destFilePath := filepath.Join(destDir, testFileName)
	_, err = os.Stat(testFilePath)
	assert.Error(t, err, "Original file should no longer exist in watchDir")
	assert.True(t, os.IsNotExist(err), "Error should be os.IsNotExist")

	_, err = os.Stat(destFilePath)
	assert.NoError(t, err, "File should exist in destDir")

	// Optional: Assert using callback channel
	// select {
	// case movedPath := <-movedChan:
	//  assert.Equal(t, destFilePath, movedPath, "Callback should report correct destination")
	// case <-time.After(50 * time.Millisecond): // Add a small extra timeout
	//  t.Fatal("Did not receive move confirmation from callback")
	// }

	// 8. Stop is handled by defer
}
