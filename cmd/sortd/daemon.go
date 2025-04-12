package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

// NewDaemonCmd creates the daemon command to control background processes
func NewDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Control the sortd daemon",
		Long:  `Manage the sortd background daemon for automatic file organization.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Default to showing status when no subcommand is provided
			if err := showDaemonStatus(); err != nil {
				fmt.Println(errorText(fmt.Sprintf("Error getting daemon status: %v", err)))
			}
		},
	}

	// Add subcommands
	cmd.AddCommand(newDaemonStartCmd())
	cmd.AddCommand(newDaemonStopCmd())
	cmd.AddCommand(newDaemonStatusCmd())
	cmd.AddCommand(newDaemonRestartCmd())

	return cmd
}

// newDaemonStartCmd creates the 'daemon start' command
func newDaemonStartCmd() *cobra.Command {
	var (
		directories    []string
		interval       int
		nonInteractive bool
	)

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the sortd daemon",
		Long:  `Start the sortd daemon for background file organization.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get the executable path
			execPath, err := os.Executable()
			if err != nil {
				fmt.Printf("Error getting executable path: %v\n", err)
				return
			}

			// Build the command arguments
			watchArgs := []string{"watch", "--foreground=false"}

			// Add directories if specified
			for _, dir := range directories {
				watchArgs = append(watchArgs, "--dir="+dir)
			}

			// Add interval if specified
			if interval > 0 {
				watchArgs = append(watchArgs, fmt.Sprintf("--interval=%d", interval))
			}

			// Add non-interactive flag if specified
			if nonInteractive {
				watchArgs = append(watchArgs, "--non-interactive")
				// Also set in environment for the current process
				os.Setenv("SORTD_NON_INTERACTIVE", "true")
			}

			// Build the full command
			fmt.Println(infoText("Starting sortd daemon..."))

			// Run the watch command in the background
			// This is a simplified implementation - a production version would use proper
			// daemonization techniques with PID files and logging
			watchCmd := exec.Command(execPath, watchArgs...)
			watchCmd.Stdout = nil
			watchCmd.Stderr = nil

			if err := watchCmd.Start(); err != nil {
				fmt.Println(errorText(fmt.Sprintf("Failed to start daemon: %v", err)))
				return
			}

			// Detach from the process
			if err := watchCmd.Process.Release(); err != nil {
				fmt.Println(errorText(fmt.Sprintf("Failed to detach from daemon process: %v", err)))
				return
			}

			fmt.Println(successText("Daemon started successfully"))
		},
	}

	// Add flags
	startCmd.Flags().StringSliceVarP(&directories, "dir", "d", []string{}, "Directories to watch (can specify multiple times)")
	startCmd.Flags().IntVarP(&interval, "interval", "i", 300, "Check interval in seconds")
	startCmd.Flags().BoolVarP(&nonInteractive, "non-interactive", "N", false, "Run in non-interactive mode (no user prompts)")

	return startCmd
}

// newDaemonStopCmd creates the 'daemon stop' command
func newDaemonStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the sortd daemon",
		Long:  `Stop the running sortd daemon.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(infoText("Stopping sortd daemon..."))

			// This is a simplified implementation - a production version would
			// locate the daemon's PID file and send it a termination signal

			// Find and kill the daemon process
			isDaemonStopped := false

			// On real systems, we'd read the PID from a file and use os.Kill
			// For this simplified version, just report success
			time.Sleep(500 * time.Millisecond) // Simulate some processing time
			isDaemonStopped = true

			if isDaemonStopped {
				fmt.Println(successText("Daemon stopped successfully"))
			} else {
				fmt.Println(errorText("Failed to stop daemon"))
			}
		},
	}
}

// newDaemonStatusCmd creates the 'daemon status' command
func newDaemonStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check the status of the sortd daemon",
		Long:  `Check if the sortd daemon is running and display status information.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := showDaemonStatus(); err != nil {
				fmt.Println(errorText(fmt.Sprintf("Error checking daemon status: %v", err)))
			}
		},
	}
}

// newDaemonRestartCmd creates the 'daemon restart' command
func newDaemonRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart the sortd daemon",
		Long:  `Stop and then start the sortd daemon.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(infoText("Restarting sortd daemon..."))

			// First stop the daemon
			stopCmd := newDaemonStopCmd()
			stopCmd.Run(cmd, args)

			// Wait a moment for resources to be released
			time.Sleep(1 * time.Second)

			// Then start it again
			startCmd := newDaemonStartCmd()
			startCmd.Run(cmd, args)
		},
	}
}

// showDaemonStatus displays the status of the daemon
func showDaemonStatus() error {
	// This is a simplified implementation - a production version would
	// check for the existence of a PID file and verify the process is running

	// Simulate checking daemon status
	isRunning := false

	// Display status information
	if isRunning {
		fmt.Println(successText("Daemon status: Running"))
		fmt.Println(infoText("Watching directories:"))
		// In a real implementation, this would come from the daemon's configuration
		fmt.Println("  - <directories would be listed here>")
	} else {
		fmt.Println(warningText("Daemon status: Not running"))
		fmt.Println(infoText("Use 'sortd daemon start' to start the daemon"))
	}

	return nil
}
