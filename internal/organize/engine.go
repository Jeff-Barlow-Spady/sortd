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
	"sortd/internal/errors"
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

func (e *Engine) OrganizeFile(path string) error {
	logger := log.LogWithFields(log.F("path", path))

	if e.config == nil {
		return errors.NewConfigError("no config set", "engine", errors.ConfigNotSet, nil)
	}

	// Delegate to OrganizeByPatterns for consistency
	logger.Debug("Organizing single file using pattern engine")
	err := e.OrganizeByPatterns([]string{path})
	if err != nil {
		// OrganizeByPatterns already logs errors for individual files, but we might want
		// to log the overall failure here if the wrapper returns an error.
		logger.With(log.F("error", err)).Error("Failed to organize single file")
		return err // Propagate the error from OrganizeByPatterns
	}

	// If OrganizeByPatterns returns nil, it means either the file was moved successfully
	// or no matching pattern was found (which isn't an error for a single file call).
	// The specific outcome (moved/no match) would already be logged by sub-calls.
	logger.Debug("Single file organization attempt completed")
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
	log.Debugf("Added pattern: match=%s, target=%s", pattern.Match, pattern.Target)
}

// findDestination determines where a file should go based on patterns
func (e *Engine) findDestination(filename string) (string, bool) {
	logger := log.LogWithFields(log.F("file", filename))

	for _, pattern := range e.patterns {
		// Check glob pattern
		matched, err := filepath.Match(pattern.Match, filepath.Base(filename))
		if err != nil {
			logger.With(
				log.F("pattern", pattern.Match),
				log.F("error", err.Error()),
			).Warn("Invalid pattern, skipping")
			continue
		}

		if !matched {
			continue
		}

		// Return the pattern target as is - path joining will be handled in the calling function
		// This allows proper handling of both absolute and relative paths
		logger.With(
			log.F("pattern", pattern.Match),
			log.F("target", pattern.Target),
		).Debug("Pattern matched")

		return pattern.Target, true
	}

	logger.Debug("No matching pattern found")
	return "", false
}

// MoveFile moves a file from source to destination, handling collisions based on config.
func (e *Engine) MoveFile(src, dest string) error {
	logger := log.LogWithFields(
		log.F("source", src),
		log.F("destination", dest),
		log.F("dry_run", e.dryRun),
	)

	// Clean paths for comparison
	cleanSrc := filepath.Clean(src)
	cleanDest := filepath.Clean(dest)

	// Check for same file
	if cleanSrc == cleanDest {
		// Moving to the same place is not an error, just do nothing.
		logger.Debug("Source and destination are the same, skipping")
		return nil
	}

	// Verify source exists and get info
	srcInfo, err := os.Stat(cleanSrc)
	if err != nil {
		return errors.NewFileError("source file error", cleanSrc, errors.FileAccessDenied, err)
	}
	if srcInfo.IsDir() {
		return errors.NewFileError("cannot move directory as file", cleanSrc, errors.InvalidOperation, nil)
	}

	// Check if destination directory exists
	destDir := filepath.Dir(cleanDest)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		// If createDirs is false, return an error
		if !e.createDirs {
			return errors.NewFileError("destination directory does not exist", destDir, errors.FileAccessDenied, nil)
		}

		// Create directory if createDirs is true
		if !e.dryRun {
			if err := os.MkdirAll(destDir, 0755); err != nil {
				return errors.NewFileError("failed to create destination directory", destDir, errors.FileCreateFailed, err)
			}
		}
	} else if err != nil {
		return errors.NewFileError("error checking destination directory", destDir, errors.FileAccessDenied, err)
	}

	// Check for dry run mode first
	if e.dryRun {
		logger.Info("Would move file (dry run)")
		return nil
	}

	// Determine final destination path with collision handling
	// This needs to be atomic with the actual move operation
	e.mu.Lock()
	finalDest, err := e.handleCollision(cleanSrc, cleanDest)
	e.mu.Unlock()

	if err != nil {
		log.LogError(err, "Collision handling failed")
		return err
	}

	// If finalDest is empty, it means we're skipping the move
	if finalDest == "" {
		logger.Info("Skipping file move due to collision handling")
		return nil
	}

	// Create backup if needed
	// Check if the destination file exists before moving/overwriting
	if e.backup {
		// Check if destination file exists directly, rather than relying on finalDest path
		// which may have been renamed by collision handling
		if _, err := os.Stat(finalDest); err == nil {
			// File exists, create backup
			if err := e.createBackup(finalDest); err != nil {
				return errors.Wrap(err, "backup failed")
			}
		}
	}

	// Move the file
	logger.With(log.F("final_destination", finalDest)).Debug("Moving file")
	if err := os.Rename(cleanSrc, finalDest); err != nil {
		return errors.NewFileError("failed to move file", cleanSrc, errors.FileOperationFailed, err)
	}

	logger.With(log.F("final_destination", finalDest)).Info("Moved file successfully")
	return nil
}

