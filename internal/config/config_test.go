package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"sortd/internal/config"
	"sortd/pkg/types" // Needed for patterns in assertions

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a temporary YAML config file
func createTestYAML(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	err = tmpFile.Close()
	require.NoError(t, err)
	return tmpFile.Name()
}

const (
	validYAML = `
organize:
  patterns:
    - match: "*.jpg"
      target: "/path/to/images"
    - match: "*.png"
      target: "/path/to/images"
    - match: "report_*.pdf"
      target: "/path/to/documents/reports"
settings:
  dry_run: false
  create_dirs: true
  collision: "rename"
watch_mode:
  enabled: true
watch_directories:
  - /home/user/downloads
  - /mnt/data/staging
`
	invalidSyntaxYAML = `
organize:
  patterns:
    - match: "*.txt"
      target: "/path/to/text
settings: # Missing closing quote and incorrect indentation
  dry_run: yes # Invalid boolean value
`
	invalidValueYAML = `
settings:
  collision: "delete" # Invalid collision strategy
`
	invalidDirsYAML = `
watch_directories:
  - ""
  - "/valid/path"
`
)

func TestLoadConfigFile(t *testing.T) {
	t.Run("load valid config", func(t *testing.T) {
		configFile := createTestYAML(t, validYAML)
		cfg, err := config.LoadConfigFile(configFile)

		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Assert specific loaded values
		assert.Len(t, cfg.Organize.Patterns, 3)
		assert.Equal(t, "*.jpg", cfg.Organize.Patterns[0].Match)
		assert.Equal(t, "/path/to/images", cfg.Organize.Patterns[0].Target)
		assert.Equal(t, "rename", cfg.Settings.Collision)
		assert.Len(t, cfg.WatchDirectories, 2)
		assert.Contains(t, cfg.WatchDirectories, "/home/user/downloads")
		assert.Contains(t, cfg.WatchDirectories, "/mnt/data/staging")
	})

	t.Run("load non-existent file", func(t *testing.T) {
		nonExistentPath := filepath.Join(t.TempDir(), "does_not_exist.yaml")
		cfg, err := config.LoadConfigFile(nonExistentPath)

		require.NoError(t, err, "Loading non-existent file should return default config, not an error")
		require.NotNil(t, cfg)

		// Check a few default values
		defaultCfg := config.New() // Get expected defaults
		assert.Equal(t, defaultCfg.Settings.DryRun, cfg.Settings.DryRun)
		assert.Equal(t, defaultCfg.Settings.Collision, cfg.Settings.Collision)
		assert.Equal(t, defaultCfg.Organize.Patterns, cfg.Organize.Patterns)
		assert.Equal(t, defaultCfg.WatchMode.Enabled, cfg.WatchMode.Enabled)
	})

	t.Run("load file with invalid YAML syntax", func(t *testing.T) {
		// Use YAML with a type mismatch (string for boolean) for a more robust error
		configFile := createTestYAML(t, invalidSyntaxYAML)
		_, err := config.LoadConfigFile(configFile)

		require.Error(t, err, "Loading invalid YAML should return an error")
		assert.Contains(t, err.Error(), "error parsing config file", "Error message should indicate parsing failure")
	})

	t.Run("load file with invalid config value (collision)", func(t *testing.T) {
		configFile := createTestYAML(t, invalidValueYAML)
		_, err := config.LoadConfigFile(configFile)

		require.Error(t, err, "Loading config with invalid value should return an error")
		assert.Contains(t, err.Error(), "invalid configuration", "Error message should indicate validation failure")
		assert.Contains(t, err.Error(), "invalid collision setting", "Error message should specify the validation issue")
	})

	t.Run("load file with invalid watch directories", func(t *testing.T) {
		configFile := createTestYAML(t, invalidDirsYAML)
		_, err := config.LoadConfigFile(configFile)

		require.Error(t, err, "Loading config with invalid watch directories should return an error")
		assert.Contains(t, err.Error(), "invalid configuration", "Error message should indicate validation failure")
		assert.Contains(t, err.Error(), "path cannot be empty", "Error message should specify the validation issue")
	})
}

// Moved from tests/config_test.go
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				Organize: struct {
					Patterns []types.Pattern `yaml:"patterns"`
				}{
					Patterns: []types.Pattern{{Match: "*", Target: "/dest"}},
				},
				Settings: struct {
					DryRun     bool   `yaml:"dry_run"`
					CreateDirs bool   `yaml:"create_dirs"`
					Backup     bool   `yaml:"backup"`
					Collision  string `yaml:"collision"`
				}{
					Collision: "overwrite",
					Backup:    false,
				},
				WatchDirectories: []string{"/watch/dir"},
			},
			wantErr: false,
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
					Collision: "invalid_strategy",
					Backup:    false,
				},
				Organize: struct {
					Patterns []types.Pattern `yaml:"patterns"`
				}{
					Patterns: []types.Pattern{{Match: "*", Target: "/dest"}},
				},
				WatchDirectories: []string{"/watch/dir"},
			},
			wantErr: true,
		},
		{
			name: "invalid empty pattern",
			config: &config.Config{
				Organize: struct {
					Patterns []types.Pattern `yaml:"patterns"`
				}{
					Patterns: []types.Pattern{{Match: " ", Target: "/dest"}},
				},
				Settings: struct {
					DryRun     bool   `yaml:"dry_run"`
					CreateDirs bool   `yaml:"create_dirs"`
					Backup     bool   `yaml:"backup"`
					Collision  string `yaml:"collision"`
				}{
					Collision: "skip",
					Backup:    false,
				},
				WatchDirectories: []string{"/watch/dir"},
			},
			wantErr: true,
		},
		{
			name: "invalid empty target",
			config: &config.Config{
				Organize: struct {
					Patterns []types.Pattern `yaml:"patterns"`
				}{
					Patterns: []types.Pattern{{Match: "*.txt", Target: " "}},
				},
				Settings: struct {
					DryRun     bool   `yaml:"dry_run"`
					CreateDirs bool   `yaml:"create_dirs"`
					Backup     bool   `yaml:"backup"`
					Collision  string `yaml:"collision"`
				}{
					Collision: "rename",
					Backup:    false,
				},
				WatchDirectories: []string{"/watch/dir"},
			},
			wantErr: true,
		},
		{
			name: "invalid empty watch directory",
			config: &config.Config{
				Organize: struct {
					Patterns []types.Pattern `yaml:"patterns"`
				}{
					Patterns: []types.Pattern{{Match: "*", Target: "/dest"}},
				},
				Settings: struct {
					DryRun     bool   `yaml:"dry_run"`
					CreateDirs bool   `yaml:"create_dirs"`
					Backup     bool   `yaml:"backup"`
					Collision  string `yaml:"collision"`
				}{
					Collision: "rename",
					Backup:    false,
				},
				WatchDirectories: []string{""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- Add tests for SaveConfig, defaultConfig etc. here ---
