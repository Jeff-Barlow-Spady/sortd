package gui

import (
	"sortd/internal/config"
	"sortd/internal/organize"
)

// Interface defines the contract for GUI operations
type Interface interface {
	Run()
	ShowError(title string, err error)
	ShowInfo(message string)
}

// Factory creates GUI instances
type Factory struct {
	config         *config.Config
	organizeEngine *organize.Engine
}

// NewFactory creates a new GUI factory
func NewFactory(cfg *config.Config, organizeEngine *organize.Engine) *Factory {
	return &Factory{
		config:         cfg,
		organizeEngine: organizeEngine,
	}
}

// Create returns a new GUI instance
func (f *Factory) Create() (Interface, error) {
	// Create the GUI implementation
	app := NewApp(f.config, f.organizeEngine)
	return app, nil
}
