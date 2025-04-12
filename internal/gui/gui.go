//go:build !nogui
// +build !nogui

package gui

import (
	"fmt"
)

// IsGUIAvailable returns whether the GUI is available in this build
func IsGUIAvailable() bool {
	return true
}

// Note: For actual implementation, you would add your real GUI code here
// This is just a placeholder assuming you already have real implementations elsewhere

// StartGUI starts the GUI interface
func StartGUI() error {
	// Your actual GUI initialization code would go here
	// This is just a placeholder
	fmt.Println("Starting GUI interface...")
	return nil
}

// ShowMessage shows a message in the GUI
func ShowMessage(title, message string) {
	// Your actual GUI message display code would go here
	fmt.Printf("[%s] %s\n", title, message)
}

// Any other GUI functions would be implemented here
