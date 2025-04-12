package learning

import (
	"fmt"
	"path/filepath"
	"time"

	"sortd/internal/log"
)

// Engine is the main component of the Smart Rule Learning system
type Engine struct {
	repo           Repository
	config         *AnalysisConfig
	logger         log.Logging
	analysisTimer  *time.Timer
	stopChan       chan struct{}
	isRunning      bool
	lastAnalysisAt time.Time
}

// NewEngine creates a new learning engine
func NewEngine(repo Repository, config *AnalysisConfig, logger log.Logging) *Engine {
	return &Engine{
		repo:      repo,
		config:    config,
		logger:    logger,
		stopChan:  make(chan struct{}),
		isRunning: false,
	}
}

// Start begins the learning engine's analysis cycle
func (e *Engine) Start() error {
	if e.isRunning {
		return fmt.Errorf("learning engine is already running")
	}

	freqField := log.F("analysisFrequency", fmt.Sprintf("%d minutes", e.config.AnalysisFrequencyMins))
	confField := log.F("minConfidence", e.config.MinConfidenceThreshold)
	e.logger.With(freqField, confField).Info("Starting Smart Rule Learning engine")

	// Start the analysis timer
	e.scheduleNextAnalysis()
	e.isRunning = true

	go func() {
		for {
			select {
			case <-e.analysisTimer.C:
				if err := e.PerformAnalysis(); err != nil {
					e.logger.With(log.F("error", err)).Error("Failed to perform pattern analysis")
				}
				e.scheduleNextAnalysis()
			case <-e.stopChan:
				if e.analysisTimer != nil {
					e.analysisTimer.Stop()
				}
				e.logger.Info("Smart Rule Learning engine stopped")
				return
			}
		}
	}()

	return nil
}

// Stop halts the learning engine's analysis cycle
func (e *Engine) Stop() {
	if !e.isRunning {
		return
	}

	e.stopChan <- struct{}{}
	e.isRunning = false
}

// scheduleNextAnalysis sets up the timer for the next analysis
func (e *Engine) scheduleNextAnalysis() {
	// Schedule the next analysis based on the configured frequency
	nextAnalysis := time.Duration(e.config.AnalysisFrequencyMins) * time.Minute
	e.analysisTimer = time.NewTimer(nextAnalysis)
	e.logger.With(log.F("nextAnalysisIn", nextAnalysis.String())).Debug("Scheduled next pattern analysis")
}

// PerformAnalysis runs the pattern detection and suggestion generation
func (e *Engine) PerformAnalysis() error {
	if !e.config.LearningEnabled {
		e.logger.Debug("Learning is disabled, skipping analysis")
		return nil
	}

	e.logger.Info("Starting pattern analysis")
	startTime := time.Now()
	e.lastAnalysisAt = startTime

	// Get recent operations
	ops, err := e.repo.GetRecentOperations(e.config.RecencyDecayDays, 1000)
	if err != nil {
		return fmt.Errorf("failed to get recent operations: %w", err)
	}

	e.logger.With(log.F("operationCount", len(ops))).Info("Analyzing operations for patterns")

	// TODO: Implement full pattern detection
	// This is a placeholder for the actual pattern detection logic
	// The implementation would analyze operations to detect patterns

	// Log the results
	duration := time.Since(startTime)
	e.logger.With(
		log.F("duration", duration.String()),
		log.F("operationsAnalyzed", len(ops)),
	).Info("Pattern analysis completed")

	return nil
}

// TrackOperation records a file operation for future pattern learning
func (e *Engine) TrackOperation(record *OperationRecord) error {
	if record == nil {
		return fmt.Errorf("operation record cannot be nil")
	}

	// Only track operations if learning is enabled
	if !e.config.LearningEnabled {
		return nil
	}

	// Set defaults if not provided
	if record.ID == "" {
		return fmt.Errorf("operation ID must be provided")
	}

	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	// Extract filename and extension if not provided
	if record.FileName == "" && record.SourcePath != "" {
		record.FileName = filepath.Base(record.SourcePath)
	}

	if record.FileExt == "" && record.SourcePath != "" {
		record.FileExt = filepath.Ext(record.SourcePath)
	}

	// Save the operation record
	if err := e.repo.SaveOperationRecord(record); err != nil {
		e.logger.With(
			log.F("error", err),
			log.F("operation", record.OperationType),
			log.F("sourcePath", record.SourcePath),
		).Error("Failed to save operation record")
		return err
	}

	e.logger.With(
		log.F("operation", record.OperationType),
		log.F("source", record.SourcePath),
		log.F("destination", record.DestinationPath),
		log.F("manual", record.Manual),
	).Debug("Tracked file operation")

	return nil
}

// ClassifyFile determines the most likely classification for a file
func (e *Engine) ClassifyFile(filePath string) ([]*ClassificationMatch, error) {
	// TODO: Implement file classification
	// This is a placeholder for the actual classification logic
	return nil, fmt.Errorf("file classification not yet implemented")
}

// GetSuggestedRules returns rule suggestions for the user
func (e *Engine) GetSuggestedRules(minConfidence float64) ([]*RuleSuggestion, error) {
	// TODO: Implement rule suggestion retrieval
	// This is a placeholder for the actual suggestion logic
	return nil, fmt.Errorf("rule suggestions not yet implemented")
}

// ProcessFeedback handles user feedback on rule suggestions
func (e *Engine) ProcessFeedback(suggestionID string, accepted bool, modifications map[string]interface{}) error {
	// TODO: Implement feedback processing
	// This is a placeholder for the actual feedback processing logic
	return fmt.Errorf("feedback processing not yet implemented")
}

// IsLearningEnabled returns whether learning is currently enabled
func (e *Engine) IsLearningEnabled() bool {
	return e.config.LearningEnabled
}

// SetLearningEnabled toggles learning on or off
func (e *Engine) SetLearningEnabled(enabled bool) {
	e.config.LearningEnabled = enabled
	e.logger.With(log.F("enabled", enabled)).Info("Learning engine state changed")
}

// GetLastAnalysisTime returns the time of the last pattern analysis
func (e *Engine) GetLastAnalysisTime() time.Time {
	return e.lastAnalysisAt
}
