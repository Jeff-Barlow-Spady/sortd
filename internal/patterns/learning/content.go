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
	// Check if we already have a signature for this file
	existingSig, err := a.repo.GetContentSignatureByPath(filePath)
	if err == nil && existingSig != nil {
		// Check if the file has been modified since signature creation
		fileInfo, err := os.Stat(filePath)
		if err == nil && fileInfo.ModTime().Before(existingSig.UpdatedAt) {
			// File hasn't changed, return existing signature
			a.logger.With(
				log.F("file", filePath),
				log.F("sigId", existingSig.ID),
				log.F("sigType", existingSig.SignatureType),
			).Debug("Retrieved existing content signature")
			return existingSig, nil
		}
	}

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
		// TODO: Enhanced text analysis could use:
		// - github.com/jdkato/prose for NLP (tokenization, entity recognition)
		// - github.com/bbalet/stopwords to remove common words
		// - github.com/james-bowman/nlp for vector space models
	case "image":
		if err := generateImageSignature(signature); err != nil {
			return nil, err
		}
		// TODO: Enhanced image analysis could use:
		// - github.com/disintegration/imaging for image processing
		// - github.com/corona10/goimagehash for perceptual hashing
		// - github.com/anthonynsimon/bild for feature extraction
	case "document":
		if err := generateDocumentSignature(signature); err != nil {
			return nil, err
		}
		// TODO: Enhanced document analysis could use:
		// - github.com/unidoc/unioffice for Office documents
		// - github.com/unidoc/unipdf for PDF documents
		// - Extract text for further content analysis
	case "audio":
		if err := generateGenericSignature(signature); err != nil {
			return nil, err
		}
		// TODO: Enhanced audio analysis could use:
		// - github.com/mjibson/go-dsp for audio fingerprinting
	case "binary":
		if err := generateBinarySignature(signature); err != nil {
			return nil, err
		}
		// TODO: Enhanced binary analysis could include:
		// - File header analysis
		// - Entropy calculation
		// - Executable format parsing
	default:
		if err := generateGenericSignature(signature); err != nil {
			return nil, err
		}
	}

	// Save the signature to the repository if enabled
	if a.settings != nil && a.settings.ContentSamplingEnabled {
		saveErr := a.repo.SaveContentSignature(signature)
		if saveErr != nil {
			a.logger.With(
				log.F("error", saveErr),
				log.F("file", filePath),
			).Warn("Failed to save content signature to repository")
			// Continue even if save fails - we still want to return the signature
		} else {
			a.logger.With(
				log.F("file", filePath),
				log.F("sigId", signature.ID),
				log.F("sigType", signature.SignatureType),
			).Debug("Saved new content signature to repository")
		}
	}

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
		// TODO: Enhanced text comparison could use vector embeddings and cosine similarity
	case "image":
		return compareImageSignatures(sig1, sig2)
		// TODO: Enhanced image comparison could use perceptual hashing or feature matching
	case "document":
		return compareDocumentSignatures(sig1, sig2)
		// TODO: Enhanced document comparison could extract and compare text content
	case "audio":
		return compareGenericSignatures(sig1, sig2)
		// TODO: Enhanced audio comparison could use acoustic fingerprinting
	case "binary":
		return compareBinarySignatures(sig1, sig2)
		// TODO: Enhanced binary comparison could use fuzzy hashing (ssdeep)
	default:
		return compareGenericSignatures(sig1, sig2)
	}
}

