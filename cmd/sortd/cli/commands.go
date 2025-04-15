package cli

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sortd/internal/config"
	"sortd/internal/organize"
	"sortd/internal/watch"

	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

// findDestination finds the destination pattern for a file
func findDestination(engine *organize.Engine, filename string) (string, bool) {
	// This is a simplified implementation for matching rules
	basename := filepath.Base(filename)

	for _, rule := range cfg.Rules {
		matched, err := filepath.Match(rule.Pattern, basename)
		if err == nil && matched {
			return rule.Target, true
		}
	}

	return "", false
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "sortd",
	Short: "Sortd helps you organize files efficiently",
	Long: func() string {
		// Display our beautiful ASCII logo
		return DrawSortdLogo() + `
Sortd is a file organization tool that provides both terminal
and graphical interfaces to organize your files.

It supports pattern-based file organization, watch mode for
continuous organization, and more.`
	}(),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load configuration
		var configErr error
		if cfgFile != "" {
			cfg, configErr = config.LoadConfigFile(cfgFile)
		} else {
			cfg, configErr = config.LoadConfig()
		}

		// If config loading fails, use default config
		if configErr != nil {
			PrintWarning(fmt.Sprintf("Warning: %v", configErr))
			PrintInfo("Using default settings. Run 'sortd setup' to configure.")
			cfg = config.New()
		}
	},
}

