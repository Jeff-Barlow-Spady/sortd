package learning

import (
	"os"
	"path/filepath"
	"sortd/internal/log"
	"sortd/pkg/types"
	"testing"
)

func TestOrganizeEngineIntegration(t *testing.T) {
	// Create a test logger
	logger := log.NewLogger()

	// Create a repository with in-memory database
	repo, err := NewSQLiteRepository("", logger)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create a config
	config := DefaultConfig()

	// Create an engine
	engine := NewEngine(repo, config, logger)

	// Create the integration helper
	integration := NewOrganizeIntegration(engine, logger)
	if integration == nil {
		t.Fatal("Failed to create integration helper")
	}

	// Test pattern generation from classification
	t.Run("GeneratePattern", func(t *testing.T) {
		// Create a test classification
		classification := &FileClassification{
			ID:                  "test-class",
			Name:                "Test Classification",
			Description:         "Classification for testing",
			ConfidenceThreshold: 0.7,
			SystemDefined:       false,
			Criteria: ClassifierCriteria{
				ExtensionPatterns: []string{".txt", ".md"},
				NamePatterns:      []string{"test", "sample"},
				MimeTypes:         []string{"text/plain"},
			},
		}

		// Generate a pattern
		pattern := integration.GeneratePattern(classification, "/test/destination")

		// Verify pattern
		if pattern == nil {
			t.Fatal("Expected pattern, got nil")
		}

		// Should use the first extension pattern
		if pattern.Match != "*.txt" {
			t.Errorf("Expected match *.txt, got %s", pattern.Match)
		}

		if pattern.Target != "/test/destination" {
			t.Errorf("Expected target /test/destination, got %s", pattern.Target)
		}

		// Test with no extension patterns
		classification.Criteria.ExtensionPatterns = []string{}
		pattern = integration.GeneratePattern(classification, "/test/destination")

		// Should use the first name pattern
		if pattern.Match != "*test*" {
			t.Errorf("Expected match *test*, got %s", pattern.Match)
		}

		// Test with no name patterns either
		classification.Criteria.NamePatterns = []string{}
		pattern = integration.GeneratePattern(classification, "/test/destination")

		// Should use the ID
		if pattern.Match != "*test-class*" {
			t.Errorf("Expected match *test-class*, got %s", pattern.Match)
		}
	})

	// Test file info enrichment
	t.Run("EnrichFileInfo", func(t *testing.T) {
		// Create a temporary text file for testing
		tempDir, err := os.MkdirTemp("", "integration_test")
		if err != nil {
			t.Fatalf("Failed to create temporary directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create a text file
		testFilePath := filepath.Join(tempDir, "test.txt")
		textContent := "This is a test file for integration testing."
		err = os.WriteFile(testFilePath, []byte(textContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create a classification for text files
		classification := &FileClassification{
			ID:                  "txt",
			Name:                "Text Files",
			Description:         "Plain text files",
			ConfidenceThreshold: 0.7,
			SystemDefined:       true,
			Criteria: ClassifierCriteria{
				ExtensionPatterns: []string{".txt"},
				NamePatterns:      []string{"text", "readme"},
				MimeTypes:         []string{"text/plain"},
			},
		}

		// Save the classification
		err = repo.SaveClassification(classification)
		if err != nil {
			t.Fatalf("Failed to save classification: %v", err)
		}

		// Create a basic file info
		fileInfo := &types.FileInfo{
			Path: testFilePath,
			Size: int64(len(textContent)),
		}

		// Generate a content signature for the file
		contentAnalyzer := engine.GetContentAnalyzer()
		_, err = contentAnalyzer.AnalyzeFile(testFilePath)
		if err != nil {
			t.Fatalf("Failed to analyze file: %v", err)
		}

		// Use the engine to classify the file
		matches, err := engine.ClassifyFile(testFilePath)
		if err != nil {
			t.Fatalf("Failed to classify file: %v", err)
		}
		if len(matches) == 0 {
			t.Log("No classifications found, test might not be complete")
		}

		// Enrich the file info
		enrichedInfo, err := integration.EnrichFileInfo(fileInfo)
		if err != nil {
			t.Fatalf("Failed to enrich file info: %v", err)
		}

		// Verify the enriched info
		if enrichedInfo == nil {
			t.Fatal("Expected enriched info, got nil")
		}

		// Check content type
		if enrichedInfo.ContentType == "" {
			t.Log("ContentType not set, expected MIME type")
		}

		// Check that we have tags
		if len(enrichedInfo.Tags) == 0 {
			t.Log("No tags were added during enrichment")
		}

		// Check metadata
		if len(enrichedInfo.Metadata) == 0 {
			t.Log("No metadata was added during enrichment")
		}
	})

	// Test tracking operations
	t.Run("TrackOrganizeOperation", func(t *testing.T) {
		// Create a test operation
		err := integration.TrackOrganizeOperation(
			"/source/testfile.txt",
			"/destination/testfile.txt",
			1024,
			"move",
			true,
		)
		if err != nil {
			t.Fatalf("Failed to track organize operation: %v", err)
		}

		// Get recent operations to verify it was tracked
		operations, err := repo.GetRecentOperations(1, 10)
		if err != nil {
			t.Fatalf("Failed to get recent operations: %v", err)
		}

		// Verify we have at least one operation
		if len(operations) == 0 {
			t.Error("Expected at least one operation, got none")
		}

		// Verify the operation details
		var found bool
		for _, op := range operations {
			if op.SourcePath == "/source/testfile.txt" &&
				op.DestinationPath == "/destination/testfile.txt" &&
				op.OperationType == "move" &&
				op.FileSize == 1024 &&
				op.Manual == true {
				found = true
				break
			}
		}

		if !found {
			t.Error("Could not find the tracked operation in recent operations")
		}
	})

	// Test classify and organize
	t.Run("ClassifyAndOrganize", func(t *testing.T) {
		// This would require a more complex setup with real files and classifications
		// For now, let's do a basic check that the function doesn't crash

		// Create a temporary text file for testing
		tempDir, err := os.MkdirTemp("", "classify_organize_test")
		if err != nil {
			t.Fatalf("Failed to create temporary directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create a text file
		testFilePath := filepath.Join(tempDir, "organize_test.txt")
		textContent := "This is a test file for classification and organization testing."
		err = os.WriteFile(testFilePath, []byte(textContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Call classify and organize
		patterns, err := integration.ClassifyAndOrganize(testFilePath, "/default/destination")
		if err != nil {
			t.Fatalf("ClassifyAndOrganize failed: %v", err)
		}

		// We might not have any patterns if no classifications match
		t.Logf("Generated %d patterns for the test file", len(patterns))
	})
}

func TestGenerateID(t *testing.T) {
	// Test ID generation
	id1 := GenerateID()
	if id1 == "" {
		t.Error("GenerateID returned empty string")
	}

	// Generate a second ID and make sure it's different
	id2 := GenerateID()
	if id1 == id2 {
		t.Error("GenerateID returned the same ID twice in quick succession")
	}
}