// FindRelatedFiles finds files related to the given file
func (a *StandardContentAnalyzer) FindRelatedFiles(filePath string, minSimilarity float64, limit int) ([]*ContentRelationship, error) {
	// First, get or create a signature for the target file
	targetSig, err := a.AnalyzeFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to analyze target file %s", filePath)
	}

	// Check if we already have relationships for this file in the repository
	existingRels, err := a.repo.GetContentRelationships(targetSig.ID, minSimilarity, limit)
	if err == nil && len(existingRels) > 0 {
		a.logger.With(
			log.F("file", filePath),
			log.F("relationshipCount", len(existingRels)),
		).Debug("Retrieved existing relationships from repository")
		return existingRels, nil
	}

	// If we're here, we need to find related files
	// Get signatures of the same type from repository
	signatures, err := a.repo.GetContentSignaturesByType(targetSig.SignatureType, 100)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get content signatures of type %s", targetSig.SignatureType)
	}

	// Filter out the target signature itself
	var candidates []*ContentSignature
	for _, sig := range signatures {
		if sig.ID != targetSig.ID {
			candidates = append(candidates, sig)
		}
	}

	if len(candidates) == 0 {
		a.logger.With(log.F("file", filePath)).Debug("No other files of similar type found")
		return []*ContentRelationship{}, nil
	}

	// Compare with candidates and build relationships
	var relationships []*ContentRelationship
	for _, candidate := range candidates {
		// Skip comparing with the same file
		if candidate.FilePath == targetSig.FilePath {
			continue
		}

		// Calculate similarity
		similarity, relationType, err := a.CompareSimilarity(targetSig, candidate)
		if err != nil {
			a.logger.With(
				log.F("error", err),
				log.F("sourceFile", targetSig.FilePath),
				log.F("targetFile", candidate.FilePath),
			).Warn("Failed to compare file similarity")
			continue
		}

		// Only include relationships above the minimum similarity threshold
		if similarity >= minSimilarity {
			relationship := &ContentRelationship{
				ID:           uuid.New().String(),
				SourceID:     targetSig.ID,
				TargetID:     candidate.ID,
				Similarity:   similarity,
				RelationType: relationType,
				CreatedAt:    time.Now(),
			}

			// Save relationship to repository
			saveErr := a.repo.SaveContentRelationship(relationship)
			if saveErr != nil {
				a.logger.With(
					log.F("error", saveErr),
					log.F("sourceFile", targetSig.FilePath),
					log.F("targetFile", candidate.FilePath),
				).Warn("Failed to save relationship to repository")
				// Continue anyway
			}

			relationships = append(relationships, relationship)
		}
	}

	// Sort relationships by similarity (highest first)
	sortRelationships(relationships)

	// Apply limit if needed
	if limit > 0 && len(relationships) > limit {
		relationships = relationships[:limit]
	}

	a.logger.With(
		log.F("file", filePath),
		log.F("candidatesChecked", len(candidates)),
		log.F("relationshipsFound", len(relationships)),
	).Debug("Found related files")

	return relationships, nil
}

// Helper function to sort relationships by similarity (highest first)
func sortRelationships(relationships []*ContentRelationship) {
	// Simple bubble sort
	for i := 0; i < len(relationships); i++ {
		for j := i + 1; j < len(relationships); j++ {
			if relationships[i].Similarity < relationships[j].Similarity {
				relationships[i], relationships[j] = relationships[j], relationships[i]
			}
		}
	}
}

