package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sortd/internal/watch"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// NewWatchCmd creates the watch command
func NewWatchCmd() *cobra.Command {
	var (
		directories     []string
		interval        int
		requireConfirm  bool
		confirmInterval int
		foreground      bool
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch directories for file changes",
		Long:  `Watch specified directories for changes and organize files automatically.`,
		Run: func(cmd *cobra.Command, args []string) {
			// If no directories specified, use config or interactive selection
			if len(directories) == 0 {
				if cfg != nil && len(cfg.Directories.Watch) > 0 {
					directories = cfg.Directories.Watch
					fmt.Println(infoText("Using watch directories from configuration"))
				} else {
					fmt.Println(infoText("No watch directories specified"))

					// Use interactive selection if not in test mode
					if os.Getenv("TESTMODE") != "true" {
						watchDir := runGumFile("--directory")
						if watchDir != "" {
							directories = append(directories, watchDir)
							fmt.Println(successText("Added directory: " + watchDir))
						}
					}

					if len(directories) == 0 {
						fmt.Println(errorText("No directories to watch. Please specify at least one directory."))
						return
					}
				}
			}

			// Create and configure the daemon
			daemon := watch.NewDaemon(cfg)

			// Set confirmation requirement
			daemon.SetRequireConfirmation(requireConfirm)

			// Set dry run from config
			if cfg.Settings.DryRun {
				daemon.SetDryRun(true)
				fmt.Println(infoText("Running in dry-run mode"))
			}

			// Add directories to watch
			for _, dir := range directories {
				// Clean and validate path
				dir = filepath.Clean(dir)
				info, err := os.Stat(dir)
				if err != nil || !info.IsDir() {
					fmt.Println(errorText(fmt.Sprintf("Invalid directory: %s", dir)))
					continue
				}

				if err := daemon.AddWatchDirectory(dir); err != nil {
					fmt.Println(errorText(fmt.Sprintf("Failed to add directory %s: %v", dir, err)))
				} else {
					fmt.Println(successText(fmt.Sprintf("Watching directory: %s", dir)))
				}
			}

			// Set callback for confirmations if required
			if requireConfirm {
				daemon.SetCallback(func(source, destination string, err error) {
					// If err is nil, this is a confirmation request
					if err == nil {
						// Get the executable path
						execPath, err := os.Executable()
						if err != nil {
							fmt.Printf("Error getting executable path: %v\n", err)
							return
						}

						// Call the confirm command as a subprocess
						fmt.Println(infoText(fmt.Sprintf("Requesting confirmation for: %s -> %s", source, destination)))

						// Format paths to handle spaces
						sourceArg := fmt.Sprintf("--source=%s", source)
						destArg := fmt.Sprintf("--destination=%s", destination)

						// Execute the confirmation command
						confirmCmd := exec.Command(execPath, "confirm", sourceArg, destArg)
						confirmCmd.Stdout = os.Stdout
						confirmCmd.Stderr = os.Stderr
						confirmCmd.Stdin = os.Stdin

						if err := confirmCmd.Run(); err != nil {
							fmt.Printf("Error running confirmation: %v\n", err)
						}
					} else {
						// This is a result notification
						if strings.Contains(err.Error(), "already exists") {
							fmt.Println(warningText(fmt.Sprintf("Skipped %s: destination already exists", source)))
						} else {
							fmt.Println(errorText(fmt.Sprintf("Failed to organize %s: %v", source, err)))
						}
					}
				})
			}

			// Start the daemon
			if err := daemon.Start(); err != nil {
				fmt.Println(errorText(fmt.Sprintf("Failed to start watch daemon: %v", err)))
				return
			}

			fmt.Println(successText("Watch daemon started"))

			// If running in foreground, wait for signal to exit
			if foreground {
				fmt.Println(infoText("Running in foreground mode. Press Ctrl+C to stop."))

				// Handle signals for graceful shutdown
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

				// Block until we receive a signal
				<-sigChan

				fmt.Println(infoText("\nStopping watch daemon..."))
				daemon.Stop()
				fmt.Println(successText("Watch daemon stopped"))
			} else {
				// Just start it and exit
				fmt.Println(infoText("Watch daemon running in background"))
				fmt.Println(infoText("Use 'sortd daemon stop' to stop the daemon"))
			}
		},
	}

	// Add flags
	cmd.Flags().StringSliceVarP(&directories, "dir", "d", []string{}, "Directories to watch (can specify multiple times)")
	cmd.Flags().IntVarP(&interval, "interval", "i", 300, "Check interval in seconds")
	cmd.Flags().BoolVarP(&requireConfirm, "require-confirmation", "c", false, "Require confirmation for file operations")
	cmd.Flags().IntVar(&confirmInterval, "confirmation-period", 60, "Period in seconds for batch confirmations")
	cmd.Flags().BoolVarP(&foreground, "foreground", "f", false, "Run in foreground (don't daemonize)")

	return cmd
}
