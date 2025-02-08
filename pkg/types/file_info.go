package types

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// FileInfo represents metadata about a file in the system
type FileInfo struct {
	Path        string   `json:"path"`
	ContentType string   `json:"type"`
	Size        int64    `json:"size"`
	Tags        []string `json:"tags,omitempty"`
}

// ToJSON converts a FileInfo to JSON format
func (fi *FileInfo) ToJSON() string {
	jsonBytes, _ := json.Marshal(fi)
	return string(jsonBytes)
}

// String converts a FileInfo to a human-readable format
func (fi *FileInfo) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("File: %s\n", fi.Path))
	sb.WriteString(fmt.Sprintf("Type: %s\n", fi.ContentType))
	sb.WriteString(fmt.Sprintf("Size: %d bytes\n", fi.Size))
	if len(fi.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(fi.Tags, ", ")))
	}
	return sb.String()
}

// IsSymlink checks if the file is a symbolic link
func (fi *FileInfo) IsSymlink() bool {
	info, err := os.Lstat(fi.Path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}
