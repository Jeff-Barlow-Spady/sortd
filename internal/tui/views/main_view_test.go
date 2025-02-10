package views

import (
	"fmt"
	"testing"

	"sortd/pkg/types"

	"github.com/stretchr/testify/assert"
)

// Mock model for testing
type mockModel struct {
	files         []types.FileEntry
	selectedFiles map[string]bool
	cursor        int
	showHelp      bool
	mode          types.Mode
	currentDir    string
}

func (m *mockModel) Files() []types.FileEntry    { return m.files }
func (m *mockModel) IsSelected(name string) bool { return m.selectedFiles[name] }
func (m *mockModel) Cursor() int                 { return m.cursor }
func (m *mockModel) ShowHelp() bool              { return m.showHelp }
func (m *mockModel) Mode() types.Mode            { return m.mode }
func (m *mockModel) CurrentDir() string          { return m.currentDir }

func TestRenderMainView(t *testing.T) {
	tests := []struct {
		name     string
		model    *mockModel
		contains []string // Strings that should be present in the output
		excludes []string // Strings that should not be present in the output
	}{
		{
			name: "empty directory",
			model: &mockModel{
				files:         []types.FileEntry{},
				selectedFiles: make(map[string]bool),
				cursor:        0,
				showHelp:      false,
				mode:          types.Normal,
				currentDir:    "/test",
			},
			contains: []string{
				"Directory: /test",
				"Name",
				"Type",
				"Size",
			},
			excludes: []string{
				"Quick Start",
			},
		},
		{
			name: "directory with files",
			model: &mockModel{
				files: []types.FileEntry{
					{Name: "test.txt", Path: "/test/test.txt", ContentType: "text/plain", Size: 1024},
					{Name: "image.jpg", Path: "/test/image.jpg", ContentType: "image/jpeg", Size: 1024 * 1024},
				},
				selectedFiles: map[string]bool{"test.txt": true},
				cursor:        0,
				showHelp:      false,
				mode:          types.Normal,
				currentDir:    "/test",
			},
			contains: []string{
				"test.txt",
				"image.jpg",
				"text/plain",
				"image/jpeg",
				"1.0KB",
				"1.0MB",
			},
		},
		{
			name: "with help shown",
			model: &mockModel{
				files:         []types.FileEntry{},
				selectedFiles: make(map[string]bool),
				cursor:        0,
				showHelp:      true,
				mode:          types.Normal,
				currentDir:    "/test",
			},
			contains: []string{
				"Navigation:",
				"Selection:",
				"Organization:",
				"Commands:",
			},
		},
		{
			name: "main menu",
			model: &mockModel{
				files:         []types.FileEntry{},
				selectedFiles: make(map[string]bool),
				cursor:        0,
				showHelp:      false,
				mode:          types.Normal,
				currentDir:    "",
			},
			contains: []string{
				"Quick Start",
				"Setup Configuration",
				"Watch Mode",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderMainView(tt.model)

			// Check required strings are present
			for _, s := range tt.contains {
				assert.Contains(t, output, s, fmt.Sprintf("output should contain '%s'", s))
			}

			// Check excluded strings are not present
			for _, s := range tt.excludes {
				assert.NotContains(t, output, s, fmt.Sprintf("output should not contain '%s'", s))
			}
		})
	}
}

func TestRenderKeyCommands(t *testing.T) {
	output := RenderKeyCommands()
	requiredKeys := []string{
		"Up", "Down", "Select", "Open", "Organize", "Back", "Quit", "Help",
	}

	for _, key := range requiredKeys {
		assert.Contains(t, output, key, fmt.Sprintf("key commands should contain '%s'", key))
	}
}

func TestRenderHelp(t *testing.T) {
	output := RenderHelp()
	sections := []string{
		"Navigation:",
		"Selection:",
		"Organization:",
		"Commands:",
	}

	for _, section := range sections {
		assert.Contains(t, output, section, fmt.Sprintf("help should contain '%s' section", section))
	}

	// Test specific key bindings
	keyBindings := []string{
		"↑/k, ↓/j: Move cursor",
		"h/←, l/→: Change directory",
		"gg: Go to top",
		"G: Go to bottom",
		"Space: Toggle selection",
		"v: Visual mode",
		"V: Visual line mode",
		"o: Organize selected files",
		"r: Refresh view",
		"q, quit: Exit",
		":: Command mode",
		"?: Toggle help",
	}

	for _, binding := range keyBindings {
		assert.Contains(t, output, binding, fmt.Sprintf("help should contain key binding '%s'", binding))
	}
}
