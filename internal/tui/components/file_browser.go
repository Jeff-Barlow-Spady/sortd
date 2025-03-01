package components

import (
	"sortd/internal/tui/styles"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type FileBrowser struct {
	viewport  viewport.Model
	fileList  *FileList
	statusBar *StatusBar
	height    int
	width     int
}

func NewFileBrowser() *FileBrowser {
	vp := viewport.New(80, 20)
	vp.Style = styles.Theme.App

	return &FileBrowser{
		viewport:  vp,
		fileList:  NewFileList(),
		statusBar: NewStatusBar(),
	}
}

func (fb *FileBrowser) SetSize(width, height int) {
	fb.width = width
	fb.height = height
	fb.viewport.Width = width
	fb.viewport.Height = height - 2 // Leave room for status bar

	// Update the list dimensions too
	if fb.fileList != nil && fb.fileList.list.Height() != height-2 {
		fb.fileList.list.SetHeight(height - 2)
		fb.fileList.list.SetWidth(width)
	}
}

func (fb *FileBrowser) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			fb.fileList.MoveCursor(-1)
		case "down", "j":
			fb.fileList.MoveCursor(1)
		case "space":
			if fb.fileList != nil {
				fb.fileList.ToggleSelected(fb.fileList.GetCursor())
			}
		}
	}

	// Update viewport
	vp, cmd := fb.viewport.Update(msg)
	fb.viewport = vp
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Update status bar
	if cmd := fb.statusBar.Update(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Update fileList
	if cmd := fb.fileList.Update(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (fb *FileBrowser) View() string {
	// Update viewport content
	fb.viewport.SetContent(fb.fileList.View())

	return fb.viewport.View() + "\n" + fb.statusBar.View()
}

func (fb *FileBrowser) Init() tea.Cmd {
	return fb.fileList.Init()
}

func (fb *FileBrowser) Copy() *FileBrowser {
	// Create a new viewport instead of reusing the existing one
	newVP := viewport.New(fb.width, fb.height-2)
	newVP.Style = styles.Theme.App

	return &FileBrowser{
		width:     fb.width,
		height:    fb.height,
		viewport:  newVP,
		fileList:  fb.fileList.Copy(), // Use the copy method
		statusBar: NewStatusBar(),
	}
}
