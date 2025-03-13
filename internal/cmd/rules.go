package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sortd/internal/config"
	"strings"

	"github.com/spf13/cobra"
)

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Manage file organization rules",
	Long:  `Add, list, and remove file organization rules.`,
}

var rulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all organization rules",
	Run: func(cmd *cobra.Command, args []string) {
		if cfg == nil || len(cfg.Rules) == 0 {
			fmt.Println("❓ No rules defined. Use 'sortd rules add' to add rules.")
			return
		}

		fmt.Println("📝 Organization Rules:")

		// Format as a table with Gum if available
		if _, err := exec.LookPath("gum"); err == nil {
			// Header
			header := runGum("join",
				"--horizontal",
				"--align", "left",
				runGum("style", "--width", "30", "--foreground", "212", "--bold", "Pattern"),
				runGum("style", "--width", "50", "--foreground", "212", "--bold", "Target Directory"))
			fmt.Println(header)

			// Rules
			for i, rule := range cfg.Rules {
				row := runGum("join",
					"--horizontal",
					"--align", "left",
					runGum("style", "--width", "30", fmt.Sprintf("%d. %s", i+1, rule.Pattern)),
					runGum("style", "--width", "50", rule.Target))
				fmt.Println(row)
			}
		} else {
			// Fallback to standard output
			fmt.Printf("%-30s %-50s\n", "Pattern", "Target Directory")
			fmt.Println(strings.Repeat("-", 80))
			for i, rule := range cfg.Rules {
				fmt.Printf("%-30s %-50s\n", fmt.Sprintf("%d. %s", i+1, rule.Pattern), rule.Target)
			}
		}
	},
}

var rulesAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new organization rule",
	Run: func(cmd *cobra.Command, args []string) {
		// Skip interactive mode in tests
		if os.Getenv("TESTMODE") == "true" {
			if len(args) >= 2 {
				pattern := args[0]
				target := args[1]
				addRule(pattern, target)
				fmt.Printf("Added rule: %s -> %s\n", pattern, target)
			} else {
				fmt.Println("Test mode requires pattern and target arguments")
			}
			return
		}

		// Ensure gum is available for interactive mode
		if _, err := exec.LookPath("gum"); err != nil {
			fmt.Println("❌ Interactive mode requires gum to be installed.")
			fmt.Println("💡 Install Gum from https://github.com/charmbracelet/gum")
			fmt.Println("Or provide arguments: sortd rules add [pattern] [target]")
			return
		}

		// Interactive rule creation
		runGum("style",
			"--foreground", "212",
			"--border", "rounded",
			"--border-foreground", "212",
			"--padding", "1",
			"--width", "70",
			"Let's create a new organization rule!")

		// Get pattern with suggestions
		fmt.Println("📋 Pattern examples:")
		fmt.Println("  • *.jpg - All JPEG images")
		fmt.Println("  • report*.pdf - PDF files starting with 'report'")
		fmt.Println("  • *.{mp3,wav,flac} - Audio files with various extensions")

		pattern := runGumInput("Enter file pattern", "")
		if pattern == "" {
			fmt.Println("❌ Pattern cannot be empty")
			return
		}

		// Get target directory
		fmt.Println("\n📂 Choose target directory:")
		fmt.Println("  • Enter path or select with file browser")

		// Let user choose between typing or browsing
		choiceMethod := runGumChoose("Enter manually", "Browse for directory")

		var target string
		if choiceMethod == "Enter manually" {
			target = runGumInput("Target directory", "")
		} else {
			target = runGumFile("--directory")
		}

		if target == "" {
			fmt.Println("❌ Target directory cannot be empty")
			return
		}

		// Confirm
		fmt.Printf("📌 New rule: Files matching '%s' will be moved to '%s'\n", pattern, target)
		if !runGumConfirm("Add this rule?") {
			fmt.Println("🛑 Cancelled")
			return
		}

		// Add the rule
		addRule(pattern, target)

		// Save the config
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("❌ Error getting home directory: %v\n", err)
			return
		}
		configPath := filepath.Join(home, ".config", "sortd", "config.yaml")
		if err := saveConfigToFile(cfg, configPath); err != nil {
			fmt.Printf("❌ Error saving config: %v\n", err)
			return
		}

		fmt.Printf("✅ Rule added successfully! Files matching '%s' will go to '%s'\n", pattern, target)
	},
}

var rulesRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove an organization rule",
	Run: func(cmd *cobra.Command, args []string) {
		if cfg == nil || len(cfg.Rules) == 0 {
			fmt.Println("❓ No rules defined to remove.")
			return
		}

		// Skip interactive mode in tests
		if os.Getenv("TESTMODE") == "true" {
			if len(args) >= 1 {
				index := 0
				fmt.Sscanf(args[0], "%d", &index)
				if index > 0 && index <= len(cfg.Rules) {
					removeRule(index - 1)
					fmt.Printf("Removed rule at index %d\n", index)
				} else {
					fmt.Println("Invalid rule index")
				}
			} else {
				fmt.Println("Test mode requires rule index argument")
			}
			return
		}

		// Ensure gum is available for interactive mode
		if _, err := exec.LookPath("gum"); err != nil {
			fmt.Println("❌ Interactive mode requires gum to be installed.")
			fmt.Println("💡 Install Gum from https://github.com/charmbracelet/gum")
			fmt.Println("Or provide arguments: sortd rules remove [index]")
			return
		}

		// List the rules first
		fmt.Println("📝 Current Rules:")
		var options []string
		for i, rule := range cfg.Rules {
			options = append(options, fmt.Sprintf("%d. %s -> %s", i+1, rule.Pattern, rule.Target))
		}

		// Let the user select a rule to remove
		fmt.Println("🗑️ Select a rule to remove:")
		selection := runGumChoose(options...)
		if selection == "" {
			fmt.Println("🛑 Cancelled")
			return
		}

		// Parse the selection to get the index
		var index int
		fmt.Sscanf(selection, "%d.", &index)
		index-- // Adjust for 0-based indexing

		// Confirm
		fmt.Printf("⚠️ Are you sure you want to remove this rule?\n%s\n", selection)
		if !runGumConfirm("Remove this rule?") {
			fmt.Println("🛑 Cancelled")
			return
		}

		// Remove the rule
		removeRule(index)

		// Save the config
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("❌ Error getting home directory: %v\n", err)
			return
		}
		configPath := filepath.Join(home, ".config", "sortd", "config.yaml")
		if err := saveConfigToFile(cfg, configPath); err != nil {
			fmt.Printf("❌ Error saving config: %v\n", err)
			return
		}

		fmt.Println("✅ Rule removed successfully!")
	},
}

func init() {
	rootCmd.AddCommand(rulesCmd)
	rulesCmd.AddCommand(rulesListCmd)
	rulesCmd.AddCommand(rulesAddCmd)
	rulesCmd.AddCommand(rulesRemoveCmd)
}

// Helper functions for managing rules

func addRule(pattern, target string) {
	if cfg == nil {
		cfg = config.New()
	}

	cfg.Rules = append(cfg.Rules, struct {
		Pattern string `yaml:"pattern"`
		Target  string `yaml:"target"`
	}{
		Pattern: pattern,
		Target:  target,
	})
}

func removeRule(index int) {
	if cfg == nil || index < 0 || index >= len(cfg.Rules) {
		return
	}

	// Remove the rule at the specified index
	cfg.Rules = append(cfg.Rules[:index], cfg.Rules[index+1:]...)
}

func saveConfigToFile(cfg *config.Config, path string) error {
	// Use the correct SaveConfig function from the config package
	return config.SaveConfig(cfg, path)
}
