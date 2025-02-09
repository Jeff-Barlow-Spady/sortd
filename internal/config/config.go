package config

import (
	"fmt"
	"os"
	"path/filepath"

	"sortd/pkg/types"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration structure.
// It defines organization patterns, settings, and watch mode parameters.
type Config struct {
	Organize struct {
		Patterns []types.Pattern `yaml:"patterns"` // File organization patterns
	} `yaml:"organize"`
	Settings struct {
		DryRun     bool   `yaml:"dry_run"`     // If true, simulate operations
		CreateDirs bool   `yaml:"create_dirs"` // Create destination directories
		Backup     bool   `yaml:"backup"`      // Create backups before moving
		Collision  string `yaml:"collision"`   // Collision strategy: rename, skip, or ask
	} `yaml:"settings"`
	Directories struct {
		Default string   `yaml:"default"` // Default working directory
		Watch   []string `yaml:"watch"`   // Directories to watch
	} `yaml:"directories"`
	Rules []struct {
		Pattern string `yaml:"pattern"` // Pattern to match
		Target  string `yaml:"target"`  // Target directory
	} `yaml:"rules"`
	WatchMode struct {
		Enabled  bool `yaml:"enabled"`  // Enable watch mode
		Interval int  `yaml:"interval"` // Watch interval in seconds
	} `yaml:"watch_mode"`
}

// LoadConfig loads configuration from the default location
// (~/.config/sortd/config.yaml).
func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(home, ".config", "sortd", "config.yaml")
	return LoadConfigFile(configPath)
}

// LoadConfigFile loads configuration from a specific file path.
// If the file doesn't exist, returns default configuration.
func LoadConfigFile(path string) (*Config, error) {
	// Start with default configuration
	cfg := defaultConfig()

	// Try to read the config file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Return defaults if file doesn't exist
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Unmarshal into a temporary config to preserve defaults for unset fields
	var tempCfg Config
	if err := yaml.Unmarshal(data, &tempCfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Merge the loaded config with defaults
	if len(tempCfg.Organize.Patterns) > 0 {
		cfg.Organize.Patterns = tempCfg.Organize.Patterns
	}
	if tempCfg.Settings.Collision != "" {
		cfg.Settings.Collision = tempCfg.Settings.Collision
	}
	cfg.Settings.DryRun = tempCfg.Settings.DryRun
	cfg.Settings.CreateDirs = tempCfg.Settings.CreateDirs
	cfg.Settings.Backup = tempCfg.Settings.Backup

	if tempCfg.Directories.Default != "" {
		cfg.Directories.Default = tempCfg.Directories.Default
	}
	if len(tempCfg.Directories.Watch) > 0 {
		cfg.Directories.Watch = tempCfg.Directories.Watch
	}
	if len(tempCfg.Rules) > 0 {
		cfg.Rules = tempCfg.Rules
	}

	cfg.WatchMode.Enabled = tempCfg.WatchMode.Enabled
	if tempCfg.WatchMode.Interval > 0 {
		cfg.WatchMode.Interval = tempCfg.WatchMode.Interval
	}

	// Validate the final configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// defaultConfig returns the default configuration with safe defaults.
func defaultConfig() *Config {
	cfg := &Config{}

	// Initialize the Organize struct with empty patterns slice
	cfg.Organize.Patterns = []types.Pattern{}

	// Set default settings
	cfg.Settings.DryRun = true     // Safe by default
	cfg.Settings.CreateDirs = true // Create destination directories
	cfg.Settings.Backup = false    // No backup by default
	cfg.Settings.Collision = "ask" // Ask on collision by default

	// Initialize directories struct
	cfg.Directories.Default = "." // Current directory by default
	cfg.Directories.Watch = []string{}

	// Initialize empty rules slice
	cfg.Rules = []struct {
		Pattern string `yaml:"pattern"`
		Target  string `yaml:"target"`
	}{}

	// Set default watch mode settings
	cfg.WatchMode.Enabled = false
	cfg.WatchMode.Interval = 5 // 5 seconds default interval

	return cfg
}

// SaveConfig saves the configuration to the default location.
// Creates the config directory if it doesn't exist.
func SaveConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("nil config")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".config", "sortd")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// Validate checks if the configuration is valid.
// Returns error if any settings are invalid.
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("nil config")
	}

	// Validate collision setting
	validCollisions := map[string]bool{"rename": true, "skip": true, "ask": true}
	if !validCollisions[c.Settings.Collision] {
		return fmt.Errorf("invalid collision setting: %s", c.Settings.Collision)
	}

	// Validate watch interval if watch mode is enabled
	if c.WatchMode.Enabled && c.WatchMode.Interval < 1 {
		return fmt.Errorf("watch interval must be >= 1 second")
	}

	// Validate patterns
	for i, pattern := range c.Organize.Patterns {
		if pattern.Match == "" {
			return fmt.Errorf("pattern %d: match pattern is required", i)
		}
		if pattern.Target == "" {
			return fmt.Errorf("pattern %d: target directory is required", i)
		}
	}

	// Validate rules
	for i, rule := range c.Rules {
		if rule.Pattern == "" {
			return fmt.Errorf("rule %d: pattern is required", i)
		}
		if rule.Target == "" {
			return fmt.Errorf("rule %d: target is required", i)
		}
	}

	// Validate directories
	if c.Directories.Default != "" {
		if _, err := os.Stat(c.Directories.Default); err != nil {
			if os.IsNotExist(err) {
				if !c.Settings.CreateDirs {
					return fmt.Errorf("default directory does not exist and create_dirs is false")
				}
			} else {
				return fmt.Errorf("error accessing default directory: %w", err)
			}
		}
	}

	return nil
}

// NewTestConfig creates a configuration instance for testing purposes.
func NewTestConfig() *Config {
	cfg := &Config{}
	cfg.Organize.Patterns = []types.Pattern{
		{Match: "*.txt", Target: "documents/"},
		{Match: "*.jpg", Target: "images/"},
	}
	cfg.Settings.DryRun = false
	cfg.Settings.CreateDirs = true
	cfg.Settings.Backup = true
	cfg.Settings.Collision = "rename"
	cfg.WatchMode.Interval = 5
	return cfg
}

// NewDefaultConfig creates a new configuration instance with default values.
// This is simply an alias for New().
func New() *Config {
	return defaultConfig()
}