// CreateContentGroup creates a group of related files
func (a *StandardContentAnalyzer) CreateContentGroup(name, description, groupType string, signatureIDs []string) (*ContentGroup, error) {
	if name == "" {
		return nil, errors.NewInvalidInputError("group name cannot be empty", nil)
	}

	if len(signatureIDs) == 0 {
		return nil, errors.NewInvalidInputError("cannot create empty group", nil)
	}

	// Create a new content group
	group := &ContentGroup{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		GroupType:   groupType,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save the group to the repository
	err := a.repo.SaveContentGroup(group)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to save content group")
	}

	// Add each signature to the group with default membership score
	for _, sigID := range signatureIDs {
		// Get the signature to verify it exists
		sig, err := a.repo.GetContentSignature(sigID)
		if err != nil {
			a.logger.With(
				log.F("error", err),
				log.F("signatureID", sigID),
				log.F("groupID", group.ID),
			).Warn("Failed to get signature when adding to group")
			continue
		}

		// Calculate membership score for each signature
		membershipScore := 1.0 // Default score for explicitly added signatures

		// Add to the group
		err = a.repo.AddToContentGroup(group.ID, sigID, membershipScore)
		if err != nil {
			a.logger.With(
				log.F("error", err),
				log.F("signatureID", sigID),
				log.F("groupID", group.ID),
			).Warn("Failed to add signature to content group")
			// Continue anyway to add as many as possible
		} else {
			a.logger.With(
				log.F("file", sig.FilePath),
				log.F("groupID", group.ID),
			).Debug("Added file to content group")
		}
	}

	a.logger.With(
		log.F("groupID", group.ID),
		log.F("groupName", name),
		log.F("memberCount", len(signatureIDs)),
	).Info("Created new content group")

	return group, nil
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
	} else if strings.HasPrefix(mimeType, "audio/") {
		return "audio"
	} else if strings.HasPrefix(mimeType, "video/") {
		return "video"
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

	// Define max sample size based on settings or default
	maxSampleSize := 4096         // Default 4KB sample
	if sig.FileSize > 1024*1024 { // For files larger than 1MB
		maxSampleSize = 8192 // Use larger 8KB sample
	}

	// Create a limited reader to avoid reading the entire file
	limitedReader := io.LimitReader(file, int64(maxSampleSize))
	scanner := bufio.NewScanner(limitedReader)
	scanner.Split(bufio.ScanWords)

	// Scan words up to our limit
	wordCount := 0
	maxWords := 1000 // Reasonable word limit for classification
	for scanner.Scan() && wordCount < maxWords {
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

	// Extract metadata about the text
	metadata := map[string]interface{}{
		"word_count":      wordCount,
		"sample_size":     maxSampleSize,
		"is_partial":      sig.FileSize > int64(maxSampleSize),
		"avg_word_length": calculateAvgWordLength(wordFreq),
	}

	// Convert metadata to JSON
	_, _ = json.Marshal(metadata)
	sig.Keywords = append(sig.Keywords, "sample_based_analysis")

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

// calculateAvgWordLength calculates the average word length from a frequency map
func calculateAvgWordLength(wordFreq map[string]int) float64 {
	totalLength := 0
	totalWords := 0

	for word, count := range wordFreq {
		totalLength += len(word) * count
		totalWords += count
	}

	if totalWords == 0 {
		return 0
	}

	return float64(totalLength) / float64(totalWords)
}

// generateImageSignature generates a signature for image files
func generateImageSignature(sig *ContentSignature) error {
	file, err := os.Open(sig.FilePath)
	if err != nil {
		return errors.NewFileError("failed to open file", sig.FilePath, errors.FileAccessDenied, err)
	}
	defer file.Close()

	// Read file header (first 512 bytes) for image metadata
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return errors.NewFileError("failed to read file header", sig.FilePath, errors.FileOperationFailed, err)
	}
	header = header[:n] // Trim to actual bytes read

	// Generate a SHA-256 hash of the header
	hash := sha256.New()
	hash.Write(header)
	headerHash := hex.EncodeToString(hash.Sum(nil))

	// Extract image metadata (would use image libraries in production)
	// Basic metadata we can get from the header
	metadata := map[string]interface{}{
		"header_hash": headerHash,
		"file_size":   sig.FileSize,
		"mime_type":   sig.MimeType,
	}

	// In a real implementation, you would extract actual image dimensions,
	// color depth, etc. using proper image libraries

	// Extract keywords based on image type
	sig.Keywords = []string{}
	if strings.Contains(sig.MimeType, "jpeg") || strings.Contains(sig.MimeType, "jpg") {
		sig.Keywords = append(sig.Keywords, "jpeg", "photo")
	} else if strings.Contains(sig.MimeType, "png") {
		sig.Keywords = append(sig.Keywords, "png")
	} else if strings.Contains(sig.MimeType, "gif") {
		sig.Keywords = append(sig.Keywords, "gif", "animation")
	} else if strings.Contains(sig.MimeType, "svg") {
		sig.Keywords = append(sig.Keywords, "svg", "vector")
	} else {
		sig.Keywords = append(sig.Keywords, "image")
	}

	// Add file size category
	if sig.FileSize < 10*1024 {
		sig.Keywords = append(sig.Keywords, "small_image")
	} else if sig.FileSize > 1024*1024 {
		sig.Keywords = append(sig.Keywords, "large_image")
	}

	// Convert metadata to JSON
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return errors.NewFileError("failed to marshal image metadata", sig.FilePath, errors.FileOperationFailed, err)
	}

	sig.Signature = string(metadataBytes)

	return nil
}

