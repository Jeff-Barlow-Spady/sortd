package tests

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestTUIInitialization(t *testing.T) {
	// TODO: Initialize the TUI component and verify its startup state.
	t.Skip("Test not implemented yet.")
}

func TestTUIInteraction(t *testing.T) {
	// TODO: Simulate user interactions with the TUI and assert expected behavior.
	t.Skip("Test not implemented yet.")
}

func TestTuiCommands(t *testing.T) {
	t.Run("help command", func(t *testing.T) {
		ui := NewTui()
		output := ui.Help()
		assert.Contains(t, output, "[o]rganize")
		assert.Contains(t, output, "[u]ndo")
	})

	t.Run("select file command", func(t *testing.T) {
		ui := NewTui()
		ui.SelectFile("file1.txt")
		assert.True(t, ui.IsSelected("file1.txt"), "File should be selected")
	})

	t.Run("deselect file command", func(t *testing.T) {
		ui := NewTui()
		ui.SelectFile("file1.txt")
		ui.DeselectFile("file1.txt")
		assert.False(t, ui.IsSelected("file1.txt"), "File should be deselected")
	})
}
