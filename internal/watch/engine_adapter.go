package watch

import (
	"os"
	"path/filepath"
	"sortd/internal/config"
	"sortd/internal/organize"
)

// EngineAdapter wraps the organize.Engine to provide additional functionality
type EngineAdapter struct {
	engine *organize.Engine
	dryRun bool
}

// NewEngineAdapter creates a new adapter around the organize engine
func NewEngineAdapter(cfg *config.Config) *EngineAdapter {
	engine := organize.NewWithConfig(cfg)

	return &EngineAdapter{
		engine: engine,
		dryRun: false,
	}
}

// SetDryRun sets whether to run in dry run mode
func (ea *EngineAdapter) SetDryRun(dryRun bool) {
	ea.engine.SetDryRun(dryRun)
	ea.dryRun = dryRun
}

// GetDryRun returns whether the engine is in dry run mode
func (ea *EngineAdapter) GetDryRun() bool {
	return ea.dryRun
}

// MoveFile moves a file from source to destination
func (ea *EngineAdapter) MoveFile(source, destination string) error {
	// Create the destination directory if it doesn't exist
	destDir := filepath.Dir(destination)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// If in dry run mode, we don't actually move the file
	if ea.dryRun {
		return nil
	}

	// Delegate to the engine to do the actual move
	return ea.engine.MoveFile(source, destination)
}
