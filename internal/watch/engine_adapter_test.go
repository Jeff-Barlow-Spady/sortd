package watch

import (
	"os"
	"path/filepath"
	"sortd/internal/config"
	"sortd/internal/patterns/learning"
	"sortd/pkg/types"
	"testing"
)

func TestEngineAdapterBasic(t *testing.T) {
	// Create basic config
	cfg := config.New()
	cfg.Settings.DryRun = false

	// Create the adapter
	adapter := NewEngineAdapter(cfg)
	if adapter == nil {
		t.Fatal("Failed to create EngineAdapter")
	}

	// Test dry run setting
	if adapter.GetDryRun() != false {
		t.Error("Expected default dry run to be false")
	}

	// Test setting dry run
	adapter.SetDryRun(true)
	if !adapter.GetDryRun() {
		t.Error("Expected dry run to be true after setting")
	}

	// Test learning enabled (should default to false)
	if adapter.GetLearningEnabled() {
		t.Error("Expected learning to be disabled by default")
	}

	// Test setting learning enabled without a learning engine
	adapter.SetLearningEnabled(true)
	// Since there's no learning engine, this should still be false
	if adapter.GetLearningEnabled() {
		t.Error("Expected learning to be disabled without a learning engine")
	}
}

func TestEngineAdapterWithLearning(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "adapter_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a basic config
	cfg := config.New()
	cfg.Settings.DryRun = false

	// Create the adapter
	adapter := NewEngineAdapter(cfg)

	// Setup learning engine
	dbPath := filepath.Join(tempDir, "test.db")
	learningCfg := learning.DefaultConfig()
	learningCfg.DatabasePath = dbPath

	// Create a repository
	repo, err := learning.NewSQLiteRepository(dbPath, adapter.logger)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create the learning engine
	learningEngine := learning.NewEngine(repo, learningCfg, adapter.logger)

	// Set the learning engine on the adapter
	adapter.SetLearningEngine(learningEngine)

	// Check that learning is now enabled
	if !adapter.GetLearningEnabled() {
		t.Error("Expected learning to be enabled after setting engine")
	}

	// Test toggling learning state
	adapter.SetLearningEnabled(false)
	if adapter.GetLearningEnabled() {
		t.Error("Expected learning to be disabled after setting to false")
	}

	adapter.SetLearningEnabled(true)
	if !adapter.GetLearningEnabled() {
		t.Error("Expected learning to be enabled after setting to true")
	}
}

