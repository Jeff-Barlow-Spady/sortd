package watch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sortd/internal/config"
	"sync"
	"time"
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
	watcher *Watcher

	// Organize engine adapter
	engine *EngineAdapter

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
	// Create a watcher
	watcher, err := New()
	if err != nil {
		log.Fatalf("Failed to create watcher for daemon: %v", err)
	}

	// Create the organization engine adapter
	engine := NewEngineAdapter(cfg)

	return &Daemon{
		config:              cfg,
		watcher:             watcher,
		engine:              engine,
		processed:           0,
		lastActivity:        time.Now(),
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
	if len(d.config.Directories.Watch) > 0 {
		for _, dir := range d.config.Directories.Watch {
			if err := d.watcher.AddDirectory(dir); err != nil {
				return fmt.Errorf("error adding watch directory %s: %w", dir, err)
			}
		}
	}

	// Make sure we have directories to watch
	if len(d.watcher.GetDirectories()) == 0 {
		return fmt.Errorf("no directories to watch")
	}

	// Start the watcher
	if err := d.watcher.Start(); err != nil {
		return fmt.Errorf("error starting watcher: %w", err)
	}

	d.running = true

	// Start processing file events
	go d.processEvents()

	return nil
}

// Stop halts the daemon process
func (d *Daemon) Stop() {
	if !d.running {
		return
	}

	// Stop the watcher
	d.watcher.Stop()
	d.running = false
}

// AddWatchDirectory adds a directory to be watched
func (d *Daemon) AddWatchDirectory(dir string) error {
	return d.watcher.AddDirectory(dir)
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
		WatchDirectories: d.watcher.GetDirectories(),
		LastActivity:     d.lastActivity,
		FilesProcessed:   d.processed,
	}
}

// processEvents handles file modification events from the watcher
func (d *Daemon) processEvents() {
	for fileEvent := range d.watcher.FileChannel() {
		// Skip directories
		if fileEvent.Info.IsDir() {
			continue
		}

		// Update last activity time
		d.mutex.Lock()
		d.lastActivity = fileEvent.Timestamp
		d.mutex.Unlock()

		// Process the file
		d.organizeFile(fileEvent.Path)
	}
}

// organizeFile processes a single file according to the rules
func (d *Daemon) organizeFile(filePath string) {
	destPattern, _, ruleMatched := d.findMatchingRule(filePath)

	// If no rule matched, skip this file
	if !ruleMatched {
		return
	}

	// Calculate destination path
	destPath := filepath.Join(destPattern, filepath.Base(filePath))

	var err error

	// If we require confirmation and have a callback, invoke it
	if d.requireConfirmation && d.callback != nil {
		// We pass an empty error here as a signal that this is a confirmation request
		d.mutex.RLock()
		cb := d.callback
		d.mutex.RUnlock()

		cb(filePath, destPath, nil)

		// The callback should handle the actual organization
		return
	}

	// Otherwise, organize the file directly
	dryRun := d.engine.GetDryRun()
	if dryRun {
		// In dry run mode, just log what would happen
		log.Printf("Would organize %s to %s", filePath, destPath)
	} else {
		// Move the file
		err = d.engine.MoveFile(filePath, destPath)

		// Update stats
		if err == nil {
			d.mutex.Lock()
			d.processed++
			d.mutex.Unlock()
		}
	}

	// If a callback is registered, notify it
	if d.callback != nil {
		d.mutex.RLock()
		cb := d.callback
		d.mutex.RUnlock()

		cb(filePath, destPath, err)
	}
}

// findMatchingRule finds the destination path for a file based on rules
func (d *Daemon) findMatchingRule(filePath string) (string, string, bool) {
	// Get just the filename
	fileName := filepath.Base(filePath)

	// Check against each rule
	for _, rule := range d.config.Rules {
		matched, err := filepath.Match(rule.Pattern, fileName)
		if err == nil && matched {
			return rule.Target, rule.Pattern, true
		}
	}

	return "", "", false
}

// OrganizeFile can be called to manually organize a file through the daemon
func (d *Daemon) OrganizeFile(filePath string) (string, error) {
	destPattern, _, ruleMatched := d.findMatchingRule(filePath)

	// If no rule matched, return an error
	if !ruleMatched {
		return "", fmt.Errorf("no matching rule found for %s", filePath)
	}

	// Calculate destination path
	destPath := filepath.Join(destPattern, filepath.Base(filePath))

	// Create destination directory if it doesn't exist
	dryRun := d.engine.GetDryRun()
	if !dryRun && d.config.Settings.CreateDirs {
		os.MkdirAll(destPattern, 0755)
	}

	var err error

	if !dryRun {
		// Move the file
		err = d.engine.MoveFile(filePath, destPath)

		// Update stats
		if err == nil {
			d.mutex.Lock()
			d.processed++
			d.mutex.Unlock()
		}
	}

	return destPath, err
}
