package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"sortd/internal/config"
	"sortd/internal/organize"
)

// Watcher watches directories for new files and organizes them
type Watcher struct {
	cfg          *config.Config
	organizer    *organize.Engine
	dirs         []string
	interval     time.Duration
	done         chan struct{}
	wg           sync.WaitGroup
	processMap   map[string]time.Time // Track processed files and their times
	mu           sync.Mutex           // Protects the processMap
	recursive    bool                 // Whether to watch directories recursively
	pendingFiles map[string]time.Time // Files pending confirmation
	pendingMu    sync.Mutex           // Protects the pendingFiles map
}

// NewWatcher creates a new directory watcher
func NewWatcher(cfg *config.Config, dirs []string, interval time.Duration) (*Watcher, error) {
	if len(dirs) == 0 {
		return nil, fmt.Errorf("no directories specified to watch")
	}

	// Validate directories exist
	for _, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			return nil, fmt.Errorf("cannot access directory %s: %w", dir, err)
		}
	}

	return &Watcher{
		cfg:          cfg,
		organizer:    organize.New(),
		dirs:         dirs,
		interval:     interval,
		done:         make(chan struct{}),
		processMap:   make(map[string]time.Time),
		pendingFiles: make(map[string]time.Time),
	}, nil
}

// Start begins watching the directories for changes
func (w *Watcher) Start() {
	w.wg.Add(1)
	defer w.wg.Done()

	// Configure the organizer with our settings
	//w.organizer.Configure(w.cfg)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Do an initial scan
	w.scanAndOrganize()

	for {
		select {
		case <-ticker.C:
			w.scanAndOrganize()
			// Process pending files that have passed their confirmation period
			if w.cfg != nil && w.cfg.WatchMode.ConfirmationPeriod > 0 {
				w.processPendingFiles()
			}
		case <-w.done:
			return
		}
	}
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	close(w.done)
	w.wg.Wait()
}

// processPendingFiles checks if any pending files have passed their confirmation period
// and processes them
func (w *Watcher) processPendingFiles() {
	now := time.Now()
	var filesToProcess []string

	w.pendingMu.Lock()
	for file, addedTime := range w.pendingFiles {
		confirmationTime := addedTime.Add(time.Duration(w.cfg.WatchMode.ConfirmationPeriod) * time.Second)
		if now.After(confirmationTime) {
			filesToProcess = append(filesToProcess, file)
			delete(w.pendingFiles, file)
		}
	}
	w.pendingMu.Unlock()

	if len(filesToProcess) > 0 {
		fmt.Printf("✨ Processing %d confirmed files\n", len(filesToProcess))

		// Configure the organizer with our settings
		if w.cfg != nil {
			w.organizer.SetConfig(w.cfg)
		}

		// Organize files using patterns
		err := w.organizer.OrganizeByPatterns(filesToProcess)
		if err != nil {
			fmt.Printf("❌ Error organizing files: %v\n", err)
		}
	}
}

