package learning

import (
	"os"
	"path/filepath"
	"testing"

	"sortd/internal/log"
)

func TestContentAnalyzer(t *testing.T) {
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

	// Create a temporary text file for testing
	tempDir, err := os.MkdirTemp("", "content_analyzer_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a text file
	textFilePath := filepath.Join(tempDir, "test.txt")
	textContent := "This is a test file for content analysis.\nIt contains multiple words for testing the text content analyzer.\nSome words are repeated to test frequency mapping.\nTest test test analysis content content signature."
	err = os.WriteFile(textFilePath, []byte(textContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test text file: %v", err)
	}

	// Test text file analysis
	t.Run("TextFileAnalysis", func(t *testing.T) {
		signature, err := analyzer.AnalyzeFile(textFilePath)
		if err != nil {
			t.Fatalf("Failed to analyze text file: %v", err)
		}

		// Verify signature
		if signature == nil {
			t.Fatal("Expected signature, got nil")
		}

		// Check MIME type detection
		if signature.MimeType != "text/plain; charset=utf-8" && signature.MimeType != "text/plain" {
			t.Errorf("Expected text/plain MIME type, got %s", signature.MimeType)
		}

		// Check signature type
		if signature.SignatureType != "text" {
			t.Errorf("Expected text signature type, got %s", signature.SignatureType)
		}

		// Check that we have keywords
		if len(signature.Keywords) == 0 {
			t.Error("Expected keywords, got none")
		}

		// Verify some common keywords are present
		foundTest := false
		foundContent := false
		for _, keyword := range signature.Keywords {
			if keyword == "test" {
				foundTest = true
			}
			if keyword == "content" {
				foundContent = true
			}
		}

		if !foundTest {
			t.Error("Expected 'test' in keywords")
		}
		if !foundContent {
			t.Error("Expected 'content' in keywords")
		}

		// Check signature is not empty
		if signature.Signature == "" {
			t.Error("Expected non-empty signature")
		}
	})

	// Create a binary file
	binaryFilePath := filepath.Join(tempDir, "test.bin")
	binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}
	err = os.WriteFile(binaryFilePath, binaryContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test binary file: %v", err)
	}

	// Test binary file analysis
	t.Run("BinaryFileAnalysis", func(t *testing.T) {
		signature, err := analyzer.AnalyzeFile(binaryFilePath)
		if err != nil {
			t.Fatalf("Failed to analyze binary file: %v", err)
		}

		// Verify signature
		if signature == nil {
			t.Fatal("Expected signature, got nil")
		}

		// Check signature type (could be binary or generic depending on how mimetype identifies it)
		if signature.SignatureType != "binary" && signature.SignatureType != "generic" {
			t.Errorf("Expected binary or generic signature type, got %s", signature.SignatureType)
		}

		// For binary files, we should have a hash-based signature
		if signature.Signature == "" {
			t.Error("Expected non-empty signature")
		}
	})

	// Test similarity comparison between identical content
	t.Run("SimilarityComparisonIdentical", func(t *testing.T) {
		// Create a duplicate file
		duplicateFilePath := filepath.Join(tempDir, "duplicate.txt")
		err = os.WriteFile(duplicateFilePath, []byte(textContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create duplicate file: %v", err)
		}

		// Get signatures
		sig1, err := analyzer.AnalyzeFile(textFilePath)
		if err != nil {
			t.Fatalf("Failed to analyze original file: %v", err)
		}

		sig2, err := analyzer.AnalyzeFile(duplicateFilePath)
		if err != nil {
			t.Fatalf("Failed to analyze duplicate file: %v", err)
		}

		// Compare
		similarity, relationType, err := analyzer.CompareSimilarity(sig1, sig2)
		if err != nil {
			t.Fatalf("Failed to compare signatures: %v", err)
		}

		// Identical content should have high similarity
		if similarity < 0.9 {
			t.Errorf("Expected high similarity (>0.9), got %f", similarity)
		}

		// Relation type should reflect high similarity
		if relationType != "nearly_identical" && relationType != "identical" {
			t.Errorf("Expected 'nearly_identical' or 'identical' relation, got %s", relationType)
		}
	})

	// Test similarity comparison between different content
	t.Run("SimilarityComparisonDifferent", func(t *testing.T) {
		// Create a different file
		differentFilePath := filepath.Join(tempDir, "different.txt")
		differentContent := "This is completely different content with no relation to the original test file."
		err = os.WriteFile(differentFilePath, []byte(differentContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create different file: %v", err)
		}

		// Get signatures
		sig1, err := analyzer.AnalyzeFile(textFilePath)
		if err != nil {
			t.Fatalf("Failed to analyze original file: %v", err)
		}

		sig2, err := analyzer.AnalyzeFile(differentFilePath)
		if err != nil {
			t.Fatalf("Failed to analyze different file: %v", err)
		}

		// Compare
		similarity, relationType, err := analyzer.CompareSimilarity(sig1, sig2)
		if err != nil {
			t.Fatalf("Failed to compare signatures: %v", err)
		}

		// Different content should have lower similarity
		if similarity > 0.5 {
			t.Errorf("Expected low similarity (<0.5), got %f", similarity)
		}

		// Verify the relation type is appropriate for the similarity score
		if similarity < 0.3 && relationType != "different" {
			t.Errorf("Expected 'different' relation for similarity %f, got %s", similarity, relationType)
		}
	})

	// Test finding related files
	t.Run("FindRelatedFiles", func(t *testing.T) {
		// Create several text files with varying content
		similarFilePath := filepath.Join(tempDir, "similar.txt")
		similarContent := "This is similar to the test file. It contains words like test and content analysis."
		err = os.WriteFile(similarFilePath, []byte(similarContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create similar file: %v", err)
		}

		// Create another similar file
		similarFilePath2 := filepath.Join(tempDir, "similar2.txt")
		similarContent2 := "More content for testing the analyzer. This has some test words repeated."
		err = os.WriteFile(similarFilePath2, []byte(similarContent2), 0644)
		if err != nil {
			t.Fatalf("Failed to create second similar file: %v", err)
		}

		// Create a completely different file
		unrelatedFilePath := filepath.Join(tempDir, "unrelated.txt")
		unrelatedContent := "This has nothing in common with the others. Different vocabulary entirely."
		err = os.WriteFile(unrelatedFilePath, []byte(unrelatedContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create unrelated file: %v", err)
		}

		// Analyze all files to populate the repository
		files := []string{textFilePath, similarFilePath, similarFilePath2, unrelatedFilePath}
		for _, file := range files {
			_, err := analyzer.AnalyzeFile(file)
			if err != nil {
				t.Fatalf("Failed to analyze file %s: %v", file, err)
			}
		}

		// Find related files
		relationships, err := analyzer.FindRelatedFiles(textFilePath, 0.3, 10)
		if err != nil {
			t.Fatalf("Failed to find related files: %v", err)
		}

		// We should find at least the similar files
		if len(relationships) < 2 {
			t.Errorf("Expected at least 2 related files, got %d", len(relationships))
		}

		// Verify relationships are ordered by similarity (highest first)
		for i := 1; i < len(relationships); i++ {
			if relationships[i-1].Similarity < relationships[i].Similarity {
				t.Errorf("Relationships not properly sorted by similarity")
				break
			}
		}
	})

	// Test content group creation
	t.Run("CreateContentGroup", func(t *testing.T) {
		// Get signatures for a few files
		sig1, err := analyzer.AnalyzeFile(textFilePath)
		if err != nil {
			t.Fatalf("Failed to get signature for file: %v", err)
		}

		sig2, err := analyzer.AnalyzeFile(filepath.Join(tempDir, "similar.txt"))
		if err != nil {
			t.Fatalf("Failed to get signature for file: %v", err)
		}

		// Create a content group
		group, err := analyzer.CreateContentGroup(
			"Test Group",
			"Test content group description",
			"test_collection",
			[]string{sig1.ID, sig2.ID},
		)
		if err != nil {
			t.Fatalf("Failed to create content group: %v", err)
		}

		// Verify group was created
		if group == nil {
			t.Fatal("Expected content group, got nil")
		}

		// Check group properties
		if group.Name != "Test Group" {
			t.Errorf("Expected group name 'Test Group', got '%s'", group.Name)
		}

		if group.GroupType != "test_collection" {
			t.Errorf("Expected group type 'test_collection', got '%s'", group.GroupType)
		}

		// Get members of the group
		members, err := repo.GetContentGroupMembers(group.ID)
		if err != nil {
			t.Fatalf("Failed to get group members: %v", err)
		}

		// Should have two members
		if len(members) != 2 {
			t.Errorf("Expected 2 group members, got %d", len(members))
		}

		// Verify the members contain our signatures
		foundSig1 := false
		foundSig2 := false
		for _, member := range members {
			if member.SignatureID == sig1.ID {
				foundSig1 = true
			}
			if member.SignatureID == sig2.ID {
				foundSig2 = true
			}
		}

		if !foundSig1 {
			t.Error("First signature not found in group members")
		}
		if !foundSig2 {
			t.Error("Second signature not found in group members")
		}
	})
}

// Helper function to verify if a string is in a slice
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
