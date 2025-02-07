package tests

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileInspection(t *testing.T) {
	t.Run("basic metadata extraction", func(t *testing.T) {
		engine := NewAnalysisEngine()
		result, err := engine.Scan("testdata/sample.txt")

		require.NoError(t, err)
		assert.Equal(t, "text/plain", result.ContentType)
		assert.Greater(t, result.Size, int64(0), "Should detect file size")
	})
}

func NewAnalysisEngine() *AnalysisEngine {
	return &AnalysisEngine{}
}

func TestMultimodalProcessing(t *testing.T) {
	t.Run("image content analysis", func(t *testing.T) {
		engine := NewAnalysisEngine()
		result, err := engine.Process("testdata/photo.jpg")

		require.NoError(t, err)
		assert.Contains(t, result.Tags, "nature", "Should detect image content")
	})
}

func TestScanModeOperations(t *testing.T) {
	t.Run("single file scan", func(t *testing.T) {
		result, err := ScanFile("document.pdf")
		require.NoError(t, err)
		assert.Equal(t, "application/pdf", result.ContentType)
		assert.Contains(t, result.Tags, "document")
	})

	t.Run("directory scan", func(t *testing.T) {
		results, err := ScanDirectory("testdata/")
		require.NoError(t, err)
		assert.Greater(t, len(results), 0, "Should find files")
		assert.Contains(t, results[0].Path, "testdata/")
	})

	t.Run("invalid path handling", func(t *testing.T) {
		_, err := ScanFile("non_existent.file")
		ErrFileNotFound := 0
		assert.ErrorIs(t, err, ErrFileNotFound)
	})
}

func ScanDirectory(s string) (any, any) {
	panic("unimplemented")
}

func ScanFile(s string) (any, any) {
	panic("unimplemented")
}

func TestScanOutputFormats(t *testing.T) {
	t.Run("JSON output", func(t *testing.T) {
		result, _ := ScanFile("test.txt")
		jsonOut := result.ToJSON()
		assert.JSONEq(t, `{
		"path": "test.txt",
		"type": "text/plain",
		"size": 1024
	}`, jsonOut)
	})

	t.Run("human-readable output", func(t *testing.T) {
		result, _ := ScanFile("image.jpg")
		output := result.String()
		assert.Contains(t, output, "Image File")
		assert.Contains(t, output, "Dimensions: 1920x1080")
	})
}

func TestCLIScanCommands(t *testing.T) {
	t.Run("single file scan", func(t *testing.T) {
		cmd := exec.Command("sortd", "scan", "file.txt")
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err)
		assert.Contains(t, string(output), "file.txt")
		assert.Contains(t, string(output), "text/plain")
	})

	t.Run("batch scan with filters", func(t *testing.T) {
		cmd := exec.Command("sortd", "scan", "./docs", "--filter", "*.md")
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err)
		assert.Contains(t, string(output), ".md")
		assert.NotContains(t, string(output), ".go")
	})

	t.Run("scan with format options", func(t *testing.T) {
		cmd := exec.Command("sortd", "scan", "data/", "--json")
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err)
		assert.JSONEq(t, string(output), `{"results":[...]}`)
	})
}

func TestScanEdgeCases(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		results, err := ScanDirectory("empty_dir/")
		assert.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("large file handling", func(t *testing.T) {
		result, err := ScanFile("large.bin")
		require.NoError(t, err)
		assert.Equal(t, "application/octet-stream", result.ContentType)
		assert.Equal(t, result.Size, int64(1024*1024*1024)) // 1GB
	})

	t.Run("symbolic links", func(t *testing.T) {
		os.Symlink("real_file.txt", "link.txt")
		defer os.Remove("link.txt")
		result, err := ScanFile("link.txt")
		assert.NoError(t, err)
		assert.True(t, result.IsSymlink)
	})
}