// OrganizeCmd represents the organize command
var OrganizeCmd = &cobra.Command{
	Use:   "organize [directory]",
	Short: "Organize files in a directory",
	Long:  `Organize files according to defined rules.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Declare variables we'll need
		var targetDir string
		var files []string
		var interactive bool
		var fileCount int

		// Get command flags
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		recursive, _ := cmd.Flags().GetBool("recursive")
		interactive, _ = cmd.Flags().GetBool("interactive")

		// Print a nice header
		PrintHeader("📂 File Organization")

		// Get directory to organize, with interactive option if available
		if len(args) > 0 {
			targetDir = args[0]
		} else if cfg.Directories.Default != "" {
			targetDir = cfg.Directories.Default
		} else if HasGum() && interactive {
			// Use Gum to interactively select a directory
			PrintInfo("Choose a directory to organize:")
			dirChoice := RunGumChoose("Enter path manually", "Browse directories")

			if dirChoice == "Browse directories" {
				targetDir = RunGumFile("--directory")
			} else {
				targetDir = RunGumInput("Enter directory path", "")
			}

			if targetDir == "" {
				PrintError("No directory selected")
				os.Exit(1)
			}
		} else {
			// Use current directory as fallback
			var err error
			targetDir, err = os.Getwd()
			if err != nil {
				PrintError(fmt.Sprintf("Error getting current directory: %v", err))
				os.Exit(1)
			}
		}

		// Validate directory exists
		dirInfo, err := os.Stat(targetDir)
		if err != nil {
			PrintError(fmt.Sprintf("Error accessing directory: %v", err))
			os.Exit(1)
		}

		if !dirInfo.IsDir() {
			PrintError(fmt.Sprintf("%s is not a directory", targetDir))
			os.Exit(1)
		}

		// Create the organize engine
		engine := organize.NewWithConfig(cfg)

		// Set dry run mode if specified
		engine.SetDryRun(dryRun)

		// Scan for files
		if recursive {
			// Get all files recursively
			err = filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					files = append(files, path)
				}
				return nil
			})

			if err != nil {
				PrintError(fmt.Sprintf("Error scanning directory: %v", err))
				os.Exit(1)
			}
		} else {
			// Just get files in the specified directory
			entries, err := os.ReadDir(targetDir)
			if err != nil {
				PrintError(fmt.Sprintf("Error reading directory: %v", err))
				os.Exit(1)
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					files = append(files, filepath.Join(targetDir, entry.Name()))
				}
			}
		}

		fileCount = len(files)

		// Give information about what we're going to do
		if dryRun {
			PrintInfo(fmt.Sprintf("Dry run mode - scanning %d files in %s", fileCount, targetDir))
		} else {
			PrintInfo(fmt.Sprintf("Organizing %d files in %s", fileCount, targetDir))
		}

		// Allow the user to select files interactively if requested
		if HasGum() && interactive && fileCount > 0 {
			// Ask if user wants to select specific files
			if RunGumConfirm("Do you want to select specific files to organize?") {
				// Show file selection UI
				var displayPaths []string
				for _, file := range files {
					// Use relative paths for display
					relPath, err := filepath.Rel(targetDir, file)
					if err == nil {
						displayPaths = append(displayPaths, relPath)
					} else {
						displayPaths = append(displayPaths, filepath.Base(file))
					}
				}

				// Select files
				selections := RunGumMultiChoose(displayPaths...)

				// Update files list with selections
				if len(selections) > 0 {
					selectedFiles := make([]string, 0, len(selections))
					for _, selection := range selections {
						for i, displayPath := range displayPaths {
							if displayPath == selection {
								selectedFiles = append(selectedFiles, files[i])
								break
							}
						}
					}

					files = selectedFiles
					fileCount = len(files)
					PrintInfo(fmt.Sprintf("Selected %d files to organize", fileCount))
				}
			}
		}

		// Organize the files using our engine
		var results []struct {
			SourcePath      string
			DestinationPath string
			Moved           bool
			Error           error
		}

		// Organize each file individually for better control and reporting
		for _, file := range files {
			// Skip directories just to be safe
			fileInfo, err := os.Stat(file)
			if err != nil || fileInfo.IsDir() {
				continue
			}

			// Find destination and apply rules
			result := struct {
				SourcePath      string
				DestinationPath string
				Moved           bool
				Error           error
			}{
				SourcePath: file,
			}

			// Use findDestination helper to match rules
			destPattern, found := findDestination(engine, file)
			if found {
				destPath := filepath.Join(destPattern, filepath.Base(file))
				result.DestinationPath = destPath

				// Move file (or simulate in dry run)
				if dryRun {
					result.Moved = false
				} else {
					err := engine.MoveFile(file, destPath)
					if err != nil {
						result.Error = err
					} else {
						result.Moved = true
					}
				}
			}

			results = append(results, result)
		}

		// Display results summary
		if dryRun {
			PrintInfo("Dry run complete - no files were actually moved")
		}

		// Count organized files and errors
		organized := 0
		errorCount := 0
		for _, result := range results {
			if result.Error != nil {
				errorCount++
			} else if result.Moved {
				organized++
			}
		}

		// Print result summary
		if dryRun {
			PrintSuccess(fmt.Sprintf("Would organize %d of %d files", organized, fileCount))
		} else {
			PrintSuccess(fmt.Sprintf("Organized %d of %d files", organized, fileCount))
		}

		if errorCount > 0 {
			PrintWarning(fmt.Sprintf("%d errors occurred during organization", errorCount))
		}

		// Detailed results if requested
		if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
			PrintHeader("Detailed Results:")
			for _, result := range results {
				if result.Error != nil {
					PrintError(fmt.Sprintf("Error with %s: %v", result.SourcePath, result.Error))
				} else if result.Moved {
					PrintSuccess(fmt.Sprintf("Moved: %s -> %s", result.SourcePath, result.DestinationPath))
				} else if result.DestinationPath != "" {
					PrintInfo(fmt.Sprintf("Would move: %s -> %s", result.SourcePath, result.DestinationPath))
				}
			}
		}
	},
}

// WatchCmd represents the watch command
var WatchCmd = &cobra.Command{
	Use:   "watch [directories...]",
	Short: "Watch directories for changes and organize automatically",
	Long:  `Watch specified directories for file changes and organize them automatically according to rules.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get command flags
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		requireConfirm, _ := cmd.Flags().GetBool("confirm")

		// Get watch directories from arguments
		watchDirs := args

		// Validate configuration
		if cfg == nil {
			PrintError("Configuration not loaded")
			os.Exit(1)
		}

		// Validate watch directories
		if len(watchDirs) == 0 {
			PrintError("No directories specified to watch")
			PrintInfo("Specify directories as arguments or in your config file")
			os.Exit(1)
		}

		// Create the daemon
		daemon, err := watch.NewDaemon(cfg)
		if err != nil {
			PrintError(fmt.Sprintf("Failed to create daemon: %v", err))
			os.Exit(1)
		}

		// Set dry run mode if specified
		daemon.SetDryRun(dryRun)

		// Set confirmation requirement if specified
		daemon.SetRequireConfirmation(requireConfirm)

		// Add watch directories
		for _, dir := range watchDirs {
			if err := daemon.AddWatchDirectory(dir); err != nil {
				PrintError(fmt.Sprintf("Failed to add watch directory %s: %v", dir, err))
				os.Exit(1)
			}
		}

		// Start the daemon
		if err := daemon.Start(); err != nil {
			PrintError(fmt.Sprintf("Failed to start daemon: %v", err))
			os.Exit(1)
		}

		PrintInfo("Press Ctrl+C to stop watching")

		// Keep the process alive
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		<-ch

		// Stop the daemon
		daemon.Stop()
	},
}

