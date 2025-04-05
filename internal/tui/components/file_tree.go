package components

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TreeNode represents a node in the file tree
type TreeNode struct {
	Name     string
	Path     string
	IsDir    bool
	Size     int64
	IsOpen   bool
	Children []*TreeNode
	Parent   *TreeNode
	Level    int
}

// FileTree is a component that displays a hierarchical file tree
type FileTree struct {
	// Core state
	Root           *TreeNode
	Cursor         int
	VisibleRows    []*TreeNode
	Selected       map[string]bool
	CurrentDir     string
	MaxHeight      int
	Height         int
	Width          int
	Offset         int  // For scrolling
	ShowHidden     bool // Whether to show hidden files
	lastUpdateTime time.Time

	// Visual enhancement for ADHD
	focusedAnimation int
	showAnimations   bool
	highlightCurrent bool
}

// NewFileTree creates a new file tree component
func NewFileTree(rootDir string) *FileTree {
	selected := make(map[string]bool)
	root := &TreeNode{
		Name:     filepath.Base(rootDir),
		Path:     rootDir,
		IsDir:    true,
		IsOpen:   true,
		Children: []*TreeNode{},
		Level:    0,
	}

	tree := &FileTree{
		Root:             root,
		Cursor:           0,
		VisibleRows:      []*TreeNode{},
		Selected:         selected,
		CurrentDir:       rootDir,
		ShowHidden:       false,
		MaxHeight:        20,
		Height:           20,
		Width:            80,
		focusedAnimation: 0,
		showAnimations:   true,
		highlightCurrent: true,
		lastUpdateTime:   time.Now(),
	}

	// Initialize the tree from the provided root directory
	tree.BuildTree(root)
	tree.UpdateVisibleRows()

	return tree
}

// BuildTree recursively builds the tree structure for a directory
func (f *FileTree) BuildTree(node *TreeNode) error {
	// Only build if this is a directory
	if !node.IsDir {
		return nil
	}

	// Read directory contents
	entries, err := os.ReadDir(node.Path)
	if err != nil {
		return err
	}

	// Sort entries - directories first, then files
	sort.Slice(entries, func(i, j int) bool {
		isDir1 := entries[i].IsDir()
		isDir2 := entries[j].IsDir()
		if isDir1 && !isDir2 {
			return true
		}
		if !isDir1 && isDir2 {
			return false
		}
		return entries[i].Name() < entries[j].Name()
	})

	// Clear existing children before rebuilding
	node.Children = []*TreeNode{}

	for _, entry := range entries {
		// Skip hidden files unless showing them is enabled
		if !f.ShowHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		var size int64
		if err == nil {
			size = info.Size()
		}

		childNode := &TreeNode{
			Name:     entry.Name(),
			Path:     filepath.Join(node.Path, entry.Name()),
			IsDir:    entry.IsDir(),
			Size:     size,
			IsOpen:   false,
			Children: []*TreeNode{},
			Parent:   node,
			Level:    node.Level + 1,
		}

		node.Children = append(node.Children, childNode)
	}

	return nil
}

// Init initializes the component
func (f *FileTree) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (f *FileTree) Update(msg tea.Msg) (*FileTree, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			f.MoveUp()
		case "down", "j":
			f.MoveDown()
		case "left", "h":
			// If folder is open, close it. Otherwise, go to parent.
			if f.Cursor < len(f.VisibleRows) {
				current := f.VisibleRows[f.Cursor]
				if current.IsDir && current.IsOpen {
					f.Toggle(false)
				} else {
					f.MoveToParent()
				}
			}
		case "right", "l", "enter":
			// If folder, open it. If file, select it.
			if f.Cursor < len(f.VisibleRows) {
				current := f.VisibleRows[f.Cursor]
				if current.IsDir {
					f.Toggle(false)
					// After toggling a directory open, move cursor down to show contents
					if current.IsOpen && len(current.Children) > 0 {
						f.MoveDown()
					}
				} else {
					// For files, toggle selection
					f.ToggleSelected()
				}
			}
		case " ":
			// Toggle selection
			f.ToggleSelected()
		}

		// Animate focus indicator on key presses
		if f.showAnimations {
			f.focusedAnimation = (f.focusedAnimation + 1) % 4
		}

		// Update current directory after key press
		if f.Cursor < len(f.VisibleRows) {
			if f.VisibleRows[f.Cursor].IsDir {
				f.CurrentDir = f.VisibleRows[f.Cursor].Path
			} else if f.VisibleRows[f.Cursor].Parent != nil {
				f.CurrentDir = f.VisibleRows[f.Cursor].Parent.Path
			}
		}

	case tea.WindowSizeMsg:
		// Adjust dimensions based on window size
		f.Width = msg.Width - 4   // Allow some margin
		f.Height = msg.Height - 8 // Allow space for headers and footers

		// Ensure cursor is visible after resize
		f.EnsureCursorVisible()

	// Add a timer to animate cursor in ADHD mode
	case time.Time:
		if f.showAnimations && time.Since(f.lastUpdateTime) > 250*time.Millisecond {
			f.focusedAnimation = (f.focusedAnimation + 1) % 4
			f.lastUpdateTime = time.Now()
			return f, tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
				return t
			})
		}
	}

	return f, cmd
}

