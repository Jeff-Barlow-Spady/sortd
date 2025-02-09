package analysis

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/toasty/sortd/internal/config"
	"github.com/toasty/sortd/pkg/types"
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
type Engine struct{}

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

	contentType := "application/octet-stream"
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read first 512 bytes to determine content type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	buffer = buffer[:n]

	contentType = http.DetectContentType(buffer)

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

	err := filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		result, err := e.Scan(currentPath)
		if err != nil {
			return err
		}

		results = append(results, result)
		return nil
	})

	if err != nil {
		return nil, &ScanError{
			Path:    path,
			Err:     fmt.Errorf("directory scan failed: %w", err),
			Context: "directory walk",
		}
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
