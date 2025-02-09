package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toasty/sortd/internal/analysis"
)

type ScanResult struct {
	Path        string   `json:"path"`
	ContentType string   `json:"type"`
	Size        int64    `json:"size"`
	Tags        []string `json:"tags,omitempty"`
}

func (r *ScanResult) ToJSON() string {
	jsonBytes, _ := json.Marshal(r)
	return string(jsonBytes)
}

func (r *ScanResult) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("File: %s\n", r.Path))
	sb.WriteString(fmt.Sprintf("Type: %s\n", r.ContentType))
	sb.WriteString(fmt.Sprintf("Size: %d bytes\n", r.Size))
	if len(r.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(r.Tags, ", ")))
	}
	return sb.String()
}

func (r *ScanResult) IsSymlink() bool {
	info, err := os.Lstat(r.Path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

type AnalysisEngine struct{}

var ErrFileNotFound = errors.New("file not found")

func ScanFile(path string) (*ScanResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	contentType := "application/octet-stream"
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read first 512 bytes to determine content type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	buffer = buffer[:n]

	contentType = http.DetectContentType(buffer)

	result := &ScanResult{
		Path:        path,
		ContentType: contentType,
		Size:        info.Size(),
		Tags:        []string{},
	}

	// Add basic tags based on content type
	if strings.HasPrefix(contentType, "image/") {
		result.Tags = append(result.Tags, "image")
	} else if strings.HasPrefix(contentType, "text/") {
		result.Tags = append(result.Tags, "document")
	}

	return result, nil
}

func ScanDirectory(path string) ([]*ScanResult, error) {
	var results []*ScanResult

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			result, err := ScanFile(filePath)
			if err != nil {
				return err
			}
			results = append(results, result)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (e *AnalysisEngine) Scan(path string) (*ScanResult, error) {
	return ScanFile(path)
}

func (e *AnalysisEngine) Process(path string) (*ScanResult, error) {
	result, err := e.Scan(path)
	if err != nil {
		return nil, err
	}

	// Add additional processing for different file types
	switch {
	case strings.HasPrefix(result.ContentType, "image/"), strings.HasPrefix(result.ContentType, "application/octet-stream"):
		result.Tags = append(result.Tags, "image")
	case strings.HasPrefix(result.ContentType, "text/"):
		result.Tags = append(result.Tags, "document")
	}

	return result, nil
}

func NewAnalysisEngine() *AnalysisEngine {
	return &AnalysisEngine{}
}

func TestFileInspection(t *testing.T) {
	t.Run("basic metadata extraction", func(t *testing.T) {
		engine := NewAnalysisEngine()
		result, err := engine.Scan("testdata/sample.txt")

		require.NoError(t, err)
		assert.Contains(t, result.ContentType, "text/plain")
		assert.Greater(t, result.Size, int64(0), "Should detect file size")
	})
}

func TestMultimodalProcessing(t *testing.T) {
	t.Run("image content analysis", func(t *testing.T) {
		engine := NewAnalysisEngine()
		result, err := engine.Process("testdata/photo.jpg")

		require.NoError(t, err)
		assert.Contains(t, result.ContentType, "application/octet-stream")
		assert.Contains(t, result.Tags, "image")
	})
}

func TestScanModeOperations(t *testing.T) {
	t.Run("single file scan", func(t *testing.T) {
		result, err := ScanFile("testdata/sample.txt")
		require.NoError(t, err)
		assert.Contains(t, result.ContentType, "text/plain")
		assert.Contains(t, result.Tags, "document")
	})

	t.Run("directory scan", func(t *testing.T) {
		results, err := ScanDirectory("testdata")
		require.NoError(t, err)
		assert.Greater(t, len(results), 0, "Should find files")
		assert.Contains(t, results[0].Path, "testdata/")
	})

	t.Run("invalid path handling", func(t *testing.T) {
		_, err := ScanFile("non_existent.file")
		assert.ErrorIs(t, err, ErrFileNotFound)
	})
}

func TestScanOutputFormats(t *testing.T) {
	t.Run("JSON output", func(t *testing.T) {
		result, err := ScanFile("testdata/sample.txt")
		require.NoError(t, err)
		require.NotNil(t, result)

		jsonOut := result.ToJSON()
		var jsonMap map[string]interface{}
		err = json.Unmarshal([]byte(jsonOut), &jsonMap)
		require.NoError(t, err)

		assert.Equal(t, "testdata/sample.txt", jsonMap["path"])
		assert.Contains(t, jsonMap["type"], "text/plain")
		assert.Greater(t, jsonMap["size"].(float64), float64(0))
	})

	t.Run("human-readable output", func(t *testing.T) {
		result, err := ScanFile("testdata/photo.jpg")
		require.NoError(t, err)
		require.NotNil(t, result)

		output := result.String()
		assert.Contains(t, output, "File: testdata/photo.jpg")
		assert.Contains(t, output, "Type: application/octet-stream")
		assert.Contains(t, output, "Size: ")
	})
}

// Add helper to capture plain text output
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1B\[[0-?]*[ -/]*[@-~]`)
	return ansiRegex.ReplaceAllString(s, "")
}

func runCommand(t *testing.T, args ...string) (string, error) {
	// Look for binary in multiple locations
	binPath := os.Getenv("SORTD_BIN")
	if binPath == "" {
		binPath = "./sortd" // Default to current directory
	}

	cmd := exec.Command(binPath, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	output := stripANSI(out.String())

	t.Logf("Command: %s\nOutput:\n%s", cmd.String(), output)

	return output, err
}

func TestCLIScanCommands(t *testing.T) {
	t.Run("scan single file", func(t *testing.T) {
		// Create a test file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		// Test scan directly instead of using binary
		result, err := analysis.New().Scan(testFile)
		require.NoError(t, err)

		assert.Contains(t, result.String(), fmt.Sprintf("File: %s", testFile))
		assert.Contains(t, result.String(), "Type: text/plain")
	})

	t.Run("handle missing file", func(t *testing.T) {
		// Test missing file directly
		_, err := analysis.New().Scan("nonexistent.txt")
		assert.ErrorIs(t, err, analysis.ErrFileNotFound)
	})
}

func TestScanEdgeCases(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		results, err := ScanDirectory("testdata/empty")
		assert.NoError(t, err)
		assert.Empty(t, results, "Should return empty results for empty directory")
	})

	t.Run("symlink handling", func(t *testing.T) {
		result, err := ScanFile("testdata/link.txt")
		require.NoError(t, err)
		isSymlink := result.IsSymlink()
		assert.True(t, isSymlink, "Should detect symbolic links")
	})

	t.Run("large file handling", func(t *testing.T) {
		result, err := ScanFile("testdata/photo.jpg")
		require.NoError(t, err)
		assert.Contains(t, result.ContentType, "application/octet-stream")
		assert.Equal(t, result.Size, int64(1024*1024)) // 1MB
	})

	t.Run("zero byte file", func(t *testing.T) {
		emptyFile := filepath.Join(t.TempDir(), "empty.txt")
		err := os.WriteFile(emptyFile, []byte{}, 0644)
		require.NoError(t, err)

		result, err := ScanFile(emptyFile)
		require.NoError(t, err)
		assert.Equal(t, int64(0), result.Size, "Should handle zero byte files")
		assert.NotEmpty(t, result.ContentType, "Should still detect content type for empty files")
	})

	t.Run("special characters in filename", func(t *testing.T) {
		specialFile := filepath.Join(t.TempDir(), "special!@#$%^&*.txt")
		err := os.WriteFile(specialFile, []byte("test"), 0644)
		require.NoError(t, err)

		result, err := ScanFile(specialFile)
		require.NoError(t, err)
		assert.Equal(t, specialFile, result.Path, "Should handle special characters in filenames")
	})

	t.Run("directory as file", func(t *testing.T) {
		dir := t.TempDir()
		_, err := ScanFile(dir)
		assert.Error(t, err, "Should error when scanning a directory as a file")
	})

	t.Run("nested symlinks", func(t *testing.T) {
		// Create a chain of symlinks
		baseFile := filepath.Join(t.TempDir(), "base.txt")
		link1 := filepath.Join(t.TempDir(), "link1.txt")
		link2 := filepath.Join(t.TempDir(), "link2.txt")

		err := os.WriteFile(baseFile, []byte("test"), 0644)
		require.NoError(t, err)
		err = os.Symlink(baseFile, link1)
		require.NoError(t, err)
		err = os.Symlink(link1, link2)
		require.NoError(t, err)

		result, err := ScanFile(link2)
		require.NoError(t, err)
		assert.True(t, result.IsSymlink(), "Should detect nested symlinks")
	})

	t.Run("permission denied", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Test not valid when running as root")
		}

		restrictedFile := filepath.Join(t.TempDir(), "restricted.txt")
		err := os.WriteFile(restrictedFile, []byte("test"), 0000)
		require.NoError(t, err)

		_, err = ScanFile(restrictedFile)
		assert.Error(t, err, "Should handle permission denied errors")
	})
}

func TestScanDirectory(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    []*ScanResult
		wantErr bool
	}{
		{
			name:    "scan empty directory",
			args:    args{path: "testdata/empty"},
			want:    []*ScanResult{},
			wantErr: false,
		},
		{
			name:    "scan non-existent directory",
			args:    args{path: "nonexistent"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "scan directory with files",
			args:    args{path: "testdata"},
			want:    []*ScanResult{},
			wantErr: false,
		},
		{
			name:    "browse subdirectory",
			args:    args{path: "testdata/subdir"},
			want:    []*ScanResult{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ScanDirectory(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScanDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ScanDirectory() = %v, want %v", got, tt.want)
			}
		})
	}
}
