package analysis

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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
	if path == "" {
		return nil, &ScanError{Path: path, Err: ErrInvalidInput, Context: "empty path"}
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &ScanError{Path: path, Err: ErrFileNotFound, Context: "file existence check"}
		}
		if os.IsPermission(err) {
			return nil, &ScanError{Path: path, Err: ErrPermissionDenied, Context: "file permissions"}
		}
		return nil, &ScanError{Path: path, Err: fmt.Errorf("filesystem error: %w", err), Context: "stat operation"}
	}

	if info.IsDir() {
		return nil, &ScanError{Path: path, Err: ErrInvalidInput, Context: "directory instead of file"}
	}

	contentType, err := e.detectContentType(path)
	if err != nil {
		return nil, &ScanError{Path: path, Err: err, Context: "content type detection"}
	}

	result := &types.FileInfo{
		Path:        path,
		ContentType: contentType,
		Size:        info.Size(),
		Tags:        []string{},
	}

	// Add basic tags based on content type
	if strings.HasPrefix(contentType, "image/") {
		result.Tags = append(result.Tags, "image")
	} else if strings.HasPrefix(contentType, "text/") {
		result.Tags = append(result.Tags, "document")
	}

	return result, nil
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

// ScanDirectory performs analysis on all files in a directory
func (e *Engine) ScanDirectory(path string) ([]*types.FileInfo, error) {
	var results []*types.FileInfo

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, &ScanError{
			Path:    path,
			Err:     fmt.Errorf("directory read failed: %w", err),
			Context: "os.ReadDir",
		}
	}

	// Sort entries to ensure consistent order for tests
	fileNames := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			fileNames = append(fileNames, entry.Name())
		}
	}
	sort.Strings(fileNames)

	// Process files in sorted order
	for _, name := range fileNames {
		currentPath := filepath.Join(path, name)
		result, err := e.Scan(currentPath)
		if err != nil {
			// Log error but continue scanning other files
			fmt.Printf("Error scanning file %s: %v\n", currentPath, err)
			continue
		}

		results = append(results, result)
	}

	// Return empty slice instead of nil for no results
	if results == nil {
		return []*types.FileInfo{}, nil
	}

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
