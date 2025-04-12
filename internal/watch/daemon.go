package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"sortd/internal/log"

	"github.com/fsnotify/fsnotify"

	"sortd/internal/config"
	"sortd/internal/errors"
	"sortd/internal/patterns/learning"
	"sortd/pkg/workflow"
)

// DaemonStatus represents the status of the watch daemon
type DaemonStatus struct {
	Running          bool
	WatchDirectories []string
	LastActivity     time.Time
	FilesProcessed   int
}

// Daemon manages a background file organization service
type Daemon struct {
	// Configuration
	config *config.Config

	// The file watcher
	watcher *fsnotify.Watcher

	// Organize engine adapter
	engine *EngineAdapter

	// Workflow manager for advanced file processing
	workflowManager *workflow.Manager

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

	// Event processing channel and workers
	eventChan  chan string
	workerWg   sync.WaitGroup
	numWorkers int
}

// NewDaemon creates a new background file organization service
func NewDaemon(cfg *config.Config) (*Daemon, error) {
	// Create a watcher using fsnotify
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		// Return the error instead of logging and exiting here
		return nil, errors.Wrap(err, "failed to create fsnotify watcher")
	}

	// Create the engine adapter
	engineAdapter := NewEngineAdapter(cfg)

	// Initialize the workflow manager
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user home directory")
	}

	// Use a fixed path under the .config/sortd directory for workflows
	workflowsDir := filepath.Join(home, ".config", "sortd", "workflows")

	// Create workflows directory if it doesn't exist
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		return nil, errors.NewFileError("failed to create workflows directory", workflowsDir, errors.FileCreateFailed, err)
	}

	// Initialize workflow manager
	workflowManager, err := workflow.NewManager(workflowsDir)
	if err != nil {
		log.LogWithFields(log.F("error", err)).Warn("Failed to initialize workflow manager")
		// Continue without workflow manager - don't fail the daemon initialization
		workflowManager = nil
	}

	return &Daemon{
		config:              cfg,
		watcher:             watcher,
		engine:              engineAdapter,
		workflowManager:     workflowManager,
		processed:           0,
		lastActivity:        time.Now(), // Initialize lastActivity
		callback:            nil,
		requireConfirmation: false,
		running:             false,
		eventChan:           make(chan string, 100), // Buffer for 100 events
		numWorkers:          4,                      // Default to 4 workers
	}, nil // Return nil error on success
}

// Start initiates the daemon process
func (d *Daemon) Start() error {
	if d.running {
		return errors.New("daemon is already running")
	}

	// Add the watch directories from config
	// Use config.WatchDirectories instead of config.Directories.Watch
	if len(d.config.WatchDirectories) > 0 {
		for _, dir := range d.config.WatchDirectories {
			if err := d.watcher.Add(dir); err != nil {
				// Use the config path for context in the error message
				log.LogWithFields(log.F("directory", dir), log.F("error", err)).Error("Error adding watch directory")
				// For now, return the error to prevent starting with incomplete watches
				return errors.NewFileError("error adding watch directory", dir, errors.FileAccessDenied, err)
			}
			log.LogWithFields(log.F("directory", dir)).Info("Watching directory")
		}
	} else {
		log.Info("No watch directories specified in configuration.")
	}

	// Make sure we have directories to watch
	// Use WatchList() for fsnotify
	if len(d.watcher.WatchList()) == 0 {
		return errors.New("no valid directories to watch")
	}

	// Start worker pool for file processing
	for i := 0; i < d.numWorkers; i++ {
		d.workerWg.Add(1)
		go d.fileProcessWorker()
	}

	// Start processing file events from the single watcher
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

	// Stop the main watcher
	if err := d.watcher.Close(); err != nil {
		log.LogWithFields(log.F("error", err)).Error("Error closing watcher")
	}

	// Close the event channel to signal workers to stop
	close(d.eventChan)

	// Wait for all workers to finish
	d.workerWg.Wait()

	d.running = false
	log.Info("Watch daemon stopped.")
}

