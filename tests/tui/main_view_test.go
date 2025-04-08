package tests

import (
	"testing"

	"sortd/internal/tui/views"
	"sortd/pkg/types"

	"github.com/stretchr/testify/assert"
)

type mockModel struct {
	files         []types.FileEntry
	selectedFiles map[string]bool
	cursor        int
	showHelp      bool
	mode          types.Mode
	currentDir    string
	viewMode      types.ViewMode
}

// Implement types.ModelReader interface for mockModel
func (m *mockModel) Files() []*types.FileInfo {
	// Convert []types.FileEntry to []*types.FileInfo for the interface
	fileInfos := make([]*types.FileInfo, len(m.files))
	for i, entry := range m.files {
		fileInfos[i] = &types.FileInfo{
			Path:        entry.Path,
			ContentType: entry.ContentType,
			Size:        entry.Size,
			Tags:        entry.Tags,
			// FileInfo doesn't have Name or IsDir, so these are omitted
		}
	}
	return fileInfos
}

func (m *mockModel) SelectedFiles() map[string]bool { return m.selectedFiles }
func (m *mockModel) Cursor() int                    { return m.cursor }
func (m *mockModel) ShowHelp() bool                 { return m.showHelp }
func (m *mockModel) Mode() types.Mode               { return m.mode }
func (m *mockModel) CurrentDir() string             { return m.currentDir }
func (m *mockModel) ViewMode() types.ViewMode       { return m.viewMode }
func (m *mockModel) IsSelected(path string) bool    { return false } // Basic implementation

func TestRenderHelp(t *testing.T) {
	help := views.RenderHelp()

	// Verify all key commands are documented
	assert.Contains(t, help, "↑/k, ↓/j: Move cursor") // Combined up/down
	assert.Contains(t, help, "h/←, l/→: Change directory") // Added
	assert.Contains(t, help, "gg: Go to top")             // Added
	assert.Contains(t, help, "G: Go to bottom")           // Added
	assert.Contains(t, help, "Space: Toggle selection")   // Updated text
	assert.Contains(t, help, "v: Visual mode")            // Added
	assert.Contains(t, help, "V: Visual line mode")       // Added
	assert.Contains(t, help, "o: Organize selected files") // Added
	assert.Contains(t, help, "r: Refresh view")           // Added
	assert.Contains(t, help, ":: Command mode")           // Added
	assert.Contains(t, help, "?: Toggle help")
	assert.Contains(t, help, "q, quit: Exit") // Updated text

	// Verify removed/changed keys are NOT present
	assert.NotContains(t, help, "enter: Open directory")
	assert.NotContains(t, help, "space: Select file") // Check old text isn't present
	assert.NotContains(t, help, "j/↓: Move down")     // Check old text isn't present
	assert.NotContains(t, help, "k/↑: Move up")       // Check old text isn't present
	assert.NotContains(t, help, "q: Quit")             // Check old text isn't present
}

func TestRenderKeyCommands(t *testing.T) {
	commands := views.RenderKeyCommands()

	// Verify all essential commands are shown
	assert.Contains(t, commands, "[↑/k] Up")
	assert.Contains(t, commands, "[↓/j] Down")
	assert.Contains(t, commands, "[Space] Select")
	assert.Contains(t, commands, "[Enter] Open")
	assert.Contains(t, commands, "[q] Quit")
	assert.Contains(t, commands, "[?] Help")
}

func TestRenderMainView(t *testing.T) {
	tests := []struct {
		name     string
		model    *mockModel
		contains []string
	}{
		{
			name: "empty_state",
			model: &mockModel{
				mode: types.Normal,
			},
			contains: []string{
				"Quick Start - Organize Files",
				"Setup Configuration",
				"3. Watch Mode (Coming Soon)",
				"4. Show Help",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := views.RenderMainView(tt.model)
			for _, s := range tt.contains {
				assert.Contains(t, result, s)
			}
		})
	}
}
