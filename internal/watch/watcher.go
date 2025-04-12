package watch

import (
	"fmt"
	"os"
	"sync"
	"time"

	"sortd/internal/log"

	"github.com/fsnotify/fsnotify"
)

// FileModification represents a file event detected by the watcher
type FileModification struct {
	Path      string
	Info      os.FileInfo
	Timestamp time.Time
	Op        fsnotify.Op
}

// Watcher monitors directories for file changes using fsnotify
type Watcher struct {
	// Directories being watched
	directories []string

	// Channel to receive file modifications
	fileModChan chan FileModification

	// Channel to signal stop
	stopChan chan struct{}

	// fsnotify watcher instance
	fsWatcher *fsnotify.Watcher

	// Lock for running state and potentially directories list if modified concurrently
	mutex sync.RWMutex

	// Whether the watcher is running
	running bool
}

// New creates a new directory watcher using fsnotify
func New() (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	return &Watcher{
		directories: []string{},
		fileModChan: make(chan FileModification, 10),
		stopChan:    make(chan struct{}),
		fsWatcher:   fsWatcher,
		running:     false,
	}, nil
}

// AddDirectory adds a directory to watch using fsnotify
func (w *Watcher) AddDirectory(dir string) error {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("error accessing directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	// Add directory to fsnotify watcher
	err = w.fsWatcher.Add(dir)
	if err != nil {
		return fmt.Errorf("failed to add directory %s to watcher: %w", dir, err)
	}

	// Keep track of directories added (optional, but useful for GetDirectories)
	w.mutex.Lock()
	// Check if already present to avoid duplicates in the list (fsnotify handles duplicates itself)
	found := false
	for _, existingDir := range w.directories {
		if existingDir == dir {
			found = true
			break
		}
	}
	if !found {
		w.directories = append(w.directories, dir)
	}
	w.mutex.Unlock()
	log.LogWithFields(log.F("directory", dir)).Info("Watching directory")
	return nil
}

// FileChannel returns the channel that delivers file modification events
func (w *Watcher) FileChannel() <-chan FileModification {
	return w.fileModChan
}

// Start begins the file watching process using fsnotify
func (w *Watcher) Start() error {
	w.mutex.Lock()
	if w.running {
		w.mutex.Unlock()
		return fmt.Errorf("watcher already running")
	}
	w.running = true
	w.mutex.Unlock()

	// Create a new stop channel each time Start is called
	w.stopChan = make(chan struct{})

	// Start the event processing loop in a separate goroutine
	go func() {
		log.Info("Watcher event loop started.")

		for {
			select {
			case event, ok := <-w.fsWatcher.Events:
				if !ok {
					log.Info("fsWatcher.Events channel closed")
					return // Channel closed
				}

				// Ignore directory events for now, handle file creation/write
				// Checking existence is crucial, event might be for a deleted file
				if event.Op.Has(fsnotify.Create) || event.Op.Has(fsnotify.Write) {
					info, err := os.Stat(event.Name)
					if err != nil {
						// File might have been quickly deleted after event, or it's a dir event we can ignore
						if !os.IsNotExist(err) {
							log.LogWithFields(log.F("file", event.Name), log.F("error", err)).Error("Error stating file")
						}
						continue // Skip this event
					}

					// Ensure it's not a directory change event if we don't want those
					if info.IsDir() {
						continue
					}

					mod := FileModification{
						Path:      event.Name,
						Info:      info,
						Timestamp: time.Now(),
						Op:        event.Op,
					}

					// Send event non-blockingly to avoid goroutine getting stuck if channel full
					select {
					case w.fileModChan <- mod:
						// Successfully sent modification to channel
					default:
						log.LogWithFields(log.F("file", event.Name)).Warn("Event channel is full, dropped event")
					}
				}

			case err, ok := <-w.fsWatcher.Errors:
				if !ok {
					log.Info("fsWatcher.Errors channel closed")
					return // Channel closed
				}
				log.LogWithFields(log.F("error", err)).Error("fsnotify watcher error")

			case <-w.stopChan:
				log.Info("Watcher event loop received stop signal.")
				return // Exit goroutine
			}
		}
	}()

	log.Info("Watcher started.")
	return nil
}

// Stop halts the file watching process
func (w *Watcher) Stop() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.running {
		return // Already stopped
	}

	// Signal the event processing goroutine to stop
	close(w.stopChan)

	// Close the underlying fsnotify watcher
	if err := w.fsWatcher.Close(); err != nil {
		log.LogWithFields(log.F("error", err)).Error("Error closing fsnotify watcher")
	}

	w.running = false

	// Close the public event channel after stopping everything else
	// Do this under the lock to prevent races with FileChannel()
	close(w.fileModChan)

	log.Info("Watcher stopped.")
}

// IsRunning returns whether the watcher is currently active
func (w *Watcher) IsRunning() bool {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.running
}

// GetDirectories returns the list of directories being watched
func (w *Watcher) GetDirectories() []string {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	dirsCopy := make([]string, len(w.directories))
	copy(dirsCopy, w.directories)
	return dirsCopy
}