// SetupCmd represents the setup command
var SetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up configuration for Sortd",
	Long:  `Interactive setup to configure Sortd preferences and rules.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Print header
		PrintHeader("🔧 Sortd Setup")

		// Load current configuration or create a new one
		if cfg == nil {
			cfg = config.New()
		}

		// Basic setup flow without Gum
		if !HasGum() {
			// Ask for default directories
			fmt.Println("Enter your default directory for organization:")
			fmt.Print("> ")
			var defaultDir string
			fmt.Scanln(&defaultDir)

			if defaultDir != "" {
				cfg.Directories.Default = defaultDir
			}

			// Set some basic settings
			fmt.Println("Enable dry run mode by default? (y/n)")
			fmt.Print("> ")
			var dryRunInput string
			fmt.Scanln(&dryRunInput)
			cfg.Settings.DryRun = dryRunInput == "y" || dryRunInput == "Y"

		} else {
			// Enhanced setup with Gum
			PrintInfo("Let's configure Sortd to match your preferences")

			// Default directory
			dirChoice := RunGumChoose("Enter path manually", "Browse directories")
			var defaultDir string

			if dirChoice == "Browse directories" {
				defaultDir = RunGumFile("--directory")
			} else {
				defaultDir = RunGumInput("Enter default directory to organize", "")
			}

			if defaultDir != "" {
				cfg.Directories.Default = defaultDir
				PrintSuccess(fmt.Sprintf("Default directory set to: %s", defaultDir))
			}

			// Settings
			PrintInfo("Configure basic settings")
			cfg.Settings.DryRun = RunGumConfirm("Enable dry run mode by default?")
			cfg.Settings.CreateDirs = RunGumConfirm("Create destination directories if they don't exist?")
			cfg.Settings.Backup = RunGumConfirm("Create backups before moving files?")

			// Collision strategy
			PrintInfo("Select a collision strategy")
			cfg.Settings.Collision = RunGumChoose("rename", "skip", "ask")

			// Watch directories
			if RunGumConfirm("Would you like to configure watch mode?") {
				cfg.WatchMode.Enabled = true

				// Configure watch interval
				intervalInput := RunGumInput("Check interval in seconds", "300")
				if interval, err := strconv.Atoi(intervalInput); err == nil && interval > 0 {
					// cfg.WatchMode.Interval = interval // REMOVED
				} else {
					// cfg.WatchMode.Interval = 300 // Default to 5 minutes // REMOVED
				}

				// Add watch directories
				PrintInfo("Add directories to watch")
				for {
					dirChoice = RunGumChoose("Enter path manually", "Browse directories", "Done adding directories")

					if dirChoice == "Done adding directories" {
						break
					}

					var watchDir string
					if dirChoice == "Browse directories" {
						watchDir = RunGumFile("--directory")
					} else {
						watchDir = RunGumInput("Directory to watch", cfg.Directories.Default)
					}

					if watchDir != "" {
						// Check if already in the list
						isDuplicate := false
						for _, dir := range cfg.Directories.Watch {
							if dir == watchDir {
								isDuplicate = true
								break
							}
						}

						if !isDuplicate {
							cfg.Directories.Watch = append(cfg.Directories.Watch, watchDir)
							PrintSuccess(fmt.Sprintf("Added %s to watch list", watchDir))
						} else {
							PrintWarning(fmt.Sprintf("%s is already in the watch list", watchDir))
						}
					}

					// Ask if they want to add more directories
					if len(cfg.Directories.Watch) > 0 && !RunGumConfirm("Add another directory to watch?") {
						break
					}
				}
			}

			// Organization rules
			if RunGumConfirm("Would you like to add organization rules?") {
				for {
					// Rule type
					ruleType := RunGumChoose("File pattern", "File extension")

					var pattern, target string

					// Get pattern
					if ruleType == "File pattern" {
						pattern = RunGumInput("File pattern (e.g., *.jpg)", "")
					} else {
						ext := RunGumInput("File extension (without dot, e.g., pdf)", "")
						if ext != "" {
							// Add the dot if needed
							if !strings.HasPrefix(ext, ".") {
								ext = "." + ext
							}
							pattern = "*" + ext
						}
					}

					// Get target
					if pattern != "" {
						dirChoice = RunGumChoose("Enter path manually", "Browse directories")

						if dirChoice == "Browse directories" {
							target = RunGumFile("--directory")
						} else {
							target = RunGumInput("Target directory", "")
						}

						// Add rule
						if pattern != "" && target != "" {
							cfg.Rules = append(cfg.Rules, struct {
								Pattern string `yaml:"pattern"`
								Target  string `yaml:"target"`
							}{
								Pattern: pattern,
								Target:  target,
							})

							PrintSuccess(fmt.Sprintf("Added rule: %s -> %s", pattern, target))
						}
					}

					// Check if we should add more rules
					if !RunGumConfirm("Add another rule?") {
						break
					}
				}
			}
		}

		// Save configuration
		err := cfg.Save()
		if err != nil {
			PrintError(fmt.Sprintf("Error saving configuration: %v", err))
			return
		}

		PrintSuccess("Configuration saved successfully!")
	},
}

// GUICmd represents the GUI command
var GUICmd = &cobra.Command{
	Use:   "gui",
	Short: "Launch the graphical user interface",
	Long:  `Launch Sortd in GUI mode with a full graphical interface.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get the path to the current executable
		execPath, err := os.Executable()
		if err != nil {
			PrintError(fmt.Sprintf("Error getting executable path: %v", err))
			return
		}

		// Prepare to launch with the --gui flag
		attr := &os.ProcAttr{
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
			Dir:   filepath.Dir(execPath),
		}

		// Launch the process with the --gui flag
		process, err := os.StartProcess(execPath, []string{execPath, "--gui"}, attr)
		if err != nil {
			PrintError(fmt.Sprintf("Error launching GUI: %v", err))
			return
		}

		// Detach the process
		if err := process.Release(); err != nil {
			PrintError(fmt.Sprintf("Error detaching GUI process: %v", err))
		} else {
			PrintSuccess("GUI launched successfully")
		}
	},
}

