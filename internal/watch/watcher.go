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
	cfg        *config.Config
	organizer  *organize.Engine
	dirs       []string
	interval   time.Duration
	done       chan struct{}
	wg         sync.WaitGroup
	processMap map[string]time.Time // Track processed files and their times
	mu         sync.Mutex           // Protects the processMap
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
		cfg:        cfg,
		organizer:  organize.New(),
		dirs:       dirs,
		interval:   interval,
		done:       make(chan struct{}),
		processMap: make(map[string]time.Time),
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
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files we can't access
			}

			// Only process regular files
			if !info.IsDir() {
				// Check if this file is new or was modified since last check
				w.mu.Lock()
				lastSeen, exists := w.processMap[path]
				w.mu.Unlock()

				if !exists || info.ModTime().After(lastSeen) {
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
			fmt.Printf("⚠️ Error scanning %s: %v\n", dir, err)
			continue
		}

		// Organize new files if we found any
		if len(newFiles) > 0 {
			fmt.Printf("✨ Found %d new files in %s\n", len(newFiles), dir)

			// Organize files
			//results, err := w.organizer.OrganizeFiles(newFiles)
			//if err != nil {
			//	fmt.Printf("❌ Error organizing files: %v\n", err)
			//	continue
			//}

			// Print results
			//fmt.Printf("📊 Files processed: %d\n", len(results))
		}
	}
}
