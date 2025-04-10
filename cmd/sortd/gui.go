package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// GUICmd launches the graphical user interface
func GUICmd() {
	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting executable path: %v\n", err)
		return
	}

	// Create a command to launch sortd with the GUI flag
	guiCmd := exec.Command(execPath, "--gui")
	guiCmd.Stdout = os.Stdout
	guiCmd.Stderr = os.Stderr
	guiCmd.Stdin = os.Stdin

	// Run the command
	if err := guiCmd.Run(); err != nil {
		fmt.Printf("Error launching GUI: %v\n", err)
		return
	}
}

// NewGUICmd creates the GUI command for the CLI
func NewGUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gui",
		Short: "Launch the graphical user interface",
		Long:  `Launch the GUI version of sortd for a more visual file organization experience.`,
		Run: func(cmd *cobra.Command, args []string) {
			GUICmd()
		},
	}
}
