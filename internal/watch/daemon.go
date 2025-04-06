package watch

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"sortd/internal/config"
	"sortd/internal/log"
	"sortd/internal/organize"
)

// DaemonStatus represents the current status of the daemon
type DaemonStatus struct {
	Running          bool      // Whether the daemon is currently active
	WatchDirectories []string  // Directories being watched
	LastActivity     time.Time // Time of last file activity
	FilesProcessed   int       // Total files processed
}

// Daemon manages a background file organization service
type Daemon struct {
	// Configuration
	config *config.Config

	// The file watcher
	watcher *fsnotify.Watcher

	// Organize engine adapter
	engine *organize.Engine

	// Statistics
	processed    int
	lastActivity time.Time

	// Callback for when a file is processed
	callback func(string, string, error)

	// Lock for modifications
	mutex sync.RWMutex

	// Whether to require confirmation before organizing
	requireConfirmation bool

	// Whether the daemon is running
	running bool
}

// NewDaemon creates a new background file organization service
func NewDaemon(cfg *config.Config) *Daemon {
	// Create a watcher using fsnotify
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("Failed to create fsnotify watcher for daemon: %v", err)
		os.Exit(1) // Exit as log.Fatalf would
	}

	// Create the organization engine using the correct constructor
	engine := organize.NewWithConfig(cfg)

	return &Daemon{
		config:              cfg,
		watcher:             watcher,
		engine:              engine,
		processed:           0,
		lastActivity:        time.Now(), // Initialize lastActivity
		callback:            nil,
		requireConfirmation: false,
		running:             false,
	}
}

// Start initiates the daemon process
func (d *Daemon) Start() error {
	if d.running {
		return fmt.Errorf("daemon is already running")
	}

	// Add the watch directories from config
	// Use config.WatchDirectories instead of config.Directories.Watch
	if len(d.config.WatchDirectories) > 0 {
		for _, dir := range d.config.WatchDirectories {
			if err := d.watcher.Add(dir); err != nil {
				// Use internal logger for error
				log.Errorf("Error adding watch directory %s: %w", dir, err)
				// Decide if we should return the error or just log it
				// For now, return the error to prevent starting with incomplete watches
				return fmt.Errorf("error adding watch directory %s: %w", dir, err)
			}
			log.Info("Watching directory: %s", dir)
		}
	} else {
		log.Info("No watch directories specified in configuration.")
	}

	// Make sure we have directories to watch
	// Use WatchList() for fsnotify
	if len(d.watcher.WatchList()) == 0 {
		return fmt.Errorf("no valid directories to watch")
	}

	// fsnotify doesn't need an explicit Start(), it starts listening on NewWatcher()
	// Remove the invalid EnableRenames call

	// Start processing file events
	go d.processEvents()

	d.running = true
	log.Info("Watch daemon started.")

	return nil
}

// Stop halts the daemon process
func (d *Daemon) Stop() {
	if !d.running {
		return
	}

	// Stop the watcher
	if err := d.watcher.Close(); err != nil {
		log.Errorf("Error closing watcher: %v", err)
	}
	d.running = false
	log.Info("Watch daemon stopped.")
}

// AddWatchDirectory adds a directory to be watched
func (d *Daemon) AddWatchDirectory(dir string) error {
	err := d.watcher.Add(dir)
	if err != nil {
		log.Errorf("Error adding watch directory dynamically %s: %v", dir, err)
	} else {
		log.Info("Dynamically added watch directory: %s", dir)
	}
	return err
}

// SetCallback sets a function to be called when a file is processed
func (d *Daemon) SetCallback(cb func(string, string, error)) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.callback = cb
}

// SetRequireConfirmation sets whether the daemon should require
// confirmation before organizing files
func (d *Daemon) SetRequireConfirmation(require bool) {
	d.requireConfirmation = require
}

// SetDryRun sets whether to run in dry run mode
func (d *Daemon) SetDryRun(dryRun bool) {
	d.engine.SetDryRun(dryRun)
}

