package types

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// KeyHandlerModel defines the interface required by key handlers to interact
// with the main TUI model. This breaks the import cycle between the model and handlers.
// It provides access to necessary components, state, and actions.
type KeyHandlerModel interface {
	// --- Required by handlers/key_handlers.go ---

	// Component Accessors
	Keys() *KeyMap // Corrected: Return pointer to KeyMap defined in types
	List() list.Model
	Viewport() viewport.Model
	Help() help.Model // Added based on handler usage

	// State Getters
	Mode() Mode // Assuming Mode is defined in types
	CurrentDir() string
	SelectedFiles() map[string]bool
	ShowFullHelp() bool    // Added based on handler usage
	CommandBuffer() string // Added based on handler usage
	VisualMode() bool      // Added based on handler usage
	ActiveView() ViewMode  // Added based on handler usage
	StatusMsg() string     // Added based on handler usage

	// State Setters & Actions
	SetMode(Mode)
	SetStatus(string) tea.Cmd
	LoadDirectory(string) tea.Cmd
	TriggerOrganizationCmd([]string) tea.Cmd
	UpdateVisualSelection()
	ToggleSelection()                 // Added based on handler usage
	ClearSelection()                  // Added based on handler usage
	SetList(list.Model)               // Added based on handler usage
	SetViewport(viewport.Model)       // Added based on handler usage
	SetHelp(help.Model)               // Added based on handler usage
	SetShowFullHelp(bool)             // Added based on handler usage
	SetCommandBuffer(string)          // Added based on handler usage
	SetVisualMode(bool)               // Added based on handler usage
	SetVisualStart(int)               // Added based on handler usage
	SetVisualEnd(int)                 // Added based on handler usage
	SetCurrentDir(string)             // Added based on handler usage
	SetSelectedFiles(map[string]bool) // Added based on handler usage
	SetOrganizing(bool)               // Added based on handler usage
	SetLoading(bool)                  // Added based on handler usage
	SetActiveView(ViewMode)           // Added based on handler usage
	UpdateViewportContent()           // Added based on handler usage

	// --- Potentially needed methods (Add if handlers require them) ---
	// GetOrganizePaths() []string // Maybe replace with direct SelectedFiles access?
}

// Ensure KeyMap is defined or adjust Keys() return type.
// If KeyMap is specific to tui, Keys() might need to return key.Binding or similar.
// Let's assume key.Binding is sufficient for now.

// Note: Item type used in handlers (selectedItem.(tui.Item)) is defined in internal/tui.
// Handlers will still need to import internal/tui specifically for this type assertion,
// or we need to define an Item interface in pkg/types. For now, handlers will keep the tui import for Item.
