package main

import (
	"fmt"
	"os"
	"sortd/cmd/sortd/cli"
	"sortd/internal/config"
	"sortd/internal/gui"
	"sortd/internal/organize"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
)

// Entry point for the application
func main() {
	// Get the root command from the factory function in root.go
	rootCmd := NewRootCmd()

	// Set the version from this file
	rootCmd.Version = version

	// Load theme preferences
	cli.LoadThemePreference()

	// Prepend logo to help message
	helpTemplate := cli.DrawSortdLogo() + "\n\n" + rootCmd.UsageTemplate()
	rootCmd.SetUsageTemplate(helpTemplate)
	rootCmd.SetHelpTemplate(helpTemplate)

	// Add commands that are only defined in main.go and not duplicated elsewhere
	rootCmd.AddCommand(guiCmd())

	// Initialize workflow commands
	initWorkflowCommands(rootCmd)

	// Execute the command with improved error handling
	if err := rootCmd.Execute(); err != nil {
		// Print to both stderr and stdout to ensure tests can capture it
		errMsg := fmt.Sprintf("Error: %s", err)
		fmt.Fprintln(os.Stderr, errMsg)
		fmt.Println(errMsg) // Also print to stdout for test capturing
		os.Exit(1)
	}
}

// runGUI launches the GUI directly
func runGUI() error {
	// Load configuration or use defaults
	cfg, err := config.LoadConfig()
	if err != nil {
		// Create a default config if loading fails
		cfg = config.New()
		// Set some reasonable defaults
		cfg.Settings.DryRun = false
		cfg.Settings.CreateDirs = true
		cfg.Settings.Collision = "rename"
		cfg.Directories.Default = "."
	}

	// Create organize engine
	organizeEngine := organize.NewWithConfig(cfg)

	// Create and run the GUI application
	guiApp := gui.NewApp(cfg, organizeEngine)
	guiApp.Run()

	return nil
}

// guiCmd creates the GUI command for the CLI
func guiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gui",
		Short: "Launch the graphical user interface",
		Long:  `Launch the GUI version of sortd for a more visual file organization experience.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Launching GUI interface...")
			if err := runGUI(); err != nil {
				fmt.Printf("Error launching GUI: %v\n", err)
				os.Exit(1)
			}
		},
	}
}

// tuiCmd is deprecated and disabled
// func tuiCmd() *cobra.Command {
//    (code removed for brevity)
// }

// All duplicated commands removed:
// - watchCmd
// - daemonCmd and associated variables (daemonStatusCmd, daemonStopCmd, daemonStartCmd)
