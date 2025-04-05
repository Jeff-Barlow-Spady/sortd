package analysis

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"sortd/internal/config"
	"sortd/pkg/types"
)

var (
	ErrFileNotFound      = errors.New("file not found")
	ErrPermissionDenied  = errors.New("permission denied")
	ErrInvalidFileType   = errors.New("invalid file type")
	ErrUnsupportedFormat = errors.New("unsupported file format")
	ErrInvalidInput      = errors.New("invalid input")
)

type ScanError struct {
	Path    string
	Err     error
	Context string
}

func (e *ScanError) Error() string {
	return fmt.Sprintf("analysis error at %s: %v (context: %s)", e.Path, e.Err, e.Context)
}

func (e *ScanError) Unwrap() error {
	return e.Err
}

// Engine handles file analysis and content detection
type Engine struct {
	config *config.Config
}

func (e *Engine) SetConfig(cfg *config.Config) {
	e.config = cfg
}

// New creates a new Analysis Engine instance
func New() *Engine {
	return &Engine{}
}

// NewWithConfig creates a new Analysis Engine instance with config settings
func NewWithConfig(cfg *config.Config) *Engine {
	return &Engine{}
}

// Scan performs basic file analysis
func (e *Engine) Scan(path string) (*types.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Open file for content type detection
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read first 512 bytes for content type detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	buffer = buffer[:n]

	// Detect content type
	contentType := http.DetectContentType(buffer)

	// For text files, try to be more specific
	if strings.HasPrefix(contentType, "text/plain") {
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".txt":
			contentType = "text/plain; charset=utf-8"
		case ".md":
			contentType = "text/markdown; charset=utf-8"
		case ".json":
			contentType = "application/json"
		case ".yaml", ".yml":
			contentType = "application/yaml"
		}
	}

	// Generate tags based on content type
	tags := make([]string, 0)
	if strings.HasPrefix(contentType, "text/") {
		tags = append(tags, "document")
	} else if strings.HasPrefix(contentType, "image/") {
		tags = append(tags, "image")
	} else if strings.HasPrefix(contentType, "video/") {
		tags = append(tags, "video")
	} else if strings.HasPrefix(contentType, "audio/") {
		tags = append(tags, "audio")
	}

	return &types.FileInfo{
		Path:        path,
		ContentType: contentType,
		Size:        info.Size(),
		Tags:        tags,
	}, nil
}

// Process performs file analysis with additional processing
func (e *Engine) Process(path string) (*types.FileInfo, error) {
	result, err := e.Scan(path)
	if err != nil {
		return nil, err
	}

	// Add additional processing for different file types
	switch {
	case strings.HasPrefix(result.ContentType, "image/"), strings.HasPrefix(result.ContentType, "application/octet-stream"):
		if !contains(result.Tags, "image") {
			result.Tags = append(result.Tags, "image")
		}
	case strings.HasPrefix(result.ContentType, "text/"):
		if !contains(result.Tags, "document") {
			result.Tags = append(result.Tags, "document")
		}
	}

	return result, nil
}

// ScanDirectory performs analysis on entries (files and dirs) in a single directory level.
func (e *Engine) ScanDirectory(dir string) ([]*types.FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Handle specific errors like permission denied or not a directory?
		return nil, fmt.Errorf("failed to read directory '%s': %w", dir, err)
	}

	var results []*types.FileInfo
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		var fileInfo *types.FileInfo
		var scanErr error

		if entry.IsDir() {
			// Create a FileInfo for the directory
			fileInfo = &types.FileInfo{
				Path:        path,
				ContentType: "inode/directory", // Convention for directories
				Size:        0,               // Directories don't have a size in this context
				Tags:        []string{"directory"}, // Add a 'directory' tag
			}
		} else {
			// It's a file, use the Scan method
			fileInfo, scanErr = e.Scan(path)
			if scanErr != nil {
				// Log or collect errors for files? For now, we skip problematic files.
				// Consider returning partial results with errors.
				fmt.Fprintf(os.Stderr, "Error scanning file %s: %v\n", path, scanErr) // Log to stderr
				continue // Skip this file
			}
		}

		results = append(results, fileInfo)
	}

	// Note: Sorting is handled by the caller (e.g., TUI) if needed.
	return results, nil
}

// Analyze performs additional analysis on a file
func (e *Engine) Analyze(path string) (*types.FileInfo, error) {
	// Implementation here
	return nil, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (e *Engine) detectContentType(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read first 512 bytes for detection
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	// Use both net/http and file extension detection
	contentType := http.DetectContentType(buffer)
	ext := filepath.Ext(path)

	// Override for known image types
	if strings.HasPrefix(contentType, "application/octet-stream") {
		switch ext {
		case ".jpg", ".jpeg":
			return "image/jpeg", nil
		case ".png":
			return "image/png", nil
		}
	}

	return contentType, nil
}
