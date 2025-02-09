package tests

import (
	"testing"

	"sortd/internal/tui/common"
	"sortd/internal/tui/views"

	"github.com/stretchr/testify/assert"
)

type mockModel struct {
	files         []common.FileEntry
	selectedFiles map[string]bool
	cursor        int
	showHelp      bool
	mode          common.Mode
	currentDir    string
}

func (m *mockModel) Files() []common.FileEntry   { return m.files }
func (m *mockModel) IsSelected(name string) bool { return m.selectedFiles[name] }
func (m *mockModel) Cursor() int                 { return m.cursor }
func (m *mockModel) ShowHelp() bool              { return m.showHelp }
func (m *mockModel) Mode() common.Mode           { return m.mode }
func (m *mockModel) CurrentDir() string          { return m.currentDir }

func TestRenderHelp(t *testing.T) {
	help := views.RenderHelp()

	// Verify all key commands are documented
	assert.Contains(t, help, "j/↓: Move down")
	assert.Contains(t, help, "k/↑: Move up")
	assert.Contains(t, help, "enter: Open directory")
	assert.Contains(t, help, "space: Select file")
	assert.Contains(t, help, "?: Toggle help")
	assert.Contains(t, help, "q: Quit")
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
				mode: common.Normal,
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
