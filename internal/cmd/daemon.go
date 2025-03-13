package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the sortd background service",
	Long:  `Start, stop, or check the status of the sortd background service.`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the sortd daemon",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if daemon is already running
		if isDaemonRunning() {
			fmt.Println("🚫 Daemon is already running")
			return
		}

		// Get the path to the daemon executable
		execPath, err := os.Executable()
		if err != nil {
			fmt.Printf("❌ Error getting executable path: %v\n", err)
			return
		}

		// The daemon executable should be in the same directory
		daemonPath := filepath.Join(filepath.Dir(execPath), "daemon")
		if _, err := os.Stat(daemonPath); os.IsNotExist(err) {
			fmt.Println("❌ Daemon executable not found")
			fmt.Printf("Expected at: %s\n", daemonPath)
			return
		}

		// Start the daemon
		fmt.Println("🚀 Starting sortd daemon...")

		// Use gum for a nice animation if available
		if _, err := exec.LookPath("gum"); err == nil {
			runGum("spin", "--spinner", "dot", "--title", "Starting daemon...", "--", daemonPath)
		} else {
			// Start the daemon in the background
			cmd := exec.Command(daemonPath)
			cmd.Stdout = nil
			cmd.Stderr = nil
			if err := cmd.Start(); err != nil {
				fmt.Printf("❌ Error starting daemon: %v\n", err)
				return
			}
		}

		// Check if it's running
		if isDaemonRunning() {
			fmt.Println("✅ Daemon started successfully")
		} else {
			fmt.Println("❌ Failed to start daemon")
		}
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the sortd daemon",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if daemon is running
		if !isDaemonRunning() {
			fmt.Println("🚫 Daemon is not running")
			return
		}

		// Find the PID of the daemon
		pid, err := getDaemonPID()
		if err != nil {
			fmt.Printf("❌ Error finding daemon process: %v\n", err)
			return
		}

		// Stop the daemon
		fmt.Println("🛑 Stopping sortd daemon...")

		// Use gum for a nice animation if available
		if _, err := exec.LookPath("gum"); err == nil {
			runGum("spin", "--spinner", "dot", "--title", "Stopping daemon...", "--", "kill", pid)
		} else {
			// Stop the daemon
			stopCmd := exec.Command("kill", pid)
			if err := stopCmd.Run(); err != nil {
				fmt.Printf("❌ Error stopping daemon: %v\n", err)
				return
			}
		}

		// Check if it's stopped
		if !isDaemonRunning() {
			fmt.Println("✅ Daemon stopped successfully")
		} else {
			fmt.Println("❌ Failed to stop daemon")
		}
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of the sortd daemon",
	Run: func(cmd *cobra.Command, args []string) {
		if isDaemonRunning() {
			pid, _ := getDaemonPID()

			// Use gum for a nice status display if available
			if _, err := exec.LookPath("gum"); err == nil {
				runGum("style",
					"--foreground", "212",
					"--border", "rounded",
					"--border-foreground", "212",
					"--padding", "1",
					"--align", "center",
					fmt.Sprintf("✅ Daemon is running (PID: %s)", pid))
			} else {
				fmt.Printf("✅ Daemon is running (PID: %s)\n", pid)
			}
		} else {
			// Use gum for a nice status display if available
			if _, err := exec.LookPath("gum"); err == nil {
				runGum("style",
					"--foreground", "212",
					"--border", "rounded",
					"--border-foreground", "212",
					"--padding", "1",
					"--align", "center",
					"❌ Daemon is not running")
			} else {
				fmt.Println("❌ Daemon is not running")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
}

// Helper functions for daemon management

func isDaemonRunning() bool {
	// Check if the daemon process is running
	cmd := exec.Command("pgrep", "-f", "sortd/daemon")
	output, err := cmd.Output()
	return err == nil && len(output) > 0
}

func getDaemonPID() (string, error) {
	// Get the PID of the daemon process
	cmd := exec.Command("pgrep", "-f", "sortd/daemon")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
