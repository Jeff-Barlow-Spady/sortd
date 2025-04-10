package types

// Pattern defines a rule for matching files and specifying their target directory.
// It is used within the application's configuration.
type Pattern struct {
	Match  string `yaml:"match"`  // Glob pattern to match filenames (e.g., "*.pdf", "report_*.docx").
	Target string `yaml:"target"` // Target directory path where matched files should be moved (e.g., "Documents/Reports", "Images/Screenshots").
}

// Note: Removed redundant fields Glob, Prefixes, Suffixes, DestDir for clarity
// and aligned struct tags with YAML configuration format.
