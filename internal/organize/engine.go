package organize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"sortd/internal/config"
	"sortd/internal/log"
	"sortd/pkg/types"
)

// Engine handles file organization operations
type Engine struct {
	files      map[string]types.FileInfo
	patterns   []types.Pattern
	dryRun     bool
	mu         sync.RWMutex // Protects files map
	createDirs bool
	backup     bool
	collision  string
	config     *config.Config
}

func (e *Engine) SetConfig(cfg *config.Config) {
	e.patterns = cfg.Organize.Patterns
	e.createDirs = cfg.Settings.CreateDirs
	e.backup = cfg.Settings.Backup
	e.collision = cfg.Settings.Collision
	e.config = cfg
}

func (e *Engine) OrganizeFile(path string) error {
	if e.config == nil {
		return fmt.Errorf("no config set")
	}

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Find matching pattern
	for _, pattern := range e.config.Organize.Patterns {
		matched, err := filepath.Match(pattern.Match, info.Name())
		if err != nil {
			return err
		}
		if matched {
			// Create target directory if needed
			targetDir := filepath.Join(filepath.Dir(path), pattern.Target)
			if e.config.Settings.CreateDirs {
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					return err
				}
			}

			// Move file
			newPath := filepath.Join(targetDir, info.Name())
			return os.Rename(path, newPath)
		}
	}

	return nil
}

// New creates a new Organization Engine instance
func New() *Engine {
	return &Engine{}
}

// NewWithConfig creates a new Organization Engine instance with configuration
func NewWithConfig(cfg *config.Config) *Engine {
	return &Engine{
		files:      make(map[string]types.FileInfo),
		patterns:   cfg.Organize.Patterns,
		dryRun:     cfg.Settings.DryRun,
		createDirs: cfg.Settings.CreateDirs,
		backup:     cfg.Settings.Backup,
		collision:  cfg.Settings.Collision,
		config:     cfg,
	}
}

// SetDryRun sets whether operations should be performed or just simulated
func (e *Engine) SetDryRun(dryRun bool) {
	e.dryRun = dryRun
}

// AddPattern adds a new organization pattern
func (e *Engine) AddPattern(pattern types.Pattern) {
	e.patterns = append(e.patterns, pattern)
	log.Debug("Added pattern: match=%s, target=%s", pattern.Match, pattern.Target)
}

// findDestination determines where a file should go based on patterns
func (e *Engine) findDestination(filename string) (string, bool) {
	for _, pattern := range e.patterns {
		// Check glob pattern
		matched, err := filepath.Match(pattern.Match, filepath.Base(filename))
		if err != nil || !matched {
			continue
		}

		return pattern.Target, true
	}
	return "", false
}

