package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type FileInfo struct {
	Path        string
	ContentType string
	Size        int64
}

type OrganizationEngine struct {
	files map[string]FileInfo
}

func NewOrganizationEngine() *OrganizationEngine {
	return &OrganizationEngine{
		files: make(map[string]FileInfo),
	}
}

func (e *OrganizationEngine) MoveFile(src, dest string) error {
	if _, exists := e.files[dest]; exists {
		return fmt.Errorf("destination file already exists: %s", dest)
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	e.files[dest] = FileInfo{
		Path: dest,
		Size: info.Size(),
	}

	return nil
}

func (e *OrganizationEngine) OrganizeFiles(files []string, destDir string) error {
	for _, file := range files {
		dest := filepath.Join(destDir, filepath.Base(file))
		if err := e.MoveFile(file, dest); err != nil {
			return err
		}
	}
	return nil
}

func TestBasicFileMove(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	destDir := filepath.Join(tmpDir, "documents")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	t.Run("move single file", func(t *testing.T) {
		engine := NewOrganizationEngine()
		err := engine.MoveFile(testFile, filepath.Join(destDir, "test.txt"))
		assert.NoError(t, err, "Should complete organization")
	})

	t.Run("prevent duplicate moves", func(t *testing.T) {
		engine := NewOrganizationEngine()
		err := engine.MoveFile(testFile, filepath.Join(destDir, "test.txt"))
		assert.NoError(t, err)
		err = engine.MoveFile(testFile, filepath.Join(destDir, "test.txt"))
		assert.Error(t, err)
	})
}

func TestFileOrganization(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	destDir := filepath.Join(tmpDir, "organized")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	t.Run("basic file move", func(t *testing.T) {
		engine := NewOrganizationEngine()
		err := engine.MoveFile(testFile, filepath.Join(destDir, "test.txt"))
		assert.NoError(t, err)
	})

	t.Run("prevent duplicate moves", func(t *testing.T) {
		engine := NewOrganizationEngine()
		err := engine.MoveFile(testFile, filepath.Join(destDir, "test.txt"))
		assert.NoError(t, err)
		err = engine.MoveFile(testFile, filepath.Join(destDir, "test.txt"))
		assert.Error(t, err, "Should prevent moving to existing destination")
	})

	t.Run("batch file organization", func(t *testing.T) {
		engine := NewOrganizationEngine()
		files := []string{
			filepath.Join(tmpDir, "test1.txt"),
			filepath.Join(tmpDir, "test2.txt"),
			filepath.Join(tmpDir, "test3.txt"),
		}
		for _, file := range files {
			err := os.WriteFile(file, []byte("test content"), 0644)
			require.NoError(t, err)
		}
		err := engine.OrganizeFiles(files, destDir)
		assert.NoError(t, err)
	})
}

func TestConflictResolution(t *testing.T) {
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "documents")
	err := os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	t.Run("batch organization", func(t *testing.T) {
		engine := NewOrganizationEngine()
		files := []string{
			filepath.Join(tmpDir, "file1.txt"),
			filepath.Join(tmpDir, "file2.txt"),
		}
		for _, file := range files {
			err := os.WriteFile(file, []byte("test content"), 0644)
			require.NoError(t, err)
		}
		err := engine.OrganizeFiles(files, destDir)
		assert.NoError(t, err, "Should organize all files")
	})
}
