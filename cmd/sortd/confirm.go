package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sortd/internal/organize"
	"strings"

	"github.com/spf13/cobra"
)

// NewConfirmCmd creates the confirm command for confirming file operations
func NewConfirmCmd() *cobra.Command {
	var (
		sourcePath      string
		destinationPath string
		action          string
	)

	cmd := &cobra.Command{
		Use:   "confirm",
		Short: "Confirm a pending file operation",
		Long:  `Confirm a pending file operation from the watch daemon.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Validate required parameters
			if sourcePath == "" || destinationPath == "" {
				fmt.Println(errorText("Missing required parameters: source and destination paths"))
				return
			}

			// Display operation for confirmation
			fmt.Println(primaryText("Confirm File Operation"))
			fmt.Println("--------------------------------")
			fmt.Printf("Action: %s\n", emphasisText(action))
			fmt.Printf("Source: %s\n", infoText(sourcePath))
			fmt.Printf("Destination: %s\n", infoText(destinationPath))
			fmt.Println("--------------------------------")

			// Get confirmation
			confirmed := runGumConfirm("Confirm this operation?")
			if !confirmed {
				fmt.Println(warningText("Operation cancelled"))
				return
			}

			// Perform the operation based on the action type
			if strings.ToLower(action) == "move" {
				// Create destination directory if it doesn't exist
				destDir := filepath.Dir(destinationPath)
				if err := os.MkdirAll(destDir, 0755); err != nil {
					fmt.Println(errorText(fmt.Sprintf("Error creating destination directory: %v", err)))
					return
				}

				// Handle the file operation
				if cfg.Settings.DryRun {
					fmt.Printf("Would move %s -> %s\n", sourcePath, destinationPath)
					fmt.Println(successText("Dry run completed successfully"))
				} else {
					// Create the organize engine and perform the operation
					organizeEngine := organize.NewWithConfig(cfg)
					err := organizeEngine.MoveFile(sourcePath, destinationPath)
					if err != nil {
						fmt.Println(errorText(fmt.Sprintf("Error moving file: %v", err)))
						return
					}
					fmt.Println(successText("File moved successfully"))
				}
			} else {
				fmt.Println(errorText(fmt.Sprintf("Unsupported action: %s", action)))
			}
		},
	}

	// Add required flags
	cmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Source file path (required)")
	cmd.Flags().StringVarP(&destinationPath, "destination", "d", "", "Destination file path (required)")
	cmd.Flags().StringVarP(&action, "action", "a", "move", "Action to perform (move, copy, etc.)")

	cmd.MarkFlagRequired("source")
	cmd.MarkFlagRequired("destination")

	return cmd
}
