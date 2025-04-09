package analysis

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"sortd/internal/config"
	"sortd/pkg/types"

	serr "sortd/internal/errors"
	log "sortd/internal/log"
)

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
	return &Engine{
		config: cfg,
	}
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

// Analyze performs additional analysis on a file
func (e *Engine) Analyze(path string) (*types.FileInfo, error) {
	logger := log.LogWithFields(log.F("path", path))

	// First do basic scanning
	fileInfo, err := e.Scan(path)
	if err != nil {
		return nil, err
	}

	// Perform deeper content type specific analysis
	switch {
	case strings.HasPrefix(fileInfo.ContentType, "image/"):
		result, err := e.analyzeImage(path, fileInfo)
		if err != nil {
			logger.With(log.F("error", err.Error())).Warn("Failed to analyze image metadata")
		}
		return result, nil

	case strings.HasPrefix(fileInfo.ContentType, "audio/"):
		result, err := e.analyzeAudio(path, fileInfo)
		if err != nil {
			logger.With(log.F("error", err.Error())).Warn("Failed to analyze audio metadata")
		}
		return result, nil

	case strings.HasPrefix(fileInfo.ContentType, "video/"):
		result, err := e.analyzeVideo(path, fileInfo)
		if err != nil {
			logger.With(log.F("error", err.Error())).Warn("Failed to analyze video metadata")
		}
		return result, nil

	case strings.HasPrefix(fileInfo.ContentType, "text/"),
		strings.HasPrefix(fileInfo.ContentType, "application/json"),
		strings.HasPrefix(fileInfo.ContentType, "application/xml"),
		strings.HasPrefix(fileInfo.ContentType, "application/yaml"):
		result, err := e.analyzeText(path, fileInfo)
		if err != nil {
			logger.With(log.F("error", err.Error())).Warn("Failed to analyze text metadata")
		}
		return result, nil

	case strings.HasPrefix(fileInfo.ContentType, "application/pdf"):
		result, err := e.analyzePDF(path, fileInfo)
		if err != nil {
			logger.With(log.F("error", err.Error())).Warn("Failed to analyze PDF metadata")
		}
		return result, nil

	default:
		logger.Debug("No specific analyzer available for content type")
		return fileInfo, nil
	}
}

// analyzeImage extracts metadata from image files (EXIF)
func (e *Engine) analyzeImage(path string, info *types.FileInfo) (*types.FileInfo, error) {
	// This is a placeholder for actual image metadata extraction
	// In a complete implementation, this would use libraries like github.com/rwcarlsen/goexif
	// to extract EXIF data from images

	// Add image-specific tags
	info.Tags = append(info.Tags, "image")

	// For now, just return the basic info
	return info, nil
}

// analyzeAudio extracts metadata from audio files (ID3, etc)
func (e *Engine) analyzeAudio(path string, info *types.FileInfo) (*types.FileInfo, error) {
	// This is a placeholder for actual audio metadata extraction
	// In a complete implementation, this would use libraries like github.com/dhowden/tag
	// to extract audio metadata (ID3 tags, etc)

	// Add audio-specific tags
	info.Tags = append(info.Tags, "audio")

	// For now, just return the basic info
	return info, nil
}

// analyzeVideo extracts metadata from video files
func (e *Engine) analyzeVideo(path string, info *types.FileInfo) (*types.FileInfo, error) {
	// This is a placeholder for actual video metadata extraction
	// In a complete implementation, this would use libraries or external tools
	// to extract video metadata

	// Add video-specific tags
	info.Tags = append(info.Tags, "video")

	// For now, just return the basic info
	return info, nil
}

// analyzeText performs analysis on text-based files
func (e *Engine) analyzeText(path string, info *types.FileInfo) (*types.FileInfo, error) {
	// This is a placeholder for actual text content analysis
	// In a complete implementation, this would examine file contents
	// to extract meaningful information

	// Add document tag if not present
	if !contains(info.Tags, "document") {
		info.Tags = append(info.Tags, "document")
	}

	// For now, just return the basic info
	return info, nil
}

// analyzePDF extracts metadata from PDF files
func (e *Engine) analyzePDF(path string, info *types.FileInfo) (*types.FileInfo, error) {
	// This is a placeholder for actual PDF metadata extraction
	// In a complete implementation, this would use libraries like github.com/ledongthuc/pdf
	// to extract PDF metadata

	// Add PDF-specific tags
	info.Tags = append(info.Tags, "document", "pdf")

	// For now, just return the basic info
	return info, nil
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
		if os.IsNotExist(err) {
			return "", serr.NewFileError("failed to open file for content type detection", path, serr.FileNotFound, err)
		}
		return "", serr.NewFileError("failed to open file for content type detection", path, serr.FileAccessDenied, err)
	}
	defer file.Close()

	// Read first 512 bytes for detection
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", serr.NewFileError("failed to read file for content type detection", path, serr.FileOperationFailed, err)
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
