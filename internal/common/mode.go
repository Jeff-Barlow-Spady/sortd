package common

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
	Files() []FileEntry
	IsSelected(name string) bool
	Cursor() int
	ShowHelp() bool
	Mode() Mode
	CurrentDir() string
}

// FileEntry represents a file in the current directory
type FileEntry struct {
	Name        string
	Path        string
	ContentType string
	Size        int64
	Tags        []string
}
