package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"sortd/internal/config"
	"sortd/internal/log"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
	Version = "0.1.0" // Adding Version definition
)

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "sortd",
		Short: "Organizing Files Doesn't Have To Suck",
		Long: func() string {
			// Create a colorful ASCII logo that respects the theme
			// This function will be called after the config is loaded
			logo := `
	` + colorize("::######:::'#######::'########::'########:'########::", Color("99")) + `
	` + colorize("'##... ##:'##.... ##: ##.... ##:... ##..:: ##.... ##:", Color("134")) + `
	` + colorize("'##:::..:: ##:::: ##: ##:::: ##:::: ##:::: ##:::: ##:", Color("171")) + `
	` + colorize(". ######:: ##:::: ##: ########::::: ##:::: ##:::: ##:", Color("213")) + `
	` + colorize("::.... ##: ##:::: ##: ##.. ##:::::: ##:::: ##:::: ##:", Color("212")) + `
	` + colorize("'##::: ##: ##:::: ##: ##::. ##::::: ##:::: ##:::: ##:", Color("211")) + `
	` + colorize(". ######::. #######:: ##:::. ##:::: ##:::: ########::", Color("204")) + `
	` + colorize(":......::::.......:::..:::::..:::::..:::::........:::", Color("203")) + `

` + primaryText("Sortd") + ` helps you organize files in an ` + emphasisText("interactive") + `, ` + colorize("fun", Color("211")) + ` way!
			`
			return logo
		}(),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Check if we're in a test environment, but only skip interactive features
			inTestMode := os.Getenv("TESTMODE") == "true"

			// Check if gum is installed (skip in test mode)
			if !inTestMode {
				_, err := exec.LookPath("gum")
				if err != nil {
					fmt.Println(warningText("Gum is not installed! Some interactive features won't work."))
					fmt.Println(infoText("Install Gum from https://github.com/charmbracelet/gum"))
				}
			}

			// Load config (always do this, even in test mode)
			var configErr error
			if cfgFile != "" {
				cfg, configErr = config.LoadConfigFile(cfgFile)
			} else {
				cfg, configErr = config.LoadConfig()
			}

			if configErr != nil {
				if !inTestMode {
					fmt.Println(warningText(fmt.Sprintf("Warning: %v", configErr)))
					fmt.Println(infoText("Using default settings. Run 'sortd setup' to configure."))
				}
				cfg = config.New()
			}
		},
		Version: Version, // Add version to the root command
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/sortd/config.yaml)")

	// Add subcommands
	rootCmd.AddCommand(NewSetupCmd())
	rootCmd.AddCommand(NewOrganizeCmd())
	rootCmd.AddCommand(NewRulesCmd())
	rootCmd.AddCommand(NewWatchCmd())
	rootCmd.AddCommand(NewDaemonCmd())
	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewThemeCmd())
	rootCmd.AddCommand(NewCloudCmd())
	rootCmd.AddCommand(NewAnalyzeCmd())
	rootCmd.AddCommand(NewScanCmd())
	rootCmd.AddCommand(NewConfirmCmd()) // Add the confirm command
	rootCmd.AddCommand(NewGUICmd())     // Add the GUI command

	return rootCmd
}