// fileProcessWorker processes files from the event channel
func (d *Daemon) fileProcessWorker() {
	defer d.workerWg.Done()

	for filePath := range d.eventChan {
		// First try workflow processing
		var workflowHandled bool = false
		if d.workflowManager != nil {
			// Create a minimal event to pass to the workflow manager
			event := fsnotify.Event{
				Name: filePath,
				Op:   fsnotify.Create, // Treat as a create event
			}

			processed, wfErr := d.workflowManager.ProcessEvent(event)
			if wfErr != nil {
				log.LogWithFields(log.F("file", filePath), log.F("error", wfErr)).Error("Error processing event with workflow manager")
				// Decide if error means we should still try patterns. For now, assume yes.
			}
			if processed {
				log.LogWithFields(log.F("file", filePath)).Debug("Event was handled by a workflow")
				workflowHandled = true
				// Explicitly skip pattern processing if workflow handled it
				continue
			}
		}

		// If no workflow handled it, try config patterns
		if !workflowHandled {
			log.LogWithFields(log.F("file", filePath)).Debug("Event not handled by workflow, trying config patterns")
			d.organizeFile(filePath)
		}
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
			log.LogWithFields(log.F("event", event.String())).Debug("Received fsnotify event")

			// We are primarily interested in Create and Write events for files
			// Note: RENAMED files trigger REMOVE on old name, CREATE on new name.
			// WRITE might occur multiple times for one save operation.
			if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
				// Check if it's a file (fsnotify doesn't guarantee IsDir reliably)
				info, err := os.Stat(event.Name)
				if err != nil {
					// File might have been removed quickly after event, log and skip
					log.LogWithFields(log.F("file", event.Name), log.F("error", err)).Debug("Failed to stat file from event")
					continue
				}
				if info.IsDir() {
					log.LogWithFields(log.F("directory", event.Name)).Debug("Skipping directory event")
					continue // Skip directories
				}

				// Update last activity time
				d.mutex.Lock()
				d.lastActivity = time.Now()
				d.mutex.Unlock()

				// Send file to worker pool for processing
				select {
				case d.eventChan <- event.Name:
					log.LogWithFields(log.F("file", event.Name)).Debug("Queued event for processing")
				default:
					log.LogWithFields(log.F("file", event.Name)).Warn("Event channel full, dropping event")
				}
			}

		case err, ok := <-d.watcher.Errors:
			if !ok {
				log.Info("Watcher error channel closed.")
				return // Exit goroutine if channel is closed
			}
			log.LogWithFields(log.F("error", err)).Error("Watcher error")
		}
	}
}

// AddWatchDirectory adds a directory to be watched
func (d *Daemon) AddWatchDirectory(dir string) error {
	err := d.watcher.Add(dir)
	if err != nil {
		log.LogWithFields(log.F("directory", dir), log.F("error", err)).Error("Error adding watch directory dynamically")
		return err
	}

	log.LogWithFields(log.F("directory", dir)).Info("Dynamically added watch directory")

	return nil
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

	// Also set dry run mode for workflow manager if available
	if d.workflowManager != nil {
		d.workflowManager.SetDryRun(dryRun)
	}
}

// Status returns the current status of the daemon
func (d *Daemon) Status() DaemonStatus {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return DaemonStatus{
		Running: d.running,
		// Use WatchList() for fsnotify
		WatchDirectories: d.watcher.WatchList(),
		LastActivity:     d.lastActivity,
		FilesProcessed:   d.processed,
	}
}