// generateDocumentSignature generates a signature for document files
func generateDocumentSignature(sig *ContentSignature) error {
	file, err := os.Open(sig.FilePath)
	if err != nil {
		return errors.NewFileError("failed to open file", sig.FilePath, errors.FileAccessDenied, err)
	}
	defer file.Close()

	// Read file header (first 512 bytes) for document metadata
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return errors.NewFileError("failed to read file header", sig.FilePath, errors.FileOperationFailed, err)
	}
	header = header[:n] // Trim to actual bytes read

	// Generate a SHA-256 hash of the header
	hash := sha256.New()
	hash.Write(header)
	headerHash := hex.EncodeToString(hash.Sum(nil))

	// Extract document metadata
	metadata := map[string]interface{}{
		"header_hash": headerHash,
		"file_size":   sig.FileSize,
		"mime_type":   sig.MimeType,
	}

	// Extract keywords based on document type
	sig.Keywords = []string{}

	// Add MIME type as a keyword
	if sig.MimeType != "" {
		sig.Keywords = append(sig.Keywords, strings.Replace(sig.MimeType, "/", "_", -1))
	}

	// Categorize by document type
	if strings.Contains(sig.MimeType, "pdf") {
		sig.Keywords = append(sig.Keywords, "pdf", "document")
	} else if strings.Contains(sig.MimeType, "msword") || strings.Contains(sig.MimeType, "officedocument.wordprocessing") {
		sig.Keywords = append(sig.Keywords, "word", "document")
	} else if strings.Contains(sig.MimeType, "spreadsheet") || strings.Contains(sig.MimeType, "excel") {
		sig.Keywords = append(sig.Keywords, "spreadsheet", "excel")
	} else if strings.Contains(sig.MimeType, "presentation") || strings.Contains(sig.MimeType, "powerpoint") {
		sig.Keywords = append(sig.Keywords, "presentation", "powerpoint")
	} else {
		sig.Keywords = append(sig.Keywords, "document")
	}

	// Add file size category
	if sig.FileSize < 100*1024 {
		sig.Keywords = append(sig.Keywords, "small_document")
	} else if sig.FileSize > 5*1024*1024 {
		sig.Keywords = append(sig.Keywords, "large_document")
	}

	// Convert metadata to JSON
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return errors.NewFileError("failed to marshal document metadata", sig.FilePath, errors.FileOperationFailed, err)
	}

	sig.Signature = string(metadataBytes)

	return nil
}

// generateBinarySignature generates a signature for binary files
func generateBinarySignature(sig *ContentSignature) error {
	file, err := os.Open(sig.FilePath)
	if err != nil {
		return errors.NewFileError("failed to open file", sig.FilePath, errors.FileAccessDenied, err)
	}
	defer file.Close()

	// Read file header (first 512 bytes max) for binary analysis
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return errors.NewFileError("failed to read file header", sig.FilePath, errors.FileOperationFailed, err)
	}
	header = header[:n] // Trim to actual bytes read

	// Generate a SHA-256 hash of the header only
	hash := sha256.New()
	hash.Write(header)
	headerHash := hex.EncodeToString(hash.Sum(nil))

	// Extract binary metadata
	metadata := map[string]interface{}{
		"header_size":   n,
		"header_hash":   headerHash,
		"file_size":     sig.FileSize,
		"is_executable": isExecutable(header),
	}

	// Convert metadata to JSON and use as signature
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return errors.NewFileError("failed to marshal binary metadata", sig.FilePath, errors.FileOperationFailed, err)
	}

	sig.Signature = string(metadataBytes)

	// Add relevant keywords based on binary analysis
	sig.Keywords = extractBinaryKeywords(header, sig.MimeType)

	return nil
}