// NewThemeCmd creates the theme command
func NewThemeCmd() *cobra.Command {
	var interactive bool

	cmd := &cobra.Command{
		Use:   "theme [theme-name]",
		Short: "Set or view the current theme",
		Long:  `Set the theme for sortd or view the current theme if no theme name is provided.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Load current config
			currentCfg, _ := config.LoadConfig()

			// If no args, just display the current theme
			if len(args) == 0 {
				// If theme name is empty, set a default
				if currentCfg.Theme.Name == "" {
					currentCfg.Theme.Name = "default"
				}

				fmt.Println(infoText("Current theme: " + currentCfg.Theme.Name))

				fmt.Println("\nAvailable themes:")
				for _, name := range config.ListThemes() {
					if name == currentCfg.Theme.Name {
						fmt.Println("  " + successText(name+" (current)"))
					} else {
						fmt.Println("  " + name)
					}
				}

				// Show a sample of the current theme colors with color blocks for better visibility
				fmt.Println("\nTheme color samples:")
				fmt.Println(colorize("  Primary color", Color(currentCfg.Theme.Primary)))
				fmt.Println(colorize("✓ Success color", Color(currentCfg.Theme.Success)))
				fmt.Println(colorize("⚠ Warning color", Color(currentCfg.Theme.Warning)))
				fmt.Println(colorize("✗ Error color", Color(currentCfg.Theme.Error)))
				fmt.Println(colorize("ℹ Info color", Color(currentCfg.Theme.Info)))
				fmt.Println(colorize("  Emphasis color", Color(currentCfg.Theme.Emphasis)))
				fmt.Print(frame("  Border color sample", Color(currentCfg.Theme.Border)))

				// If interactive mode is enabled, allow the user to choose a theme
				if interactive {
					// Check if gum is installed
					if _, err := exec.LookPath("gum"); err != nil {
						fmt.Println(errorText("Interactive mode requires gum to be installed."))
						fmt.Println(infoText("Install Gum from https://github.com/charmbracelet/gum"))
						return
					}

					fmt.Println(infoText("\nSelect a theme to apply (Press Ctrl+C to exit):"))

					// Create a channel to handle interruption signals
					sigChan := make(chan os.Signal, 1)
					signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

					// Set a timeout for theme selection
					timeout := time.NewTimer(time.Second * 120) // 2 minute timeout

					// Run the theme selection in a goroutine
					themeChan := make(chan string, 1)
					go func() {
						selectedTheme := runGumChoose(config.ListThemes()...)
						themeChan <- selectedTheme
					}()

					// Wait for either a theme selection, signal, or timeout
					select {
					case selectedTheme := <-themeChan:
						if selectedTheme == "" {
							// User cancelled the selection
							fmt.Println(infoText("\nTheme selection cancelled"))
							return
						}

						// Apply the selected theme
						currentCfg.ApplyTheme(selectedTheme)

						// Save the updated config
						home, _ := os.UserHomeDir()
						configPath := filepath.Join(home, ".config", "sortd", "config.yaml")
						if err := config.SaveConfig(currentCfg, configPath); err != nil {
							fmt.Println(errorText("Failed to save theme: " + err.Error()))
							return
						}

						// Apply the theme to the current session
						cfg = currentCfg

						fmt.Println(successText("Theme set to " + selectedTheme))
						fmt.Println(infoText("Theme applied to current session"))

					case <-sigChan:
						fmt.Println(infoText("\nTheme selection cancelled"))
						return

					case <-timeout.C:
						fmt.Println(infoText("\nTheme selection timed out"))
						return
					}
				}

				return
			}

			// Get the new theme name
			themeName := args[0]

			// Check if the theme exists
			themes := config.ListThemes()
			validTheme := false
			for _, name := range themes {
				if name == themeName {
					validTheme = true
					break
				}
			}

			if !validTheme {
				fmt.Println(errorText("Invalid theme: " + themeName))
				fmt.Println(infoText("Available themes: " + strings.Join(themes, ", ")))
				return
			}

			// Apply the new theme to the config
			currentCfg.ApplyTheme(themeName)

			// Save the updated config
			home, _ := os.UserHomeDir()
			configPath := filepath.Join(home, ".config", "sortd", "config.yaml")
			if err := config.SaveConfig(currentCfg, configPath); err != nil {
				fmt.Println(errorText("Failed to save theme: " + err.Error()))
				return
			}

			// Apply the theme to the current session
			cfg = currentCfg

			fmt.Println(successText("Theme set to " + themeName))
			fmt.Println(infoText("Theme applied to current session"))
		},
	}

	// Add interactive flag
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Choose theme interactively")

	return cmd
}

// NewVersionCmd creates the version command
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of sortd",
		Long:  `All software has versions. This is sortd's.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Use our new style helpers
			versionText := bold(emphasisText("sortd version 0.1.0"))
			fmt.Print(frame(versionText, Color(cfg.Theme.Border)))
		},
	}
}

// Helper functions to run Gum commands
func runGum(args ...string) string {
	// Check if Gum is available
	if _, err := exec.LookPath("gum"); err != nil {
		fmt.Println(warningText("Gum is not installed. Some interactive features won't work."))
		fmt.Println(infoText("Install Gum from https://github.com/charmbracelet/gum"))
		return ""
	}

	cmd := exec.Command("gum", args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		// Log the error but don't crash the program
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Println(errorText(fmt.Sprintf("Gum command failed: %s", string(exitErr.Stderr))))
		} else {
			fmt.Println(errorText(fmt.Sprintf("Gum command failed: %s", err)))
		}
		return ""
	}
	return strings.TrimSpace(string(output))
}

// runGumWithDefault runs a Gum command but provides a default value if Gum fails or isn't installed
func runGumWithDefault(defaultValue string, args ...string) string {
	result := runGum(args...)
	if result == "" {
		return defaultValue
	}
	return result
}

func runGumInput(prompt, defaultValue string) string {
	args := []string{"input", "--placeholder", prompt}
	if defaultValue != "" {
		args = append(args, "--value", defaultValue)
	}
	return runGumWithDefault(defaultValue, args...)
}

func runGumConfirm(prompt string) bool {
	args := []string{"confirm", prompt}
	result := runGumWithDefault("n", args...)
	return result == "0" // Gum returns exit code 0 for "yes"
}