// organizeFile processes a single file according to the rules
func (d *Daemon) organizeFile(filePath string) {
	log.LogWithFields(log.F("file", filePath)).Debug("Attempting to organize file via config patterns")

	// Use OrganizeByPatterns which returns only an error
	err := d.engine.OrganizeByPatterns([]string{filePath})
	log.LogWithFields(log.F("file", filePath), log.F("error", err)).Debug("Result from engine.OrganizeByPatterns")

	// If error occurred during organization (including no pattern match implicitly? Check engine impl if needed)
	if err != nil {
		log.LogWithFields(log.F("file", filePath), log.F("error", err)).Error("Error organizing file")
		// Execute callback with the error
		d.mutex.RLock()
		cb := d.callback
		d.mutex.RUnlock()
		if cb != nil {
			// Pass empty destPath as organization failed or didn't happen
			log.LogWithFields(log.F("file", filePath), log.F("error", err)).Debug("Invoking callback for")
			cb(filePath, "", err)
		}
		return
	}

	// If no error, organization was successful (or file was skipped by engine logic)
	// Update stats assuming a move happened if no error (Might need refinement if engine skips silently)
	d.mutex.Lock()
	d.processed++
	d.mutex.Unlock()

	log.LogWithFields(log.F("file", filePath)).Info("Successfully organized file (or skipped by engine rules)")

	// If a callback is registered, notify it of success (nil error)
	// We don't know the exact destination path from OrganizeByPatterns easily.
	// We could try to find it again, but for now, pass empty string.
	d.mutex.RLock()
	cb := d.callback
	d.mutex.RUnlock()
	if cb != nil {
		log.LogWithFields(log.F("file", filePath)).Debug("Invoking callback for")
		cb(filePath, "", nil) // Indicate success with nil error, empty dest path
	}
}

// OrganizeFile can be called to manually organize a file through the daemon
func (d *Daemon) OrganizeFile(filePath string) (string, error) {
	log.LogWithFields(log.F("file", filePath)).Debug("Manual organize task triggered for")

	// First check if the file exists
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return "", errors.NewFileError("file not found", filePath, errors.FileNotFound, err)
		}
		return "", errors.NewFileError("error accessing file", filePath, errors.FileAccessDenied, err)
	}

	// Delegate directly to the engine using OrganizeByPatterns
	err := d.engine.OrganizeByPatterns([]string{filePath})
	if err != nil {
		log.LogWithFields(log.F("file", filePath), log.F("error", err)).Error("Error during manual organization")
		return "", errors.Wrapf(err, "failed to organize file %s", filePath)
	}

	// Update stats on success
	d.mutex.Lock()
	d.processed++
	d.mutex.Unlock()

	log.LogWithFields(log.F("file", filePath)).Info("Successfully manually organized file")

	// We don't know the destination path easily from OrganizeByPatterns.
	// Return empty string for path and nil for error on success.
	return "", nil
}

// NewDaemonWithWorkflowPath creates a new daemon with a custom workflow path
func NewDaemonWithWorkflowPath(cfg *config.Config, workflowPath string) (*Daemon, error) {
	// Create a watcher using fsnotify
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fsnotify watcher")
	}

	// Create the engine adapter
	engineAdapter := NewEngineAdapter(cfg)

	// Initialize workflow manager with the custom path
	workflowManager, err := workflow.NewManager(workflowPath)
	if err != nil {
		log.LogWithFields(log.F("error", err), log.F("path", workflowPath)).Warn("Failed to initialize workflow manager")
		// Continue without workflow manager
		workflowManager = nil
	}

	return &Daemon{
		config:              cfg,
		watcher:             watcher,
		engine:              engineAdapter,
		workflowManager:     workflowManager,
		processed:           0,
		lastActivity:        time.Now(),
		callback:            nil,
		requireConfirmation: false,
		running:             false,
		eventChan:           make(chan string, 100),
		numWorkers:          4,
	}, nil
}

// SetupLearningEngine initializes the smart rule learning engine
func (d *Daemon) SetupLearningEngine(dbPath string) error {
	if d.engine == nil {
		return fmt.Errorf("engine adapter not initialized")
	}

	// Create a logger for the learning engine
	logger := log.LogWithFields(log.F("component", "learning_engine"))

	// Set up learning config
	learningConfig := learning.DefaultConfig()
	learningConfig.DatabasePath = dbPath

	// Initialize database connection and repository
	repo, err := learning.NewSQLiteRepository(dbPath, logger)
	if err != nil {
		logger.With(log.F("error", err)).Error("Failed to initialize learning repository")
		return err
	}

	// Create the learning engine
	learningEngine := learning.NewEngine(repo, learningConfig, logger)

	// Set the learning engine in the adapter
	d.engine.SetLearningEngine(learningEngine)

	// Start the learning engine
	if err := learningEngine.Start(); err != nil {
		logger.With(log.F("error", err)).Error("Failed to start learning engine")
		return err
	}

	logger.Info("Content analysis and smart rule learning engine started successfully")
	return nil
}
