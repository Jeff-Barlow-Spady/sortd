package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sortd/internal/analysis"
	serr "sortd/internal/errors"
	"sortd/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileInspection(t *testing.T) {
	t.Run("basic metadata extraction", func(t *testing.T) {
		// Create test file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "sample.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

		engine := analysis.New()
		result, err := engine.Scan(testFile)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Contains(t, result.ContentType, "text/plain")
		assert.Equal(t, int64(12), result.Size, "Should detect file size")
		assert.Contains(t, result.Tags, "document")
	})
}

func TestMultimodalProcessing(t *testing.T) {
	t.Run("image content analysis", func(t *testing.T) {
		// Create test image
		tmpDir := t.TempDir()
		imgPath := filepath.Join(tmpDir, "test.jpg")
		createTestImage(t, imgPath)

		engine := analysis.New()
		result, err := engine.Process(imgPath)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Contains(t, result.ContentType, "image/")
		assert.Contains(t, result.Tags, "image")
	})
}

func createTestImage(t *testing.T, path string) {
	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()

	// Write minimal JPEG header
	_, err = file.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46})
	require.NoError(t, err)
}

func TestScanModeOperations(t *testing.T) {
	testDataDir := "testdata"
	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		require.NoError(t, os.Mkdir(testDataDir, 0755))
	}
	txtFile := filepath.Join(testDataDir, "sample.txt")
	jpgFile := filepath.Join(testDataDir, "photo.jpg")
	require.NoError(t, os.WriteFile(txtFile, []byte("test content"), 0644))
	createTestImage(t, jpgFile)
	t.Cleanup(func() {
		os.RemoveAll(testDataDir)
	})

	engine := analysis.New()

	t.Run("single file scan", func(t *testing.T) {
		result, err := engine.Scan(txtFile)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Contains(t, result.ContentType, "text/plain")
		assert.Contains(t, result.Tags, "document")
	})

	t.Run("directory scan", func(t *testing.T) {
		results, err := engine.ScanDirectory(testDataDir)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 2, "Should find at least two files")

		foundTxt := false
		foundJpg := false
		for _, res := range results {
			if strings.HasSuffix(res.Path, "sample.txt") {
				foundTxt = true
				assert.Contains(t, res.ContentType, "text/plain")
			} else if strings.HasSuffix(res.Path, "photo.jpg") {
				foundJpg = true
				assert.Contains(t, res.ContentType, "image/jpeg")
			}
		}
		assert.True(t, foundTxt, "Did not find sample.txt in directory scan")
		assert.True(t, foundJpg, "Did not find photo.jpg in directory scan")
	})

	t.Run("invalid path handling", func(t *testing.T) {
		_, err := engine.Scan("non_existent.file")
		assert.Error(t, err, "Expected error for non-existent file")
		assert.True(t, serr.IsFileNotFound(err), "Error should be a FileNotFound error")
	})
}

func TestScanOutputFormats(t *testing.T) {
	testDataDir := "testdata"
	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		require.NoError(t, os.Mkdir(testDataDir, 0755))
	}
	txtFile := filepath.Join(testDataDir, "sample.txt")
	jpgFile := filepath.Join(testDataDir, "photo.jpg")
	require.NoError(t, os.WriteFile(txtFile, []byte("test content"), 0644))
	createTestImage(t, jpgFile)
	t.Cleanup(func() { os.RemoveAll(testDataDir) })

	engine := analysis.New()

	t.Run("JSON output check", func(t *testing.T) {
		result, err := engine.Scan(txtFile)
		require.NoError(t, err)
		require.NotNil(t, result)

		jsonBytes, err := json.Marshal(result)
		require.NoError(t, err, "Failed to marshal FileInfo to JSON")

		var unmarshaledResult types.FileInfo
		err = json.Unmarshal(jsonBytes, &unmarshaledResult)
		require.NoError(t, err, "Failed to unmarshal JSON back to FileInfo")

		assert.Equal(t, txtFile, unmarshaledResult.Path)
		assert.Contains(t, unmarshaledResult.ContentType, "text/plain")
		assert.Equal(t, int64(12), unmarshaledResult.Size)
		assert.Contains(t, unmarshaledResult.Tags, "document")
	})

	t.Run("human-readable output check", func(t *testing.T) {
		result, err := engine.Scan(jpgFile)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, jpgFile, result.Path)
		assert.Contains(t, result.ContentType, "image/jpeg")
		assert.Greater(t, result.Size, int64(0))
		assert.Contains(t, result.Tags, "image")
	})
}
