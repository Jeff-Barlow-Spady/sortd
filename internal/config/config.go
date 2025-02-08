package config

import (
	"os"
	"path/filepath"

	"github.com/toasty/sortd/pkg/types"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Organize struct {
		Patterns []types.Pattern `yaml:"patterns"`
	} `yaml:"organize"`
	Settings struct {
		DryRun     bool   `yaml:"dry_run"`
		CreateDirs bool   `yaml:"create_dirs"`
		Backup     bool   `yaml:"backup"`
		Collision  string `yaml:"collision"` // rename, skip, or ask
	} `yaml:"settings"`
}

// LoadConfig loads configuration from the default location
func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(home, ".config", "sortd", "config.yaml")
	return LoadConfigFile(configPath)
}

// LoadConfigFile loads configuration from a specific file
func LoadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return defaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// defaultConfig returns the default configuration
func defaultConfig() *Config {
	cfg := &Config{}
	cfg.Settings.DryRun = true     // Safe by default
	cfg.Settings.CreateDirs = true // Create destination directories
	cfg.Settings.Backup = false    // No backup by default
	cfg.Settings.Collision = "ask" // Ask on collision by default
	return cfg
}
