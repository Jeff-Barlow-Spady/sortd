package components

import (
	"sortd/internal/tui/styles"
	"sortd/pkg/types"
)

type FileList struct {
	files    []types.FileEntry
	cursor   int
	selected map[string]bool
}

func NewFileList() *FileList {
	return &FileList{
		files:    make([]types.FileEntry, 0),
		selected: make(map[string]bool),
	}
}

// Add file list rendering logic...

func (f *FileList) Render() string {
	return styles.FileListStyle.Render(f.files)
}
