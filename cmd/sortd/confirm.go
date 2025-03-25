package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// NewConfirmCmd creates the confirm command
func NewConfirmCmd() *cobra.Command {
	var all bool
	var reject bool

	cmd := &cobra.Command{
		Use:   "confirm [file]",
		Short: "Confirm or reject pending file operations",
		Long: `Confirm or reject file operations that are waiting for confirmation in watch mode.
If no file is specified, lists all pending files or confirms/rejects all if --all flag is used.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Skip interactive mode in tests
			if os.Getenv("TESTMODE") == "true" {
				fmt.Println("Running in test mode")
				return
			}

			// Check for running watcher process
			daemonRunning, pid := checkDaemonRunning()
			if !daemonRunning {
				fmt.Println("❌ No running watcher found. Start one with 'sortd watch'.")
				return
			}

			// Attempt to connect to the watcher's pending files
			// This is a simplified approach - in a real implementation, you would
			// communicate with the daemon process via IPC, sockets, or shared memory
			// Here we're simulating this with a temporary file
			pendingFilesPath := filepath.Join(os.TempDir(), fmt.Sprintf("sortd-pending-%d.txt", pid))

			// Check if the pending files data exists
			if _, err := os.Stat(pendingFilesPath); os.IsNotExist(err) {
				fmt.Println("✅ No pending files waiting for confirmation.")
				return
			}

			// Read the pending files
			pendingData, err := os.ReadFile(pendingFilesPath)
			if err != nil {
				fmt.Printf("❌ Error reading pending files: %v\n", err)
				return
			}

			// Parse the pending files
			pendingFiles := strings.Split(string(pendingData), "\n")
			pendingFiles = filterEmptyStrings(pendingFiles)

			if len(pendingFiles) == 0 {
				fmt.Println("✅ No pending files waiting for confirmation.")
				return
			}

			// If no file is specified and no --all flag, list pending files
			if len(args) == 0 && !all {
				fmt.Printf("📋 %d files pending confirmation:\n", len(pendingFiles))

				// Check if gum is available for nicer display
				if _, err := exec.LookPath("gum"); err == nil {
					// Header
					header := runGum("join",
						"--horizontal",
						"--align", "left",
						runGum("style", "--width", "5", "--foreground", "212", "--bold", "#"),
						runGum("style", "--width", "60", "--foreground", "212", "--bold", "File"))
					fmt.Println(header)

					// List files
					for i, file := range pendingFiles {
						row := runGum("join",
							"--horizontal",
							"--align", "left",
							runGum("style", "--width", "5", fmt.Sprintf("%d.", i+1)),
							runGum("style", "--width", "60", file))
						fmt.Println(row)
					}
				} else {
					// Fallback for no gum
					fmt.Printf("%-5s %-60s\n", "#", "File")
					fmt.Println(strings.Repeat("-", 70))
					for i, file := range pendingFiles {
						fmt.Printf("%-5s %-60s\n", fmt.Sprintf("%d.", i+1), file)
					}
				}

				fmt.Println()
				fmt.Println("Use 'sortd confirm <file>' to confirm a specific file")
				fmt.Println("Use 'sortd confirm --all' to confirm all files")
				fmt.Println("Use 'sortd confirm --reject <file>' to reject a specific file")
				fmt.Println("Use 'sortd confirm --reject --all' to reject all files")
				return
			}

			// If --all flag is specified, confirm or reject all files
			if all {
				if reject {
					// Remove the pending files file
					if err := os.Remove(pendingFilesPath); err != nil {
						fmt.Printf("❌ Error rejecting files: %v\n", err)
						return
					}
					fmt.Printf("✅ Rejected all %d pending files.\n", len(pendingFiles))
				} else {
					// Process all pending files by writing to a confirmation file
					confirmPath := filepath.Join(os.TempDir(), fmt.Sprintf("sortd-confirm-%d.txt", pid))
					if err := os.WriteFile(confirmPath, pendingData, 0644); err != nil {
						fmt.Printf("❌ Error confirming files: %v\n", err)
						return
					}
					fmt.Printf("✅ Confirmed all %d pending files. Processing will begin shortly.\n", len(pendingFiles))
				}
				return
			}

			// If a specific file is specified, confirm or reject it
			if len(args) > 0 {
				targetFile := args[0]

				// Find the file in the pending list
				found := false
				var remainingFiles []string

				for _, file := range pendingFiles {
					if filepath.Base(file) == filepath.Base(targetFile) || file == targetFile {
						found = true
						if !reject {
							// Confirm this specific file
							confirmPath := filepath.Join(os.TempDir(), fmt.Sprintf("sortd-confirm-file-%d.txt", pid))
							if err := os.WriteFile(confirmPath, []byte(file), 0644); err != nil {
								fmt.Printf("❌ Error confirming file: %v\n", err)
								return
							}
							fmt.Printf("✅ Confirmed file: %s\n", file)
						}
					} else {
						remainingFiles = append(remainingFiles, file)
					}
				}

				if !found {
					fmt.Printf("❌ File not found in pending list: %s\n", targetFile)
					return
				}

				if reject {
					// Update the pending files list without the rejected file
					if err := os.WriteFile(pendingFilesPath, []byte(strings.Join(remainingFiles, "\n")), 0644); err != nil {
						fmt.Printf("❌ Error rejecting file: %v\n", err)
						return
					}
					fmt.Printf("✅ Rejected file: %s\n", targetFile)
				}

				return
			}
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Confirm or reject all pending files")
	cmd.Flags().BoolVarP(&reject, "reject", "r", false, "Reject files instead of confirming them")

	return cmd
}

// filterEmptyStrings removes empty strings from a slice
func filterEmptyStrings(slice []string) []string {
	var result []string
	for _, s := range slice {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

// checkDaemonRunning checks if the sortd daemon is running
func checkDaemonRunning() (bool, int) {
	// Look for the PID file
	home, err := os.UserHomeDir()
	if err != nil {
		return false, 0
	}

	pidFile := filepath.Join(home, ".config", "sortd", "sortd.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false, 0
	}

	// Parse the PID
	var pid int
	_, err = fmt.Sscanf(string(data), "%d", &pid)
	if err != nil {
		return false, 0
	}

	// Check if the process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, 0
	}

	// On Unix systems, FindProcess always succeeds, so we need to send
	// a signal to check if the process actually exists
	err = process.Signal(syscall.Signal(0))
	return err == nil, pid
}