// scanAndOrganize checks the watched directories for new files and organizes them
func (w *Watcher) scanAndOrganize() {
	fmt.Println("🔍 Scanning watched directories...")

	// Process each directory
	for _, dir := range w.dirs {
		// Skip if directory no longer exists
		if _, err := os.Stat(dir); err != nil {
			fmt.Printf("⚠️ Warning: Cannot access %s: %v\n", dir, err)
			continue
		}

		// Find files
		var newFiles []string

		if w.recursive {
			// Recursively walk the directory tree
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // Skip files we can't access
				}

				// Only process regular files
				if !info.IsDir() {
					// Check if this file is new or was modified since last check
					w.mu.Lock()
					lastModTime, exists := w.processMap[path]
					w.mu.Unlock()

					if !exists || info.ModTime().After(lastModTime) {
						newFiles = append(newFiles, path)

						// Update the processed map
						w.mu.Lock()
						w.processMap[path] = time.Now()
						w.mu.Unlock()
					}
				}
				return nil
			})

			if err != nil {
				fmt.Printf("⚠️ Error scanning directory %s: %v\n", dir, err)
			}
		} else {
			// Non-recursive mode: only check files in the top directory
			entries, err := os.ReadDir(dir)
			if err != nil {
				fmt.Printf("⚠️ Error reading directory %s: %v\n", dir, err)
				continue
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue // Skip subdirectories
				}

				path := filepath.Join(dir, entry.Name())
				info, err := entry.Info()
				if err != nil {
					continue // Skip if we can't get file info
				}

				// Check if this file is new or was modified since last check
				w.mu.Lock()
				lastModTime, exists := w.processMap[path]
				w.mu.Unlock()

				if !exists || info.ModTime().After(lastModTime) {
					newFiles = append(newFiles, path)

					// Update the processed map
					w.mu.Lock()
					w.processMap[path] = time.Now()
					w.mu.Unlock()
				}
			}
		}

		// Handle new files if we found any
		if len(newFiles) > 0 {
			fmt.Printf("✨ Found %d new files in %s\n", len(newFiles), dir)

			// Check if confirmation period is enabled
			if w.cfg != nil && (w.cfg.WatchMode.ConfirmationPeriod > 0 || w.cfg.WatchMode.RequireConfirmation) {
				// Add files to pending list
				w.pendingMu.Lock()
				for _, file := range newFiles {
					w.pendingFiles[file] = time.Now()
				}
				w.pendingMu.Unlock()

				if w.cfg.WatchMode.RequireConfirmation {
					fmt.Printf("⏳ Files queued for confirmation. Use 'sortd confirm' to approve.\n")
				} else {
					fmt.Printf("⏳ Files will be processed after %d seconds confirmation period.\n",
						w.cfg.WatchMode.ConfirmationPeriod)
				}
			} else {
				// Configure the organizer with our settings
				if w.cfg != nil {
					w.organizer.SetConfig(w.cfg)
				}

				// Organize files using patterns
				err := w.organizer.OrganizeByPatterns(newFiles)
				if err != nil {
					fmt.Printf("❌ Error organizing files: %v\n", err)
				}
			}
		}
	}
}

// SetRecursive sets whether to watch directories recursively
func (w *Watcher) SetRecursive(recursive bool) {
	w.recursive = recursive
}

// GetPendingFiles returns the list of files pending confirmation
func (w *Watcher) GetPendingFiles() []string {
	w.pendingMu.Lock()
	defer w.pendingMu.Unlock()

	var files []string
	for file := range w.pendingFiles {
		files = append(files, file)
	}
	return files
}

// ConfirmFile marks a file as confirmed for processing
func (w *Watcher) ConfirmFile(file string) bool {
	w.pendingMu.Lock()
	defer w.pendingMu.Unlock()

	_, exists := w.pendingFiles[file]
	if !exists {
		return false
	}

	// Set confirmation time to now minus confirmation period to process immediately
	w.pendingFiles[file] = time.Now().Add(-time.Duration(w.cfg.WatchMode.ConfirmationPeriod+1) * time.Second)
	return true
}

// ConfirmAllFiles marks all pending files as confirmed
func (w *Watcher) ConfirmAllFiles() int {
	w.pendingMu.Lock()
	defer w.pendingMu.Unlock()

	count := len(w.pendingFiles)
	for file := range w.pendingFiles {
		// Set confirmation time to now minus confirmation period to process immediately
		w.pendingFiles[file] = time.Now().Add(-time.Duration(w.cfg.WatchMode.ConfirmationPeriod+1) * time.Second)
	}
	return count
}

// RejectFile removes a file from the pending list without processing it
func (w *Watcher) RejectFile(file string) bool {
	w.pendingMu.Lock()
	defer w.pendingMu.Unlock()

	_, exists := w.pendingFiles[file]
	if !exists {
		return false
	}

	delete(w.pendingFiles, file)
	return true
}

// RejectAllFiles removes all files from the pending list
func (w *Watcher) RejectAllFiles() int {
	w.pendingMu.Lock()
	defer w.pendingMu.Unlock()

	count := len(w.pendingFiles)
	w.pendingFiles = make(map[string]time.Time)
	return count
}
