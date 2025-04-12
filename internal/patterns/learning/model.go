package learning

import (
	"encoding/json"
	"time"
)

// OperationRecord tracks file organization operations for learning patterns
type OperationRecord struct {
	ID              string    `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	OperationType   string    `json:"operation_type"` // "move", "copy", "rename", etc.
	SourcePath      string    `json:"source_path"`
	DestinationPath string    `json:"destination_path"`
	FileName        string    `json:"file_name"`
	FileExt         string    `json:"file_ext"`
	FileSize        int64     `json:"file_size"`
	Manual          bool      `json:"manual"` // Whether operation was performed manually or via rule
	Success         bool      `json:"success"`
}

// DetectedPattern represents a pattern identified by the learning system
type DetectedPattern struct {
	ID                   string    `json:"id"`
	PatternType          string    `json:"pattern_type"` // "extension", "name", "time", "content"
	PatternValue         string    `json:"pattern_value"`
	DestinationPath      string    `json:"destination_path"`
	Confidence           float64   `json:"confidence"`
	OccurrenceCount      int       `json:"occurrence_count"`
	FirstSeen            time.Time `json:"first_seen"`
	LastSeen             time.Time `json:"last_seen"`
	MatchCount           int       `json:"match_count"`            // Only in memory
	TotalPossibleMatches int       `json:"total_possible_matches"` // Only in memory
}

// ClassifierCriteria defines how files are matched against classifications
type ClassifierCriteria struct {
	ExtensionPatterns []string `json:"extension_patterns"`
	NamePatterns      []string `json:"name_patterns"`
	ContentSignatures []string `json:"content_signatures"`
	MinFileSize       int64    `json:"min_file_size"`
	MaxFileSize       int64    `json:"max_file_size"`
	MimeTypes         []string `json:"mime_types"`
}

// FileClassification defines a type of file with its identification criteria
type FileClassification struct {
	ID                  string             `json:"id"`
	Name                string             `json:"name"`
	Description         string             `json:"description"`
	Criteria            ClassifierCriteria `json:"criteria"`
	ConfidenceThreshold float64            `json:"confidence_threshold"`
	SystemDefined       bool               `json:"system_defined"`
}

// ClassificationMatch represents a file matching a classification
type ClassificationMatch struct {
	FilePath         string    `json:"file_path"`
	ClassificationID string    `json:"classification_id"`
	Confidence       float64   `json:"confidence"`
	Timestamp        time.Time `json:"timestamp"`
}

// ContentSignature represents a content-based signature of a file
type ContentSignature struct {
	ID            string    `json:"id"`
	FilePath      string    `json:"file_path"`
	MimeType      string    `json:"mime_type"`
	SignatureType string    `json:"signature_type"` // "text", "image", "document", "binary", etc.
	Signature     string    `json:"signature"`      // JSON or binary encoded signature
	Keywords      []string  `json:"keywords"`       // For text and document content
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	FileSize      int64     `json:"file_size"`
}

// ContentRelationship represents a relationship between two files based on content
type ContentRelationship struct {
	ID           string    `json:"id"`
	SourceID     string    `json:"source_id"`     // ContentSignature ID
	TargetID     string    `json:"target_id"`     // ContentSignature ID
	Similarity   float64   `json:"similarity"`    // 0.0 to 1.0
	RelationType string    `json:"relation_type"` // "similar", "derived", "contains", etc.
	CreatedAt    time.Time `json:"created_at"`
}

// ContentGroup represents a group of related files
type ContentGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	GroupType   string    `json:"group_type"` // "similarity", "project", "version", etc.
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ContentGroupMember represents a file's membership in a content group
type ContentGroupMember struct {
	GroupID         string    `json:"group_id"`
	SignatureID     string    `json:"signature_id"`
	MembershipScore float64   `json:"membership_score"` // How strongly the file belongs to the group
	JoinedAt        time.Time `json:"joined_at"`
}

// RuleSuggestion represents a suggested organization rule
type RuleSuggestion struct {
	ID                string                 `json:"id"`
	PatternID         string                 `json:"pattern_id"`
	ClassificationID  string                 `json:"classification_id"`
	SuggestedRuleJSON string                 `json:"suggested_rule_json"`
	Confidence        float64                `json:"confidence"`
	Status            string                 `json:"status"` // "pending", "accepted", "rejected", "modified"
	CreatedAt         time.Time              `json:"created_at"`
	RespondedAt       time.Time              `json:"responded_at"`
	UserModifications map[string]interface{} `json:"user_modifications"`
}

// SuggestedRule represents the actual rule being suggested
type SuggestedRule struct {
	PatternType     string   `json:"pattern_type"`
	PatternValue    string   `json:"pattern_value"`
	DestinationPath string   `json:"destination_path"`
	Description     string   `json:"description"`
	Classification  string   `json:"classification"`
	Confidence      float64  `json:"confidence"`
	Examples        []string `json:"examples"`
}

// ToJSON serializes a struct to JSON
func ToJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON deserializes JSON to a struct
func FromJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}

// LearningSettings stores configuration for the learning engine
type LearningSettings struct {
	ID                      string  `json:"id"`
	MinConfidenceThreshold  float64 `json:"min_confidence_threshold"`
	LearningEnabled         bool    `json:"learning_enabled"`
	AnalysisFrequencyMins   int     `json:"analysis_frequency_minutes"`
	MaxSuggestionsPerDay    int     `json:"max_suggestions_per_day"`
	ContentSamplingEnabled  bool    `json:"content_sampling_enabled"`
	ExtensionPatternsWeight float64 `json:"extension_patterns_weight"`
	NamePatternsWeight      float64 `json:"name_patterns_weight"`
	ContentPatternsWeight   float64 `json:"content_patterns_weight"`
	TimePatternsWeight      float64 `json:"time_patterns_weight"`
	MinOperationsForPattern int     `json:"min_operations_for_pattern"`
	RecencyDecayDays        int     `json:"recency_decay_days"`
	ContentSampleMaxBytes   int     `json:"content_sample_max_bytes"`
}