// MoveFile moves a file from source to destination, handling collisions based on config.
func (e *Engine) MoveFile(src, dest string) error {
	// Clean paths for comparison
	cleanSrc := filepath.Clean(src)
	cleanDest := filepath.Clean(dest)

	// Check for same file
	if cleanSrc == cleanDest {
		// Moving to the same place is not an error, just do nothing.
		log.Debug("Source and destination are the same, skipping: %s", src)
		return nil
	}

	// Verify source exists and get info
	srcInfo, err := os.Stat(cleanSrc)
	if err != nil {
		return fmt.Errorf("source file error: %w", err)
	}
	if srcInfo.IsDir() {
		return fmt.Errorf("cannot move directory as file: %s", src)
	}

	// Ensure destination directory exists before checking destination file
	destDir := filepath.Dir(cleanDest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// --- Collision Handling ---
	collisionStrategy := e.collision // Get from engine config
	if _, err := os.Stat(cleanDest); err == nil {
		// Destination exists, handle collision
		log.Warn("Destination file %s already exists. Handling collision with strategy: %s", cleanDest, collisionStrategy)
		switch collisionStrategy {
		case "skip":
			log.Info("Skipping move for %s due to collision (strategy: skip)", cleanSrc)
			return nil // Not an error, just skipped
		case "overwrite":
			log.Warn("Overwriting %s (strategy: overwrite)", cleanDest)
			// Proceed to rename
		case "rename":
			originalDest := cleanDest
			counter := 1
			for {
				ext := filepath.Ext(originalDest)
				base := strings.TrimSuffix(originalDest, ext)
				newName := fmt.Sprintf("%s_(%d)%s", base, counter, ext)
				if _, err := os.Stat(newName); os.IsNotExist(err) {
					cleanDest = newName // Found a non-existent name
					log.Info("Renaming destination to %s due to collision (strategy: rename)", cleanDest)
					break
				}
				counter++
				if counter > 1000 { // Safety break
					return fmt.Errorf("failed to find unique name for %s after 1000 attempts", originalDest)
				}
			}
			// Proceed to rename with the new cleanDest
		case "ask":
			// TODO: Implement interactive 'ask' functionality. Requires CLI interaction.
			// For now, treat 'ask' like 'skip'
			log.Warn("Collision strategy 'ask' not implemented, treating as 'skip'.")
			return nil
		default:
			return fmt.Errorf("unknown collision strategy: %s", collisionStrategy)
		}
	} else if !os.IsNotExist(err) {
		// Error other than "does not exist" when checking destination
		return fmt.Errorf("error checking destination %s: %w", cleanDest, err)
	}
	// Destination does not exist or collision handled (overwrite/rename)

	// Lock for thread safety (should this be around the whole operation?)
	// e.mu.Lock()
	// defer e.mu.Unlock()
	// TODO: Review locking strategy for potential race conditions, especially with 'rename'

	if e.dryRun {
		log.Info("Would move %s -> %s", src, cleanDest)
		return nil
	}

	// --- Backup Logic (Optional) ---
	if e.backup {
		// TODO: Implement backup logic if enabled
		log.Debug("Backup logic not implemented.")
	}

	// --- Move the file ---
	log.Debug("Attempting to move %s to %s", cleanSrc, cleanDest)
	if err := os.Rename(cleanSrc, cleanDest); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	// Record the move (Consider if locking is needed here)
	// e.files[cleanDest] = types.FileInfo{ ... }
	log.Info("Moved %s -> %s", src, cleanDest)

	return nil
}

// OrganizeFiles moves multiple files to a destination directory
func (e *Engine) OrganizeFiles(files []string, destDir string) error {
	for _, file := range files {
		dest := filepath.Join(destDir, filepath.Base(file))
		if err := e.MoveFile(file, dest); err != nil {
			return fmt.Errorf("failed to move %s: %w", file, err)
		}
	}
	return nil
}

// OrganizeByPatterns organizes files according to defined patterns
func (e *Engine) OrganizeByPatterns(files []string) error {
	log.Info("Organizing %d files using patterns", len(files))
	for _, file := range files {
		if destDir, found := e.findDestination(file); found {
			dest := filepath.Join(destDir, filepath.Base(file))
			if err := e.MoveFile(file, dest); err != nil {
				return fmt.Errorf("failed to move %s: %w", file, err)
			}
		} else {
			log.Debug("No pattern match for file: %s", file)
		}
	}
	return nil
}

// Add directory organization method
func (e *Engine) OrganizeDir(dir string) ([]string, error) {
	var organized []string
	// Implementation here
	return organized, nil
}

// OrganizeDirectory organizes all files in a directory according to the configured patterns
func (e *Engine) OrganizeDirectory(directory string) ([]types.OrganizeResult, error) {
	var results []types.OrganizeResult

	// Check if directory exists
	dirInfo, err := os.Stat(directory)
	if err != nil {
		return nil, fmt.Errorf("error accessing directory: %w", err)
	}

	if !dirInfo.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", directory)
	}

	// Read directory contents
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %w", err)
	}

	// Process each file
	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		// Get full file path
		filePath := filepath.Join(directory, entry.Name())

		// Find destination using patterns
		destDir, found := e.findDestination(filePath)
		if !found {
			// Skip files that don't match any pattern
			continue
		}

		// Build destination path
		destPath := filepath.Join(destDir, entry.Name())

		// Create result object
		result := types.OrganizeResult{
			SourcePath:      filePath,
			DestinationPath: destPath,
		}

		// Try to move the file
		err := e.MoveFile(filePath, destPath)
		if err != nil {
			result.Error = err
		} else {
			result.Moved = !e.dryRun
		}

		// Add to results
		results = append(results, result)
	}

	return results, nil
}
