package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

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

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Println(colorGreen + "✓ " + message + colorReset)
}

// PrintError prints an error message
func PrintError(message string) {
	fmt.Println(colorRed + "✗ " + message + colorReset)
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	fmt.Println(colorYellow + "! " + message + colorReset)
}

// PrintInfo prints an informational message
func PrintInfo(message string) {
	fmt.Println(colorBlue + "ℹ " + message + colorReset)
}

// PrintHeader prints a section header
func PrintHeader(message string) {
	fmt.Println("\n" + colorBold + colorCyan + message + colorReset)
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
   █████████     ███████    ███████████   ███████████ ██████████  
 ███░░░░░███   ███░░░░░███ ░░███░░░░░███ ░█░░░███░░░█░░███░░░░███ 
░███     ░░░  ███     ░░███ ░███    ░███ ░    ░███  ░  ░███    ░░███
░░█████████  ░███      ░███ ░██████████       ░███     ░███    ░███
 ░░░░░░░░███ ░███      ░███ ░███░░░░░███      ░███     ░███    ░███
 ███     ███ ░░███     ███  ░███    ░███      ░███     ░███    ███ 
░░█████████   ░░░███████░   █████   █████     █████     ██████████  
 ░░░░░░░░░       ░░░░░░░    ░░░░░   ░░░░░     ░░░░░     ░░░░░░░░░░   
`

	return colorCyan + logo + colorReset
}

// ChooseDirectory prompts the user to select a directory using Gum.
func ChooseDirectory() string {
	return RunGumFile("--directory")
}