// Status returns the current status of the daemon
func (d *Daemon) Status() DaemonStatus {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return DaemonStatus{
		Running:          d.running,
		// Use WatchList() for fsnotify
		WatchDirectories: d.watcher.WatchList(),
		LastActivity:     d.lastActivity,
		FilesProcessed:   d.processed,
	}
}

// processEvents handles file modification events from the watcher
func (d *Daemon) processEvents() {
	for {
		select {
		case event, ok := <-d.watcher.Events:
			if !ok {
				log.Info("Watcher event channel closed.")
				return // Exit goroutine if channel is closed
			}

			// Log the raw event for debugging
			log.Debugf("Received fsnotify event: %s", event.String())

			// We are primarily interested in Create and Write events for files
			// Note: RENAMED files trigger REMOVE on old name, CREATE on new name.
			// WRITE might occur multiple times for one save operation.
			if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
				// Check if it's a file (fsnotify doesn't guarantee IsDir reliably)
				info, err := os.Stat(event.Name)
				if err != nil {
					// File might have been removed quickly after event, log and skip
					log.Debugf("Failed to stat file from event %s: %v", event.Name, err)
					continue
				}
				if info.IsDir() {
					log.Debugf("Skipping directory event: %s", event.Name)
					continue // Skip directories
				}

				// Update last activity time
				d.mutex.Lock()
				d.lastActivity = time.Now()
				d.mutex.Unlock()

				// Process the file
				log.Debugf("Processing file event for: %s", event.Name)
				d.organizeFile(event.Name)
			}

		case err, ok := <-d.watcher.Errors:
			if !ok {
				log.Info("Watcher error channel closed.")
				return // Exit goroutine if channel is closed
			}
			log.Errorf("Watcher error: %v", err)
		}
	}
}

// organizeFile processes a single file according to the rules
func (d *Daemon) organizeFile(filePath string) {
	log.Debugf("Organize task triggered for: %s", filePath)

	// Use OrganizeByPatterns which returns only an error
	err := d.engine.OrganizeByPatterns([]string{filePath})

	// If error occurred during organization (including no pattern match implicitly? Check engine impl if needed)
	if err != nil {
		log.Errorf("Error organizing file %s: %v", filePath, err)
		// Execute callback with the error
		if d.callback != nil {
			d.mutex.RLock()
			cb := d.callback
			d.mutex.RUnlock()
			// Pass empty destPath as organization failed or didn't happen
			cb(filePath, "", err)
		}
		return
	}

	// If no error, organization was successful (or file was skipped by engine logic)
	// Update stats assuming a move happened if no error (Might need refinement if engine skips silently)
	d.mutex.Lock()
	d.processed++
	d.mutex.Unlock()

	log.Info("Successfully organized file: %s (or skipped by engine rules)", filePath)

	// If a callback is registered, notify it of success (nil error)
	// We don't know the exact destination path from OrganizeByPatterns easily.
	// We could try to find it again, but for now, pass empty string.
	if d.callback != nil {
		d.mutex.RLock()
		cb := d.callback
		d.mutex.RUnlock()
		cb(filePath, "", nil) // Indicate success with nil error, empty dest path
	}
}

// OrganizeFile can be called to manually organize a file through the daemon
func (d *Daemon) OrganizeFile(filePath string) (string, error) {
	log.Debugf("Manual organize task triggered for: %s", filePath)

	// Delegate directly to the engine using OrganizeByPatterns
	err := d.engine.OrganizeByPatterns([]string{filePath})
	if err != nil {
		log.Errorf("Error during manual organization of %s: %v", filePath, err)
		return "", err // Return the engine error directly
	}

	// Update stats on success
	d.mutex.Lock()
	d.processed++
	d.mutex.Unlock()

	log.Info("Successfully manually organized file: %s", filePath)

	// We don't know the destination path easily from OrganizeByPatterns.
	// Return empty string for path and nil for error on success.
	return "", nil
}
