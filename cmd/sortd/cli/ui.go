package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ColorTheme represents a set of colors for the CLI
type ColorTheme struct {
	Name        string
	Success     string
	Error       string
	Warning     string
	Info        string
	Header      string
	Logo        string
	BoxOutline  string
	Highlight   string
	Normal      string
	Description string
}

// Available themes
var (
	// Default Orange theme
	DefaultTheme = ColorTheme{
		Name:        "default",
		Success:     colorGreen,
		Error:       colorRed,
		Warning:     colorYellow,
		Info:        colorBlue,
		Header:      colorCyan + colorBold,
		Logo:        colorCyan,
		BoxOutline:  colorCyan,
		Highlight:   colorPurple,
		Normal:      colorWhite,
		Description: "Default orange theme",
	}

	// Blue theme
	BlueTheme = ColorTheme{
		Name:        "blue",
		Success:     colorGreen,
		Error:       colorRed,
		Warning:     colorYellow,
		Info:        colorBlue,
		Header:      colorBlue + colorBold,
		Logo:        colorBlue,
		BoxOutline:  colorBlue,
		Highlight:   colorCyan,
		Normal:      colorWhite,
		Description: "Blue focused theme",
	}

	// Green theme
	GreenTheme = ColorTheme{
		Name:        "green",
		Success:     colorGreen,
		Error:       colorRed,
		Warning:     colorYellow,
		Info:        colorCyan,
		Header:      colorGreen + colorBold,
		Logo:        colorGreen,
		BoxOutline:  colorGreen,
		Highlight:   colorCyan,
		Normal:      colorWhite,
		Description: "Nature-inspired green theme",
	}

	// Dark theme
	DarkTheme = ColorTheme{
		Name:        "dark",
		Success:     colorGreen,
		Error:       colorRed,
		Warning:     colorYellow,
		Info:        colorPurple,
		Header:      colorWhite + colorBold,
		Logo:        colorWhite,
		BoxOutline:  colorPurple,
		Highlight:   colorBlue,
		Normal:      colorWhite,
		Description: "Dark mode theme",
	}

	// Gruvbox theme
	GruvboxTheme = ColorTheme{
		Name:        "gruvbox",
		Success:     "\033[38;5;142m",             // Gruvbox yellow/green
		Error:       "\033[38;5;167m",             // Gruvbox red
		Warning:     "\033[38;5;214m",             // Gruvbox orange
		Info:        "\033[38;5;109m",             // Gruvbox blue
		Header:      "\033[38;5;208m" + colorBold, // Gruvbox orange bold
		Logo:        "\033[38;5;208m",             // Gruvbox orange
		BoxOutline:  "\033[38;5;142m",             // Gruvbox yellow/green
		Highlight:   "\033[38;5;175m",             // Gruvbox purple
		Normal:      "\033[38;5;223m",             // Gruvbox light foreground
		Description: "Warm, earthy color scheme (gruvbox)",
	}

	// Kanagawa theme
	KanagawaTheme = ColorTheme{
		Name:        "kanagawa",
		Success:     "\033[38;5;108m",             // Kanagawa soft green
		Error:       "\033[38;5;203m",             // Kanagawa peach red
		Warning:     "\033[38;5;179m",             // Kanagawa soft yellow
		Info:        "\033[38;5;110m",             // Kanagawa light blue
		Header:      "\033[38;5;105m" + colorBold, // Kanagawa purple bold
		Logo:        "\033[38;5;105m",             // Kanagawa purple
		BoxOutline:  "\033[38;5;110m",             // Kanagawa light blue
		Highlight:   "\033[38;5;217m",             // Kanagawa sakura
		Normal:      "\033[38;5;252m",             // Kanagawa light foreground
		Description: "Japanese traditional colors (kanagawa)",
	}

	// Tokyo Night theme
	TokyoNightTheme = ColorTheme{
		Name:        "tokyo-night",
		Success:     "\033[38;5;115m",             // Tokyo night teal
		Error:       "\033[38;5;203m",             // Tokyo night red
		Warning:     "\033[38;5;222m",             // Tokyo night yellow
		Info:        "\033[38;5;110m",             // Tokyo night blue
		Header:      "\033[38;5;139m" + colorBold, // Tokyo night purple bold
		Logo:        "\033[38;5;139m",             // Tokyo night purple
		BoxOutline:  "\033[38;5;110m",             // Tokyo night blue
		Highlight:   "\033[38;5;216m",             // Tokyo night orange
		Normal:      "\033[38;5;253m",             // Tokyo night light foreground
		Description: "Neon Tokyo-inspired dark theme",
	}
)

// List of all available themes
var AvailableThemes = []ColorTheme{
	DefaultTheme,
	BlueTheme,
	GreenTheme,
	DarkTheme,
	GruvboxTheme,
	KanagawaTheme,
	TokyoNightTheme,
}

// Current active theme, starts with default
var CurrentTheme = DefaultTheme

// Terminal colors
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
)

// SetTheme sets the current theme by name
func SetTheme(themeName string) bool {
	for _, theme := range AvailableThemes {
		if theme.Name == themeName {
			CurrentTheme = theme
			return true
		}
	}
	return false
}

