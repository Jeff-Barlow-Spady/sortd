package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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
			// Create a colorful ASCII logo in DOS Rebel style
			logo := `
    ` + colorize("██████   ██████  ██████  ████████ ██████", Color("99")) + `
    ` + colorize("██       ██    ██ ██   ██    ██    ██   ██", Color("134")) + `
    ` + colorize("███████  ██    ██ ██████     ██    ██   ██", Color("171")) + `
    ` + colorize("     ██  ██    ██ ██   ██    ██    ██   ██", Color("213")) + `
    ` + colorize("██████    ██████  ██   ██    ██    ██████", Color("212")) + `
    ` + colorize("                                         ", Color("211")) + `
    ` + colorize("      FILE ORGANIZATION SYSTEM           ", Color("213")) + `

` + primaryText("Sortd") + ` helps you organize files with vim-like keybindings and powerful rules.

` + successText("QUICK START:") + `
  • Run ` + emphasisText("sortd setup") + ` to configure your preferences
  • Use ` + emphasisText("sortd organize ~/Downloads") + ` to organize a directory
  • Start ` + emphasisText("sortd") + ` without arguments to enter the TUI file browser
  • Try ` + emphasisText("sortd watch") + ` to automatically organize files as they arrive
  • Launch the GUI with ` + emphasisText("sortd gui") + `

` + infoText("TIP:") + ` The TUI uses vim-style keybindings (j/k navigate, space selects files, ? for help)
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
	// rootCmd.AddCommand(NewThemeCmd()) // Theme functionality is not yet implemented
	rootCmd.AddCommand(NewCloudCmd())
	rootCmd.AddCommand(NewAnalyzeCmd())
	rootCmd.AddCommand(NewScanCmd())
	rootCmd.AddCommand(NewConfirmCmd())
	rootCmd.AddCommand(NewGUICmd())

	return rootCmd
}

// NewThemeCmd creates the theme command
/*
func NewThemeCmd() *cobra.Command {
	// This function is commented out until the theme functionality is implemented in the config package
	return nil
}
*/

// NewVersionCmd creates the version command
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of sortd",
		Long:  `All software has versions. This is sortd's.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Use our new style helpers with a fixed color instead of cfg.Theme.Border
			versionText := bold(emphasisText("sortd version 0.1.0"))
			fmt.Print(frame(versionText, Color("39"))) // Use a light blue color instead
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

	// Check if this appears to be a theme choice
	if isThemeChoice(options) {
		// Use enhanced styling for theme selection
		args := []string{"choose",
			"--header", "Use arrow keys to navigate, space to select",
			"--header.foreground", "39", // Blue for header
			"--cursor.foreground", "213", // Highlight for cursor
			"--selected.foreground", "114", // Green for selected item
			"--height", "10"}
		args = append(args, options...)
		return runGumWithDefault(defaultOption, args...)
	}

	// Standard implementation for non-theme choices
	args := []string{"choose"}
	args = append(args, options...)
	return runGumWithDefault(defaultOption, args...)
}

// Helper to determine if this is a theme choice (list of theme names)
func isThemeChoice(options []string) bool {
	// Simple heuristic: if the list contains common theme names, it's likely a theme choice
	themeNames := []string{"default", "dark", "light", "monokai", "nord", "solarized"}

	matches := 0
	for _, option := range options {
		for _, theme := range themeNames {
			if strings.Contains(strings.ToLower(option), theme) {
				matches++
				break
			}
		}
	}

	return matches >= len(options)/3 // If at least 1/3 of options match theme names
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
	// Always use default color
	return colorize("✓ "+text, "114") // Default green
}

// warningText formats text as a warning message
func warningText(text string) string {
	// Always use default color
	return colorize("⚠ "+text, "220") // Default yellow
}

// errorText formats text as an error message
func errorText(text string) string {
	// Always use default color
	return colorize("✗ "+text, "196") // Default red
}

// infoText formats text as an informational message
func infoText(text string) string {
	// Always use default color
	return colorize("ℹ "+text, "39") // Default blue
}

// primaryText formats text with the primary theme color
func primaryText(text string) string {
	// Always use default color
	return colorize(text, "213") // Default purple
}

// emphasisText formats text with the emphasis color
func emphasisText(text string) string {
	// Always use default color
	return colorize(text, "212") // Default light pink
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	rootCmd := NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
