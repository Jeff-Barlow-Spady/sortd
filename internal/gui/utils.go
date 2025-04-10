package gui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"sortd/internal/config"
	"sortd/internal/log"

	"fyne.io/fyne/v2"
	"gopkg.in/yaml.v3"
)

// parseImportedConfig parses an imported configuration file
func parseImportedConfig(reader fyne.URIReadCloser) (*config.Config, error) {
	// Read the entire file
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Determine format based on extension
	format := filepath.Ext(reader.URI().Name())
	var cfg config.Config

	switch strings.ToLower(format) {
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("error parsing JSON: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("error parsing YAML: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported file format: %s", format)
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// exportConfig exports the configuration to a file
func exportConfig(cfg *config.Config, writer fyne.URIWriteCloser, format string) error {
	var data []byte
	var err error

	switch strings.ToLower(format) {
	case "json":
		data, err = json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("error encoding to JSON: %w", err)
		}
	case "yaml":
		data, err = yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("error encoding to YAML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}

	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	return nil
}

// refreshDirectoryPreview updates the directory preview
func refreshDirectoryPreview(dirPath string) (string, error) {
	if dirPath == "" {
		return "No directory selected", nil
	}

	// Check if directory exists
	if _, err := os.Stat(dirPath); err != nil {
		return fmt.Sprintf("Error accessing directory: %v", err), err
	}

	// Get directory contents
	files, err := filepath.Glob(filepath.Join(dirPath, "*"))
	if err != nil {
		return fmt.Sprintf("Error reading directory: %v", err), err
	}

	if len(files) == 0 {
		return "Directory is empty", nil
	}

	// Build preview text
	sb := strings.Builder{}
	sb.WriteString("Directory Contents:\n")

	// Sort files alphabetically
	sort.Strings(files)

	// Show first 10 files
	maxFilesToDisplay := 10
	for i, file := range files {
		if i >= maxFilesToDisplay {
			sb.WriteString(fmt.Sprintf("... and %d more files\n", len(files)-maxFilesToDisplay))
			break
		}

		info, err := os.Stat(file)
		if err != nil {
			log.Warnf("Error getting stats for file %s: %v", file, err)
			continue
		}

		fileType := "File"
		if info.IsDir() {
			fileType = "Dir"
		}

		sb.WriteString(fmt.Sprintf("%s: %s (%d bytes)\n", fileType, filepath.Base(file), info.Size()))
	}

	return sb.String(), nil
}