// Toggle expands or collapses the node at the current cursor position
func (f *FileTree) Toggle(expandChildren bool) {
	if f.Cursor < 0 || f.Cursor >= len(f.VisibleRows) {
		return
	}

	node := f.VisibleRows[f.Cursor]
	if !node.IsDir {
		return
	}

	node.IsOpen = !node.IsOpen

	// If we're expanding the node
	if node.IsOpen {
		// Only build children if they haven't been built yet
		if len(node.Children) == 0 {
			f.BuildTree(node)
		}

		// If expandChildren is true, recursively open all children
		if expandChildren {
			f.expandAllChildren(node)
		}
	}

	// Update the visible rows to reflect the change
	f.UpdateVisibleRows()
}

// expandAllChildren recursively opens all child directories
func (f *FileTree) expandAllChildren(node *TreeNode) {
	for _, child := range node.Children {
		if child.IsDir {
			child.IsOpen = true
			if len(child.Children) == 0 {
				f.BuildTree(child)
			}
			f.expandAllChildren(child)
		}
	}
}

// UpdateVisibleRows updates the list of visible rows based on which nodes are open
func (f *FileTree) UpdateVisibleRows() {
	f.VisibleRows = []*TreeNode{}
	f.addVisibleNode(f.Root)

	// Adjust cursor if it's out of bounds
	if f.Cursor >= len(f.VisibleRows) {
		f.Cursor = max(0, len(f.VisibleRows)-1)
	}
}

// addVisibleNode recursively adds visible nodes to the VisibleRows slice
func (f *FileTree) addVisibleNode(node *TreeNode) {
	f.VisibleRows = append(f.VisibleRows, node)
	if node.IsOpen {
		for _, child := range node.Children {
			f.addVisibleNode(child)
		}
	}
}

// MoveUp moves the cursor up one row
func (f *FileTree) MoveUp() {
	if f.Cursor > 0 {
		f.Cursor--
	}
	f.EnsureCursorVisible()
}

// MoveDown moves the cursor down one row
func (f *FileTree) MoveDown() {
	if f.Cursor < len(f.VisibleRows)-1 {
		f.Cursor++
	}
	f.EnsureCursorVisible()
}

// MoveToParent moves the cursor to the parent of the current node
func (f *FileTree) MoveToParent() {
	if f.Cursor < 0 || f.Cursor >= len(f.VisibleRows) {
		return
	}

	node := f.VisibleRows[f.Cursor]
	if node.Parent == nil {
		return
	}

	// Find the parent's index in the visible rows
	for i, row := range f.VisibleRows {
		if row == node.Parent {
			f.Cursor = i
			break
		}
	}

	f.EnsureCursorVisible()
}

// EnsureCursorVisible makes sure the cursor is visible by adjusting the scroll offset
func (f *FileTree) EnsureCursorVisible() {
	if f.Height <= 0 {
		return
	}

	// If cursor is above current view
	if f.Cursor < f.Offset {
		f.Offset = f.Cursor
	}

	// If cursor is below current view
	if f.Cursor >= f.Offset+f.Height-2 { // -2 to account for scroll indicators
		f.Offset = f.Cursor - f.Height + 3 // +3 for scroll indicators and buffer
	}

	// Ensure offset is never negative
	if f.Offset < 0 {
		f.Offset = 0
	}

	// Ensure offset isn't too far (showing empty space when unnecessary)
	maxOffset := max(0, len(f.VisibleRows)-f.Height)
	if f.Offset > maxOffset {
		f.Offset = maxOffset
	}
}

// ToggleSelected toggles the selection state of the current item
func (f *FileTree) ToggleSelected() {
	if f.Cursor < 0 || f.Cursor >= len(f.VisibleRows) {
		return
	}

	node := f.VisibleRows[f.Cursor]
	if f.Selected[node.Path] {
		delete(f.Selected, node.Path)
	} else {
		f.Selected[node.Path] = true
	}
}

// GetSelectedFiles returns a list of selected file paths
func (f *FileTree) GetSelectedFiles() []string {
	files := make([]string, 0, len(f.Selected))
	for path := range f.Selected {
		files = append(files, path)
	}
	return files
}

