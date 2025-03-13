package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sortd/internal/config"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard for sortd",
	Long:  `A fun, interactive setup wizard to configure sortd with your preferences.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Skip interactive mode in tests
		if os.Getenv("TESTMODE") == "true" {
			fmt.Println("Setup complete (test mode)")
			return
		}

		// Ensure gum is available
		if _, err := exec.LookPath("gum"); err != nil {
			fmt.Println("❌ This command requires gum to be installed.")
			fmt.Println("💡 Install Gum from https://github.com/charmbracelet/gum")
			return
		}

		fmt.Println("🪄 Welcome to the sortd setup wizard!")

		// Use gum style for a styled header
		runGum("style",
			"--foreground", "212",
			"--border", "rounded",
			"--border-foreground", "212",
			"--padding", "1",
			"--width", "70",
			"Let's configure sortd to match your preferences!")

		// Create a new config
		newConfig := config.New()

		// Get default directory
		defaultDir := runGumInput("Enter default directory to organize", getHomeDir())
		newConfig.Directories.Default = defaultDir

		// Ask about settings
		fmt.Println("📋 Configure basic settings:")
		newConfig.Settings.CreateDirs = runGumConfirm("Create destination directories if they don't exist?")
		newConfig.Settings.Backup = runGumConfirm("Create backups before moving files?")

		// Configure collision strategy
		fmt.Println("🔄 What should happen when files would be overwritten?")
		newConfig.Settings.Collision = runGumChoose("rename", "skip", "ask")

		// Ask about watch mode
		newConfig.WatchMode.Enabled = runGumConfirm("Enable watch mode for automatic organizing?")
		if newConfig.WatchMode.Enabled {
			// Show formatted input for interval
			interval := runGumInput("Check interval in seconds", "300")
			newConfig.WatchMode.Interval = parseInterval(interval)

			// Add a directory to watch
			dirs := strings.Split(runGumInput("Directories to watch (comma separated)", defaultDir), ",")
			for _, dir := range dirs {
				dir = strings.TrimSpace(dir)
				if dir != "" {
					newConfig.Directories.Watch = append(newConfig.Directories.Watch, dir)
				}
			}
		}

		// Add a basic rule - this could be expanded to a more interactive rule builder
		fmt.Println("📝 Let's add your first rule:")
		pattern := runGumInput("File pattern (e.g., *.jpg)", "")
		if pattern != "" {
			target := runGumInput("Target directory", "")
			if target != "" {
				newConfig.Rules = append(newConfig.Rules, struct {
					Pattern string `yaml:"pattern"`
					Target  string `yaml:"target"`
				}{
					Pattern: pattern,
					Target:  target,
				})
			}
		}

		// Save config
		configDir := filepath.Join(getHomeDir(), ".config", "sortd")
		os.MkdirAll(configDir, 0755)
		configPath := filepath.Join(configDir, "config.yaml")

		if err := config.SaveConfig(newConfig, configPath); err != nil {
			fmt.Printf("❌ Error saving config: %v\n", err)
			os.Exit(1)
		}

		// Success!
		runGum("style",
			"--foreground", "212",
			"--border", "rounded",
			"--border-foreground", "212",
			"--padding", "1",
			"--width", "70",
			fmt.Sprintf("✨ Setup complete! Configuration saved to %s", configPath))

		// Show helpful next steps
		fmt.Println("\n🚀 Next steps:")
		fmt.Println("  • Run 'sortd organize' to organize files")
		fmt.Println("  • Run 'sortd rules add' to add more organization rules")
		fmt.Println("  • Run 'sortd watch' to start watching directories")
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

// Helper functions to run Gum commands

func runGum(args ...string) string {
	cmd := exec.Command("gum", args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		// Fallback for errors
		return ""
	}
	return strings.TrimSpace(string(output))
}

func runGumInput(prompt, defaultValue string) string {
	return runGum("input", "--placeholder", prompt, "--value", defaultValue)
}

func runGumConfirm(prompt string) bool {
	result := runGum("confirm", prompt)
	return result == ""
}

func runGumChoose(options ...string) string {
	args := []string{"choose"}
	args = append(args, options...)
	return runGum(args...)
}

func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

func parseInterval(input string) int {
	var interval int
	fmt.Sscanf(input, "%d", &interval)
	if interval <= 0 {
		return 300 // Default to 5 minutes
	}
	return interval
}

// This function needs to be added to the config package
// Save it in a new function in config.go
func SaveConfig(cfg *config.Config, path string) error {
	// Stub for now - will implement in config.go
	return nil
}
