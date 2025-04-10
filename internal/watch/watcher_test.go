package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatcherFsnotify(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create the watcher
	w, err := New()
	require.NoError(t, err, "New watcher creation failed")

	// Add the temporary directory to watch
	err = w.AddDirectory(tempDir)
	require.NoError(t, err, "Failed to add directory to watcher")

	// Start the watcher
	err = w.Start()
	require.NoError(t, err, "Failed to start watcher")
	defer w.Stop() // Ensure watcher is stopped even on test failure

	// Get the event channel immediately after starting
	evChan := w.FileChannel()
	require.NotNil(t, evChan, "Event channel should not be nil after start")

	// Allow a brief moment for fsnotify to initialize watches
	// This can sometimes be needed depending on the OS and load
	time.Sleep(100 * time.Millisecond)

	// --- Test File Creation ---
	testFilePath := filepath.Join(tempDir, "testfile.txt")
	t.Logf("Creating file: %s", testFilePath)
	file, err := os.Create(testFilePath)
	require.NoError(t, err, "Failed to create test file")
	err = file.Close()
	require.NoError(t, err, "Failed to close test file")

	// Expect a CREATE event
	select {
	case event, ok := <-evChan:
		require.True(t, ok, "Event channel closed unexpectedly during create test")
		t.Logf("Received event: %+v", event)
		assert.Equal(t, testFilePath, event.Path, "Event path mismatch")
		assert.True(t, event.Op.Has(fsnotify.Create), "Expected Create operation")
		require.NotNil(t, event.Info, "Event info should not be nil")
		assert.Equal(t, filepath.Base(testFilePath), event.Info.Name(), "Event info name mismatch")
	case <-time.After(3 * time.Second): // Increased timeout slightly
		t.Fatal("Timeout waiting for CREATE event")
	}

	// --- Test File Write ---
	t.Logf("Writing to file: %s", testFilePath)
	err = os.WriteFile(testFilePath, []byte("hello world"), 0644)
	require.NoError(t, err, "Failed to write to test file")

	// Expect a WRITE event (or potentially multiple events)
	foundWrite := false
	timeout := time.After(3 * time.Second) // Reset timeout
LoopWrite: // Label for breaking out of the loop
	for {
		select {
		case event, ok := <-evChan:
			require.True(t, ok, "Event channel closed unexpectedly during write test")
			t.Logf("Received event: %+v", event)
			if event.Path == testFilePath && event.Op.Has(fsnotify.Write) {
				assert.Equal(t, testFilePath, event.Path, "Event path mismatch")
				assert.True(t, event.Op.Has(fsnotify.Write), "Expected Write operation")
				require.NotNil(t, event.Info, "Event info should not be nil")
				assert.Equal(t, filepath.Base(testFilePath), event.Info.Name(), "Event info name mismatch")
				foundWrite = true
				// We found the write event we care about, but other events might follow.
				// Break inner select, could continue loop or break outer
				// break // just break select
				break LoopWrite // Exit loop once write is found
			} // else: could be other events (like CHMOD), ignore them for this specific check
		case <-timeout:
			// Only fail if we didn't find the write event within the timeout
			if !foundWrite {
				t.Fatal("Timeout waiting for WRITE event")
			}
			break LoopWrite // Exit loop on timeout
		}
	}
	assert.True(t, foundWrite, "Did not find the expected WRITE event")

	// --- Test Watcher Stop ---
	t.Log("Stopping watcher")
	w.Stop()

	// Drain any potentially remaining buffered events after stopping
	// This prevents the final check from failing on a buffered event
DrainLoop:
	for {
		select {
		case _, ok := <-evChan:
			if !ok {
				break DrainLoop // Channel closed and drained
			}
			// Consume/ignore the event
		default:
			// No more events buffered right now, channel might be closed or not.
			break DrainLoop
		}
	}

	// Verify the channel is definitely closed now by trying one more read
	// It might take a moment for the goroutine defer to run and close the channel
	select {
	case _, ok := <-evChan:
		assert.False(t, ok, "Event channel should be closed after stop")
	case <-time.After(1 * time.Second):
		// If we timeout here, it means Stop() didn't close the channel properly
		t.Error("Timeout waiting for event channel to close after stop")
	}
}
