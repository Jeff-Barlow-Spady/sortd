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
	t.Run("organization with config", func(t *testing.T) {
		cfg := config.NewTestConfig()

		// Create test files
		testDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(testDir, "test.txt"), []byte("test"), 0644))

		// Initialize organizer with config
		organizer := organize.NewWithConfig(cfg)
		// Test organization
		_, err := organizer.OrganizeDir(testDir)
		require.NoError(t, err)

		// Verify file was moved according to pattern
		assert.FileExists(t, filepath.Join(testDir, "documents", "test.txt"))
	})

	t.Run("analysis with config", func(t *testing.T) {
		cfg := config.NewTestConfig()

		// Create test file
		testDir := t.TempDir()
		testFile := filepath.Join(testDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

		// Initialize analyzer with config
		analyzer := analysis.NewWithConfig(cfg)

		// Test analysis
		result, err := analyzer.Analyze(testFile)
		require.NoError(t, err)
		assert.Equal(t, "text/plain", result.ContentType)
	})
}
