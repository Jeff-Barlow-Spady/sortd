package common

type Mode int

const (
	Normal Mode = iota
	Command
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

type FileEntry struct {
	Name string
	Path string
}