// handleCollision implements collision resolution strategies.
// It returns the final destination path and an error if any.
// If the file should be skipped, it returns an empty string and nil error.
func (e *Engine) handleCollision(src, dest string) (string, error) {
	logger := log.LogWithFields(
		log.F("source", src),
		log.F("destination", dest),
		log.F("strategy", e.collision),
	)

	// Check if destination already exists
	_, err := os.Stat(dest)
	if os.IsNotExist(err) {
		// No collision, use dest as is
		return dest, nil
	}
	if err != nil {
		// Some other error occurred
		return "", errors.NewFileError("error checking destination", dest, errors.FileAccessDenied, err)
	}

	// Handle collision based on strategy
	logger.Warn("Destination file already exists, handling collision")

	// Use default strategy if none is specified
	collisionStrategy := e.collision
	if collisionStrategy == "" {
		collisionStrategy = "skip" // Use skip as the default
		logger.Info("Using default collision strategy: skip")
	}

	switch collisionStrategy {
	case "skip":
		logger.Info("Skipping move due to collision")
		return "", nil // Empty string signals skip

	case "overwrite":
		logger.Warn("Overwriting destination file")
		return dest, nil // Return original dest for overwriting

	case "rename":
		// Find a new name by incrementing counter
		return e.findUniqueDestName(dest)

	case "ask":
		// For now, skip when ask is specified
		logger.Warn("Collision strategy 'ask' not implemented, treating as 'skip'")
		return "", nil

	default:
		return "", errors.NewConfigError("unknown collision strategy: "+collisionStrategy, collisionStrategy, errors.InvalidConfig, nil)
	}
}

// findUniqueDestName finds a unique filename by adding counter to the basename
func (e *Engine) findUniqueDestName(originalPath string) (string, error) {
	logger := log.LogWithFields(log.F("original_path", originalPath))

	ext := filepath.Ext(originalPath)
	base := strings.TrimSuffix(originalPath, ext)

	for counter := 1; counter <= 1000; counter++ {
		newName := fmt.Sprintf("%s_(%d)%s", base, counter, ext)

		// Check if this name exists
		_, err := os.Stat(newName)
		if os.IsNotExist(err) {
			// Found a name that doesn't exist
			logger.With(log.F("new_name", newName)).Debug("Found unique destination name")
			return newName, nil
		}
	}
	logger.Warn("Could not find unique filename after 1000 attempts")
	return "", errors.New("couldn't find a unique name after 1000 attempts")
}

