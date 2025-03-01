package components

import (
	"sortd/internal/tui/styles"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StatusBar struct {
	text    string
	style   lipgloss.Style
	spinner spinner.Model
	loading bool
}

func NewStatusBar() *StatusBar {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Theme.Help

	return &StatusBar{
		style:   styles.Theme.Help,
		spinner: s,
	}
}

func (s *StatusBar) SetLoading(loading bool) {
	s.loading = loading
}

func (s *StatusBar) SetText(text string) {
	s.text = text
}

func (s *StatusBar) Update(msg tea.Msg) tea.Cmd {
	if s.loading {
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return cmd
	}
	return nil
}

func (s *StatusBar) View() string {
	if s.text == "" && !s.loading {
		return ""
	}

	if s.loading {
		return s.style.Render(s.spinner.View() + " " + s.text)
	}
	return s.style.Render(s.text)
}

// Copy returns a copy of the StatusBar
func (sb *StatusBar) Copy() *StatusBar {
	newSB := NewStatusBar()
	// If StatusBar has a text field, copy it
	if sb.text != "" {
		newSB.SetText(sb.text)
	}
	return newSB
}
