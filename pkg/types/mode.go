package types

// Mode represents the current mode of the TUI
type Mode int

const (
	// Normal is the default mode for file navigation and selection
	Normal Mode = iota
	// Setup is the initial mode when the application starts
	Setup
	// Command is the mode for entering commands
	Command
	// Visual is the mode for visual selection
	Visual
)

// ModelReader defines the interface that views use to read model state
type ModelReader interface {
	Files() []*FileInfo
	IsSelected(name string) bool
	Cursor() int
	ShowHelp() bool
	Mode() Mode
	CurrentDir() string
}

// FileEntry represents a file or directory in the TUI list.
// Note: This might be similar but distinct from FileInfo used by analysis.
type FileEntry struct {
	Name        string
	Path        string
	ContentType string // e.g., "image/jpeg", "text/plain", "inode/directory"
	Size        int64
	Tags        []string
	IsDir       bool // Added field to distinguish files and directories
}

// FilterValue is required by the list component for filtering.
type FilterValue struct {
	Value string
}
