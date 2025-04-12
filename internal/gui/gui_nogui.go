//go:build nogui
// +build nogui

package gui

import (
	"fmt"
)

// StartGUI is a stub implementation for builds with GUI disabled
func StartGUI() error {
	fmt.Println("GUI is disabled in this build. Please use the CLI interface.")
	return fmt.Errorf("GUI not available in this build")
}

// IsGUIAvailable returns whether the GUI is available in this build
func IsGUIAvailable() bool {
	return false
}

// ShowMessage shows a message in the GUI (stub implementation)
func ShowMessage(title, message string) {
	fmt.Printf("[%s] %s\n", title, message)
}

// Any other GUI functions should have stub implementations here
