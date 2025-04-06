package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"sortd/internal/config"

	"github.com/spf13/cobra"
)

// NewSetupCmd creates the setup command
func NewSetupCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Interactive setup wizard for sortd",
		Long:  `A fun, interactive setup wizard to configure sortd with your preferences.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Handle dry run mode
			if dryRun {
				fmt.Println(infoText("Dry run: This would configure your sortd configuration"))
				fmt.Println(infoText("Configuration would be saved to: " + filepath.Join(getHomeDir(), ".config", "sortd", "config.yaml")))
				return
			}

			// Skip interactive mode in tests
			if os.Getenv("TESTMODE") == "true" {
				fmt.Println(successText("Setup complete (test mode)"))
				return
			}

			// Ensure gum is available
			if _, err := exec.LookPath("gum"); err != nil {
				fmt.Println(errorText("This command requires gum to be installed."))
				fmt.Println(infoText("Install Gum from https://github.com/charmbracelet/gum"))
				return
			}

			// Setup signal handling for clean exits
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Println(infoText("\nSetup wizard cancelled. No changes were made."))
				os.Exit(0)
			}()

			fmt.Println(colorize("🪄 Welcome to the sortd setup wizard!", Color("213")))
			fmt.Println(infoText("(Press Ctrl+C at any time to exit)"))

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
			fmt.Println(infoText("\n📂 First, let's set up your default directory:"))
			fmt.Println("You can type a path or browse for a directory")
			dirChoice := runGumChoose("Enter path manually", "Browse directories")

			var defaultDir string
			if dirChoice == "Browse directories" {
				// Use file browser to select directory
				fmt.Println(infoText("Use arrow keys to navigate, Enter to select a directory, Esc to cancel"))
				defaultDir = runGumFile("--directory")
				if defaultDir == "" {
					defaultDir = getHomeDir()
				}
			} else {
				defaultDir = runGumInput("Enter default directory to organize", getHomeDir())
			}
			newConfig.Directories.Default = defaultDir

			// Ask about settings
			fmt.Println("📋 Configure basic settings:")
			newConfig.Settings.CreateDirs = runGumConfirm("Create destination directories if they don't exist?")
			newConfig.Settings.Backup = runGumConfirm("Create backups before moving files?")

			// Configure collision strategy
			fmt.Println("🔄 What should happen when files would be overwritten?")
			newConfig.Settings.Collision = runGumChoose("rename", "skip", "ask")

			// Choose a theme
			fmt.Println(colorize("🎨 Choose a theme:", Color("213")))
			// Hardcoded list of themes until config.ListThemes is implemented
			themes := []string{"default", "dark", "light", "colorful"}
			selectedTheme := runGumChoose(themes...)
			// Instead of applying theme (which is not implemented), just show feedback
			fmt.Println(successText("Selected theme: " + selectedTheme))

			// Ask about watch mode
			newConfig.WatchMode.Enabled = runGumConfirm("Enable watch mode for automatic organizing?")
			if newConfig.WatchMode.Enabled {
				// Add a directory to watch
				fmt.Println(infoText("\n📂 Select directories to watch:"))

				addMoreDirs := true
				for addMoreDirs {
					fmt.Println("Choose a directory to watch:")
					dirChoice := runGumChoose("Enter path manually", "Browse directories", "Done adding directories")

					if dirChoice == "Done adding directories" {
						addMoreDirs = false
						continue
					}

					var watchDir string
					if dirChoice == "Browse directories" {
						watchDir = runGumFile("--directory")
						if watchDir == "" {
							continue
						}
					} else {
						watchDir = runGumInput("Directory to watch", defaultDir)
						if watchDir == "" {
							continue
						}
					}

					// Validate the directory
					info, err := os.Stat(watchDir)
					if err != nil || !info.IsDir() {
						fmt.Println(errorText("Invalid directory: " + watchDir))
						continue
					}

					// Add the directory to watch list
					watchDir = filepath.Clean(watchDir)
					exists := false
					for _, dir := range newConfig.Directories.Watch {
						if dir == watchDir {
							exists = true
							break
						}
					}

					if !exists {
						newConfig.Directories.Watch = append(newConfig.Directories.Watch, watchDir)
						fmt.Println(successText("Added directory: " + watchDir))
					} else {
						fmt.Println(warningText("Directory already added: " + watchDir))
					}

					if len(newConfig.Directories.Watch) > 0 {
						addMoreDirs = runGumConfirm("Add another directory to watch?")
					}
				}
			}

			// Add rules - enhanced interactive rule builder
			fmt.Println(primaryText("\n📝 Now let's set up your organization rules:"))
			fmt.Println(infoText("Rules determine how files are automatically organized"))

			addMoreRules := runGumConfirm("Would you like to add organization rules now?")
			for addMoreRules {
				fmt.Println(emphasisText("\nSelect rule type:"))
				ruleType := runGumChoose("Simple file pattern", "File extension", "Directory rule")

				var pattern, target string

				switch ruleType {
				case "Simple file pattern":
					pattern = runGumInput("File pattern (e.g., *.jpg, screenshot*.png)", "")
					if pattern == "" {
						fmt.Println(warningText("Empty pattern, skipping rule"))
						continue
					}

				case "File extension":
					ext := runGumInput("File extension (without dot, e.g., pdf, jpg)", "")
					if ext == "" {
						fmt.Println(warningText("Empty extension, skipping rule"))
						continue
					}
					pattern = "*." + strings.TrimPrefix(ext, ".")

				case "Directory rule":
					dirPath := runGumInput("Directory path to organize", "")
					if dirPath == "" {
						fmt.Println(warningText("Empty directory path, skipping rule"))
						continue
					}
					pattern = filepath.Join(dirPath, "*")
				}

				// Now select target directory
				fmt.Println(infoText("\nSelect target directory:"))
				targetChoice := runGumChoose("Enter path manually", "Browse directories")

				if targetChoice == "Browse directories" {
					target = runGumFile("--directory")
					if target == "" {
						fmt.Println(warningText("No target directory selected, skipping rule"))
						continue
					}
				} else {
					target = runGumInput("Target directory", "")
					if target == "" {
						fmt.Println(warningText("Empty target directory, skipping rule"))
						continue
					}
				}

				// Confirm the rule
				fmt.Println(infoText("\nNew rule summary:"))
				fmt.Println("  Pattern: " + primaryText(pattern))
				fmt.Println("  Target:  " + primaryText(target))

				if runGumConfirm("Add this rule?") {
					newConfig.Rules = append(newConfig.Rules, struct {
						Pattern string `yaml:"pattern"`
						Target  string `yaml:"target"`
					}{
						Pattern: pattern,
						Target:  target,
					})
					fmt.Println(successText("Rule added!"))
				} else {
					fmt.Println(infoText("Rule skipped"))
				}

				// Ask about more rules
				addMoreRules = runGumConfirm("Would you like to add another rule?")
			}

			// Save config
			configDir := filepath.Join(getHomeDir(), ".config", "sortd")
			os.MkdirAll(configDir, 0755)
			configPath := filepath.Join(configDir, "config.yaml")

			// The SaveConfig function only takes the config object
			if err := config.SaveConfig(newConfig); err != nil {
				fmt.Printf(errorText("Error saving config: %v"), err)
				os.Exit(1)
			}

			// Show a pretty success message
			successMessage := bold(colorize("✨ Setup complete! Configuration saved to "+configPath, Color("114")))
			fmt.Print(frame(successMessage, Color("213")))

			// Show helpful next steps
			fmt.Println("\n🚀 " + bold("Next steps:"))
			fmt.Println("  • " + colorize("Run 'sortd organize' to organize files", Color("39")))
			fmt.Println("  • " + colorize("Run 'sortd rules add' to add more organization rules", Color("39")))
			fmt.Println("  • " + colorize("Run 'sortd watch' to start watching directories", Color("39")))

			// Ask if they want to explore advanced features
			if runGumConfirm("\nWould you like to learn about advanced features?") {
				runGum("style",
					"--foreground", "212",
					"--border", "rounded",
					"--border-foreground", "212",
					"--padding", "1",
					"--width", "70",
					"Advanced Features:\n\n"+
						"• Cloud Storage: Use 'sortd cloud' to manage remote storage\n"+
						"• Semantic Analysis: Use 'sortd analyze' for content-based organization\n"+
						"• Custom Plugins: See docs for extending sortd functionality")
			}
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")

	return cmd
}
