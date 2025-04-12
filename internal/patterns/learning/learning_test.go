package learning

import (
	"os"
	"testing"
	"time"

	"sortd/internal/log"
)

func TestDatabaseInitialization(t *testing.T) {
	// Test in-memory database initialization
	db, err := InitDatabase("")
	if err != nil {
		t.Fatalf("Failed to initialize in-memory database: %v", err)
	}
	defer db.Close()

	// Verify database is functional by executing a simple query
	var version string
	err = db.QueryRow("SELECT sqlite_version()").Scan(&version)
	if err != nil {
		t.Fatalf("Failed to query SQLite version: %v", err)
	}

	if version == "" {
		t.Error("Expected non-empty SQLite version")
	}
}

func TestRepositoryBasicOperations(t *testing.T) {
	// Create a test logger
	logger := log.NewLogger()

	// Create a repository with in-memory database
	repo, err := NewSQLiteRepository("", logger)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create a test operation record
	testRecord := &OperationRecord{
		ID:              "test-op-1",
		Timestamp:       time.Now(),
		OperationType:   "move",
		SourcePath:      "/path/to/source/file.txt",
		DestinationPath: "/path/to/destination/file.txt",
		FileName:        "file.txt",
		FileExt:         ".txt",
		FileSize:        1024,
		Manual:          true,
		Success:         true,
	}

	// Save the operation record
	err = repo.SaveOperationRecord(testRecord)
	if err != nil {
		t.Fatalf("Failed to save operation record: %v", err)
	}

	// Retrieve the operation by type
	operations, err := repo.GetOperationsByType("move", 10)
	if err != nil {
		t.Fatalf("Failed to retrieve operations by type: %v", err)
	}

	if len(operations) == 0 {
		t.Error("Expected at least one operation, got none")
	}

	// Verify retrieved operation matches the saved one
	if len(operations) > 0 {
		retrievedOp := operations[0]
		if retrievedOp.ID != testRecord.ID {
			t.Errorf("Expected ID %s, got %s", testRecord.ID, retrievedOp.ID)
		}
		if retrievedOp.OperationType != testRecord.OperationType {
			t.Errorf("Expected operation type %s, got %s", testRecord.OperationType, retrievedOp.OperationType)
		}
	}
}

func TestEngineBasic(t *testing.T) {
	// Create a test logger
	logger := log.NewLogger()

	// Create a repository
	repo, err := NewSQLiteRepository("", logger)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create a config
	config := DefaultConfig()

	// Create an engine
	engine := NewEngine(repo, config, logger)

	// Test tracking an operation
	testRecord := &OperationRecord{
		ID:              "test-op-engine-1",
		Timestamp:       time.Now(),
		OperationType:   "copy",
		SourcePath:      "/path/to/source/image.jpg",
		DestinationPath: "/path/to/destination/image.jpg",
		FileName:        "image.jpg",
		FileExt:         ".jpg",
		FileSize:        2048,
		Manual:          true,
		Success:         true,
	}

	err = engine.TrackOperation(testRecord)
	if err != nil {
		t.Fatalf("Failed to track operation: %v", err)
	}

	// Check if learning is enabled by default
	if !engine.IsLearningEnabled() {
		t.Error("Expected learning to be enabled by default")
	}

	// Toggle learning off
	engine.SetLearningEnabled(false)
	if engine.IsLearningEnabled() {
		t.Error("Expected learning to be disabled after setting it to false")
	}
}

func TestFileClassificationBasic(t *testing.T) {
	// Get all classifications
	logger := log.NewLogger()
	repo, err := NewSQLiteRepository("", logger)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Get all classifications
	classifications, err := repo.GetAllClassifications()
	if err != nil {
		t.Fatalf("Failed to get classifications: %v", err)
	}

	// Verify that we have some base classifications from the embedded SQL
	if len(classifications) == 0 {
		t.Error("Expected to find base classifications, got none")
	}

	// Check if we can find the 'document' classification
	var foundDoc bool
	for _, c := range classifications {
		if c.ID == "doc" {
			foundDoc = true
			if c.Name != "Documents" {
				t.Errorf("Expected Documents name, got %s", c.Name)
			}
			if len(c.Criteria.ExtensionPatterns) == 0 {
				t.Error("Expected extension patterns, got none")
			}
			break
		}
	}

	if !foundDoc {
		t.Error("Document classification not found")
	}
}

func TestCleanup(t *testing.T) {
	// Clean up any test files if they exist
	tempDBPath := "test_sortd.db"
	_, err := os.Stat(tempDBPath)
	if err == nil {
		os.Remove(tempDBPath)
	}
}
