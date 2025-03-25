package organize

import (
	"sortd/internal/config"
	"sortd/pkg/types"
)

// Organizer defines the interface for file organization operations
// This allows for dependency injection in tests and other parts of the application
type Organizer interface {
	// SetConfig sets the config for the organizer
	SetConfig(cfg *config.Config)

	// SetDryRun sets whether operations should be performed or just simulated
	SetDryRun(dryRun bool)

	// AddPattern adds a new organization pattern
	AddPattern(pattern types.Pattern)

	// OrganizeFile organizes a single file based on organization rules
	OrganizeFile(path string) error

	// MoveFile moves a file from source to destination with safety checks
	MoveFile(src, dest string) error

	// OrganizeFiles organizes files to a specific destination directory
	OrganizeFiles(files []string, destDir string) error

	// OrganizeByPatterns organizes files according to the configured patterns
	OrganizeByPatterns(files []string) error

	// OrganizeDir organizes all files in a directory by patterns
	OrganizeDir(dir string) ([]string, error)
}

// Ensure Engine implements the Organizer interface
var _ Organizer = (*Engine)(nil)
