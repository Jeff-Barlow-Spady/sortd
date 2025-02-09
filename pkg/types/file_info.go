package types

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileInfo represents analyzed file information
type FileInfo struct {
	Path        string   `json:"path"`
	ContentType string   `json:"type"`
	Size        int64    `json:"size"`
	Tags        []string `json:"tags,omitempty"`
}

// Name returns the base name of the file
func (f *FileInfo) Name() string {
	return filepath.Base(f.Path)
}

// ToJSON converts FileInfo to JSON string
func (f *FileInfo) ToJSON() string {
	jsonBytes, _ := json.Marshal(f)
	return string(jsonBytes)
}

// String returns a human-readable representation
func (f *FileInfo) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("File: %s\n", f.Path))
	sb.WriteString(fmt.Sprintf("Type: %s\n", f.ContentType))
	sb.WriteString(fmt.Sprintf("Size: %d bytes\n", f.Size))
	if len(f.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(f.Tags, ", ")))
	}
	return sb.String()
}

// IsSymlink checks if the file is a symbolic link
func (f *FileInfo) IsSymlink() bool {
	info, err := os.Lstat(f.Path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}
