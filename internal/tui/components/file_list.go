package components

import "sortd/internal/tui/common"

type FileList struct {
	files    []common.FileEntry
	cursor   int
	selected map[string]bool
}

func NewFileList() *FileList {
	return &FileList{
		files:    make([]common.FileEntry, 0),
		selected: make(map[string]bool),
	}
}

// Add file list rendering logic...