func runGumChoose(options ...string) string {
	// If no options provided, return empty
	if len(options) == 0 {
		return ""
	}

	// Default to the first option if Gum fails
	defaultOption := options[0]

	// Basic implementation for non-theme use cases
	if !isThemeChoice(options) {
		args := []string{"choose"}
		args = append(args, options...)
		return runGumWithDefault(defaultOption, args...)
	}

	// Enhanced implementation for theme selection
	// Prepare a list of themed options with their respective colors
	var themedOptions []string
	for _, themeName := range options {
		theme := config.GetTheme(themeName)

		// Style the theme name with its primary color
		themedOption := colorizeWithColor(themeName, theme["primary"])
		themedOptions = append(themedOptions, themedOption)
	}

	// Use gum choose with styled options and header
	args := []string{"choose",
		"--header", "Use arrow keys to navigate, space to select",
		"--header.foreground", "39", // Blue for header
		"--cursor.foreground", "213", // Highlight for cursor
		"--selected.foreground", "114", // Green for selected item
		"--height", "10"}
	args = append(args, themedOptions...)

	// Run gum with styled options
	result := runGumWithDefault("", args...)

	// If Gum failed and returned the default (empty string), use the first theme
	if result == "" {
		fmt.Println(infoText(fmt.Sprintf("Falling back to default theme: %s", defaultOption)))
		return defaultOption
	}

	// Extract the theme name by stripping ANSI codes
	for _, themeName := range options {
		if strings.Contains(result, themeName) {
			return themeName
		}
	}

	// If we can't find the theme name, fallback to the default
	fmt.Println(infoText(fmt.Sprintf("Could not parse theme selection, using default: %s", defaultOption)))
	return defaultOption
}

// Helper to determine if this is a theme choice (list of theme names)
func isThemeChoice(options []string) bool {
	if len(options) == 0 {
		return false
	}

	// Get the list of valid themes
	themes := config.ListThemes()

	// Check if the first few options are valid themes
	themeCount := 0
	for _, option := range options {
		for _, theme := range themes {
			if option == theme {
				themeCount++
				break
			}
		}
	}

	// If most options are valid themes, consider it a theme choice
	return themeCount >= len(options)/2
}

// Helper function to colorize text with a specified color
func colorizeWithColor(text, colorCode string) string {
	return fmt.Sprintf("\033[38;5;%sm%s\033[0m", colorCode, text)
}

func runGumFile(args ...string) string {
	fileArgs := []string{"file"}
	fileArgs = append(fileArgs, args...)
	result := runGumWithDefault("", fileArgs...)

	// Verify the result is a valid file or directory
	if result != "" {
		if _, err := os.Stat(result); err != nil {
			fmt.Println(errorText(fmt.Sprintf("Invalid file selection: %s", err)))
			return ""
		}
	}

	return result
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

// Style functions for consistent visual appearance
type Color string

// colorize returns text with the specified color
func colorize(text string, color Color) string {
	// Ensure we have a valid color
	if color == "" {
		color = "7" // Default to light grey if no color provided
	}

	// Clean the color string to ensure it's just a number
	cleanColor := strings.TrimSpace(string(color))

	// Log color usage for debugging
	log.Debug("Using color: " + cleanColor + " for text: " + text)

	// Use 24-bit color for more vivid displays if supported
	return fmt.Sprintf("\033[38;5;%sm%s\033[0m", cleanColor, text)
}

// bold makes text bold
func bold(text string) string {
	return fmt.Sprintf("\033[1m%s\033[0m", text)
}

// frame puts text in a colored frame
func frame(text string, color Color) string {
	width := len(text) + 4 // Add padding
	top := fmt.Sprintf("┌%s┐", strings.Repeat("─", width))
	bottom := fmt.Sprintf("└%s┘", strings.Repeat("─", width))

	var result strings.Builder
	result.WriteString("\n")
	result.WriteString(colorize(top, color) + "\n")
	result.WriteString(colorize("│", color) + " " + text + " " + colorize("│", color) + "\n")
	result.WriteString(colorize(bottom, color) + "\n")

	return result.String()
}

// successText formats text as a success message
func successText(text string) string {
	if cfg == nil {
		return colorize("✓ "+text, "114") // Default green
	}
	return colorize("✓ "+text, Color(cfg.Theme.Success))
}

// warningText formats text as a warning message
func warningText(text string) string {
	if cfg == nil {
		return colorize("⚠ "+text, "220") // Default yellow
	}
	return colorize("⚠ "+text, Color(cfg.Theme.Warning))
}

// errorText formats text as an error message
func errorText(text string) string {
	if cfg == nil {
		return colorize("✗ "+text, "196") // Default red
	}
	return colorize("✗ "+text, Color(cfg.Theme.Error))
}

// infoText formats text as an informational message
func infoText(text string) string {
	if cfg == nil {
		return colorize("ℹ "+text, "39") // Default blue
	}
	return colorize("ℹ "+text, Color(cfg.Theme.Info))
}

// primaryText formats text with the primary theme color
func primaryText(text string) string {
	if cfg == nil {
		return colorize(text, "213") // Default purple
	}
	return colorize(text, Color(cfg.Theme.Primary))
}

// emphasisText formats text with the emphasis color
func emphasisText(text string) string {
	if cfg == nil {
		return colorize(text, "212") // Default light pink
	}
	return colorize(text, Color(cfg.Theme.Emphasis))
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	rootCmd := NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
