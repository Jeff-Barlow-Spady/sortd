package analysis

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"

	"sortd/internal/config"
	serr "sortd/internal/errors"
	log "sortd/internal/log"
	"sortd/pkg/types"
)

// Analyzer defines the interface for file type specific analyzers
type Analyzer interface {
	// CanHandle checks if this analyzer is suitable for the given content type
	CanHandle(contentType string) bool
	// Analyze performs the specific analysis and updates FileInfo
	Analyze(path string, info *types.FileInfo) (*types.FileInfo, error)
}

// --- Concrete Analyzer Implementations ---

// ImageAnalyzer handles analysis for image files using EXIF data
type ImageAnalyzer struct{}

// CanHandle checks if the content type is an image type that might contain EXIF data
func (a *ImageAnalyzer) CanHandle(contentType string) bool {
	// Be somewhat lenient: check for image/ prefix, but also common types
	// that might contain EXIF even if detected differently (e.g., octet-stream for some RAWs)
	// This might need refinement based on observed file types.
	return strings.HasPrefix(contentType, "image/") || contentType == "application/octet-stream"
}

// Analyze extracts EXIF metadata from image files
func (a *ImageAnalyzer) Analyze(path string, info *types.FileInfo) (*types.FileInfo, error) {
	logger := log.LogWithFields(log.F("path", path))

	// Ensure base image tag is present
	if !contains(info.Tags, "image") {
		info.Tags = append(info.Tags, "image")
	}
	// Ensure metadata map is initialized
	if info.Metadata == nil {
		info.Metadata = make(map[string]string)
	}

	file, err := os.Open(path)
	if err != nil {
		return info, fmt.Errorf("failed to open image file for exif: %w", err)
	}
	defer file.Close()

	x, err := exif.Decode(file)
	if err != nil {
		logger.Debugf("No EXIF data found or failed to decode for %s: %v", path, err)
		return info, nil // Not an error if no EXIF data
	}

	// Extract specific fields
	if dt, err := x.Get(exif.DateTimeOriginal); err == nil {
		dtStr, _ := dt.StringVal()
		if dtStr != "" {
			info.Metadata["DateTimeOriginal"] = dtStr
			logger.Debugf("Found DateTimeOriginal: %s", dtStr)
		}
	}
	if model, err := x.Get(exif.Model); err == nil {
		modelStr, _ := model.StringVal()
		if modelStr != "" {
			info.Metadata["CameraModel"] = modelStr
			logger.Debugf("Found CameraModel: %s", modelStr)
		}
	}

	return info, nil
}

// TODO: Implement other analyzers like AudioAnalyzer, PDFAnalyzer here...

// --- Engine Implementation ---

// Engine handles file analysis and content detection
type Engine struct {
	config    *config.Config
	analyzers []Analyzer // List of registered analyzers
}

func (e *Engine) SetConfig(cfg *config.Config) {
	e.config = cfg
}

// registerAnalyzer adds an analyzer to the engine's list
func (e *Engine) registerAnalyzer(analyzer Analyzer) {
	if e.analyzers == nil {
		e.analyzers = []Analyzer{}
	}
	e.analyzers = append(e.analyzers, analyzer)
}

// New creates a new Analysis Engine instance and registers default analyzers
func New() *Engine {
	exif.RegisterParsers(mknote.All...)
	engine := &Engine{}
	engine.registerAnalyzer(&ImageAnalyzer{}) // Register image analyzer
	// TODO: Register other analyzers when implemented
	return engine
}

// NewWithConfig creates a new Analysis Engine instance with config settings
func NewWithConfig(cfg *config.Config) *Engine {
	engine := New()
	engine.config = cfg
	return engine
}

// Scan performs basic file analysis
func (e *Engine) Scan(path string) (*types.FileInfo, error) {
	logger := log.LogWithFields(log.F("path", path))

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, serr.NewFileError("failed to stat file", path, serr.FileNotFound, err)
		}
		return nil, serr.NewFileError("failed to stat file", path, serr.FileAccessDenied, err)
	}

	// Open file for content type detection
	file, err := os.Open(path)
	if err != nil {
		return nil, serr.NewFileError("failed to open file", path, serr.FileAccessDenied, err)
	}
	defer file.Close()

	// Read first 512 bytes for content type detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, serr.NewFileError("failed to read file", path, serr.FileOperationFailed, err)
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

	logger.Info("File scanned successfully")
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
	logger := log.LogWithFields(log.F("directory", dir))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, serr.NewFileError("failed to read directory", dir, serr.FileAccessDenied, err)
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
				ContentType: "inode/directory",     // Convention for directories
				Size:        0,                     // Directories don't have a size in this context
				Tags:        []string{"directory"}, // Add a 'directory' tag
			}
		} else {
			// It's a file, use the Scan method
			fileInfo, scanErr = e.Scan(path)
			if scanErr != nil {
				logger.ErrorWithStack(scanErr, "Error scanning file")
				continue // Skip this file
			}
		}

		results = append(results, fileInfo)
	}

	// Note: Sorting is handled by the caller (e.g., TUI) if needed.
	return results, nil
}

// Analyze performs analysis by delegating to registered analyzers
func (e *Engine) Analyze(path string) (*types.FileInfo, error) {
	logger := log.LogWithFields(log.F("path", path))
	fileInfo, err := e.Scan(path)
	if err != nil {
		return nil, err
	}
	if fileInfo.Metadata == nil {
		fileInfo.Metadata = make(map[string]string)
	}

	var analysisErr error
	foundAnalyzer := false
	for _, analyzer := range e.analyzers {
		if analyzer.CanHandle(fileInfo.ContentType) {
			foundAnalyzer = true
			logger.Debugf("Using analyzer %T for content type %s", analyzer, fileInfo.ContentType)
			fileInfo, analysisErr = analyzer.Analyze(path, fileInfo)
			if analysisErr != nil {
				logger.With(log.F("analyzer", fmt.Sprintf("%T", analyzer)), log.F("error", analysisErr.Error())).Warn("Analyzer failed, returning partial info")
				return fileInfo, nil // Return info obtained so far even if analyzer fails
			}
			break
		}
	}

	if !foundAnalyzer {
		logger.Debugf("No specific analyzer registered for content type: %s", fileInfo.ContentType)
	}

	// General text analysis placeholder (Consider a TextAnalyzer struct)
	if strings.HasPrefix(fileInfo.ContentType, "text/") {
		if !contains(fileInfo.Tags, "document") {
			fileInfo.Tags = append(fileInfo.Tags, "document")
		}
	}

	return fileInfo, nil
}

// contains helper function (keep for now, consider moving to utils later)
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Removed old analyzeText, analyzeAudio, analyzeVideo, analyzePDF placeholders
