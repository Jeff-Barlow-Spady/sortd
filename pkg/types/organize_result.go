package types

// OrganizeResult represents the result of organizing a file
type OrganizeResult struct {
	SourcePath      string `json:"source_path"`
	DestinationPath string `json:"destination_path"`
	Moved           bool   `json:"moved"`
	Error           error  `json:"error,omitempty"`
}
