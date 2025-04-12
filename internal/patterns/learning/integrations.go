// Package learning provides smart rule learning capabilities
package learning

import (
	"fmt"
	"path/filepath"
	"sortd/internal/log"
	"sortd/pkg/types"
	"strings"
	"time"
)

// OrganizeEngineIntegration provides integration helpers between the learning and organization engines
type OrganizeEngineIntegration struct {
	engine *Engine
	logger log.Logging
}

// NewOrganizeIntegration creates a new integration helper
func NewOrganizeIntegration(engine *Engine, logger log.Logging) *OrganizeEngineIntegration {
	return &OrganizeEngineIntegration{
		engine: engine,
		logger: logger,
	}
}

// GeneratePattern converts a classification to an organization pattern
func (i *OrganizeEngineIntegration) GeneratePattern(classification *FileClassification, destination string) *types.Pattern {
	if classification == nil {
		return nil
	}

	// Create a match string from extension patterns
	var match string
	if len(classification.Criteria.ExtensionPatterns) > 0 {
		// Use the first extension pattern for now
		// Could be improved to handle multiple patterns
		match = "*" + classification.Criteria.ExtensionPatterns[0]
	} else {
		// Fallback to a generic match based on name patterns
		if len(classification.Criteria.NamePatterns) > 0 {
			match = "*" + classification.Criteria.NamePatterns[0] + "*"
		} else {
			// Last resort - use ID as part of the pattern
			match = "*" + classification.ID + "*"
		}
	}

	// Based on checking the types.Pattern struct, it only has Match and Target fields
	return &types.Pattern{
		Match:  match,
		Target: destination,
	}
}

// ClassifyAndOrganize classifies a file and returns appropriate organization patterns
func (i *OrganizeEngineIntegration) ClassifyAndOrganize(filePath string, defaultDestDir string) ([]types.Pattern, error) {
	logger := i.logger.With(log.F("file", filePath))
	logger.Debug("Classifying file for organization")

	// Get classifications for the file
	matches, err := i.engine.ClassifyFile(filePath)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		logger.Debug("No classifications found for file")
		return nil, nil
	}

	var patterns []types.Pattern
	for _, match := range matches {
		// Get the full classification details
		classification, err := i.engine.repo.GetClassificationByID(match.ClassificationID)
		if err != nil {
			logger.With(log.F("classificationID", match.ClassificationID), log.F("error", err)).
				Warn("Failed to get classification details")
			continue
		}

		// Default destination is classification ID subdirectory
		destination := filepath.Join(defaultDestDir, classification.ID)

		// Create a pattern from this classification
		pattern := i.GeneratePattern(classification, destination)
		if pattern != nil {
			// We can't add confidence to description as Pattern doesn't have that field
			// Let's log the confidence instead
			logger.With(
				log.F("classification", classification.Name),
				log.F("confidence", match.Confidence),
			).Debug("Adding pattern based on classification")

			patterns = append(patterns, *pattern)
		}
	}

	logger.With(log.F("patternCount", len(patterns))).Info("Generated organization patterns from classifications")
	return patterns, nil
}

// SuggestDestination suggests a destination directory for a file based on its content and classifications
func (i *OrganizeEngineIntegration) SuggestDestination(filePath string, defaultDir string) string {
	// Get classifications
	matches, err := i.engine.ClassifyFile(filePath)
	if err != nil || len(matches) == 0 {
		return defaultDir
	}

	// Use the highest confidence classification
	var bestMatch *ClassificationMatch
	for _, match := range matches {
		if bestMatch == nil || match.Confidence > bestMatch.Confidence {
			bestMatch = match
		}
	}

	// Get classification details
	classification, err := i.engine.repo.GetClassificationByID(bestMatch.ClassificationID)
	if err != nil {
		return defaultDir
	}

	// Create destination path from classification
	return filepath.Join(defaultDir, classification.ID)
}

// Helper function to check if a string is in a slice
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// EnrichFileInfo adds classification information to FileInfo
func (i *OrganizeEngineIntegration) EnrichFileInfo(info *types.FileInfo) (*types.FileInfo, error) {
	if info == nil {
		return nil, nil
	}

	// Get classifications
	matches, err := i.engine.ClassifyFile(info.Path)
	if err != nil || len(matches) == 0 {
		return info, nil
	}

	// Initialize tags if needed
	if info.Tags == nil {
		info.Tags = []string{}
	}

	// Initialize metadata if needed
	if info.Metadata == nil {
		info.Metadata = make(map[string]string)
	}

	// Add classification information
	for _, match := range matches {
		// Get classification details
		classification, err := i.engine.repo.GetClassificationByID(match.ClassificationID)
		if err != nil {
			continue
		}

		// Add classification ID as a tag if not already present
		if !containsString(info.Tags, classification.ID) {
			info.Tags = append(info.Tags, classification.ID)
		}

		// Add confidence to metadata
		info.Metadata["Classification_"+classification.ID] = fmt.Sprintf("%.2f", match.Confidence)
	}

	// Get content signature if available
	signature, err := i.engine.repo.GetContentSignatureByPath(info.Path)
	if err == nil && signature != nil {
		// Add signature type as a tag
		if !containsString(info.Tags, signature.SignatureType) {
			info.Tags = append(info.Tags, signature.SignatureType)
		}

		// Add MIME type to metadata if not already set
		if info.ContentType == "" {
			info.ContentType = signature.MimeType
		}

		// Add keywords to metadata
		if len(signature.Keywords) > 0 {
			info.Metadata["Keywords"] = strings.Join(signature.Keywords, ", ")
		}
	}

	return info, nil
}

// TrackOrganizeOperation tracks a file organization operation in the learning system
func (i *OrganizeEngineIntegration) TrackOrganizeOperation(
	sourcePath, destPath string,
	fileSize int64,
	operationType string,
	isManual bool) error {

	// Create operation record
	record := &OperationRecord{
		ID:              GenerateID(),
		Timestamp:       time.Now(),
		OperationType:   operationType,
		SourcePath:      sourcePath,
		DestinationPath: destPath,
		FileName:        filepath.Base(sourcePath),
		FileExt:         filepath.Ext(sourcePath),
		FileSize:        fileSize,
		Manual:          isManual,
		Success:         true,
	}

	// Track the operation
	return i.engine.TrackOperation(record)
}

// GenerateID generates a unique ID for operations
func GenerateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
