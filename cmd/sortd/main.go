package main

import (
	"fmt"
	"os"
	"sortd/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Determine if we're running in test mode
	testMode := os.Getenv("TESTMODE") == "true"

	var p *tea.Program
	if testMode {
		// Run in test mode without alt screen
		p = tea.NewProgram(tui.New(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	} else {
		p = tea.NewProgram(tui.New(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	}

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
