package learning

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AnalysisConfig contains configuration parameters for the learning system
type AnalysisConfig struct {
	// Minimum confidence for pattern suggestion
	MinConfidenceThreshold float64 `json:"min_confidence_threshold"`

	// Feature toggles
	LearningEnabled        bool `json:"learning_enabled"`
	ContentSamplingEnabled bool `json:"content_sampling_enabled"`

	// Analysis parameters
	AnalysisFrequencyMins int `json:"analysis_frequency_minutes"`
	MaxSuggestionsPerDay  int `json:"max_suggestions_per_day"`

	// Confidence weights
	ExtensionPatternsWeight float64 `json:"extension_patterns_weight"`
	NamePatternsWeight      float64 `json:"name_patterns_weight"`
	ContentPatternsWeight   float64 `json:"content_patterns_weight"`
	TimePatternsWeight      float64 `json:"time_patterns_weight"`

	// Pattern detection parameters
	MinOperationsForPattern int `json:"min_operations_for_pattern"`
	RecencyDecayDays        int `json:"recency_decay_days"`
	ContentSampleMaxBytes   int `json:"content_sample_max_bytes"`

	// Database configuration
	DatabasePath string `json:"database_path"`
}

// DefaultConfig creates a default configuration
func DefaultConfig() *AnalysisConfig {
	return &AnalysisConfig{
		MinConfidenceThreshold:  0.6,
		LearningEnabled:         true,
		ContentSamplingEnabled:  true,
		AnalysisFrequencyMins:   60,
		MaxSuggestionsPerDay:    5,
		ExtensionPatternsWeight: 0.4,
		NamePatternsWeight:      0.3,
		ContentPatternsWeight:   0.2,
		TimePatternsWeight:      0.1,
		MinOperationsForPattern: 3,
		RecencyDecayDays:        30,
		ContentSampleMaxBytes:   4096,
		DatabasePath:            "",
	}
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(configPath string) (*AnalysisConfig, error) {
	// Start with default config
	config := DefaultConfig()

	// If no config path specified, return defaults
	if configPath == "" {
		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file exists, save defaults and return
			err = SaveConfig(config, configPath)
			return config, err
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse config file
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate and normalize config
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveConfig saves configuration to a JSON file
func SaveConfig(config *AnalysisConfig, configPath string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Validate config
	if err := ValidateConfig(config); err != nil {
		return err
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ValidateConfig validates the configuration
func ValidateConfig(config *AnalysisConfig) error {
	// Validate confidence threshold
	if config.MinConfidenceThreshold < 0 || config.MinConfidenceThreshold > 1 {
		return fmt.Errorf("confidence threshold must be between 0 and 1")
	}

	// Validate weights sum to 1.0 (with small tolerance for floating point)
	weightSum := config.ExtensionPatternsWeight +
		config.NamePatternsWeight +
		config.ContentPatternsWeight +
		config.TimePatternsWeight

	if weightSum < 0.99 || weightSum > 1.01 {
		return fmt.Errorf("pattern weights must sum to 1.0")
	}

	// Validate min operations is positive
	if config.MinOperationsForPattern < 1 {
		return fmt.Errorf("minimum operations for pattern must be at least 1")
	}

	// Validate recency decay is positive
	if config.RecencyDecayDays < 1 {
		return fmt.Errorf("recency decay days must be at least 1")
	}

	// Validate analysis frequency
	if config.AnalysisFrequencyMins < 5 {
		return fmt.Errorf("analysis frequency must be at least 5 minutes")
	}

	// Normalize database path
	if config.DatabasePath != "" {
		config.DatabasePath = filepath.Clean(config.DatabasePath)
	}

	return nil
}
