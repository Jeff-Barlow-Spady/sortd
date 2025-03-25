package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"sortd/internal/config"
	"sortd/internal/watch"

	"github.com/spf13/cobra"
)

// NewWatchCmd creates the watch command
func NewWatchCmd() *cobra.Command {
	var (
		dryRun              bool
		recursive           bool
		timeout             int
		confirmationPeriod  int
		requireConfirmation bool
	)

	cmd := &cobra.Command{
		Use:   "watch [directories...]",
		Short: "Watch directories for changes and organize files automatically",
		Long: `Start watching one or more directories for new files and organize them automatically
according to your organization rules.`,
		Run: func(cmd *cobra.Command, args []string) {
			var dirsToWatch []string

			// Determine which directories to watch
			if len(args) > 0 {
				dirsToWatch = args
			} else if cfg != nil && len(cfg.Directories.Watch) > 0 {
				dirsToWatch = cfg.Directories.Watch
			} else if cfg != nil && cfg.Directories.Default != "" {
				dirsToWatch = []string{cfg.Directories.Default}
			} else {
				// If in test mode, use current directory
				if os.Getenv("TESTMODE") == "true" {
					wd, _ := os.Getwd()
					dirsToWatch = []string{wd}
				} else {
					// Use Gum to let the user choose directories
					fmt.Println("📂 Choose directories to watch:")

					if _, err := exec.LookPath("gum"); err == nil {
						// Let user input comma-separated list of directories or browse
						choiceMethod := runGumChoose("Enter manually", "Browse for directory")

						if choiceMethod == "Enter manually" {
							dirs := runGumInput("Directories to watch (comma separated)", "")
							for _, dir := range strings.Split(dirs, ",") {
								dir = strings.TrimSpace(dir)
								if dir != "" {
									dirsToWatch = append(dirsToWatch, dir)
								}
							}
						} else {
							dir := runGumFile("--directory")
							if dir != "" {
								dirsToWatch = []string{dir}
							}
						}
					} else {
						// Fallback for no gum
						fmt.Println("Enter directories to watch (comma separated):")
						var dirs string
						fmt.Scanln(&dirs)
						for _, dir := range strings.Split(dirs, ",") {
							dir = strings.TrimSpace(dir)
							if dir != "" {
								dirsToWatch = append(dirsToWatch, dir)
							}
						}
					}
				}
			}

			if len(dirsToWatch) == 0 {
				fmt.Println("❌ No directories specified to watch")
				return
			}

			// Validate directories
			var validDirs []string
			for _, dir := range dirsToWatch {
				info, err := os.Stat(dir)
				if err != nil {
					fmt.Printf("⚠️ Warning: Cannot access %s: %v\n", dir, err)
					continue
				}
				if !info.IsDir() {
					fmt.Printf("⚠️ Warning: %s is not a directory\n", dir)
					continue
				}
				validDirs = append(validDirs, dir)
			}

			if len(validDirs) == 0 {
				fmt.Println("❌ No valid directories to watch")
				return
			}

			// Determine interval
			interval := 300 // Default 5 minutes
			if cfg != nil && cfg.WatchMode.Interval > 0 {
				interval = cfg.WatchMode.Interval
			}

			// Apply command line flags to config
			if cfg == nil {
				cfg = config.New()
			}

			// Apply confirmation settings from command line flags
			if confirmationPeriod > 0 {
				cfg.WatchMode.ConfirmationPeriod = confirmationPeriod
			}

			if requireConfirmation {
				cfg.WatchMode.RequireConfirmation = true
			}

			// Create watcher
			watcher, err := watch.NewWatcher(cfg, validDirs, time.Duration(interval)*time.Second)
			if err != nil {
				fmt.Printf("❌ Error creating watcher: %v\n", err)
				return
			}

			// Apply command line flags
			if dryRun {
				fmt.Println("🔍 Dry run mode: files will not be moved")
				fmt.Println("Running in dry mode - no changes will be made")
				if cfg != nil {
					cfg.Settings.DryRun = true
				}
			}

			if recursive {
				fmt.Println("🔍 Watching directories recursively")
				watcher.SetRecursive(true)
			}

			// Format for display
			dirList := strings.Join(validDirs, ", ")

			// Start watching
			fmt.Printf("👀 Watching %s (checking every %d seconds)\n", dirList, interval)

			// Show confirmation settings if enabled
			if cfg.WatchMode.ConfirmationPeriod > 0 {
				fmt.Printf("⏳ Files will be processed after a %d second confirmation period\n",
					cfg.WatchMode.ConfirmationPeriod)
			}
			if cfg.WatchMode.RequireConfirmation {
				fmt.Println("✋ Files will require manual confirmation with 'sortd confirm'")
			}

			fmt.Println("Press Ctrl+C to stop")

			if _, err := exec.LookPath("gum"); err == nil {
				// Use gum style for a nice watching indicator
				runGum("style",
					"--foreground", "212",
					"--border", "rounded",
					"--border-foreground", "212",
					"--padding", "1",
					"--align", "center",
					fmt.Sprintf("Watching %d directories\n\n%s", len(validDirs), dirList))
			}

			// Create the PID file for daemon/confirmation communication
			if err := createPIDFile(); err != nil {
				fmt.Printf("⚠️ Warning: Failed to create PID file: %v\n", err)
			}

			// Start a background goroutine to check for confirmation files
			confirmChan := make(chan string)
			go monitorConfirmationFiles(confirmChan)

			// Start the watcher
			go watcher.Start()

			// Setup signal catching for clean shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			// Set timeout if specified
			if timeout > 0 {
				fmt.Printf("⏱️ Watcher will run for %d seconds\n", timeout)
				go func() {
					time.Sleep(time.Duration(timeout) * time.Second)
					sigChan <- syscall.SIGINT
				}()
			}

			// Start a goroutine to periodically write pending files to a file for the confirm command
			go func() {
				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						// Get pending files from watcher
						pendingFiles := watcher.GetPendingFiles()
						if len(pendingFiles) > 0 {
							// Write them to a file for the confirm command to read
							pid := os.Getpid()
							pendingFilesPath := filepath.Join(os.TempDir(), fmt.Sprintf("sortd-pending-%d.txt", pid))
							// Join the files with newlines
							pendingData := strings.Join(pendingFiles, "\n")
							if err := os.WriteFile(pendingFilesPath, []byte(pendingData), 0644); err != nil {
								fmt.Printf("⚠️ Warning: Failed to write pending files: %v\n", err)
							}
						}

					case file := <-confirmChan:
						if file == "all" {
							count := watcher.ConfirmAllFiles()
							fmt.Printf("✅ Confirmed all %d pending files\n", count)
						} else if file != "" {
							if watcher.ConfirmFile(file) {
								fmt.Printf("✅ Confirmed file: %s\n", file)
							}
						}
					}
				}
			}()

			// Wait for a signal
			<-sigChan

			fmt.Println("\n🛑 Stopping watcher...")
			watcher.Stop()
			fmt.Println("✅ Watch mode ended")

			// Clean up the PID file and any pending files
			cleanupPIDFile()
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate watching without moving files")
	cmd.Flags().BoolVar(&recursive, "recursive", false, "Watch directories recursively")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Stop watching after specified seconds")
	cmd.Flags().IntVar(&confirmationPeriod, "confirmation-period", 0, "Wait this many seconds before processing files (0 = disabled)")
	cmd.Flags().BoolVar(&requireConfirmation, "require-confirmation", false, "Require manual confirmation before processing files")

	return cmd
}

