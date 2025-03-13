package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"sortd/internal/watch"

	"github.com/spf13/cobra"
)

// NewWatchCmd creates the watch command
func NewWatchCmd() *cobra.Command {
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

			// Create watcher
			watcher, err := watch.NewWatcher(cfg, validDirs, time.Duration(interval)*time.Second)
			if err != nil {
				fmt.Printf("❌ Error creating watcher: %v\n", err)
				return
			}

			// Format for display
			dirList := strings.Join(validDirs, ", ")

			// Start watching
			fmt.Printf("👀 Watching %s (checking every %d seconds)\n", dirList, interval)
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

			// Start the watcher
			go watcher.Start()

			// Setup signal catching for clean shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			// Wait for a signal
			<-sigChan

			fmt.Println("\n🛑 Stopping watcher...")
			watcher.Stop()
			fmt.Println("✅ Watch mode ended")
		},
	}

	return cmd
}
