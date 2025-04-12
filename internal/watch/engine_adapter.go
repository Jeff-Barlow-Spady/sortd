package watch

import (
	"os"
	"path/filepath"
	"sortd/internal/config"
	"sortd/internal/log"
	"sortd/internal/organize"
	"sortd/internal/patterns/learning"
	"sortd/pkg/types"
	"strings"
)

// EngineAdapter wraps the organize.Engine to provide additional functionality
type EngineAdapter struct {
	engine              *organize.Engine
	dryRun              bool
	learningEngine      *learning.Engine
	learningEnabled     bool
	learningIntegration *learning.OrganizeEngineIntegration
	logger              log.Logging
}

// NewEngineAdapter creates a new adapter around the organize engine
func NewEngineAdapter(cfg *config.Config) *EngineAdapter {
	engine := organize.NewWithConfig(cfg)
	logger := log.NewLogger()

	adapter := &EngineAdapter{
		engine:          engine,
		dryRun:          false,
		learningEnabled: false,
		logger:          logger,
	}

	return adapter
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

	// Track this operation in the learning system if enabled
	if ea.learningEnabled && ea.learningIntegration != nil {
		// Get file size
		fileInfo, err := os.Stat(source)
		if err == nil {
			// Track the operation (don't return error as this is optional)
			_ = ea.learningIntegration.TrackOrganizeOperation(
				source,
				destination,
				fileInfo.Size(),
				"move",
				false, // Not a manual operation
			)
		}
	}

	// Delegate to the engine to do the actual move
	return ea.engine.MoveFile(source, destination)
}

// SetLearningEngine sets the learning engine for enhanced organization
func (ea *EngineAdapter) SetLearningEngine(learningEngine *learning.Engine) {
	ea.learningEngine = learningEngine

	if learningEngine != nil {
		ea.learningIntegration = learning.NewOrganizeIntegration(learningEngine, ea.logger)
		ea.learningEnabled = true
	} else {
		ea.learningIntegration = nil
		ea.learningEnabled = false
	}
}

// GetLearningEnabled returns whether learning is enabled
func (ea *EngineAdapter) GetLearningEnabled() bool {
	return ea.learningEnabled
}

// SetLearningEnabled toggles learning on or off
func (ea *EngineAdapter) SetLearningEnabled(enabled bool) {
	ea.learningEnabled = enabled

	if ea.learningEngine != nil {
		ea.learningEngine.SetLearningEnabled(enabled)
	}
}

// SuggestDestination uses learning to suggest a destination
func (ea *EngineAdapter) SuggestDestination(filePath string, defaultDir string) string {
	if !ea.learningEnabled || ea.learningIntegration == nil {
		return defaultDir
	}

	return ea.learningIntegration.SuggestDestination(filePath, defaultDir)
}

// EnrichFileInfo adds classification data to file info
func (ea *EngineAdapter) EnrichFileInfo(fileInfo *types.FileInfo) (*types.FileInfo, error) {
	if !ea.learningEnabled || ea.learningIntegration == nil {
		return fileInfo, nil
	}

	return ea.learningIntegration.EnrichFileInfo(fileInfo)
}

// OrganizeByPatterns forwards the organization request to the engine
func (ea *EngineAdapter) OrganizeByPatterns(files []string) error {
	// If content analysis is enabled, try to enhance the classification
	if ea.learningEnabled && ea.learningIntegration != nil {
		for _, filePath := range files {
			// First check if the file exists and get info
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				// Skip this file but continue with others
				continue
			}

			// Skip directories
			if fileInfo.IsDir() {
				continue
			}

			// Get file info with content analysis
			enrichedInfo, err := ea.learningIntegration.EnrichFileInfo(&types.FileInfo{
				Path: filePath,
				Size: fileInfo.Size(),
			})

			if err == nil && enrichedInfo != nil && len(enrichedInfo.Tags) > 0 {
				// Log the enhanced classification
				tags := strings.Join(enrichedInfo.Tags, ", ")
				ea.logger.With(log.F("file", filePath), log.F("tags", tags)).
					Debug("Enhanced file classification before organization")
			}
		}
	}

	// Forward to the organize engine
	return ea.engine.OrganizeByPatterns(files)
}