// createPIDFile creates a PID file to facilitate communication with the confirm command
func createPIDFile() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".config", "sortd")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	pidFile := filepath.Join(configDir, "sortd.pid")
	return os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
}

// cleanupPIDFile removes the PID file and any pending files
func cleanupPIDFile() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	pidFile := filepath.Join(home, ".config", "sortd", "sortd.pid")
	os.Remove(pidFile)

	// Clean up temp files
	pid := os.Getpid()
	pendingFilesPath := filepath.Join(os.TempDir(), fmt.Sprintf("sortd-pending-%d.txt", pid))
	os.Remove(pendingFilesPath)

	confirmPath := filepath.Join(os.TempDir(), fmt.Sprintf("sortd-confirm-%d.txt", pid))
	os.Remove(confirmPath)
}

// monitorConfirmationFiles watches for files that indicate manual confirmation
func monitorConfirmationFiles(confirmChan chan<- string) {
	pid := os.Getpid()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Check for all-files confirmation
		confirmPath := filepath.Join(os.TempDir(), fmt.Sprintf("sortd-confirm-%d.txt", pid))
		if _, err := os.Stat(confirmPath); err == nil {
			data, err := os.ReadFile(confirmPath)
			if err == nil {
				lines := strings.Split(string(data), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" {
						confirmChan <- line
					}
				}
			}
			os.Remove(confirmPath)
			confirmChan <- "all"
		}

		// Check for single-file confirmation
		singleConfirmPath := filepath.Join(os.TempDir(), fmt.Sprintf("sortd-confirm-file-%d.txt", pid))
		if _, err := os.Stat(singleConfirmPath); err == nil {
			data, err := os.ReadFile(singleConfirmPath)
			if err == nil {
				file := strings.TrimSpace(string(data))
				if file != "" {
					confirmChan <- file
				}
			}
			os.Remove(singleConfirmPath)
		}
	}
}
