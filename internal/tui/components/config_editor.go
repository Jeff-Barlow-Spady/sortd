package components

import (
	"fmt"
	"sortd/internal/config"
	"sortd/internal/tui/messages"
	"sortd/internal/tui/styles"
	"sortd/pkg/types"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type ConfigEditor struct {
	inputs    []textinput.Model
	cursor    int
	config    *config.Config
	statusBar *StatusBar
}

func NewConfigEditor(cfg *config.Config) *ConfigEditor {
	ce := &ConfigEditor{
		inputs:    make([]textinput.Model, 0),
		config:    cfg,
		statusBar: NewStatusBar(),
	}

	// Create inputs for each pattern
	for _, pattern := range cfg.Organize.Patterns {
		input := textinput.New()
		input.Placeholder = "Pattern glob (e.g. *.jpg)"
		input.SetValue(pattern.Glob)
		input.Width = 40
		ce.inputs = append(ce.inputs, input)

		destInput := textinput.New()
		destInput.Placeholder = "Destination directory"
		destInput.SetValue(pattern.DestDir)
		destInput.Width = 40
		ce.inputs = append(ce.inputs, destInput)
	}

	if len(ce.inputs) > 0 {
		ce.inputs[0].Focus()
	}

	return ce
}

func (ce *ConfigEditor) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the submit button was focused?
			if s == "enter" && ce.cursor == len(ce.inputs) {
				return ce.Save
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				ce.cursor--
			} else {
				ce.cursor++
			}

			if ce.cursor >= len(ce.inputs) {
				ce.cursor = 0
			} else if ce.cursor < 0 {
				ce.cursor = len(ce.inputs) - 1
			}

			for i := 0; i < len(ce.inputs); i++ {
				if i == ce.cursor {
					cmds = append(cmds, ce.inputs[i].Focus())
				} else {
					ce.inputs[i].Blur()
				}
			}

			return tea.Batch(cmds...)
		}
	}

	// Update all textinputs
	for i := range ce.inputs {
		var cmd tea.Cmd
		ce.inputs[i], cmd = ce.inputs[i].Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return tea.Batch(cmds...)
}

func (ce *ConfigEditor) Init() tea.Cmd {
	return nil
}

func (ce *ConfigEditor) View() string {
	var s strings.Builder

	s.WriteString(styles.Theme.Title.Render("Configuration Editor\n\n"))

	for i := 0; i < len(ce.inputs); i += 2 {
		s.WriteString("Pattern:\n")
		s.WriteString(ce.inputs[i].View() + "\n")
		s.WriteString("Destination:\n")
		s.WriteString(ce.inputs[i+1].View() + "\n\n")
	}

	s.WriteString("\n" + ce.statusBar.View())

	return s.String()
}

func (ce *ConfigEditor) Save() tea.Msg {
	// Update config from inputs
	patterns := make([]types.Pattern, 0)
	for i := 0; i < len(ce.inputs); i += 2 {
		patterns = append(patterns, types.Pattern{
			Glob:    ce.inputs[i].Value(),
			DestDir: ce.inputs[i+1].Value(),
		})
	}

	ce.config.Organize.Patterns = patterns

	return messages.ConfigUpdateMsg{Config: ce.config}
}

func (ce *ConfigEditor) Copy() *ConfigEditor {
	fmt.Println("Starting ConfigEditor.Copy()")
	newCE := &ConfigEditor{
		inputs:    make([]textinput.Model, len(ce.inputs)),
		cursor:    ce.cursor,
		config:    ce.config,
		statusBar: NewStatusBar(), // Create new instead of copying
	}

	// Just create a shallow copy of the inputs
	copy(newCE.inputs, ce.inputs)

	fmt.Println("Finished ConfigEditor.Copy()")
	return newCE
}
