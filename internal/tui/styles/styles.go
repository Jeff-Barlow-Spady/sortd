package styles

import "github.com/charmbracelet/lipgloss"

// Color constants for the application
var (
	ColorPrimary    = lipgloss.Color("#7B61FF") // Primary accent color
	ColorSecondary  = lipgloss.Color("#5F87FF") // Secondary accent color
	ColorBackground = lipgloss.Color("#282a36") // Dark background color
	ColorText       = lipgloss.Color("#f8f8f2") // Light text color
	ColorHighlight  = lipgloss.Color("#ff79c6") // Highlight color for actions
	ColorSuccess    = lipgloss.Color("#73F59F") // Success color
	ColorWarning    = lipgloss.Color("#FFB86C") // Warning color
	ColorError      = lipgloss.Color("#FF5555") // Error color
	ColorSubtle     = lipgloss.Color("#6272A4") // Subtle color for less important elements
	ColorBorder     = lipgloss.Color("#44475A") // Border color
)

// Styles defines the core UI styles
var (
	App = lipgloss.NewStyle().
		Padding(1, 2).
		Background(ColorBackground).
		Foreground(ColorText)

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Background(ColorBackground).
		MarginBottom(1)

	Selected = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	Unselected = lipgloss.NewStyle().
			Foreground(ColorSubtle)

	Help = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Italic(true)

	// New styles for enhanced TUI
	LogoStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(ColorBackground).
			MarginTop(1).
			MarginBottom(1)

	CommandPrompt = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)

	SectionStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true).
			Underline(true)

	TipStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true)

	StatusMsg = lipgloss.NewStyle().
			Foreground(ColorWarning)

	EmptyStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle).
			Italic(true)

	DirStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	// Version number style
	VersionStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Italic(true)

	// Panel styles
	PanelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1).
			Background(ColorBackground)

	// Header Bar styles
	HeaderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1).
			Bold(true)
)

// FileListStyle defines the style for the file list
var FileListStyle = lipgloss.NewStyle().
	Padding(1, 2).
	Border(lipgloss.RoundedBorder()).
	BorderForeground(ColorPrimary).
	Background(ColorBackground)

// DefaultTheme encapsulates all styles for easy access across the application
var DefaultTheme = struct {
	App           lipgloss.Style
	Title         lipgloss.Style
	Selected      lipgloss.Style
	Unselected    lipgloss.Style
	Help          lipgloss.Style
	Logo          lipgloss.Style
	CommandPrompt lipgloss.Style
	Section       lipgloss.Style
	Tip           lipgloss.Style
	Status        lipgloss.Style
	Empty         lipgloss.Style
	Dir           lipgloss.Style
	Version       lipgloss.Style
	Panel         lipgloss.Style
	Header        lipgloss.Style
	FileList      lipgloss.Style
	// Colors for direct use
	ColorPrimary    lipgloss.Color
	ColorSecondary  lipgloss.Color
	ColorBackground lipgloss.Color
	ColorText       lipgloss.Color
	ColorHighlight  lipgloss.Color
	ColorSuccess    lipgloss.Color
	ColorWarning    lipgloss.Color
	ColorError      lipgloss.Color
	ColorSubtle     lipgloss.Color
	ColorBorder     lipgloss.Color
}{
	App:           App,
	Title:         TitleStyle,
	Selected:      Selected,
	Unselected:    Unselected,
	Help:          Help,
	Logo:          LogoStyle,
	CommandPrompt: CommandPrompt,
	Section:       SectionStyle,
	Tip:           TipStyle,
	Status:        StatusMsg,
	Empty:         EmptyStyle,
	Dir:           DirStyle,
	Version:       VersionStyle,
	Panel:         PanelStyle,
	Header:        HeaderStyle,
	FileList:      FileListStyle,
	// Colors
	ColorPrimary:    ColorPrimary,
	ColorSecondary:  ColorSecondary,
	ColorBackground: ColorBackground,
	ColorText:       ColorText,
	ColorHighlight:  ColorHighlight,
	ColorSuccess:    ColorSuccess,
	ColorWarning:    ColorWarning,
	ColorError:      ColorError,
	ColorSubtle:     ColorSubtle,
	ColorBorder:     ColorBorder,
}
