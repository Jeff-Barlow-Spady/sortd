package learning

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	// Get the default config
	config := DefaultConfig()

	// Verify default values
	if config.MinConfidenceThreshold != 0.6 {
		t.Errorf("Expected MinConfidenceThreshold 0.6, got %f", config.MinConfidenceThreshold)
	}

	if !config.LearningEnabled {
		t.Error("Expected LearningEnabled to be true")
	}

	if !config.ContentSamplingEnabled {
		t.Error("Expected ContentSamplingEnabled to be true")
	}

	if config.AnalysisFrequencyMins != 60 {
		t.Errorf("Expected AnalysisFrequencyMins 60, got %d", config.AnalysisFrequencyMins)
	}

	if config.MaxSuggestionsPerDay != 5 {
		t.Errorf("Expected MaxSuggestionsPerDay 5, got %d", config.MaxSuggestionsPerDay)
	}

	// Check weights
	weightsSum := config.ExtensionPatternsWeight +
		config.NamePatternsWeight +
		config.ContentPatternsWeight +
		config.TimePatternsWeight

	if weightsSum < 0.99 || weightsSum > 1.01 {
		t.Errorf("Expected weights to sum to 1.0, got %f", weightsSum)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a config file path
	configPath := filepath.Join(tempDir, "test_config.json")

	// Get default config
	config := DefaultConfig()

	// Modify some values
	config.MinConfidenceThreshold = 0.75
	config.LearningEnabled = false
	config.AnalysisFrequencyMins = 30
	config.MaxSuggestionsPerDay = 10
	config.ExtensionPatternsWeight = 0.5
	config.NamePatternsWeight = 0.3
	config.ContentPatternsWeight = 0.1
	config.TimePatternsWeight = 0.1
	config.DatabasePath = filepath.Join(tempDir, "test.db")

	// Save the config
	err = SaveConfig(config, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", configPath)
	}

	// Load the config back
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values match what we saved
	if loadedConfig.MinConfidenceThreshold != config.MinConfidenceThreshold {
		t.Errorf("Expected MinConfidenceThreshold %f, got %f",
			config.MinConfidenceThreshold, loadedConfig.MinConfidenceThreshold)
	}

	if loadedConfig.LearningEnabled != config.LearningEnabled {
		t.Errorf("Expected LearningEnabled %v, got %v",
			config.LearningEnabled, loadedConfig.LearningEnabled)
	}

	if loadedConfig.AnalysisFrequencyMins != config.AnalysisFrequencyMins {
		t.Errorf("Expected AnalysisFrequencyMins %d, got %d",
			config.AnalysisFrequencyMins, loadedConfig.AnalysisFrequencyMins)
	}

	if loadedConfig.MaxSuggestionsPerDay != config.MaxSuggestionsPerDay {
		t.Errorf("Expected MaxSuggestionsPerDay %d, got %d",
			config.MaxSuggestionsPerDay, loadedConfig.MaxSuggestionsPerDay)
	}

	if loadedConfig.DatabasePath != config.DatabasePath {
		t.Errorf("Expected DatabasePath %s, got %s",
			config.DatabasePath, loadedConfig.DatabasePath)
	}

	// Verify that non-existent file returns defaults
	nonExistentPath := filepath.Join(tempDir, "non_existent.json")
	defaultedConfig, err := LoadConfig(nonExistentPath)
	if err != nil {
		t.Fatalf("Failed when loading non-existent config: %v", err)
	}

	// Verify the config file was created with defaults
	if _, err := os.Stat(nonExistentPath); os.IsNotExist(err) {
		t.Fatalf("Default config file was not created at %s", nonExistentPath)
	}

	// Check that we got default values
	if defaultedConfig.MinConfidenceThreshold != 0.6 {
		t.Errorf("Expected default MinConfidenceThreshold 0.6, got %f",
			defaultedConfig.MinConfidenceThreshold)
	}
}

