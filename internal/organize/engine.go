package organize

import (
	"fmt"
	"os"
	"path/filepath"
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

// MoveFile moves a file from source to destination with safety checks
func (e *Engine) MoveFile(src, dest string) error {
	// Clean paths for comparison
	cleanSrc := filepath.Clean(src)
	cleanDest := filepath.Clean(dest)

	// Check for same file
	if cleanSrc == cleanDest {
		return fmt.Errorf("source and destination are the same file: %s", src)
	}

	// Verify source exists and get info
	srcInfo, err := os.Stat(cleanSrc)
	if err != nil {
		return fmt.Errorf("source file error: %w", err)
	}

	if srcInfo.IsDir() {
		return fmt.Errorf("cannot move directory as file: %s", src)
	}

	// Check if destination exists on filesystem
	if _, err := os.Stat(cleanDest); err == nil {
		return fmt.Errorf("destination file already exists on filesystem: %s", dest)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking destination: %w", err)
	}

	// Lock for thread safety
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if destination is tracked in our files map
	if _, exists := e.files[cleanDest]; exists {
		return fmt.Errorf("destination file already exists in tracking: %s", dest)
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(cleanDest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	if e.dryRun {
		log.Info("Would move %s -> %s", src, dest)
		return nil
	}

	// Move the file
	if err := os.Rename(cleanSrc, cleanDest); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	// Record the move
	e.files[cleanDest] = types.FileInfo{
		Path: cleanDest,
		Size: srcInfo.Size(),
	}
	log.Info("Moved %s -> %s", src, dest)

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
