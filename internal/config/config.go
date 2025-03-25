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
		DryRun                 bool   `yaml:"dry_run"`                 // If true, simulate operations
		CreateDirs             bool   `yaml:"create_dirs"`             // Create destination directories
		Backup                 bool   `yaml:"backup"`                  // Create backups before moving
		Collision              string `yaml:"collision"`               // Collision strategy: rename, skip, or ask
		ImprovedCategorization bool   `yaml:"improved_categorization"` // Use improved file type categorization
	} `yaml:"settings"`
	Directories struct {
		Default string   `yaml:"default"` // Default working directory
		Watch   []string `yaml:"watch"`   // Directories to watch
	} `yaml:"directories"`
	Rules []struct {
		Pattern string `yaml:"pattern"` // Pattern to match
		Target  string `yaml:"target"`  // Target directory
	} `yaml:"rules"`
	// Directory-specific rules allow different rule sets for different directories
	DirectoryRules map[string][]struct {
		Pattern string `yaml:"pattern"` // Pattern to match
		Target  string `yaml:"target"`  // Target directory
	} `yaml:"directory_rules"`
	WatchMode struct {
		Enabled             bool `yaml:"enabled"`              // Enable watch mode
		Interval            int  `yaml:"interval"`             // Watch interval in seconds
		ConfirmationPeriod  int  `yaml:"confirmation_period"`  // Confirmation period in seconds (0 = disabled)
		RequireConfirmation bool `yaml:"require_confirmation"` // Require confirmation before executing rules
	} `yaml:"watch_mode"`
	Theme struct {
		Name     string `yaml:"name"`     // Theme name (default, dark, light, etc.)
		Primary  string `yaml:"primary"`  // Primary color for branding
		Success  string `yaml:"success"`  // Success message color
		Warning  string `yaml:"warning"`  // Warning message color
		Error    string `yaml:"error"`    // Error message color
		Info     string `yaml:"info"`     // Informational message color
		Emphasis string `yaml:"emphasis"` // Emphasis color for text that should stand out
		Border   string `yaml:"border"`   // Border color for frames
	} `yaml:"theme"`
	Templates []struct {
		Name        string `yaml:"name"`        // Template name
		Description string `yaml:"description"` // Template description
		Rules       []struct {
			Pattern string `yaml:"pattern"` // Pattern to match
			Target  string `yaml:"target"`  // Target directory
		} `yaml:"rules"`
	} `yaml:"templates"` // Rule templates for common use cases
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
	cfg.Settings.ImprovedCategorization = tempCfg.Settings.ImprovedCategorization

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
	cfg.WatchMode.ConfirmationPeriod = tempCfg.WatchMode.ConfirmationPeriod
	cfg.WatchMode.RequireConfirmation = tempCfg.WatchMode.RequireConfirmation

	// Initialize directory-specific rules
	cfg.DirectoryRules = make(map[string][]struct {
		Pattern string `yaml:"pattern"`
		Target  string `yaml:"target"`
	})

	// Merge directory-specific rules
	for dir, rules := range tempCfg.DirectoryRules {
		cfg.DirectoryRules[dir] = rules
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
	cfg.Settings.DryRun = true                 // Safe by default
	cfg.Settings.CreateDirs = true             // Create destination directories
	cfg.Settings.Backup = false                // No backup by default
	cfg.Settings.Collision = "ask"             // Ask on collision by default
	cfg.Settings.ImprovedCategorization = true // Enable improved categorization by default

	// Initialize directories struct
	cfg.Directories.Default = "." // Current directory by default
	cfg.Directories.Watch = []string{}

	// Initialize empty rules slice
	cfg.Rules = []struct {
		Pattern string `yaml:"pattern"`
		Target  string `yaml:"target"`
	}{}

	// Initialize directory-specific rules
	cfg.DirectoryRules = make(map[string][]struct {
		Pattern string `yaml:"pattern"`
		Target  string `yaml:"target"`
	})

	// Set default watch mode settings
	cfg.WatchMode.Enabled = false
	cfg.WatchMode.Interval = 5                // 5 seconds default interval
	cfg.WatchMode.ConfirmationPeriod = 0      // Disabled by default
	cfg.WatchMode.RequireConfirmation = false // Disabled by default

	// Initialize default templates
	cfg.Templates = []struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Rules       []struct {
			Pattern string `yaml:"pattern"`
			Target  string `yaml:"target"`
		} `yaml:"rules"`
	}{
		{
			Name:        "documents",
			Description: "Common document file organization",
			Rules: []struct {
				Pattern string `yaml:"pattern"`
				Target  string `yaml:"target"`
			}{
				{Pattern: "*.pdf", Target: "Documents/PDFs"},
				{Pattern: "*.doc*", Target: "Documents/Word"},
				{Pattern: "*.xls*", Target: "Documents/Excel"},
				{Pattern: "*.ppt*", Target: "Documents/Presentations"},
				{Pattern: "*.txt", Target: "Documents/Text"},
			},
		},
		{
			Name:        "media",
			Description: "Media file organization (images, video, audio)",
			Rules: []struct {
				Pattern string `yaml:"pattern"`
				Target  string `yaml:"target"`
			}{
				{Pattern: "*.{jpg,jpeg,png,gif,bmp,tiff}", Target: "Media/Images"},
				{Pattern: "*.{mp4,mov,avi,mkv,wmv}", Target: "Media/Videos"},
				{Pattern: "*.{mp3,wav,flac,aac,ogg}", Target: "Media/Audio"},
			},
		},
		{
			Name:        "downloads",
			Description: "Common downloads organization",
			Rules: []struct {
				Pattern string `yaml:"pattern"`
				Target  string `yaml:"target"`
			}{
				{Pattern: "*.{zip,tar,gz,rar,7z}", Target: "Downloads/Archives"},
				{Pattern: "*.{exe,msi,deb,rpm}", Target: "Downloads/Installers"},
				{Pattern: "*.{iso,img}", Target: "Downloads/Disk Images"},
			},
		},
	}

	return cfg
}