// isExecutable checks if a file header indicates an executable format
func isExecutable(header []byte) bool {
	if len(header) < 4 {
		return false
	}

	// Check for common executable signatures
	// MZ header (Windows executable)
	if header[0] == 0x4D && header[1] == 0x5A {
		return true
	}

	// ELF header (Linux executable)
	if header[0] == 0x7F && header[1] == 0x45 && header[2] == 0x4C && header[3] == 0x46 {
		return true
	}

	// Mach-O header (macOS executable)
	if (header[0] == 0xFE && header[1] == 0xED && header[2] == 0xFA && header[3] == 0xCE) ||
		(header[0] == 0xCE && header[1] == 0xFA && header[2] == 0xED && header[3] == 0xFE) {
		return true
	}

	return false
}

// extractBinaryKeywords extracts keywords based on binary header analysis
func extractBinaryKeywords(header []byte, mimeType string) []string {
	keywords := []string{}

	// Add MIME type as a keyword
	if mimeType != "" {
		keywords = append(keywords, strings.Replace(mimeType, "/", "_", -1))
	}

	// Check for common file signatures
	if len(header) >= 4 {
		if header[0] == 0x4D && header[1] == 0x5A {
			keywords = append(keywords, "executable", "windows", "pe")
		} else if header[0] == 0x7F && header[1] == 0x45 && header[2] == 0x4C && header[3] == 0x46 {
			keywords = append(keywords, "executable", "linux", "elf")
		} else if (header[0] == 0xFE && header[1] == 0xED) || (header[0] == 0xCE && header[1] == 0xFA) {
			keywords = append(keywords, "executable", "macos", "mach-o")
		} else if header[0] == 0x50 && header[1] == 0x4B {
			keywords = append(keywords, "archive", "zip")
		} else if header[0] == 0x1F && header[1] == 0x8B {
			keywords = append(keywords, "archive", "gzip")
		} else if header[0] == 0xFF && header[1] == 0xD8 {
			keywords = append(keywords, "jpeg", "image")
		} else if header[0] == 0x89 && header[1] == 0x50 && header[2] == 0x4E && header[3] == 0x47 {
			keywords = append(keywords, "png", "image")
		}
	}

	return keywords
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
	// Check if these are identical signatures (exact content match)
	if sig1.Signature == sig2.Signature {
		return 1.0, "identical", nil
	}

	// Parse the stored signature JSON into maps
	var freqMap1, freqMap2 map[string]int

	if err := json.Unmarshal([]byte(sig1.Signature), &freqMap1); err != nil {
		return 0.0, "", fmt.Errorf("failed to parse first signature: %w", err)
	}

	if err := json.Unmarshal([]byte(sig2.Signature), &freqMap2); err != nil {
		return 0.0, "", fmt.Errorf("failed to parse second signature: %w", err)
	}

	// Calculate cosine similarity between the word frequency maps
	similarity := calculateCosineSimilarity(freqMap1, freqMap2)
	relationType := determineRelationType(similarity)

	return similarity, relationType, nil
}

