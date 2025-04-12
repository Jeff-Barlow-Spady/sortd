package learning

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"sortd/internal/log"
)

// Engine is the main component of the Smart Rule Learning system
type Engine struct {
	repo            Repository
	config          *AnalysisConfig
	logger          log.Logging
	analysisTimer   *time.Timer
	stopChan        chan struct{}
	isRunning       bool
	lastAnalysisAt  time.Time
	contentAnalyzer ContentAnalyzer
}

// NewEngine creates a new learning engine
func NewEngine(repo Repository, config *AnalysisConfig, logger log.Logging) *Engine {
	e := &Engine{
		repo:      repo,
		config:    config,
		logger:    logger,
		stopChan:  make(chan struct{}),
		isRunning: false,
	}

	// Initialize content analyzer
	settings := &LearningSettings{
		ContentSamplingEnabled: config.ContentSamplingEnabled,
		ContentSampleMaxBytes:  config.ContentSampleMaxBytes,
	}
	e.contentAnalyzer = NewContentAnalyzer(repo, logger, settings)

	return e
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
	logger := e.logger.With(log.F("file", filePath))
	logger.Debug("Classifying file based on patterns and content analysis")

	// Get existing file classifications from the database first
	existingMatches, err := e.repo.GetFileClassifications(filePath)
	if err == nil && len(existingMatches) > 0 {
		logger.With(log.F("matchCount", len(existingMatches))).Debug("Using existing classifications from database")
		return existingMatches, nil
	}

	// Get all available classifications
	classifications, err := e.repo.GetAllClassifications()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve classifications: %w", err)
	}

	if len(classifications) == 0 {
		logger.Warn("No classifications available for matching")
		return nil, nil
	}

	// Generate content signature for the file
	var signature *ContentSignature
	if e.config.ContentSamplingEnabled {
		// First check if we already have a signature
		signature, err = e.repo.GetContentSignatureByPath(filePath)
		if err != nil || signature == nil {
			// If not, generate one
			signature, err = e.contentAnalyzer.AnalyzeFile(filePath)
			if err != nil {
				logger.With(log.F("error", err)).Warn("Failed to generate content signature, will classify without content analysis")
			}
		}
	}

	// Prepare file info for classification
	fileName := filepath.Base(filePath)
	fileExt := filepath.Ext(filePath)

	// Calculate match for each classification
	var matches []*ClassificationMatch
	now := time.Now()

	for _, classification := range classifications {
		confidence := 0.0

		// Match extension patterns
		for _, extPattern := range classification.Criteria.ExtensionPatterns {
			if extPattern == fileExt {
				confidence += 0.4 // Weight for extension match
				break
			}
		}

		// Match name patterns
		for _, namePattern := range classification.Criteria.NamePatterns {
			if strings.Contains(strings.ToLower(fileName), strings.ToLower(namePattern)) {
				confidence += 0.3 // Weight for name match
				break
			}
		}

		// Match content signatures if available
		if signature != nil {
			// Check for MIME type match
			for _, mimeType := range classification.Criteria.MimeTypes {
				if mimeType == signature.MimeType {
					confidence += 0.2 // Weight for MIME type match
					break
				}
			}

			// Use content signature matches from classification criteria
			for _, sigPattern := range classification.Criteria.ContentSignatures {
				if strings.Contains(signature.Signature, sigPattern) {
					confidence += 0.1 // Weight for content signature match
					break
				}
			}
		}

		// Add match if confidence exceeds threshold
		if confidence >= classification.ConfidenceThreshold {
			match := &ClassificationMatch{
				FilePath:         filePath,
				ClassificationID: classification.ID,
				Confidence:       confidence,
				Timestamp:        now,
			}
			matches = append(matches, match)

			// Save the match to the database
			if err := e.repo.SaveClassificationMatch(match); err != nil {
				logger.With(log.F("error", err)).Warn("Failed to save classification match")
			}
		}
	}

	// If content analysis is enabled, find related files
	if signature != nil && e.config.ContentSamplingEnabled {
		// Look for content relationships to improve classifications
		relationships, err := e.contentAnalyzer.FindRelatedFiles(filePath, 0.7, 5)
		if err == nil && len(relationships) > 0 {
			logger.With(log.F("relationCount", len(relationships))).Debug("Found related files that may improve classification")

			// Analyze classifications of related files to potentially add new classifications
			for _, rel := range relationships {
				// Get target file path from relationship
				var targetSig *ContentSignature
				targetSig, err = e.repo.GetContentSignature(rel.TargetID)
				if err != nil {
					continue
				}

				// Get classifications for the related file
				relatedMatches, err := e.repo.GetFileClassifications(targetSig.FilePath)
				if err != nil || len(relatedMatches) == 0 {
					continue
				}

				// Consider adding these classifications to our file with adjusted confidence
				for _, relMatch := range relatedMatches {
					// Check if we already have this classification
					alreadyHas := false
					for _, existingMatch := range matches {
						if existingMatch.ClassificationID == relMatch.ClassificationID {
							alreadyHas = true
							break
						}
					}

					if !alreadyHas {
						// Add this classification with reduced confidence based on similarity
						newConfidence := relMatch.Confidence * rel.Similarity

						// Get the classification to check threshold
						var relClassification *FileClassification
						relClassification, err = e.repo.GetClassificationByID(relMatch.ClassificationID)
						if err != nil {
							continue
						}

						// Add match if confidence exceeds threshold
						if newConfidence >= relClassification.ConfidenceThreshold {
							match := &ClassificationMatch{
								FilePath:         filePath,
								ClassificationID: relMatch.ClassificationID,
								Confidence:       newConfidence,
								Timestamp:        now,
							}
							matches = append(matches, match)

							// Save the match to the database
							if err := e.repo.SaveClassificationMatch(match); err != nil {
								logger.With(log.F("error", err)).Warn("Failed to save derived classification match")
							}
						}
					}
				}
			}
		}
	}

	logger.With(log.F("matchCount", len(matches))).Info("File classification completed")
	return matches, nil
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

// GetContentAnalyzer returns the content analyzer instance
func (e *Engine) GetContentAnalyzer() ContentAnalyzer {
	return e.contentAnalyzer
}

// AnalyzeContent is a convenience method to analyze a file's content
func (e *Engine) AnalyzeContent(filePath string) (*ContentSignature, error) {
	return e.contentAnalyzer.AnalyzeFile(filePath)
}

// FindSimilarFiles finds files similar to the given file
func (e *Engine) FindSimilarFiles(filePath string, minSimilarity float64, limit int) ([]*ContentRelationship, error) {
	return e.contentAnalyzer.FindRelatedFiles(filePath, minSimilarity, limit)
}

// CreateContentGroup creates a group of related files
func (e *Engine) CreateContentGroup(name, description, groupType string, filePaths []string) (*ContentGroup, error) {
	// Convert file paths to signature IDs
	var signatureIDs []string
	for _, path := range filePaths {
		// Get or create content signature
		sig, err := e.contentAnalyzer.AnalyzeFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze file %s: %w", path, err)
		}
		signatureIDs = append(signatureIDs, sig.ID)
	}

	return e.contentAnalyzer.CreateContentGroup(name, description, groupType, signatureIDs)
}
