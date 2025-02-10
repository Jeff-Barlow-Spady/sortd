package testutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// CreateTestFilesWithContent creates test files with specific content
func CreateTestFilesWithContent(t *testing.T, dir string, files map[string]string) {
	for name, content := range files {
		err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
		require.NoError(t, err)
	}
}

// CreateTestFilesWithDefault creates test files with default content
func CreateTestFilesWithDefault(t *testing.T, dir string) {
	files := map[string]string{
		"test1.txt": "test content 1",
		"test2.txt": "test content 2",
		"test3.jpg": "image content",
	}
	CreateTestFilesWithContent(t, dir, files)
}

// StripANSI removes ANSI escape sequences from a string
func StripANSI(str string) string {
	// Simple ANSI escape sequence stripping
	// This is a basic implementation - you might want to use a more robust solution
	var result []rune
	inEscape := false
	for _, r := range str {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		result = append(result, r)
	}
	return string(result)
}