// compareImageSignatures compares two image signatures
func compareImageSignatures(sig1, sig2 *ContentSignature) (float64, string, error) {
	// Check if signatures are identical
	if sig1.Signature == sig2.Signature {
		return 1.0, "identical", nil
	}

	// Parse metadata from signatures
	var meta1, meta2 map[string]interface{}

	if err := json.Unmarshal([]byte(sig1.Signature), &meta1); err != nil {
		return 0.0, "", fmt.Errorf("failed to parse first signature: %w", err)
	}

	if err := json.Unmarshal([]byte(sig2.Signature), &meta2); err != nil {
		return 0.0, "", fmt.Errorf("failed to parse second signature: %w", err)
	}

	// Compare header hashes
	headerHash1, ok1 := meta1["header_hash"].(string)
	headerHash2, ok2 := meta2["header_hash"].(string)

	if ok1 && ok2 && headerHash1 == headerHash2 {
		return 0.95, "nearly_identical", nil
	}

	// Compare MIME types
	mimeType1, hasMime1 := meta1["mime_type"].(string)
	mimeType2, hasMime2 := meta2["mime_type"].(string)

	// If MIME types match, there's some similarity
	if hasMime1 && hasMime2 && mimeType1 == mimeType2 {
		return 0.3, "same_format", nil
	}

	// Check if both are image formats but different types
	if hasMime1 && hasMime2 &&
		strings.HasPrefix(mimeType1, "image/") &&
		strings.HasPrefix(mimeType2, "image/") {
		return 0.2, "both_images", nil
	}

	// If we get here, the images are different
	return 0.0, "different", nil
}

// compareDocumentSignatures compares two document signatures
func compareDocumentSignatures(sig1, sig2 *ContentSignature) (float64, string, error) {
	// Check if signatures are identical
	if sig1.Signature == sig2.Signature {
		return 1.0, "identical", nil
	}

	// Parse metadata from signatures
	var meta1, meta2 map[string]interface{}

	if err := json.Unmarshal([]byte(sig1.Signature), &meta1); err != nil {
		return 0.0, "", fmt.Errorf("failed to parse first signature: %w", err)
	}

	if err := json.Unmarshal([]byte(sig2.Signature), &meta2); err != nil {
		return 0.0, "", fmt.Errorf("failed to parse second signature: %w", err)
	}

	// Compare header hashes
	headerHash1, ok1 := meta1["header_hash"].(string)
	headerHash2, ok2 := meta2["header_hash"].(string)

	if ok1 && ok2 && headerHash1 == headerHash2 {
		return 0.95, "nearly_identical", nil
	}

	// Compare MIME types
	mimeType1, hasMime1 := meta1["mime_type"].(string)
	mimeType2, hasMime2 := meta2["mime_type"].(string)

	// If MIME types match, there's some similarity
	if hasMime1 && hasMime2 && mimeType1 == mimeType2 {
		// For same document type, give a base similarity score
		return 0.3, "same_type", nil
	}

	// If we get here, the documents are different
	return 0.0, "different", nil
}

// compareBinarySignatures compares two binary signatures
func compareBinarySignatures(sig1, sig2 *ContentSignature) (float64, string, error) {
	// Check if signatures are identical
	if sig1.Signature == sig2.Signature {
		return 1.0, "identical", nil
	}

	// Parse metadata from signatures
	var meta1, meta2 map[string]interface{}

	if err := json.Unmarshal([]byte(sig1.Signature), &meta1); err != nil {
		return 0.0, "", fmt.Errorf("failed to parse first signature: %w", err)
	}

	if err := json.Unmarshal([]byte(sig2.Signature), &meta2); err != nil {
		return 0.0, "", fmt.Errorf("failed to parse second signature: %w", err)
	}

	// Compare header hashes
	headerHash1, ok1 := meta1["header_hash"].(string)
	headerHash2, ok2 := meta2["header_hash"].(string)

	if ok1 && ok2 && headerHash1 == headerHash2 {
		return 0.95, "nearly_identical", nil
	}

	// Check executable status
	isExec1, hasExec1 := meta1["is_executable"].(bool)
	isExec2, hasExec2 := meta2["is_executable"].(bool)

	// If both are executables, there's some base similarity
	if hasExec1 && hasExec2 && isExec1 && isExec2 {
		return 0.2, "both_executable", nil
	}

	// If MIME types match, there's some similarity
	mimeType1, hasMime1 := meta1["mime_type"].(string)
	mimeType2, hasMime2 := meta2["mime_type"].(string)

	if hasMime1 && hasMime2 && mimeType1 == mimeType2 {
		return 0.3, "same_type", nil
	}

	// If we get here, the files are different
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
