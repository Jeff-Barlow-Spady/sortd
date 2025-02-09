package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/toasty/sortd/internal/analysis"
	"github.com/toasty/sortd/internal/config"
	"github.com/toasty/sortd/internal/organize"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.Config
		wantError bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				Settings: struct {
					DryRun     bool   `yaml:"dry_run"`
					CreateDirs bool   `yaml:"create_dirs"`
					Backup     bool   `yaml:"backup"`
					Collision  string `yaml:"collision"`
				}{
					Collision: "rename",
				},
			},
			wantError: false,
		},
		{
			name: "invalid collision",
			config: &config.Config{
				Settings: struct {
					DryRun     bool   `yaml:"dry_run"`
					CreateDirs bool   `yaml:"create_dirs"`
					Backup     bool   `yaml:"backup"`
					Collision  string `yaml:"collision"`
				}{
					Collision: "invalid",
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

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

	// Create test files
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

	// Create destination directories
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "documents"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "images"), 0755))

	t.Run("organization_with_config", func(t *testing.T) {
		cfg, err := config.LoadConfigFile(configPath)
		require.NoError(t, err)

		engine := organize.New()
		engine.SetConfig(cfg)

		err = engine.OrganizeFile(testFile)
		require.NoError(t, err)

		// Verify file was moved
		_, err = os.Stat(filepath.Join(tmpDir, "documents", "test.txt"))
		require.NoError(t, err)
	})

	t.Run("analysis_with_config", func(t *testing.T) {
		cfg, err := config.LoadConfigFile(configPath)
		require.NoError(t, err)

		engine := analysis.New()
		engine.SetConfig(cfg)

		info, err := engine.Scan(testFile)
		require.NoError(t, err)
		assert.Equal(t, "text/plain; charset=utf-8", info.Type)
		assert.Contains(t, info.Tags, "document")
	})
}