// SaveConfig saves the configuration to the specified file.
// It creates parent directories if they don't exist.
func SaveConfig(cfg *Config, path string) error {
	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal the config to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write the data to the file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
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

	// Validate confirmation period
	if c.WatchMode.ConfirmationPeriod < 0 {
		return fmt.Errorf("confirmation period must be >= 0 seconds")
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

	// Validate directory-specific rules
	for dir, rules := range c.DirectoryRules {
		if dir == "" {
			return fmt.Errorf("directory-specific rule: directory path cannot be empty")
		}
		for i, rule := range rules {
			if rule.Pattern == "" {
				return fmt.Errorf("directory rule %s/%d: pattern is required", dir, i)
			}
			if rule.Target == "" {
				return fmt.Errorf("directory rule %s/%d: target is required", dir, i)
			}
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

// GetTheme returns a predefined theme configuration by name.
// If the theme doesn't exist, returns the default theme.
func GetTheme(name string) map[string]string {
	themes := map[string]map[string]string{
		"default": {
			"primary":  "213", // Purple
			"success":  "114", // Green
			"warning":  "220", // Yellow
			"error":    "196", // Red
			"info":     "39",  // Blue
			"emphasis": "212", // Light Pink
			"border":   "213", // Purple
		},
		"dark": {
			"primary":  "105", // Dark Blue
			"success":  "78",  // Dark Green
			"warning":  "214", // Dark Yellow
			"error":    "160", // Dark Red
			"info":     "33",  // Dark Blue
			"emphasis": "147", // Light Blue
			"border":   "105", // Dark Blue
		},
		"light": {
			"primary":  "135", // Light Purple
			"success":  "150", // Light Green
			"warning":  "222", // Light Yellow
			"error":    "210", // Light Red
			"info":     "117", // Light Blue
			"emphasis": "219", // Very Light Pink
			"border":   "135", // Light Purple
		},
		"monochrome": {
			"primary":  "245", // Light Grey
			"success":  "252", // White
			"warning":  "241", // Medium Grey
			"error":    "232", // Black
			"info":     "248", // Grey
			"emphasis": "255", // Bright White
			"border":   "245", // Light Grey
		},
		"ocean": {
			"primary":  "31",  // Teal
			"success":  "36",  // Green-Blue
			"warning":  "220", // Yellow
			"error":    "196", // Red
			"info":     "33",  // Blue
			"emphasis": "51",  // Cyan
			"border":   "31",  // Teal
		},
		"sunset": {
			"primary":  "208", // Orange
			"success":  "154", // Green
			"warning":  "214", // Dark Yellow
			"error":    "196", // Red
			"info":     "69",  // Light Green
			"emphasis": "203", // Pink-Orange
			"border":   "208", // Orange
		},
	}

	if theme, exists := themes[name]; exists {
		return theme
	}

	return themes["default"]
}

// ApplyTheme sets the theme in the configuration.
// It updates the theme colors based on the theme name.
func (c *Config) ApplyTheme(name string) {
	theme := GetTheme(name)

	c.Theme.Name = name
	c.Theme.Primary = theme["primary"]
	c.Theme.Success = theme["success"]
	c.Theme.Warning = theme["warning"]
	c.Theme.Error = theme["error"]
	c.Theme.Info = theme["info"]
	c.Theme.Emphasis = theme["emphasis"]
	c.Theme.Border = theme["border"]
}

// ListThemes returns a list of available theme names.
func ListThemes() []string {
	return []string{"default", "dark", "light", "monochrome", "ocean", "sunset"}
}
