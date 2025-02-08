package organize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/toasty/sortd/internal/log"
	"github.com/toasty/sortd/pkg/types"
)

// Engine handles file organization operations
type Engine struct {
	files    map[string]types.FileInfo
	patterns []types.Pattern
	dryRun   bool
	mu       sync.RWMutex // Protects files map
}

// New creates a new Organization Engine instance
func New() *Engine {
	return &Engine{
		files:    make(map[string]types.FileInfo),
		patterns: make([]types.Pattern, 0),
		dryRun:   false,
	}
}

// SetDryRun sets whether operations should be performed or just simulated
func (e *Engine) SetDryRun(dryRun bool) {
	e.dryRun = dryRun
}

// AddPattern adds a new organization pattern
func (e *Engine) AddPattern(pattern types.Pattern) {
	e.patterns = append(e.patterns, pattern)
	log.Debug("Added pattern: glob=%s, dest=%s", pattern.Glob, pattern.DestDir)
}

// findDestination determines where a file should go based on patterns
func (e *Engine) findDestination(filename string) (string, bool) {
	for _, pattern := range e.patterns {
		// Check glob pattern
		matched, err := filepath.Match(pattern.Glob, filepath.Base(filename))
		if err != nil || !matched {
			continue
		}

		// Check prefixes
		name := strings.ToLower(filepath.Base(filename))
		for _, prefix := range pattern.Prefixes {
			if strings.HasPrefix(name, strings.ToLower(prefix)) {
				log.Debug("File %s matched prefix %s", filename, prefix)
				return pattern.DestDir, true
			}
		}

		// Check suffixes
		for _, suffix := range pattern.Suffixes {
			if strings.HasSuffix(strings.TrimSuffix(name, filepath.Ext(name)), strings.ToLower(suffix)) {
				log.Debug("File %s matched suffix %s", filename, suffix)
				return pattern.DestDir, true
			}
		}

		// If no prefixes/suffixes defined, glob match is enough
		if len(pattern.Prefixes) == 0 && len(pattern.Suffixes) == 0 {
			log.Debug("File %s matched glob %s", filename, pattern.Glob)
			return pattern.DestDir, true
		}
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
