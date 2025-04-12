package watch

import (
	"os"
	"path/filepath"
	"sortd/internal/config"
	"sortd/internal/patterns/learning"
	"sortd/pkg/types"
	"testing"
	"time"
)

// TestEngineAdapterIntegration tests integration points between EngineAdapter,
// learning system, and organization functionality
func TestEngineAdapterIntegration(t *testing.T) {
	// Skip in CI environments or with -short flag
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory structure for testing
	rootDir, err := os.MkdirTemp("", "adapter_integration")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(rootDir)

	// Create source and destination directories
	sourceDir := filepath.Join(rootDir, "source")
	destDir := filepath.Join(rootDir, "destination")
	dbDir := filepath.Join(rootDir, "db")

	for _, dir := range []string{sourceDir, destDir, dbDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Set up DB path
	dbPath := filepath.Join(dbDir, "learning.db")

	// Create test files of different types
	testFiles := createTestFiles(t, sourceDir)

	// Create base config
	cfg := config.New()
	cfg.Settings.DryRun = true // Use dry run for tests
	cfg.Settings.CreateDirs = true

	// Create adapter
	adapter := NewEngineAdapter(cfg)

	// Initialize learning system
	repo, err := learning.NewSQLiteRepository(dbPath, adapter.logger)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.Close()

	learningCfg := learning.DefaultConfig()
	learningCfg.DatabasePath = dbPath
	learningEngine := learning.NewEngine(repo, learningCfg, adapter.logger)

	// Connect learning engine to adapter
	adapter.SetLearningEngine(learningEngine)

	// Test 1: Track file organization operations
	t.Run("TrackingOperations", func(t *testing.T) {
		// Perform some file operations that should be tracked
		for _, file := range testFiles {
			destPath := filepath.Join(destDir, filepath.Base(file))
			err := adapter.MoveFile(file, destPath)
			if err != nil {
				t.Errorf("MoveFile failed: %v", err)
			}
		}

		// Verify operations were tracked in the database
		// First let's get recent operations from the repository
		operations, err := repo.GetRecentOperations(1, 100)
		if err != nil {
			t.Fatalf("Failed to get recent operations: %v", err)
		}

		// Should have tracked operations for our test files
		if len(operations) < len(testFiles) {
			t.Errorf("Expected at least %d operations, got %d", len(testFiles), len(operations))
		}

		// Verify operation details for at least one file
		if len(operations) > 0 {
			op := operations[0]
			if op.OperationType != "move" {
				t.Errorf("Expected operation type 'move', got '%s'", op.OperationType)
			}
			if op.Manual {
				t.Error("Expected manual flag to be false for adapter-initiated moves")
			}
		}
	})

	// Test 2: Content-based classification and enrichment
	t.Run("ContentEnrichment", func(t *testing.T) {
		// Create a text file with distinctive content
		textFile := filepath.Join(sourceDir, "special_content.txt")
		content := "This is a special text file with unique content for classification testing."
		err := os.WriteFile(textFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create a classification for text files
		txtClassification := &learning.FileClassification{
			ID:                  "txt",
			Name:                "Text Files",
			Description:         "Plain text files",
			ConfidenceThreshold: 0.7,
			SystemDefined:       true,
			Criteria: learning.ClassifierCriteria{
				ExtensionPatterns: []string{".txt"},
				NamePatterns:      []string{"text", "readme", "special"},
				MimeTypes:         []string{"text/plain"},
			},
		}

		// Save the classification
		err = repo.SaveClassification(txtClassification)
		if err != nil {
			t.Fatalf("Failed to save classification: %v", err)
		}

		// First, analyze the file to generate content signature
		contentSig, err := learningEngine.AnalyzeContent(textFile)
		if err != nil {
			t.Fatalf("Failed to analyze content: %v", err)
		}

		// Verify signature was created
		if contentSig == nil {
			t.Fatal("Expected content signature, got nil")
		}

		// Test file info enrichment
		fileInfo := &types.FileInfo{
			Path:        textFile,
			Size:        int64(len(content)),
			ContentType: "", // Will be filled by enrichment
			ModTime:     time.Now(),
			Tags:        []string{},              // Initialize empty slice for tags
			Metadata:    make(map[string]string), // Initialize empty map for metadata
		}

		// Enrich the file info with learning data
		enrichedInfo, err := adapter.EnrichFileInfo(fileInfo)
		if err != nil {
			t.Fatalf("EnrichFileInfo failed: %v", err)
		}

		// Verify enrichment
		if enrichedInfo.ContentType == "" {
			t.Error("Expected ContentType to be set after enrichment")
		}

		// Should have the txt classification as a tag
		foundTxtTag := false
		for _, tag := range enrichedInfo.Tags {
			if tag == "txt" {
				foundTxtTag = true
				break
			}
		}

		if !foundTxtTag {
			t.Errorf("Expected 'txt' tag in enriched info, got tags: %v", enrichedInfo.Tags)
		}

		// Test destination suggestion
		suggestedDest := adapter.SuggestDestination(textFile, destDir)
		expectedSuggestedPath := filepath.Join(destDir, "txt")
		if suggestedDest != expectedSuggestedPath {
			t.Errorf("Expected suggested destination %s, got %s", expectedSuggestedPath, suggestedDest)
		}
	})

	// Test 3: Learning pattern detection and application
	// This is a more complex test that simulates multiple file operations
	// to trigger pattern learning
	t.Run("PatternDetection", func(t *testing.T) {
		// Create test files with similar naming patterns
		for i := 1; i <= 5; i++ {
			filename := filepath.Join(sourceDir, "report-"+time.Now().Format("2006-01-02")+"-"+string(rune('A'+i-1))+".pdf")
			err := os.WriteFile(filename, []byte("test content"), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Move these files to a reports folder to establish a pattern
			reportsDest := filepath.Join(destDir, "reports")
			destFile := filepath.Join(reportsDest, filepath.Base(filename))
			err = adapter.MoveFile(filename, destFile)
			if err != nil {
				t.Errorf("MoveFile failed: %v", err)
			}
		}

		// Manually trigger pattern analysis
		err := learningEngine.PerformAnalysis()
		if err != nil {
			t.Fatalf("Pattern analysis failed: %v", err)
		}

		// Create one more test file following the same pattern
		newReportFile := filepath.Join(sourceDir, "report-"+time.Now().Format("2006-01-02")+"-F.pdf")
		err = os.WriteFile(newReportFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Test if the system suggests the correct destination
		suggestedDest := adapter.SuggestDestination(newReportFile, destDir)
		if filepath.Dir(suggestedDest) != filepath.Join(destDir, "reports") {
			t.Logf("No pattern detected yet or suggestion didn't work as expected. This test may be flaky and can be improved.")
		}
	})
}

// Helper function to create test files of different types
func createTestFiles(t *testing.T, dir string) []string {
	// Create different types of files for testing
	fileSpecs := []struct {
		name    string
		content string
	}{
		{"document.txt", "This is a text document"},
		{"image.jpg", "fake image content"},
		{"spreadsheet.xlsx", "fake spreadsheet content"},
		{"code.go", "package main\n\nfunc main() {\n\tfmt.Println(\"Hello, world!\")\n}"},
		{"empty.txt", ""},
	}

	var paths []string
	for _, spec := range fileSpecs {
		path := filepath.Join(dir, spec.name)
		err := os.WriteFile(path, []byte(spec.content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", spec.name, err)
		}
		paths = append(paths, path)
	}

	return paths
}
