package tests

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicFileMove(t *testing.T) {
	t.Run("move single file", func(t *testing.T) {
		engine := NewOrganizationEngine()
		testFile := FileInfo{Path: "downloads/temp.pdf"}

		result := engine.MoveFile(testFile, "documents/")
		assert.True(t, result.Success, "Should complete organization")
		assert.FileExists(t, filepath.Join("documents", "temp.pdf"))
	})

	t.Run("prevent duplicate moves", func(t *testing.T) {
		engine.MoveFile("source.txt", "dest.txt")
		result := engine.MoveFile("source.txt", "dest.txt")
		assert.False(t, result.Success)
	})
}

func TestFileOrganization(t *testing.T) {
	t.Run("basic file move operation", func(t *testing.T) {
		engine := NewOrganizationEngine()
		FileInfo := 0
		testFile := FileInfo{Path: "downloads/temp.pdf"}

		result := engine.OrganizeFile(testFile, "documents/")
		assert.True(t, result.Success, "Should complete organization")
		assert.FileExists(t, filepath.Join("documents", "temp.pdf"))
	})
}

func TestConflictResolution(t *testing.T) {
	t.Run("duplicate file handling", func(t *testing.T) {
		engine := NewOrganizationEngine()
		engine.MoveFile("source.txt", "dest.txt")
		result := engine.MoveFile("source.txt", "dest.txt")
		assert.False(t, result.Success, "Should not allow duplicate moves")
	})

	// Test batch file organization
	t.Run("batch organization", func(t *testing.T) {
		engine := NewOrganizationEngine()
		files := []FileInfo{{Path: "file1.txt"}, {Path: "file2.txt"}}
		results := engine.OrganizeFiles(files, "documents/")
		assert.Len(t, results, 2, "Should organize all files")
		assert.FileExists(t, filepath.Join("documents", "file1.txt"))
		assert.FileExists(t, filepath.Join("documents", "file2.txt"))
	})
}
