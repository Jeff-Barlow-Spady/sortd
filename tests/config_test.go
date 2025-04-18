package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sortd/internal/analysis"
	"sortd/internal/config"
	"sortd/internal/organize"
)

// TestConfigValidation removed - moved to internal/config/config_test.go

func TestConfigIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test config
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
organize:
  patterns:
    - match: "*.txt"
      target: "documents/"
    - match: "*.jpg"
      target: "images/"
settings:
  dry_run: false
  create_dirs: true
  backup: false
  collision: "rename"
directories:
  default: "` + tmpDir + `"
  watch:
    - "` + tmpDir + `"
watch_mode:
  enabled: true
  interval: 5
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Create destination directories
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "documents"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "images"), 0755))

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("test content")
	require.NoError(t, os.WriteFile(testFile, testContent, 0644))

	// First run analysis
	t.Run("analysis_with_config", func(t *testing.T) {
		cfg, err := config.LoadConfigFile(configPath)
		require.NoError(t, err)

		engine := analysis.New()
		engine.SetConfig(cfg)

		// Verify file exists before analysis
		_, err = os.Stat(testFile)
		require.NoError(t, err, "Test file should exist before analysis")

		info, err := engine.Scan(testFile)
		require.NoError(t, err)
		assert.Equal(t, "text/plain; charset=utf-8", info.ContentType)
		assert.Contains(t, info.Tags, "document")
	})

	// Then run organization
	t.Run("organization_with_config", func(t *testing.T) {
		cfg, err := config.LoadConfigFile(configPath)
		require.NoError(t, err)

		engine := organize.NewWithConfig(cfg)

		err = engine.OrganizeFile(testFile)
		require.NoError(t, err)

		// Verify file was moved
		movedFile := filepath.Join(tmpDir, "documents", "test.txt")
		_, err = os.Stat(movedFile)
		require.NoError(t, err)

		// Verify content was preserved
		content, err := os.ReadFile(movedFile)
		require.NoError(t, err)
		assert.Equal(t, testContent, content)
	})
}
