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
	// Convert the file entries to a readable string format
	var fileListStr string
	for i, file := range f.files {
		prefix := "  "
		if i == f.cursor {
			prefix = "> "
		}

		if f.selected[file.Path] {
			prefix += "* "
		} else {
			prefix += "  "
		}

		fileListStr += prefix + file.Name + "\n"
	}

	return styles.FileListStyle.Render(fileListStr)
}