func TestValidateConfig(t *testing.T) {
	// Test valid config
	config := DefaultConfig()
	if err := ValidateConfig(config); err != nil {
		t.Errorf("ValidateConfig failed on default config: %v", err)
	}

	// Test invalid confidence threshold
	t.Run("InvalidConfidenceThreshold", func(t *testing.T) {
		invalidConfig := DefaultConfig()
		invalidConfig.MinConfidenceThreshold = 1.5

		err := ValidateConfig(invalidConfig)
		if err == nil {
			t.Error("Expected error for confidence > 1.0, got nil")
		}

		invalidConfig.MinConfidenceThreshold = -0.1
		err = ValidateConfig(invalidConfig)
		if err == nil {
			t.Error("Expected error for confidence < 0.0, got nil")
		}
	})

	// Test invalid weights
	t.Run("InvalidWeights", func(t *testing.T) {
		invalidConfig := DefaultConfig()
		invalidConfig.ExtensionPatternsWeight = 0.5
		invalidConfig.NamePatternsWeight = 0.5
		invalidConfig.ContentPatternsWeight = 0.5
		invalidConfig.TimePatternsWeight = 0.5

		err := ValidateConfig(invalidConfig)
		if err == nil {
			t.Error("Expected error for weights > 1.0, got nil")
		}
	})

	// Test invalid min operations
	t.Run("InvalidMinOperations", func(t *testing.T) {
		invalidConfig := DefaultConfig()
		invalidConfig.MinOperationsForPattern = 0

		err := ValidateConfig(invalidConfig)
		if err == nil {
			t.Error("Expected error for min operations < 1, got nil")
		}
	})

	// Test invalid recency decay
	t.Run("InvalidRecencyDecay", func(t *testing.T) {
		invalidConfig := DefaultConfig()
		invalidConfig.RecencyDecayDays = 0

		err := ValidateConfig(invalidConfig)
		if err == nil {
			t.Error("Expected error for recency decay < 1, got nil")
		}
	})

	// Test invalid analysis frequency
	t.Run("InvalidAnalysisFrequency", func(t *testing.T) {
		invalidConfig := DefaultConfig()
		invalidConfig.AnalysisFrequencyMins = 1

		err := ValidateConfig(invalidConfig)
		if err == nil {
			t.Error("Expected error for analysis frequency < 5, got nil")
		}
	})

	// Test database path normalization
	t.Run("DatabasePathNormalization", func(t *testing.T) {
		config := DefaultConfig()
		config.DatabasePath = "test/./db/../db/test.db"

		err := ValidateConfig(config)
		if err != nil {
			t.Errorf("ValidateConfig failed on valid config: %v", err)
		}

		// Check path was normalized
		expected := filepath.Clean("test/./db/../db/test.db")
		if config.DatabasePath != expected {
			t.Errorf("Expected normalized path %s, got %s", expected, config.DatabasePath)
		}
	})
}

func TestConfigJSON(t *testing.T) {
	// Test that config can be properly serialized to JSON
	config := DefaultConfig()

	// Convert to JSON
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config to JSON: %v", err)
	}

	// Convert back from JSON
	var parsedConfig AnalysisConfig
	err = json.Unmarshal(data, &parsedConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal config from JSON: %v", err)
	}

	// Verify values were preserved
	if parsedConfig.MinConfidenceThreshold != config.MinConfidenceThreshold {
		t.Errorf("JSON round-trip changed MinConfidenceThreshold from %f to %f",
			config.MinConfidenceThreshold, parsedConfig.MinConfidenceThreshold)
	}

	if parsedConfig.LearningEnabled != config.LearningEnabled {
		t.Errorf("JSON round-trip changed LearningEnabled from %v to %v",
			config.LearningEnabled, parsedConfig.LearningEnabled)
	}

	// Check that all fields were preserved through JSON serialization
	if parsedConfig.ExtensionPatternsWeight != config.ExtensionPatternsWeight ||
		parsedConfig.NamePatternsWeight != config.NamePatternsWeight ||
		parsedConfig.ContentPatternsWeight != config.ContentPatternsWeight ||
		parsedConfig.TimePatternsWeight != config.TimePatternsWeight {
		t.Error("JSON round-trip changed pattern weights")
	}

	if parsedConfig.MinOperationsForPattern != config.MinOperationsForPattern ||
		parsedConfig.RecencyDecayDays != config.RecencyDecayDays ||
		parsedConfig.ContentSampleMaxBytes != config.ContentSampleMaxBytes {
		t.Error("JSON round-trip changed analysis parameters")
	}
}