// ThemeCmd represents the theme command
var ThemeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Change or view the CLI theme",
	Long:  `Change or view the color theme for the CLI interface.`,
	Run: func(cmd *cobra.Command, args []string) {
		list, _ := cmd.Flags().GetBool("list")
		interactive, _ := cmd.Flags().GetBool("interactive")
		themeName, _ := cmd.Flags().GetString("name")

		if list {
			// List all available themes
			PrintHeader("Available Themes")
			for _, theme := range AvailableThemes {
				fmt.Printf("%s%s%s - %s\n",
					theme.Highlight,
					theme.Name,
					colorReset,
					theme.Description)
			}
			return
		}

		if interactive && HasGum() {
			// Use Gum to interactively select a theme
			PrintHeader("Theme Selection")

			var options []string
			for _, theme := range AvailableThemes {
				options = append(options, theme.Name)
			}

			selected := RunGumChoose(options...)
			if selected != "" {
				if SetTheme(selected) {
					PrintSuccess(fmt.Sprintf("Theme changed to '%s'", selected))

					// Save the theme preference
					if err := SaveThemePreference(); err != nil {
						PrintWarning(fmt.Sprintf("Failed to save theme preference: %v", err))
					}

					// Display theme demo
					showThemeDemo()
				} else {
					PrintError(fmt.Sprintf("Failed to set theme '%s'", selected))
				}
			} else {
				PrintWarning("Theme selection cancelled")
			}
			return
		}

		if themeName != "" {
			// Set theme by name
			if SetTheme(themeName) {
				PrintSuccess(fmt.Sprintf("Theme changed to '%s'", themeName))

				// Save the theme preference
				if err := SaveThemePreference(); err != nil {
					PrintWarning(fmt.Sprintf("Failed to save theme preference: %v", err))
				}

				// Display theme demo if not in non-interactive mode
				if interactive {
					showThemeDemo()
				}
			} else {
				PrintError(fmt.Sprintf("Unknown theme: '%s'", themeName))
				PrintInfo("Use 'sortd theme --list' to see available themes")
			}
			return
		}

		// No options provided, show current theme and help
		PrintHeader("Current Theme")
		fmt.Printf("Current theme: %s%s%s\n",
			CurrentTheme.Highlight,
			CurrentTheme.Name,
			colorReset)

		PrintInfo("Use 'sortd theme --list' to see available themes")
		PrintInfo("Use 'sortd theme --name=THEME' to change the theme")
		PrintInfo("Use 'sortd theme --interactive' to select a theme interactively")

		// Display theme demo
		showThemeDemo()
	},
}

