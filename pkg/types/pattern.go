package types

// Pattern defines rules for matching files
type Pattern struct {
	Glob     string   `json:"glob"`     // e.g., "*.pdf"
	Prefixes []string `json:"prefixes"` // e.g., ["invoice", "receipt"]
	Suffixes []string `json:"suffixes"` // e.g., ["2024", "final"]
	DestDir  string   `json:"dest_dir"` // destination directory for matched files
	Match    string   `json:"match"`    // e.g., "*.pdf"
	Target   string   `json:"target"`   // e.g., "documents/"
}
