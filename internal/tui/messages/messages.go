package messages

import (
	"sortd/internal/config"
	"sortd/internal/tui/common"
	"sortd/pkg/types"
)

type ErrorMsg struct {
	Err error
}

type OrganizeCompleteMsg struct{}

type ScanCompleteMsg struct {
	Files []common.FileEntry
	Error error
}

type AnalysisCompleteMsg struct {
	Results []*types.FileInfo
}

type ConfigUpdateMsg struct {
	Config *config.Config
}

type DirectoryChangeMsg struct {
	Path  string
	Error error
}
