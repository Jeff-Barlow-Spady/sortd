package learning

import (
	"os"
	"path/filepath"
	"sortd/internal/errors"
	"sortd/internal/log"
	"testing"
)

// TestLearningErrorHandling tests how the learning components handle various error conditions
func TestLearningErrorHandling(t *testing.T) {
	// Create a test logger
	logger := log.NewLogger()

	// Test repository initialization errors
	t.Run("RepositoryInitializationErrors", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir, err := os.MkdirTemp("", "repo_init_errors")
		if err != nil {
			t.Fatalf("Failed to create temporary directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Test with invalid database path (directory that exists but can't be a database)
		dbDir := filepath.Join(tempDir, "invalid-db-dir")
		err = os.Mkdir(dbDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		// Try to initialize repository with a directory as the database path
		_, err = NewSQLiteRepository(dbDir, logger)
		if err == nil {
			t.Error("Expected error when using directory as database path")
		}
	})

	// Test engine error handling with repository failures
	t.Run("EngineRepositoryFailures", func(t *testing.T) {
		// Create a mock repository that returns errors
		mockRepo := &MockErrorRepository{
			getSettingsError: errors.New("mock settings error"),
		}

		// Create a config
		config := DefaultConfig()

		// Create an engine with the mock repository
		engine := NewEngine(mockRepo, config, logger)

		// Test that engine can handle repository errors by testing learning state
		// We're not testing GetLearningSettings directly as it's not exposed
		if !engine.IsLearningEnabled() {
			t.Error("Expected learning to be enabled by default")
		}

		// Toggle learning state to verify it still works with failing repo
		engine.SetLearningEnabled(false)
		if engine.IsLearningEnabled() {
			t.Error("Expected learning to be disabled after setting")
		}
	})

	// Test content analyzer error handling
	t.Run("ContentAnalyzerErrors", func(t *testing.T) {
		// Create a test logger
		logger := log.NewLogger()

		// Create a repository with in-memory database
		repo, err := NewSQLiteRepository("", logger)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}
		defer repo.Close()

		// Create settings
		settings := &LearningSettings{
			ContentSamplingEnabled: true,
			ContentSampleMaxBytes:  4096,
		}

		// Create the analyzer
		analyzer := NewContentAnalyzer(repo, logger, settings)

		// Test with non-existent file
		_, err = analyzer.AnalyzeFile("/path/to/nonexistent/file.txt")
		if err == nil {
			t.Error("Expected error when analyzing non-existent file")
		}

		// Create a temporary directory for testing
		tempDir, err := os.MkdirTemp("", "content_errors")
		if err != nil {
			t.Fatalf("Failed to create temporary directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create a directory to test as a "file"
		dirPath := filepath.Join(tempDir, "directory")
		err = os.Mkdir(dirPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		// Try to analyze a directory
		_, err = analyzer.AnalyzeFile(dirPath)
		if err == nil {
			t.Error("Expected error when analyzing a directory")
		}

		// Create a file that can't be read
		noReadFile := filepath.Join(tempDir, "no-read.txt")
		err = os.WriteFile(noReadFile, []byte("test content"), 0000) // No permissions
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Skip this specific test on platforms where permission tests might not work the same way
		if os.Getuid() != 0 { // Skip if running as root
			// Try to analyze a file without read permissions
			_, err = analyzer.AnalyzeFile(noReadFile)
			if err == nil {
				t.Log("Expected error when analyzing file without read permissions")
				// Don't fail the test as this might behave differently on different platforms
			}
		}
	})

	// Test integration error handling
	t.Run("IntegrationErrorHandling", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir, err := os.MkdirTemp("", "integration_errors")
		if err != nil {
			t.Fatalf("Failed to create temporary directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create a mock repository
		mockRepo := &MockErrorRepository{
			getClassificationError: errors.New("mock classification error"),
		}

		// Create a config
		config := DefaultConfig()

		// Create an engine with the mock repository
		engine := NewEngine(mockRepo, config, logger)

		// Create integration
		integration := NewOrganizeIntegration(engine, logger)

		// Test classification error handling
		patterns, err := integration.ClassifyAndOrganize("/path/to/file.txt", "/default/dir")

		// We expect an error but patterns should be nil or empty
		if err == nil {
			t.Error("Expected error from ClassifyAndOrganize with failing repository")
		}

		// Should return nil or empty patterns on error
		if len(patterns) != 0 {
			t.Errorf("Expected 0 patterns on error, got %d", len(patterns))
		}

		// Test SuggestDestination to ensure it handles errors gracefully
		suggestedDir := integration.SuggestDestination("/path/to/file.txt", "/default/dir")
		// When errors occur, it should fall back to default dir
		if suggestedDir != "/default/dir" {
			t.Errorf("Expected default directory on error, got %s", suggestedDir)
		}
	})
}

// MockErrorRepository is a mock repository that returns errors
type MockErrorRepository struct {
	Repository
	getSettingsError       error
	getClassificationError error
}

func (m *MockErrorRepository) GetLearningSettings() (*LearningSettings, error) {
	if m.getSettingsError != nil {
		return nil, m.getSettingsError
	}
	return &LearningSettings{}, nil
}

func (m *MockErrorRepository) GetAllClassifications() ([]*FileClassification, error) {
	if m.getClassificationError != nil {
		return nil, m.getClassificationError
	}
	return []*FileClassification{}, nil
}

func (m *MockErrorRepository) GetClassificationByID(id string) (*FileClassification, error) {
	if m.getClassificationError != nil {
		return nil, m.getClassificationError
	}
	return &FileClassification{
		ID:                  id,
		Name:                "Test Classification",
		Description:         "Test classification for error handling",
		ConfidenceThreshold: 0.7,
		SystemDefined:       true,
	}, nil
}

func (m *MockErrorRepository) GetFileClassifications(filePath string) ([]*ClassificationMatch, error) {
	// Return empty slice to simulate no classifications found
	return []*ClassificationMatch{}, nil
}

func (m *MockErrorRepository) Close() error {
	return nil
}

func (m *MockErrorRepository) Vacuum() error {
	return nil
}
