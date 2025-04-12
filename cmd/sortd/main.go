package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sortd/internal/analysis"
	"sortd/internal/organize"
	"sortd/internal/watch"

	"sortd/internal/config"
	"sortd/internal/gui"

	"sortd/cmd/sortd/cli"

	log "github.com/sirupsen/logrus"
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

	// Add commands that are only defined in main.go
	rootCmd.AddCommand(analyzeCmd())
	rootCmd.AddCommand(organizeCmd())
	// rootCmd.AddCommand(tuiCmd())
	rootCmd.AddCommand(guiCmd())
	rootCmd.AddCommand(watchCmd())
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(cli.ThemeCmd)

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

// organizeCmd represents the organize command - REINSTATED
func organizeCmd() *cobra.Command {
	var dir string
	var dryRun bool
	var nonInteractive bool

	cmd := &cobra.Command{
		Use:   "organize [directory]",
		Short: "Organize files in a directory",
		Long:  `Organize files in a directory based on patterns and content analysis.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine the target directory
			targetDir := dir // Use flag first
			if targetDir == "" && len(args) > 0 {
				targetDir = args[0] // Use argument if flag not set
			}
			if targetDir == "" {
				var err error
				targetDir, err = os.Getwd() // Default to current directory
				if err != nil {
					return fmt.Errorf("error getting current directory: %w", err)
				}
			}

			// Load configuration
			cfg, err := config.LoadConfig()
			if err != nil {
				fmt.Printf("Warning: Could not load config: %v. Using default settings.\n", err)
				cfg = config.New()
			}
			if cmd.Flags().Changed("dry-run") {
				cfg.Settings.DryRun = dryRun
			}
			if cmd.Flags().Changed("non-interactive") {
				cfg.Settings.NonInteractive = nonInteractive
			}

			// Create the organize engine
			engine := organize.NewWithConfig(cfg)

			// Perform organization
			if cfg.Settings.DryRun {
				fmt.Printf("Dry run: Planning organization for directory '%s'\n", targetDir)
			} else {
				fmt.Printf("Organizing directory '%s'\n", targetDir)
			}

			results, err := engine.OrganizeDirectory(targetDir)
			if err != nil {
				return fmt.Errorf("error organizing directory: %w", err)
			}

			// Print results
			if len(results) == 0 {
				fmt.Println("No files needed organization.")
			} else {
				fmt.Printf("Organization Summary (%d actions taken):\n", len(results))
				for _, res := range results {
					status := "Moved"
					if !res.Moved {
						status = "Skipped"
						if res.Error != nil {
							status = fmt.Sprintf("Error: %v", res.Error)
						}
					}
					fmt.Printf("  - %s -> %s (%s)\n", res.SourcePath, res.DestinationPath, status)
				}
			}

			if cfg.Settings.DryRun {
				fmt.Println("\nDry run complete. No files were moved.")
			} else {
				fmt.Println("\nOrganization complete.")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&dir, "directory", "d", "", "Directory to organize (overrides argument, defaults to current directory)")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be done without actually moving files")
	cmd.Flags().BoolVarP(&nonInteractive, "non-interactive", "N", false, "Run in non-interactive mode (no prompts)")

	return cmd
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

// analyzeCmd represents the analyze command
func analyzeCmd() *cobra.Command {
	var dir string
	var detailed bool

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze files in a directory",
		Long:  `Analyze files in a directory to suggest organization.`,
		Run: func(cmd *cobra.Command, args []string) {
			if dir == "" {
				var err error
				dir, err = os.Getwd()
				if err != nil {
					fmt.Println("Error getting current directory:", err)
					return
				}
			}

			// Create the analysis engine and run the analysis
			engine := analysis.New()
			result, err := engine.ScanDirectory(dir)
			if err != nil {
				fmt.Printf("Error analyzing directory: %v\n", err)
				return
			}

			// Display the results
			fmt.Printf("== Analysis for %s ==\n\n", dir)
			fmt.Printf("Total files: %d\n", len(result))

			// Group files by type
			fileTypes := make(map[string]int)
			filesByType := make(map[string][]string)

			for _, file := range result {
				fileType := file.ContentType
				if fileType == "" {
					fileType = "unknown"
				}

				fileTypes[fileType]++
				filesByType[fileType] = append(filesByType[fileType], file.Path)
			}

			// Sort file types by count for nicer display
			var types []struct {
				Type  string
				Count int
			}
			for t, c := range fileTypes {
				types = append(types, struct {
					Type  string
					Count int
				}{t, c})
			}
			sort.Slice(types, func(i, j int) bool {
				return types[i].Count > types[j].Count
			})

			fmt.Println("\nFile types:")
			for _, t := range types {
				fmt.Printf("  %s: %d files\n", t.Type, t.Count)
			}

			// If detailed is true, show file listing by type
			if detailed {
				fmt.Println("\nDetailed listing:")
				for _, t := range types {
					fmt.Printf("\n== %s files ==\n", t.Type)
					for _, f := range filesByType[t.Type] {
						fmt.Printf("  %s\n", filepath.Base(f))
					}
				}
			}
		},
	}

	cmd.Flags().StringVarP(&dir, "directory", "d", "", "Directory to analyze (default is current directory)")
	cmd.Flags().BoolVarP(&detailed, "detailed", "v", false, "Show detailed listing of files")

	return cmd
}

// tuiCmd is deprecated and disabled
// func tuiCmd() *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "tui",
// 		Short: "Start the terminal user interface",
// 		Long:  `Start the terminal user interface for interactive file organization.`,
// 		Run: func(cmd *cobra.Command, args []string) {
// 			// Create and run the TUI with the version
// 			m := tui.New(version)
// 			// Initialize Bubble Tea program WITHOUT alt screen for potentially better compatibility in non-TTY environments
// 			p := tea.NewProgram(m)
// 			if _, err := p.Run(); err != nil {
// 				fmt.Printf("Error running TUI: %v\n", err)
// 				os.Exit(1)
// 			}
// 		},
// 	}

// 	return cmd
// }

// watchCmd creates a command for watch mode
func watchCmd() *cobra.Command {
	var background bool // Keep background flag for now
	var nonInteractive bool

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch configured directories for changes and organize automatically",
		Long: `Watch directories specified in the configuration for new or modified files
and organize them automatically based on defined rules and patterns.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Load configuration
			cfg, err := config.LoadConfig()
			if err != nil {
				fmt.Printf("Error loading config: %v. Using default settings.\n", err)
				cfg = config.New()
				// Ensure some watch directories exist, maybe default to Downloads?
				// For now, let's just warn if none are configured.
				if len(cfg.WatchDirectories) == 0 {
					fmt.Println("Warning: No watch directories configured. Exiting.")
					os.Exit(1)
				}
			} else if len(cfg.WatchDirectories) == 0 {
				fmt.Println("No watch directories specified in the configuration. Nothing to watch.")
				fmt.Println("Please add directories to the 'watch_directories' section of your config file.")
				os.Exit(0)
			}

			// Handle non-interactive flag
			if cmd.Flags().Changed("non-interactive") {
				cfg.Settings.NonInteractive = nonInteractive
			}

			// Create the watch daemon - Pass only config, returns (*Daemon, error)
			_, err = watch.NewDaemon(cfg)
			if err != nil {
				log.Errorf("Failed to create watch daemon: %v", err)
				return
			}

			// Handle background mode (basic implementation)
			if background {
				fmt.Println("Starting watch daemon in background...")
				if err := watch.DaemonControl(cfg, false); err != nil {
					log.Errorf("Failed to start daemon in background: %v", err)
					return
				}
				fmt.Println("Daemon started. Logs will be written to sortd.log")
				fmt.Println("Use 'sortd daemon stop' to stop the daemon")
				return // Exit after starting the daemon
			}

			// Run watch mode in foreground
			fmt.Println("Starting watch daemon in foreground. Press Ctrl+C to stop.")
			fmt.Printf("Watching directories: %v\n", cfg.WatchDirectories)

			// Start the daemon in foreground mode
			if err := watch.DaemonControl(cfg, true); err != nil {
				log.Errorf("Failed to start daemon in foreground: %v", err)
				return
			}
		},
	}

	// Remove old flags
	// cmd.Flags().StringVarP(&dir, "dir", "d", "", "Directory to watch (default is current directory)") // Removed
	// cmd.Flags().IntVarP(&interval, "interval", "i", 300, "Watch interval in seconds (default 300)") // Removed

	// Keep background flag for now, although true daemonization is needed later
	cmd.Flags().BoolVarP(&background, "background", "b", false, "Run in background mode (basic implementation)")
	cmd.Flags().BoolVarP(&nonInteractive, "non-interactive", "N", false, "Run in non-interactive mode (no prompts)")

	return cmd
}

