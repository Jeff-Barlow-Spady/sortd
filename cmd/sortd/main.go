package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sortd/internal/analysis"
	"sortd/internal/organize"
	"sortd/internal/tui"
	"strings"

	"sortd/internal/config"
	"sortd/internal/gui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
)

// Entry point for the application
func main() {
	// Global flags for all commands
	var (
		guiMode        bool
		backgroundMode bool
		watchMode      bool
		watchDir       string
		watchInterval  int
	)

	// Create the root command
	rootCmd := &cobra.Command{
		Use:     "sortd",
		Short:   "A file sorting utility",
		Long:    `Sortd helps you organize files by pattern, content, and more.`,
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			// Check special modes in order of precedence
			if guiMode {
				// Launch the GUI interface
				fmt.Println("Launching GUI interface...")

				if err := runGUI(); err != nil {
					fmt.Printf("Error launching GUI: %v\n", err)
					os.Exit(1)
				}
				return
			}

			if watchMode {
				// Start in watch mode
				fmt.Println("Starting watch mode...")

				// Set up watch directory
				if watchDir == "" {
					var err error
					watchDir, err = os.Getwd()
					if err != nil {
						fmt.Printf("Error getting current directory: %v\n", err)
						os.Exit(1)
					}
				}

				fmt.Printf("Watching directory: %s (interval: %d seconds)\n", watchDir, watchInterval)

				if backgroundMode {
					fmt.Println("Running in background mode")
					// Here we'd daemonize the process for background watching
					// For now, just simulate with a simple message
					fmt.Println("Background mode simulation - would fork to background here")
				} else {
					// Run watch mode in foreground
					fmt.Println("Running in foreground mode. Press Ctrl+C to stop.")
					// Here we'd implement an actual watch loop
					fmt.Println("Watch mode simulation - would start watching now")
				}
				return
			}

			// Default to TUI if no special flags
			fmt.Println("Starting TUI mode...")
			tuiCmd().Run(cmd, args)
		},
	}

	// Add global flags
	rootCmd.Flags().BoolVar(&guiMode, "gui", false, "Start in GUI mode with system tray icon")
	rootCmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "Start in watch mode to automatically organize files")
	rootCmd.Flags().StringVarP(&watchDir, "dir", "d", "", "Directory to watch (default is current directory)")
	rootCmd.Flags().IntVarP(&watchInterval, "interval", "i", 300, "Watch interval in seconds (default 300)")
	rootCmd.Flags().BoolVarP(&backgroundMode, "background", "b", false, "Run in background mode")

	// Add subcommands
	rootCmd.AddCommand(analyzeCmd())
	rootCmd.AddCommand(organizeCmd())
	rootCmd.AddCommand(tuiCmd())
	rootCmd.AddCommand(guiCmd())
	rootCmd.AddCommand(watchCmd())

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
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

// organizeCmd represents the organize command
func organizeCmd() *cobra.Command {
	var dir string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "organize",
		Short: "Organize files in a directory",
		Long:  `Organize files in a directory based on patterns and content analysis.`,
		Run: func(cmd *cobra.Command, args []string) {
			if dir == "" {
				var err error
				dir, err = os.Getwd()
				if err != nil {
					fmt.Println("Error getting current directory:", err)
					return
				}
			}

			// Create the organize engine
			engine := organize.New()

			// Get all files in the directory
			var files []string
			err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() && !strings.HasPrefix(filepath.Base(path), ".") {
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				fmt.Printf("Error walking directory: %v\n", err)
				return
			}

			// If dry run, just show what would happen
			if dryRun {
				fmt.Println("Dry run: No files will be moved")

				// Since we don't have a PlanOrganization method, we'll just show what files would be organized
				fmt.Println("\nFiles that would be organized:")
				for _, file := range files {
					fmt.Printf("  %s\n", file)
				}
				return
			}

			// Actually organize the files
			fmt.Println("Organizing files...")
			err = engine.OrganizeByPatterns(files)
			if err != nil {
				fmt.Printf("Error organizing files: %v\n", err)
				return
			}
			fmt.Println("Organization complete!")
		},
	}

	cmd.Flags().StringVarP(&dir, "directory", "d", "", "Directory to organize (default is current directory)")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be done without actually moving files")

	return cmd
}

// tuiCmd represents the TUI command
func tuiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Start the terminal user interface",
		Long:  `Start the terminal user interface for interactive file organization.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create and run the TUI with the version
			m := tui.New(version)
			p := tea.NewProgram(m, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				fmt.Printf("Error running TUI: %v\n", err)
				os.Exit(1)
			}
		},
	}

	return cmd
}

// watchCmd creates a command for watch mode
func watchCmd() *cobra.Command {
	var (
		dir        string
		interval   int
		background bool
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch directory for changes and organize automatically",
		Long:  `Watch the specified directory for new files and organize them automatically based on patterns.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Set default directory if none provided
			if dir == "" {
				var err error
				dir, err = os.Getwd()
				if err != nil {
					fmt.Printf("Error getting current directory: %v\n", err)
					os.Exit(1)
				}
			}

			fmt.Printf("Watching directory: %s (interval: %d seconds)\n", dir, interval)

			if background {
				fmt.Println("Running in background mode")
				// Here we'd daemonize the process for background watching
				// For now, just simulate with a simple message
				fmt.Println("Background mode simulation - would fork to background here")
			} else {
				// Run watch mode in foreground
				fmt.Println("Running in foreground mode. Press Ctrl+C to stop.")
				// Here we'd implement an actual watch loop
				fmt.Println("Watch mode simulation - would start watching now")

				// Block to simulate running
				fmt.Println("Press Ctrl+C to exit...")
				// Block indefinitely
				select {}
			}
		},
	}

	// Add watch mode flags
	cmd.Flags().StringVarP(&dir, "dir", "d", "", "Directory to watch (default is current directory)")
	cmd.Flags().IntVarP(&interval, "interval", "i", 300, "Watch interval in seconds (default 300)")
	cmd.Flags().BoolVarP(&background, "background", "b", false, "Run in background mode")

	return cmd
}