// createBackup creates a backup of a file before moving/overwriting it
func (e *Engine) createBackup(dest string) error {
	// Check if file exists first
	_, err := os.Stat(dest)
	if os.IsNotExist(err) {
		// Nothing to backup
		return nil
	}
	if err != nil {
		return errors.NewFileError("failed to check file for backup", dest, errors.FileAccessDenied, err)
	}

	// Create backup directory if it doesn't exist
	backupDir := filepath.Dir(dest)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return errors.NewFileError("failed to create backup directory", backupDir, errors.FileCreateFailed, err)
	}

	// Create backup with timestamp and nanosecond precision for uniqueness
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	baseBackupPath := fmt.Sprintf("%s.bak.%d", dest, timestamp)
	backupPath := baseBackupPath

	// Check if the backup path already exists and add counter if needed
	counter := 1
	for {
		_, err := os.Stat(backupPath)
		if os.IsNotExist(err) {
			break // Found a unique name
		}

		// Add counter to ensure uniqueness
		backupPath = fmt.Sprintf("%s_%d", baseBackupPath, counter)
		counter++

		// Safety check to prevent infinite loop
		if counter > 1000 {
			return errors.New("couldn't find a unique backup name after 1000 attempts")
		}
	}

	srcFile, err := os.Open(dest)
	if err != nil {
		return errors.NewFileError("failed to open source file for backup", dest, errors.FileAccessDenied, err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(backupPath)
	if err != nil {
		return errors.NewFileError("failed to create backup file", backupPath, errors.FileCreateFailed, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return errors.NewFileError("failed to copy file content to backup", dest, errors.FileOperationFailed, err)
	}

	log.Info("Created backup: %s", backupPath)
	return nil
}

// OrganizeFiles moves multiple files to a destination directory
func (e *Engine) OrganizeFiles(files []string, destDir string) error {
	logger := log.LogWithFields(
		log.F("dest_dir", destDir),
		log.F("file_count", len(files)),
	)
	logger.Info("Organizing files to destination directory")

	for _, file := range files {
		dest := filepath.Join(destDir, filepath.Base(file))
		if err := e.MoveFile(file, dest); err != nil {
			return errors.Wrapf(err, "failed to move %s", file)
		}
	}
	return nil
}

// OrganizeByPatterns organizes files according to defined patterns
func (e *Engine) OrganizeByPatterns(files []string) error {
	logger := log.LogWithFields(log.F("file_count", len(files)))
	logger.Info("Organizing files using patterns")
	var firstError error // Keep track of the first error encountered

	for _, file := range files {
		if destDir, found := e.findDestination(file); found {
			// Construct proper destination path - use absolute path if destDir is absolute
			var dest string
			if filepath.IsAbs(destDir) {
				dest = filepath.Join(destDir, filepath.Base(file))
			} else {
				// For relative paths, join with the source file's directory
				dest = filepath.Join(filepath.Dir(file), destDir, filepath.Base(file))
			}

			if err := e.MoveFile(file, dest); err != nil {
				wrappedErr := errors.Wrapf(err, "failed to move %s to %s", file, dest)
				log.LogError(wrappedErr, "Error during pattern organization") // Log the specific error
				if firstError == nil {
					firstError = wrappedErr // Store the first error
				}
				// Continue processing other files even if one fails
				continue
			}
		} else {
			log.LogWithFields(log.F("file", file)).Debug("No pattern match for file")
		}
	}

	// Return the first error encountered, if any
	if firstError != nil {
		// Add additional context about the overall operation
		return errors.Wrap(firstError, "pattern-based organization had errors")
	}
	return nil
}

// OrganizeDir organizes all files in a directory according to patterns
// and returns a list of successfully organized files
func (e *Engine) OrganizeDir(dir string) ([]string, error) {
	logger := log.LogWithFields(log.F("directory", dir))
	logger.Info("Organizing directory using patterns")

	// Check if directory exists first
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewFileError("directory not found", dir, errors.FileNotFound, err)
		}
		return nil, errors.NewFileError("failed to access directory", dir, errors.FileAccessDenied, err)
	}

	results, err := e.OrganizeDirectory(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to organize directory '%s'", dir)
	}

	// Convert the results to a simple list of organized files
	var organized []string
	for _, result := range results {
		if result.Error == nil && result.Moved {
			organized = append(organized, result.SourcePath)
		}
	}

	logger.With(log.F("organized_count", len(organized))).Info("Directory organization complete")
	return organized, nil
}

// OrganizeDirectory organizes all files in a directory according to the configured patterns
func (e *Engine) OrganizeDirectory(directory string) ([]types.OrganizeResult, error) {
	logger := log.LogWithFields(log.F("directory", directory))
	var results []types.OrganizeResult

	// Check if directory exists
	dirInfo, err := os.Stat(directory)
	if err != nil {
		return nil, errors.NewFileError("failed to access directory", directory, errors.FileAccessDenied, err)
	}

	if !dirInfo.IsDir() {
		return nil, errors.NewFileError("path is not a directory", directory, errors.InvalidOperation, nil)
	}

	// Read directory contents
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, errors.NewFileError("failed to read directory", directory, errors.FileAccessDenied, err)
	}

	logger.With(log.F("file_count", len(entries))).Info("Organizing directory")

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

			// Create destination path, handling both absolute and relative paths
			var destPath string
			if filepath.IsAbs(pattern.Target) {
				// If target is absolute, use it directly
				destPath = filepath.Join(pattern.Target, entry.Name())
			} else {
				// If target is relative, join with the source directory
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
