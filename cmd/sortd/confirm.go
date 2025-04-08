package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// NewConfirmCmd creates the confirm command for interactive confirmations
func NewConfirmCmd() *cobra.Command {
	var (
		message  string
		title    string
		autoYes  bool
		autoNo   bool
		exitCode int
	)

	cmd := &cobra.Command{
		Use:   "confirm",
		Short: "Display an interactive confirmation prompt",
		Long: `Show a styled confirmation dialog to the user and exit with the appropriate code.
This is useful for scripting interactive workflows or building custom user interfaces.`,
		Run: func(cmd *cobra.Command, args []string) {
			// If message not specified but args provided, use the args
			if message == "" && len(args) > 0 {
				message = strings.Join(args, " ")
			}

			// Default message if none provided
			if message == "" {
				message = "Confirm this action?"
			}

			// Print the title if specified
			if title != "" {
				fmt.Println(primaryText(title))
			}

			// If auto-yes is set, skip the confirmation
			if autoYes {
				fmt.Println(successText("✓ " + message + " (auto-approved)"))
				os.Exit(0)
				return
			}

			// If auto-no is set, skip the confirmation
			if autoNo {
				fmt.Println(errorText("✗ " + message + " (auto-rejected)"))
				os.Exit(exitCode)
				return
			}

			// Check if gum is installed for interactive confirmation
			_, err := exec.LookPath("gum")
			if err != nil {
				// Fallback to basic confirmation if gum isn't available
				fmt.Println(message + " (y/n)")
				var response string
				fmt.Scanln(&response)
				response = strings.ToLower(strings.TrimSpace(response))

				if response == "y" || response == "yes" {
					os.Exit(0)
				} else {
					os.Exit(exitCode)
				}
				return
			}

			// Use gum for styled confirmation
			if runGumConfirm(message) {
				os.Exit(0)
			} else {
				os.Exit(exitCode)
			}
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&message, "message", "m", "", "The confirmation message to display")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Optional title to display above the message")
	cmd.Flags().BoolVarP(&autoYes, "yes", "y", false, "Automatically answer yes without prompting")
	cmd.Flags().BoolVarP(&autoNo, "no", "n", false, "Automatically answer no without prompting")
	cmd.Flags().IntVarP(&exitCode, "exit-code", "e", 1, "Exit code to use for 'no' responses")

	return cmd
}