// SetDirectory sets the current directory and rebuilds the tree
func (f *FileTree) SetDirectory(dir string) error {
	// Verify the directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	// Update the root node
	f.Root = &TreeNode{
		Name:     filepath.Base(dir),
		Path:     dir,
		IsDir:    true,
		IsOpen:   true,
		Children: []*TreeNode{},
		Level:    0,
	}

	f.CurrentDir = dir
	f.Cursor = 0
	f.Offset = 0

	// Build the tree structure
	f.BuildTree(f.Root)
	f.UpdateVisibleRows()

	return nil
}

// View returns the rendered view of the file tree
func (f *FileTree) View() string {
	var b strings.Builder

	// If no visible rows, show a message
	if len(f.VisibleRows) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true).
			Align(lipgloss.Center).
			Width(f.Width)

		return emptyStyle.Render("✨ No files found ✨")
	}

	// Calculate which nodes are in view based on scroll position
	startIdx := f.Offset
	endIdx := min(len(f.VisibleRows), f.Offset+f.Height-2) // Reserve space for scroll indicators

	// Define styles
	cursor := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#6B5ECD")).
		Bold(true)

	selected := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#73F59F"))

	dirStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#81A1C1")).
		Bold(true)

	fileStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D8DEE9"))

	focusIndicators := []string{"▶ ", "▷ ", "▸ ", "▹ "}

	// Add scroll indicator if needed
	if startIdx > 0 {
		scrollUpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Align(lipgloss.Center).
			Width(f.Width)
		b.WriteString(scrollUpStyle.Render("↑ More ↑") + "\n")
	}

	// Render only the visible portion of the tree
	for i := startIdx; i < endIdx; i++ {
		node := f.VisibleRows[i]

		// Base indentation
		indent := strings.Repeat("  ", node.Level)

		// Add tree symbols for better visual hierarchy
		if node.Level > 0 {
			if i == len(f.VisibleRows)-1 || node.Parent != f.VisibleRows[i+1].Parent {
				// Last child
				indent = strings.Repeat("  ", node.Level-1) + "└─ "
			} else {
				// Middle child
				indent = strings.Repeat("  ", node.Level-1) + "├─ "
			}
		}

		// Icon based on file type and open/closed state
		var icon string
		if node.IsDir {
			if node.IsOpen {
				icon = "📂 "
			} else {
				icon = "📁 "
			}
		} else {
			// Determine file type icon
			switch filepath.Ext(strings.ToLower(node.Name)) {
			case ".jpg", ".jpeg", ".png", ".gif", ".webp":
				icon = "🖼️ "
			case ".mp4", ".avi", ".mov", ".mkv":
				icon = "🎬 "
			case ".mp3", ".wav", ".flac", ".ogg":
				icon = "🎵 "
			case ".pdf":
				icon = "📕 "
			case ".zip", ".tar", ".gz", ".rar":
				icon = "🗜️ "
			case ".txt", ".md", ".go", ".js", ".py":
				icon = "📝 "
			default:
				icon = "📄 "
			}
		}

		// Node name with icon
		var name string

		// Add animated cursor for ADHD focus enhancement
		if i == f.Cursor && f.highlightCurrent {
			if f.showAnimations {
				name = indent + focusIndicators[f.focusedAnimation] + icon + node.Name
			} else {
				name = indent + "▶ " + icon + node.Name
			}
		} else {
			name = indent + "  " + icon + node.Name
		}

		// Add selection marker with emphasis for ADHD visibility
		if f.Selected[node.Path] {
			name += " ✅"
		}

		// Apply styles based on state
		var renderedName string
		if i == f.Cursor {
			renderedName = cursor.Render(name)
		} else if f.Selected[node.Path] {
			renderedName = selected.Render(name)
		} else if node.IsDir {
			renderedName = dirStyle.Render(name)
		} else {
			renderedName = fileStyle.Render(name)
		}

		// Ensure the line doesn't exceed the available width
		if len(renderedName) > f.Width {
			renderedName = renderedName[:f.Width-3] + "..."
		}

		b.WriteString(renderedName + "\n")
	}

	// Add scroll indicator if needed
	if endIdx < len(f.VisibleRows) {
		scrollDownStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Align(lipgloss.Center).
			Width(f.Width)
		b.WriteString(scrollDownStyle.Render("↓ More ↓") + "\n")

		// Add count of remaining items for better pagination awareness
		if len(f.VisibleRows)-endIdx > 5 {
			countStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Align(lipgloss.Center).
				Width(f.Width)
			b.WriteString(countStyle.Render(fmt.Sprintf("(%d more items)", len(f.VisibleRows)-endIdx)) + "\n")
		}
	}

	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