// GetThemeNames returns all available theme names
func GetThemeNames() []string {
	var names []string
	for _, theme := range AvailableThemes {
		names = append(names, theme.Name)
	}
	return names
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Println(CurrentTheme.Success + "✓ " + message + colorReset)
}

// PrintError prints an error message
func PrintError(message string) {
	fmt.Println(CurrentTheme.Error + "✗ " + message + colorReset)
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	fmt.Println(CurrentTheme.Warning + "! " + message + colorReset)
}

// PrintInfo prints an informational message
func PrintInfo(message string) {
	fmt.Println(CurrentTheme.Info + "ℹ " + message + colorReset)
}

// PrintHeader prints a section header
func PrintHeader(message string) {
	fmt.Println("\n" + CurrentTheme.Header + message + colorReset)
	fmt.Println(strings.Repeat("─", len(message)))
}

// DrawBox creates a colored box around content
func DrawBox(content, color string) string {
	lines := strings.Split(content, "\n")
	maxLen := 0
	for _, line := range lines {
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}

	result := color + "┌" + strings.Repeat("─", maxLen+2) + "┐\n"
	for _, line := range lines {
		result += "│ " + line + strings.Repeat(" ", maxLen-len(line)) + " │\n"
	}
	result += "└" + strings.Repeat("─", maxLen+2) + "┘" + colorReset

	return result
}

// DrawBoxWithTheme creates a colored box using the current theme
func DrawBoxWithTheme(content string) string {
	return DrawBox(content, CurrentTheme.BoxOutline)
}

// HasGum checks if the Gum CLI tool is installed
func HasGum() bool {
	_, err := exec.LookPath("gum")
	return err == nil
}

// RunGumInput runs Gum input and returns the result
func RunGumInput(prompt, defaultValue string) string {
	args := []string{"input"}
	if prompt != "" {
		args = append(args, "--prompt", prompt+": ")
	}
	if defaultValue != "" {
		args = append(args, "--value", defaultValue)
	}

	cmd := exec.Command("gum", args...)
	cmd.Stdin = os.Stdin
	output, err := cmd.Output()
	if err != nil {
		return defaultValue
	}
	return strings.TrimSpace(string(output))
}

// RunGumConfirm runs Gum confirm and returns the result
func RunGumConfirm(prompt string) bool {
	if prompt == "" {
		prompt = "Confirm?"
	}

	cmd := exec.Command("gum", "confirm", prompt)
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	return err == nil
}

// RunGumChoose runs Gum choose and returns the selected option
func RunGumChoose(options ...string) string {
	args := append([]string{"choose"}, options...)
	cmd := exec.Command("gum", args...)
	cmd.Stdin = os.Stdin
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// RunGumMultiChoose runs Gum choose with multi-select and returns the selected options
func RunGumMultiChoose(options ...string) []string {
	if len(options) == 0 {
		return []string{}
	}

	args := append([]string{"choose", "--no-limit"}, options...)
	cmd := exec.Command("gum", args...)
	cmd.Stdin = os.Stdin
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	selected := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(selected) == 1 && selected[0] == "" {
		return []string{}
	}
	return selected
}

// RunGumFile runs Gum file and returns the selected file
func RunGumFile(args ...string) string {
	cmdArgs := append([]string{"file"}, args...)
	cmd := exec.Command("gum", cmdArgs...)
	cmd.Stdin = os.Stdin
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// DrawSortdLogo generates the ASCII art logo for sortd.
func DrawSortdLogo() string {
	// Define the logo as a multiline string
	logo := `
  █████████     ███████    ███████████   ███████████ ██████████  
 ███░░░░░███  ███░░░░░███ ░░███░░░░░███ ░█░░░███░░░█░░███░░░░███ 
░███    ░░░  ███     ░░███ ░███    ░███ ░   ░███  ░  ░███   ░░███
░░█████████ ░███      ░███ ░██████████      ░███     ░███    ░███
 ░░░░░░░░███░███      ░███ ░███░░░░░███     ░███     ░███    ░███
 ███    ░███░░███     ███  ░███    ░███     ░███     ░███    ███ 
░░█████████  ░░░███████░   █████   █████    █████    ██████████  
 ░░░░░░░░░     ░░░░░░░    ░░░░░   ░░░░░    ░░░░░    ░░░░░░░░░░   
                                                                 
                                                                 
                                                                 
`

	return CurrentTheme.Logo + logo + colorReset
}

// ChooseDirectory prompts the user to select a directory using Gum.
func ChooseDirectory() string {
	return RunGumFile("--directory")
}

// SaveThemePreference saves the current theme preference to a file
func SaveThemePreference() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	configDir := filepath.Join(homeDir, ".config", "sortd")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Create theme file
	themeFile := filepath.Join(configDir, "theme.txt")
	return os.WriteFile(themeFile, []byte(CurrentTheme.Name), 0644)
}

// LoadThemePreference loads the theme preference from a file
func LoadThemePreference() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Try to read theme file
	themeFile := filepath.Join(homeDir, ".config", "sortd", "theme.txt")
	themeName, err := os.ReadFile(themeFile)
	if err != nil {
		return // Use default theme
	}

	// Set theme if found
	SetTheme(strings.TrimSpace(string(themeName)))
}