// showThemeDemo displays a demo of all the theme colors
func showThemeDemo() {
	PrintHeader("Theme Demo")

	// Display all the styled text
	fmt.Println(CurrentTheme.Success + "This is Success text" + colorReset)
	fmt.Println(CurrentTheme.Error + "This is Error text" + colorReset)
	fmt.Println(CurrentTheme.Warning + "This is Warning text" + colorReset)
	fmt.Println(CurrentTheme.Info + "This is Info text" + colorReset)
	fmt.Println(CurrentTheme.Header + "This is Header text" + colorReset)
	fmt.Println(CurrentTheme.Highlight + "This is Highlight text" + colorReset)
	fmt.Println(CurrentTheme.Normal + "This is Normal text" + colorReset)

	// Display a box
	box := "This is a themed box\nIt uses the theme's box outline color"
	fmt.Println(DrawBoxWithTheme(box))

	// Display logo in theme color
	fmt.Println(DrawSortdLogo())
}

// Ensure ThemeCmd is initialized
func init() {
	ThemeCmd.Flags().BoolP("list", "l", false, "List all available themes")
	ThemeCmd.Flags().StringP("name", "n", "", "Set theme by name")
	ThemeCmd.Flags().BoolP("interactive", "i", false, "Select theme interactively")

	// Load saved theme preference
	LoadThemePreference()
}

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() error {
	// Add subcommands
	RootCmd.AddCommand(OrganizeCmd)
	RootCmd.AddCommand(WatchCmd)
	RootCmd.AddCommand(SetupCmd)
	RootCmd.AddCommand(GUICmd)
	RootCmd.AddCommand(ThemeCmd)

	// Add global flags
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.config/sortd/config.yaml)")

	// Add organize command flags
	OrganizeCmd.Flags().BoolP("dry-run", "d", false, "Simulate operations without making changes")
	OrganizeCmd.Flags().BoolP("recursive", "r", false, "Process directories recursively")
	OrganizeCmd.Flags().BoolP("interactive", "i", false, "Enable interactive mode for file selection")
	OrganizeCmd.Flags().BoolP("verbose", "v", false, "Show detailed information about each file")

	// Add watch command flags
	WatchCmd.Flags().BoolP("dry-run", "d", false, "Simulate operations without making changes")
	WatchCmd.Flags().BoolP("confirm", "c", false, "Require confirmation before organizing files")

	// Add setup command flags
	SetupCmd.Flags().BoolP("force", "f", false, "Overwrite existing configuration")

	return RootCmd.Execute()
}