func TestEngineAdapterFileOperations(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "file_operations_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create subdirectories
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	err = os.Mkdir(sourceDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create a test file
	testFilePath := filepath.Join(sourceDir, "test.txt")
	err = os.WriteFile(testFilePath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a basic config pointing to our test directories
	cfg := config.New()
	cfg.Settings.DryRun = false

	// Create the adapter
	adapter := NewEngineAdapter(cfg)

	// Set to dry run mode to avoid actual file operations
	adapter.SetDryRun(true)

	// Test moving the file (in dry run mode)
	destFilePath := filepath.Join(destDir, "test.txt")
	err = adapter.MoveFile(testFilePath, destFilePath)
	if err != nil {
		t.Fatalf("MoveFile failed in dry run mode: %v", err)
	}

	// In dry run mode, the file should not actually be moved
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		t.Error("File should not be moved in dry run mode")
	}

	if _, err := os.Stat(destFilePath); err == nil {
		t.Error("Destination file should not exist in dry run mode")
	}
}

func TestEngineAdapterSuggestDestination(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "suggest_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a basic config
	cfg := config.New()
	cfg.Settings.DryRun = false

	// Create the adapter
	adapter := NewEngineAdapter(cfg)

	// Without learning, it should return the default destination
	defaultDest := filepath.Join(tempDir, "default")
	suggestedDest := adapter.SuggestDestination("test.txt", defaultDest)
	if suggestedDest != defaultDest {
		t.Errorf("Expected default destination, got %s", suggestedDest)
	}

	// Setup learning engine
	dbPath := filepath.Join(tempDir, "test.db")
	learningCfg := learning.DefaultConfig()
	learningCfg.DatabasePath = dbPath

	// Create a repository
	repo, err := learning.NewSQLiteRepository(dbPath, adapter.logger)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create the learning engine
	learningEngine := learning.NewEngine(repo, learningCfg, adapter.logger)

	// Set the learning engine on the adapter
	adapter.SetLearningEngine(learningEngine)

	// Even with learning, without classifications it should still return default
	suggestedDest = adapter.SuggestDestination("test.txt", defaultDest)
	if suggestedDest != defaultDest {
		t.Errorf("Expected default destination, got %s", suggestedDest)
	}
}

func TestEngineAdapterEnrichFileInfo(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "enrich_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a basic config
	cfg := config.New()
	cfg.Settings.DryRun = false

	// Create the adapter
	adapter := NewEngineAdapter(cfg)

	// Create a test file
	testFilePath := filepath.Join(tempDir, "enrich.txt")
	err = os.WriteFile(testFilePath, []byte("test content for enrichment"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create basic file info
	fileInfo := &types.FileInfo{
		Path: testFilePath,
		Size: 25, // Length of "test content for enrichment"
	}

	// Without learning, it should return the same file info
	enrichedInfo, err := adapter.EnrichFileInfo(fileInfo)
	if err != nil {
		t.Fatalf("EnrichFileInfo failed: %v", err)
	}

	if enrichedInfo == nil {
		t.Fatal("Expected enriched info, got nil")
	}

	// Should be the same object
	if enrichedInfo != fileInfo {
		t.Error("Without learning, should return the same file info object")
	}

	// Setup learning engine
	dbPath := filepath.Join(tempDir, "test.db")
	learningCfg := learning.DefaultConfig()
	learningCfg.DatabasePath = dbPath

	// Create a repository
	repo, err := learning.NewSQLiteRepository(dbPath, adapter.logger)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create the learning engine
	learningEngine := learning.NewEngine(repo, learningCfg, adapter.logger)

	// Set the learning engine on the adapter
	adapter.SetLearningEngine(learningEngine)

	// With learning, it should try to enrich the file info
	// but since we have no classifications yet, it should still be similar
	enrichedInfo, err = adapter.EnrichFileInfo(fileInfo)
	if err != nil {
		t.Fatalf("EnrichFileInfo with learning failed: %v", err)
	}

	if enrichedInfo == nil {
		t.Fatal("Expected enriched info with learning, got nil")
	}

	// The object might be different now, but it should have the same basic properties
	if enrichedInfo.Path != fileInfo.Path {
		t.Errorf("Expected path %s, got %s", fileInfo.Path, enrichedInfo.Path)
	}

	if enrichedInfo.Size != fileInfo.Size {
		t.Errorf("Expected size %d, got %d", fileInfo.Size, enrichedInfo.Size)
	}
}

func TestEngineAdapterOrganizeByPatterns(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "organize_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFilePath := filepath.Join(tempDir, "organize.txt")
	err = os.WriteFile(testFilePath, []byte("test content for organization"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a basic config
	cfg := config.New()
	cfg.Settings.DryRun = true // Use dry run to avoid actual file operations

	// Create the adapter
	adapter := NewEngineAdapter(cfg)

	// Test organizing without learning
	err = adapter.OrganizeByPatterns([]string{testFilePath})
	if err != nil {
		t.Fatalf("OrganizeByPatterns failed: %v", err)
	}

	// Setup learning engine
	dbPath := filepath.Join(tempDir, "test.db")
	learningCfg := learning.DefaultConfig()
	learningCfg.DatabasePath = dbPath

	// Create a repository
	repo, err := learning.NewSQLiteRepository(dbPath, adapter.logger)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create the learning engine
	learningEngine := learning.NewEngine(repo, learningCfg, adapter.logger)

	// Set the learning engine on the adapter
	adapter.SetLearningEngine(learningEngine)

	// Test organizing with learning
	err = adapter.OrganizeByPatterns([]string{testFilePath})
	if err != nil {
		t.Fatalf("OrganizeByPatterns with learning failed: %v", err)
	}

	// File should still exist since we used dry run mode
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		t.Error("Test file should still exist in dry run mode")
	}
}
