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

func TestNewDaemon(t *testing.T) {
	cfg := &config.Config{
		WatchDirectories: []string{"/tmp/test"},
		Settings: config.Settings{
			DryRun: false,
		},
	}

	// Use a temporary directory for workflows in this test to avoid interference
	tmpWorkflowsDir := t.TempDir()
	daemon, err := watch.NewDaemonWithWorkflowPath(cfg, tmpWorkflowsDir)
	if err != nil {
		t.Errorf("NewDaemon failed: %v", err)
	}

	if daemon == nil {
		t.Error("NewDaemon returned nil daemon")
	}
}

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
		{Match: "*.txt", Target: "../destdir"},
	}
	cfg.Settings.CreateDirs = true // Important for the test
	cfg.Settings.DryRun = false    // Ensure files are actually moved

	// 3. Initialize Daemon with an empty workflow directory
	tmpWorkflowsDir := t.TempDir()
	daemon, err := watch.NewDaemonWithWorkflowPath(cfg, tmpWorkflowsDir)
	require.NoError(t, err, "NewDaemon should not return an error")
	require.NotNil(t, daemon, "NewDaemon should not return a nil daemon")

	// Optional: Setup a callback for debugging or finer-grained assertions
	// movedChan := make(chan string, 1)
	// daemon.SetCallback(func(src, dest string, err error) {
	//  if err == nil && dest != "" {
	//   movedChan <- dest
	//  }
	// })

	// 4. Start Daemon
	err = daemon.Start()
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
	time.Sleep(500 * time.Millisecond) // Increased wait time for worker pool

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

func TestDaemon_WorkflowIntegration(t *testing.T) {
	// 1. Setup temporary directories
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "watchdir")
	destDir := filepath.Join(tmpDir, "destdir")
	workflowDestDir := filepath.Join(tmpDir, "workflow_dest")
	workflowsDir := filepath.Join(tmpDir, "workflows")

	// Create necessary directories
	require.NoError(t, os.Mkdir(watchDir, 0755))
	require.NoError(t, os.Mkdir(workflowsDir, 0755))

	// 2. Create a test workflow file
	workflowID := "test-workflow"
	workflowContent := []byte(`
id: "test-workflow"
name: "Test Workflow"
description: "Test workflow for integration testing"
enabled: true
priority: 10

trigger:
  type: "file_created"
  pattern: "*.log"

actions:
  - type: "move"
    target: "` + workflowDestDir + `"
    options:
      createTargetDir: "true"
      overwrite: "false"
`)

	workflowPath := filepath.Join(workflowsDir, workflowID+".yaml")
	require.NoError(t, os.WriteFile(workflowPath, workflowContent, 0644))

	// 3. Create config with a pattern for txt files and point workflow dir to our temp dir
	cfg := &config.Config{}
	cfg.WatchDirectories = []string{watchDir}
	cfg.Organize.Patterns = []types.Pattern{
		{Match: "*.txt", Target: "../destdir"},
	}
	cfg.Settings.CreateDirs = true
	cfg.Settings.DryRun = false

	// Set environment variables to override workflow directory (if your daemon checks for this)
	origHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", origHome) })
	os.Setenv("HOME", tmpDir)

	// 4. Create a custom init function for testing - this is needed to override the config path
	// We'll need to expose a testing helper in the daemon package
	daemon, err := watch.NewDaemonWithWorkflowPath(cfg, workflowsDir)
	require.NoError(t, err, "NewDaemonWithWorkflowPath should not return an error")
	require.NotNil(t, daemon, "NewDaemonWithWorkflowPath should not return a nil daemon")

	// 5. Start the daemon
	err = daemon.Start()
	require.NoError(t, err, "Daemon should start without error")
	defer daemon.Stop()

	// 6. Create two test files - one for each system
	// This one for the organize engine pattern
	txtFilePath := filepath.Join(watchDir, "test.txt")
	require.NoError(t, os.WriteFile(txtFilePath, []byte("test content for txt"), 0644))

	// This one for the workflow trigger
	logFilePath := filepath.Join(watchDir, "test.log")
	require.NoError(t, os.WriteFile(logFilePath, []byte("test content for log"), 0644))

	// 7. Wait for processing
	time.Sleep(500 * time.Millisecond)

	// 8. Check results for organize engine
	txtDestPath := filepath.Join(destDir, "test.txt")
	_, err = os.Stat(txtFilePath)
	assert.True(t, os.IsNotExist(err), "Original txt file should be moved")

	_, err = os.Stat(txtDestPath)
	assert.NoError(t, err, "Txt file should exist in destination")

	// 9. Check results for workflow
	logDestPath := filepath.Join(workflowDestDir, "test.log")
	_, err = os.Stat(logFilePath)
	assert.True(t, os.IsNotExist(err), "Original log file should be moved")

	_, err = os.Stat(logDestPath)
	assert.NoError(t, err, "Log file should exist in workflow destination")
}
