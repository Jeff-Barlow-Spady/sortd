package types

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the keybindings for the application modes.
// It's moved to pkg/types to be shared between model and handlers.
type KeyMap struct {
	// General
	Help key.Binding
	Quit key.Binding

	// Navigation
	Up           key.Binding
	Down         key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding
	ChangeDir    key.Binding // Typically 'l' or Enter on a directory
	GoBack       key.Binding // Typically 'h' or Backspace
	Filter       key.Binding
	ClearFilter  key.Binding

	// Selection & Actions
	Select        key.Binding // Select/deselect single item
	SelectVisual  key.Binding // Start/adjust visual selection
	SelectAll     key.Binding
	ClearSelection key.Binding
	Organize      key.Binding // Trigger organization
	ToggleHidden  key.Binding
	EnterCmdMode  key.Binding // Enter Command mode (:)

	// Command Mode Specific
	ExecuteCmd key.Binding // Execute command (Enter)
	ExitCmdMode key.Binding // Exit Command mode (Esc)
}
