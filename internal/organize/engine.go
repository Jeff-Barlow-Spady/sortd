package organize

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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

// IsDryRun returns whether the engine is in dry run mode
func (e *Engine) IsDryRun() bool {
	return e.dryRun
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

		// Construct the full destination path relative to the source file's directory
		sourceDir := filepath.Dir(filename)
		return filepath.Join(sourceDir, pattern.Target), true
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

	// Lock for thread safety when checking and potentially modifying destination
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check for dry run mode first
	if e.dryRun {
		log.Info("Would move %s -> %s", src, cleanDest)
		return nil
	}

	// Collision handling
	finalDest, err := e.handleCollision(cleanSrc, cleanDest)
	if err != nil {
		return err
	}
	// If finalDest is empty, it means we're skipping the move
	if finalDest == "" {
		return nil
	}

	// Create backup if needed
	if e.backup {
		if err := e.createBackup(finalDest); err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
	}

	// Move the file
	log.Debug("Moving %s to %s", cleanSrc, finalDest)
	if err := os.Rename(cleanSrc, finalDest); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	log.Info("Moved %s -> %s", src, finalDest)
	return nil
}

// handleCollision implements collision resolution strategies.
// It returns the final destination path and an error if any.
// If the file should be skipped, it returns an empty string and nil error.
func (e *Engine) handleCollision(src, dest string) (string, error) {
	// Check if destination already exists
	_, err := os.Stat(dest)
	if os.IsNotExist(err) {
		// No collision, use dest as is
		return dest, nil
	}
	if err != nil {
		// Some other error occurred
		return "", fmt.Errorf("error checking destination %s: %w", dest, err)
	}

	// Handle collision based on strategy
	log.Warn("Destination file %s already exists. Handling collision with strategy: %s", dest, e.collision)

	switch e.collision {
	case "skip":
		log.Info("Skipping move for %s due to collision (strategy: skip)", src)
		return "", nil // Empty string signals skip

	case "overwrite":
		log.Warn("Overwriting %s (strategy: overwrite)", dest)
		return dest, nil // Return original dest for overwriting

	case "rename":
		// Find a new name by incrementing counter
		return e.findUniqueDestName(dest)

	case "ask":
		// For now, skip when ask is specified
		log.Warn("Collision strategy 'ask' not implemented, treating as 'skip'")
		return "", nil

	default:
		return "", fmt.Errorf("unknown collision strategy: %s", e.collision)
	}
}

// findUniqueDestName finds a unique filename by adding counter to the basename
func (e *Engine) findUniqueDestName(originalPath string) (string, error) {
	ext := filepath.Ext(originalPath)
	base := strings.TrimSuffix(originalPath, ext)

	for counter := 1; counter <= 1000; counter++ {
		newName := fmt.Sprintf("%s_(%d)%s", base, counter, ext)

		if _, err := os.Stat(newName); os.IsNotExist(err) {
			log.Info("Renaming destination to %s due to collision (strategy: rename)", newName)
			return newName, nil
		}
	}

	return "", fmt.Errorf("failed to find unique name for %s after 1000 attempts", originalPath)
}

// createBackup creates a backup of the destination file if it exists
func (e *Engine) createBackup(dest string) error {
	// Check if file exists first
	_, err := os.Stat(dest)
	if os.IsNotExist(err) {
		// Nothing to backup
		return nil
	}
	if err != nil {
		return err
	}

	// Create backup with timestamp
	backupPath := fmt.Sprintf("%s.bak.%d", dest, time.Now().Unix())
	srcFile, err := os.Open(dest)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	log.Info("Created backup: %s", backupPath)
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

		// For each file, check all patterns
		for _, pattern := range e.patterns {
			// Check glob pattern
			matched, err := filepath.Match(pattern.Match, entry.Name())
			if err != nil || !matched {
				continue
			}

			// Create destination path
			// Here we need to handle the case where pattern.Target is an absolute path
			var destPath string
			if filepath.IsAbs(pattern.Target) {
				destPath = filepath.Join(pattern.Target, entry.Name())
			} else {
				destPath = filepath.Join(directory, pattern.Target, entry.Name())
			}

			// Create result object
			result := types.OrganizeResult{
				SourcePath:      filePath,
				DestinationPath: destPath,
			}

			// Try to move the file
			err = e.MoveFile(filePath, destPath)
			if err != nil {
				result.Error = err
			} else {
				result.Moved = !e.dryRun
			}

			// Add to results and break out of the pattern loop
			results = append(results, result)
			break
		}
	}

	return results, nil
}