// Add daemon control commands
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Control the watch daemon",
	Long:  `Control the background watch daemon process.`,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Errorf("Failed to load config: %v", err)
			return
		}
		status, err := watch.Status(cfg)
		if err != nil {
			log.Errorf("Failed to get daemon status: %v", err)
			return
		}

		fmt.Printf("Daemon Status:\n")
		fmt.Printf("  Running: %v\n", status.Running)
		fmt.Printf("  Watch Directories: %v\n", status.WatchDirectories)
		fmt.Printf("  Last Activity: %v\n", status.LastActivity)
		fmt.Printf("  Files Processed: %d\n", status.FilesProcessed)
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running daemon",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Errorf("Failed to load config: %v", err)
			return
		}
		if err := watch.StopDaemon(cfg); err != nil {
			log.Errorf("Failed to stop daemon: %v", err)
			return
		}
		fmt.Println("Daemon stopped successfully")
	},
}

// Create a start command for daemon to replace using "watch -b"
var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the daemon in background mode",
	Run: func(cmd *cobra.Command, args []string) {
		// Get non-interactive flag value
		nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

		// Load configuration
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Errorf("Failed to load config: %v", err)
			return
		}

		// Set non-interactive mode if flag is provided
		if cmd.Flags().Changed("non-interactive") {
			cfg.Settings.NonInteractive = nonInteractive
		}

		// Check if watch directories are configured
		if len(cfg.WatchDirectories) == 0 {
			log.Error("No watch directories configured. Please update your config.")
			return
		}

		// Start the daemon
		fmt.Println("Starting watch daemon in background...")
		if err := watch.DaemonControl(cfg, false); err != nil {
			log.Errorf("Failed to start daemon: %v", err)
			return
		}
		fmt.Println("Daemon started successfully. Logs will be written to sortd.log")
	},
}

func init() {
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStartCmd)

	// Add non-interactive flag to daemon start command
	daemonStartCmd.Flags().BoolP("non-interactive", "N", false, "Run in non-interactive mode (no prompts)")
}
