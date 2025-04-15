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

// LogOperation logs an operation with status to the logger
func (ea *EngineAdapter) LogOperation(source, destination, status string, err error) {
	if status == "start" {
		ea.logger.With(
			log.F("source", source),
			log.F("destination", destination),
		).Info("→ Starting file operation")
	} else if status == "complete" {
		ea.logger.With(
			log.F("source", source),
			log.F("destination", destination),
		).Info("✓ Operation completed successfully")
	} else if status == "error" && err != nil {
		ea.logger.With(
			log.F("source", source),
			log.F("destination", destination),
			log.F("error", err.Error()),
		).Error("✗ Operation failed")
	}
}

// MoveFile moves a file from source to destination
func (ea *EngineAdapter) MoveFile(source, destination string) error {
	// Log the start of the operation
	ea.LogOperation(source, destination, "start", nil)

	// Create the destination directory if it doesn't exist
	destDir := filepath.Dir(destination)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		ea.LogOperation(source, destination, "error", err)
		return err
	}

	// If in dry run mode, we don't actually move the file
	if ea.dryRun {
		ea.LogOperation(source, destination, "complete", nil)
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
	err := ea.engine.MoveFile(source, destination)
	if err != nil {
		ea.LogOperation(source, destination, "error", err)
		return err
	}

	// Log the completion of the operation
	ea.LogOperation(source, destination, "complete", nil)
	return nil
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
	// Should only return true if both the integration exists and learning is enabled
	return ea.learningEnabled && ea.learningIntegration != nil
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

	// Log the start of the enrichment
	ea.logger.With(log.F("file", fileInfo.Path)).Info("→ Starting file classification")

	enrichedInfo, err := ea.learningIntegration.EnrichFileInfo(fileInfo)

	if err != nil {
		ea.logger.With(
			log.F("file", fileInfo.Path),
			log.F("error", err.Error()),
		).Error("✗ Classification failed")
		return fileInfo, err
	}

	if enrichedInfo != nil && len(enrichedInfo.Tags) > 0 {
		ea.logger.With(
			log.F("file", fileInfo.Path),
			log.F("tags", strings.Join(enrichedInfo.Tags, ", ")),
		).Info("✓ Classification completed")
	}

	return enrichedInfo, nil
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

			// Log the analysis step
			ea.logger.With(log.F("file", filePath)).Info("→ Analyzing file content")

			// Get file info with content analysis
			enrichedInfo, err := ea.learningIntegration.EnrichFileInfo(&types.FileInfo{
				Path: filePath,
				Size: fileInfo.Size(),
			})

			if err != nil {
				ea.logger.With(
					log.F("file", filePath),
					log.F("error", err.Error()),
				).Warn("⚠ Content analysis failed, continuing with basic organization")
			} else if enrichedInfo != nil && len(enrichedInfo.Tags) > 0 {
				// Log the enhanced classification
				tags := strings.Join(enrichedInfo.Tags, ", ")
				ea.logger.With(
					log.F("file", filePath),
					log.F("tags", tags),
				).Info("✓ Enhanced classification completed")
			}
		}
	}

	// Forward to the organize engine
	ea.logger.Info("→ Organizing files by patterns")
	err := ea.engine.OrganizeByPatterns(files)
	if err != nil {
		ea.logger.With(log.F("error", err.Error())).Error("✗ Organization failed")
		return err
	}

	ea.logger.Info("✓ Organization completed successfully")
	return nil
}

// SetupLearningEngine initializes the learning engine with a database path
func (ea *EngineAdapter) SetupLearningEngine(dbPath string) error {
	ea.logger.With(log.F("dbPath", dbPath)).Info("→ Setting up learning engine")

	// If dbPath is empty or invalid, use an in-memory database
	if dbPath == "" {
		dbPath = ":memory:"
		ea.logger.Info("Using in-memory database for learning engine")
	}

	// Create a repository
	repo, err := learning.NewSQLiteRepository(dbPath, ea.logger)
	if err != nil {
		ea.logger.With(log.F("error", err.Error())).Error("✗ Failed to create repository")
		return err
	}

	// Create a config
	config := learning.DefaultConfig()
	config.DatabasePath = dbPath

	// Create and set the engine
	engine := learning.NewEngine(repo, config, ea.logger)
	ea.SetLearningEngine(engine)

	// Enable learning by default
	engine.SetLearningEnabled(true)

	ea.logger.Info("✓ Learning engine setup completed successfully")
	return nil
}
