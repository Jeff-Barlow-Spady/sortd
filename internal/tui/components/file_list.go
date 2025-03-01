package components

import (
	"fmt"
	"os"
	"sortd/internal/tui/common"
	"sortd/internal/tui/styles"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
)

type FileList struct {
	list       list.Model
	files      []common.FileEntry
	selected   map[string]bool
	cursor     int
	currentDir string
}

func NewFileList() *FileList {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	return &FileList{
		list:     l,
		selected: make(map[string]bool),
		cursor:   0,
	}
}

func (fl *FileList) Init() tea.Cmd {
	return nil
}

func (fl *FileList) SetFiles(files []common.FileEntry) {
	fl.files = files
	items := make([]list.Item, len(files))
	for i, f := range files {
		items[i] = item(f.Name)
	}
	fl.list.SetItems(items)
}

func (fl *FileList) SetCurrentDir(dir string) {
	fl.currentDir = dir
}

type item string

func (i item) FilterValue() string { return string(i) }

func (fl *FileList) View() string {
	var s strings.Builder

	// Add header
	s.WriteString(styles.Theme.Help.Render("Directory: " + fl.currentDir + "\n\n"))

	if len(fl.files) == 0 {
		s.WriteString("No files found\n")
		return s.String()
	}

	// File listing with details
	for i, file := range fl.files {
		style := styles.Theme.Unselected
		if fl.selected[file.Name] {
			style = styles.Theme.Selected
		}

		cursor := " "
		if i == fl.cursor {
			cursor = ">"
		}

		// Add file details
		details := ""
		if info, err := os.Stat(file.Path); err == nil {
			size := humanize.Bytes(uint64(info.Size()))
			modTime := info.ModTime().Format("2006-01-02 15:04:05")
			details = fmt.Sprintf(" %8s  %s", size, modTime)
		}

		s.WriteString(fmt.Sprintf("%s %s%s\n",
			cursor,
			style.Render(file.Name),
			style.Render(details)))
	}

	return s.String()
}

func (fl *FileList) MoveCursor(delta int) {
	newPos := fl.cursor + delta
	if newPos >= 0 && newPos < len(fl.files) {
		fl.cursor = newPos
	}
}

func (fl *FileList) ToggleSelected(index int) {
	if index >= 0 && index < len(fl.files) {
		name := fl.files[index].Name
		fl.selected[name] = !fl.selected[name]
	}
}

func (fl *FileList) Copy() *FileList {
	// Create a new list.Model instead of reusing the existing one
	newList := list.New(make([]list.Item, 0), list.NewDefaultDelegate(), 0, 0)

	// Maintain dimensions from the original
	newList.SetHeight(fl.list.Height())
	newList.SetWidth(fl.list.Width())

	// Copy the items
	items := make([]list.Item, len(fl.files))
	for i, f := range fl.files {
		items[i] = item(f.Name)
	}
	newList.SetItems(items)

	newFL := &FileList{
		list:       newList,
		files:      make([]common.FileEntry, len(fl.files)),
		selected:   make(map[string]bool),
		cursor:     fl.cursor,
		currentDir: fl.currentDir,
	}

	// Copy the files and selected state
	copy(newFL.files, fl.files)
	for k, v := range fl.selected {
		newFL.selected[k] = v
	}

	return newFL
}

func (fl *FileList) GetSelected() map[string]bool {
	return fl.selected
}

func (fl *FileList) GetCursor() int {
	return fl.cursor
}

func (fl *FileList) Files() []common.FileEntry {
	return fl.files
}

func (fl *FileList) CurrentDir() string {
	return fl.currentDir
}

func (fl *FileList) GetCurrentFile() *common.FileEntry {
	if fl.cursor >= 0 && fl.cursor < len(fl.files) {
		return &fl.files[fl.cursor]
	}
	return nil
}

func (fl *FileList) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	// Update the underlying list model
	fl.list, cmd = fl.list.Update(msg)

	// Handle any FileList specific updates here
	// (most key handling is already in FileBrowser.Update)

	return cmd
}

// Add file list rendering logic...
