package types

// OrganizeResult holds the outcome of an organization attempt for a single file
type OrganizeResult struct {
	SourcePath      string `json:"source_path"`
	DestinationPath string `json:"destination_path"`
	Moved           bool   `json:"moved"`
	Error           error  `json:"error,omitempty"`
}
