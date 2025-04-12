package learning

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"sortd/internal/errors"
	"sortd/internal/log"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
)

// ContentAnalyzer is the interface for analyzing file content
type ContentAnalyzer interface {
	// AnalyzeFile generates a content signature for a file
	AnalyzeFile(filePath string) (*ContentSignature, error)

	// CompareSimilarity calculates the similarity between two signatures
	CompareSimilarity(sig1, sig2 *ContentSignature) (float64, string, error)

	// FindRelatedFiles finds files related to the given file
	FindRelatedFiles(filePath string, minSimilarity float64, limit int) ([]*ContentRelationship, error)

	// CreateContentGroup creates a group of related files
	CreateContentGroup(name, description, groupType string, signatureIDs []string) (*ContentGroup, error)
}

// StandardContentAnalyzer implements ContentAnalyzer interface
type StandardContentAnalyzer struct {
	repo     Repository
	logger   log.Logging
	settings *LearningSettings
}

// NewContentAnalyzer creates a new content analyzer
func NewContentAnalyzer(repo Repository, logger log.Logging, settings *LearningSettings) ContentAnalyzer {
	return &StandardContentAnalyzer{
		repo:     repo,
		logger:   logger,
		settings: settings,
	}
}

// AnalyzeFile generates a content signature for a file
func (a *StandardContentAnalyzer) AnalyzeFile(filePath string) (*ContentSignature, error) {
	// Get file information
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, errors.NewFileError("failed to get file info", filePath, errors.FileNotFound, err)
	}

	// Check file size
	if fileInfo.Size() == 0 {
		return nil, errors.NewFileError("empty file", filePath, errors.InvalidOperation, nil)
	}

	// Detect MIME type
	mime, err := detectMimeType(filePath)
	if err != nil {
		return nil, err
	}

	// Create a new signature
	signature := &ContentSignature{
		ID:            uuid.New().String(),
		FilePath:      filePath,
		MimeType:      mime,
		SignatureType: determineSigType(mime),
		FileSize:      fileInfo.Size(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Generate content signature based on file type
	switch signature.SignatureType {
	case "text":
		if err := generateTextSignature(signature); err != nil {
			return nil, err
		}
	case "image":
		if err := generateImageSignature(signature); err != nil {
			return nil, err
		}
	case "document":
		if err := generateDocumentSignature(signature); err != nil {
			return nil, err
		}
	case "binary":
		if err := generateBinarySignature(signature); err != nil {
			return nil, err
		}
	default:
		if err := generateGenericSignature(signature); err != nil {
			return nil, err
		}
	}

	// TODO: Save the signature to the repository when implemented

	a.logger.With(
		log.F("file", filePath),
		log.F("mimeType", mime),
		log.F("sigType", signature.SignatureType),
	).Debug("Generated content signature")

	return signature, nil
}

// CompareSimilarity calculates the similarity between two signatures
func (a *StandardContentAnalyzer) CompareSimilarity(sig1, sig2 *ContentSignature) (float64, string, error) {
	// Skip comparison if signature types don't match
	if sig1.SignatureType != sig2.SignatureType {
		return 0.0, "different_types", nil
	}

	// Calculate similarity based on signature type
	switch sig1.SignatureType {
	case "text":
		return compareTextSignatures(sig1, sig2)
	case "image":
		return compareImageSignatures(sig1, sig2)
	case "document":
		return compareDocumentSignatures(sig1, sig2)
	case "binary":
		return compareBinarySignatures(sig1, sig2)
	default:
		return compareGenericSignatures(sig1, sig2)
	}
}

// FindRelatedFiles finds files related to the given file
func (a *StandardContentAnalyzer) FindRelatedFiles(filePath string, minSimilarity float64, limit int) ([]*ContentRelationship, error) {
	// TODO: Implement this method once repository methods are complete
	return nil, errors.New("FindRelatedFiles not yet implemented")
}

// CreateContentGroup creates a group of related files
func (a *StandardContentAnalyzer) CreateContentGroup(name, description, groupType string, signatureIDs []string) (*ContentGroup, error) {
	// TODO: Implement this method once repository methods are complete
	return nil, errors.New("CreateContentGroup not yet implemented")
}

// Helper functions

// detectMimeType determines the MIME type of a file
func detectMimeType(filePath string) (string, error) {
	mime, err := mimetype.DetectFile(filePath)
	if err != nil {
		return "", errors.NewFileError("failed to detect MIME type", filePath, errors.FileOperationFailed, err)
	}
	return mime.String(), nil
}

// determineSigType determines the signature type based on MIME type
func determineSigType(mimeType string) string {
	if strings.HasPrefix(mimeType, "text/") {
		return "text"
	} else if strings.HasPrefix(mimeType, "image/") {
		return "image"
	} else if strings.Contains(mimeType, "document") ||
		strings.Contains(mimeType, "pdf") ||
		strings.Contains(mimeType, "msword") ||
		strings.Contains(mimeType, "officedocument") {
		return "document"
	} else if strings.HasPrefix(mimeType, "application/") {
		return "binary"
	}
	return "generic"
}

// generateTextSignature generates a signature for text files
func generateTextSignature(sig *ContentSignature) error {
	file, err := os.Open(sig.FilePath)
	if err != nil {
		return errors.NewFileError("failed to open file", sig.FilePath, errors.FileAccessDenied, err)
	}
	defer file.Close()

	// Create a frequency map of words
	wordFreq := make(map[string]int)
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)

	// Scan up to 10000 words max
	wordCount := 0
	for scanner.Scan() && wordCount < 10000 {
		word := strings.ToLower(scanner.Text())
		// Skip very short words and non-alphanumeric characters
		if len(word) > 2 {
			wordFreq[word]++
			wordCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.NewFileError("error scanning text file", sig.FilePath, errors.FileOperationFailed, err)
	}

	// Extract keywords (top frequency words)
	keywords := extractKeywords(wordFreq, 20)
	sig.Keywords = keywords

	// Create signature as JSON of frequency map (top 100 words max)
	sigMap := extractTopWords(wordFreq, 100)
	sigBytes, err := json.Marshal(sigMap)
	if err != nil {
		return errors.NewFileError("failed to marshal signature", sig.FilePath, errors.FileOperationFailed, err)
	}
	sig.Signature = string(sigBytes)

	return nil
}

// generateImageSignature generates a signature for image files
func generateImageSignature(sig *ContentSignature) error {
	// For now, just generate a SHA-256 hash of the file
	// In a real implementation, you'd use image processing to extract features
	hash, err := calculateFileHash(sig.FilePath)
	if err != nil {
		return err
	}
	sig.Signature = hash

	// TODO: In a real implementation, extract image metadata, color histograms, etc.
	return nil
}

// generateDocumentSignature generates a signature for document files
func generateDocumentSignature(sig *ContentSignature) error {
	// For now, just generate a SHA-256 hash of the file
	// In a real implementation, you'd extract text content and use that for analysis
	hash, err := calculateFileHash(sig.FilePath)
	if err != nil {
		return err
	}
	sig.Signature = hash

	// TODO: In a real implementation, extract document metadata, text content, etc.
	return nil
}

// generateBinarySignature generates a signature for binary files
func generateBinarySignature(sig *ContentSignature) error {
	// Generate a SHA-256 hash of the file
	hash, err := calculateFileHash(sig.FilePath)
	if err != nil {
		return err
	}
	sig.Signature = hash

	// TODO: In a real implementation, extract file headers, structure information, etc.
	return nil
}

// generateGenericSignature generates a signature for any file type
func generateGenericSignature(sig *ContentSignature) error {
	// Generate a SHA-256 hash of the file
	hash, err := calculateFileHash(sig.FilePath)
	if err != nil {
		return err
	}
	sig.Signature = hash
	return nil
}

// calculateFileHash calculates a SHA-256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", errors.NewFileError("failed to open file for hashing", filePath, errors.FileAccessDenied, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", errors.NewFileError("failed to hash file", filePath, errors.FileOperationFailed, err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// extractKeywords extracts the top N frequent words as keywords
func extractKeywords(wordFreq map[string]int, n int) []string {
	// Convert map to slice of key-value pairs
	type wordFreqPair struct {
		word  string
		count int
	}
	pairs := make([]wordFreqPair, 0, len(wordFreq))
	for word, count := range wordFreq {
		pairs = append(pairs, wordFreqPair{word, count})
	}

	// Sort by frequency (highest first)
	// Simple bubble sort for now
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[i].count < pairs[j].count {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	// Extract top N words
	limit := n
	if len(pairs) < limit {
		limit = len(pairs)
	}

	keywords := make([]string, limit)
	for i := 0; i < limit; i++ {
		keywords[i] = pairs[i].word
	}

	return keywords
}

// extractTopWords extracts the top N frequent words
func extractTopWords(wordFreq map[string]int, n int) map[string]int {
	result := make(map[string]int)

	// Convert map to slice of key-value pairs
	type wordFreqPair struct {
		word  string
		count int
	}
	pairs := make([]wordFreqPair, 0, len(wordFreq))
	for word, count := range wordFreq {
		pairs = append(pairs, wordFreqPair{word, count})
	}

	// Sort by frequency (highest first)
	// Simple bubble sort for now
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[i].count < pairs[j].count {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	// Extract top N words
	limit := n
	if len(pairs) < limit {
		limit = len(pairs)
	}

	for i := 0; i < limit; i++ {
		result[pairs[i].word] = pairs[i].count
	}

	return result
}

// Comparison functions

// compareTextSignatures compares two text signatures
func compareTextSignatures(sig1, sig2 *ContentSignature) (float64, string, error) {
	// Parse signatures
	var freqMap1, freqMap2 map[string]int
	if err := json.Unmarshal([]byte(sig1.Signature), &freqMap1); err != nil {
		return 0.0, "", fmt.Errorf("failed to parse first signature: %w", err)
	}
	if err := json.Unmarshal([]byte(sig2.Signature), &freqMap2); err != nil {
		return 0.0, "", fmt.Errorf("failed to parse second signature: %w", err)
	}

	// Calculate cosine similarity
	similarity := calculateCosineSimilarity(freqMap1, freqMap2)
	relationType := determineRelationType(similarity)

	return similarity, relationType, nil
}

// compareImageSignatures compares two image signatures
func compareImageSignatures(sig1, sig2 *ContentSignature) (float64, string, error) {
	// For now, just compare hashes
	// In a real implementation, you'd compare image features
	if sig1.Signature == sig2.Signature {
		return 1.0, "identical", nil
	}
	return 0.0, "different", nil
}

// compareDocumentSignatures compares two document signatures
func compareDocumentSignatures(sig1, sig2 *ContentSignature) (float64, string, error) {
	// For now, just compare hashes
	// In a real implementation, you'd compare document content
	if sig1.Signature == sig2.Signature {
		return 1.0, "identical", nil
	}
	return 0.0, "different", nil
}

// compareBinarySignatures compares two binary signatures
func compareBinarySignatures(sig1, sig2 *ContentSignature) (float64, string, error) {
	// Compare hashes
	if sig1.Signature == sig2.Signature {
		return 1.0, "identical", nil
	}
	return 0.0, "different", nil
}

// compareGenericSignatures compares two generic signatures
func compareGenericSignatures(sig1, sig2 *ContentSignature) (float64, string, error) {
	// Compare hashes
	if sig1.Signature == sig2.Signature {
		return 1.0, "identical", nil
	}
	return 0.0, "different", nil
}

// calculateCosineSimilarity calculates cosine similarity between two frequency maps
func calculateCosineSimilarity(map1, map2 map[string]int) float64 {
	// Calculate dot product
	dotProduct := 0.0
	for word, count1 := range map1 {
		if count2, found := map2[word]; found {
			dotProduct += float64(count1 * count2)
		}
	}

	// Calculate magnitudes
	magnitude1 := 0.0
	for _, count := range map1 {
		magnitude1 += float64(count * count)
	}
	magnitude1 = float64(magnitude1)

	magnitude2 := 0.0
	for _, count := range map2 {
		magnitude2 += float64(count * count)
	}
	magnitude2 = float64(magnitude2)

	// Calculate cosine similarity
	if magnitude1 == 0 || magnitude2 == 0 {
		return 0.0
	}

	return dotProduct / (magnitude1 * magnitude2)
}

// determineRelationType determines the type of relationship based on similarity
func determineRelationType(similarity float64) string {
	if similarity >= 0.9 {
		return "nearly_identical"
	} else if similarity >= 0.7 {
		return "very_similar"
	} else if similarity >= 0.5 {
		return "similar"
	} else if similarity >= 0.3 {
		return "somewhat_similar"
	}
	return "different"
}
