package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sortd/internal/watch"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// NewWatchCmd creates the watch command
func NewWatchCmd() *cobra.Command {
	var (
		requireConfirm  bool
		confirmInterval int
		foreground      bool
		background      bool
		nonInteractive  bool
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch directories for file changes",
		Long:  `Watch specified directories for changes and organize files automatically based on configuration.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Set non-interactive mode in environment for consistent access across functions
			if nonInteractive {
				os.Setenv("SORTD_NON_INTERACTIVE", "true")
			}

			// 1. Ensure configuration is loaded (should be done by root PersistentPreRun)
			if cfg == nil {
				fmt.Println(errorText("Configuration not loaded. Cannot start watch command."))
				fmt.Println(infoText("Try running 'sortd setup' or specify a valid config with --config"))
				return
			}

			// 2. Check if watch directories are configured
			if len(cfg.WatchDirectories) == 0 {
				fmt.Println(errorText("No watch directories specified in the configuration file."))
				fmt.Println(infoText("Please add directories under 'watch_directories:' in your config."))
				return
			}
			fmt.Println(infoText("Using watch directories from configuration:"))
			for _, dir := range cfg.WatchDirectories {
				fmt.Printf("  - %s\n", dir)
			}

			// Create the watch daemon - Pass only config, returns (*Daemon, error)
			daemon, err := watch.NewDaemon(cfg)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}

			// Set confirmation requirement
			daemon.SetRequireConfirmation(requireConfirm)

			// Set dry run from config
			if cfg.Settings.DryRun {
				daemon.SetDryRun(true)
				fmt.Println(infoText("Running in dry-run mode"))
			}

			// Set non-interactive mode if specified
			if nonInteractive {
				cfg.Settings.NonInteractive = true
				if requireConfirm {
					fmt.Println(warningText("Warning: non-interactive mode will override confirmation requirements"))
					requireConfirm = false
					daemon.SetRequireConfirmation(false)
				}
			}

			// Set callback for confirmations if required
			if requireConfirm {
				daemon.SetCallback(func(source, destination string, err error) {
					if err == nil {
						execPath, err := os.Executable()
						if err != nil {
							fmt.Printf("Error getting executable path: %v\n", err)
							return
						}

						fmt.Println(infoText(fmt.Sprintf("Requesting confirmation for: %s -> %s", source, destination)))

						sourceArg := fmt.Sprintf("--source=%s", source)
						destArg := fmt.Sprintf("--destination=%s", destination)

						confirmCmd := exec.Command(execPath, "confirm", sourceArg, destArg)
						confirmCmd.Stdout = os.Stdout
						confirmCmd.Stderr = os.Stderr
						confirmCmd.Stdin = os.Stdin

						if err := confirmCmd.Run(); err != nil {
							fmt.Printf("Error running confirmation: %v\n", err)
						}
					} else {
						if strings.Contains(err.Error(), "already exists") {
							fmt.Println(warningText(fmt.Sprintf("Skipped %s: destination already exists", source)))
						} else {
							fmt.Println(errorText(fmt.Sprintf("Failed to organize %s: %v", source, err)))
						}
					}
				})
			}

			// Handle background mode
			if background {
				fmt.Println("Starting watch daemon in background...")
				if err := watch.DaemonControl(cfg, true); err != nil {
					fmt.Printf("Error: %v\n", err)
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
			if err := watch.DaemonControl(cfg, false); err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}

			// 4. Handle foreground/background
			if foreground {
				fmt.Println(infoText("Running in foreground mode. Press Ctrl+C to stop."))

				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

				<-sigChan

				fmt.Println(infoText("\nStopping watch daemon..."))
				daemon.Stop()
				fmt.Println(successText("Watch daemon stopped"))
			} else {
				fmt.Println(infoText("Watch daemon running in background"))
				fmt.Println(infoText("Use 'sortd daemon stop' to stop the daemon"))
			}
		},
	}

	cmd.Flags().BoolVarP(&requireConfirm, "require-confirmation", "c", false, "Require confirmation for file operations")
	cmd.Flags().IntVar(&confirmInterval, "confirmation-period", 60, "Period in seconds for batch confirmations")
	cmd.Flags().BoolVarP(&foreground, "foreground", "f", false, "Run in foreground (don't daemonize)")
	cmd.Flags().BoolVarP(&background, "background", "b", false, "Run in background (daemonize)")
	cmd.Flags().BoolVarP(&nonInteractive, "non-interactive", "N", false, "Run in non-interactive mode (no user prompts)")

	return cmd
}
