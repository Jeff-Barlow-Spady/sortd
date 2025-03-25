package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sortd/internal/config"
	"strings"

	"github.com/spf13/cobra"
)

// Common file types for improved categorization
var fileTypeCategories = map[string][]string{
	"Images": {
		"*.jpg", "*.jpeg", "*.png", "*.gif", "*.bmp", "*.tiff", "*.webp", "*.svg", "*.ico", "*.raw",
	},
	"Documents": {
		"*.pdf", "*.doc", "*.docx", "*.txt", "*.rtf", "*.odt", "*.md", "*.tex", "*.csv", "*.xls",
		"*.xlsx", "*.ppt", "*.pptx", "*.odp", "*.ods", "*.epub", "*.mobi",
	},
	"Audio": {
		"*.mp3", "*.wav", "*.flac", "*.aac", "*.ogg", "*.m4a", "*.wma", "*.aiff", "*.alac",
	},
	"Video": {
		"*.mp4", "*.mov", "*.avi", "*.mkv", "*.wmv", "*.flv", "*.webm", "*.m4v", "*.3gp", "*.mpeg", "*.mpg",
	},
	"Archives": {
		"*.zip", "*.rar", "*.tar", "*.gz", "*.7z", "*.bz2", "*.xz", "*.tgz", "*.iso",
	},
	"Code": {
		"*.py", "*.js", "*.html", "*.css", "*.java", "*.c", "*.cpp", "*.h", "*.go", "*.rs",
		"*.php", "*.rb", "*.pl", "*.swift", "*.ts", "*.json", "*.xml", "*.yaml", "*.yml",
	},
	"Executables": {
		"*.exe", "*.msi", "*.deb", "*.rpm", "*.dmg", "*.pkg", "*.app", "*.apk",
	},
	"Fonts": {
		"*.ttf", "*.otf", "*.woff", "*.woff2", "*.eot",
	},
}

// NewRulesCmd creates a new rules command
func NewRulesCmd() *cobra.Command {
	rulesCmd := &cobra.Command{
		Use:   "rules",
		Short: "Manage file organization rules",
		Long:  `Add, list, and remove file organization rules.`,
	}

	rulesListCmd := &cobra.Command{
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

			// If there are directory-specific rules, also show them
			if cfg.DirectoryRules != nil && len(cfg.DirectoryRules) > 0 {
				fmt.Println("\n📂 Directory-Specific Rules:")

				for dir, rules := range cfg.DirectoryRules {
					fmt.Printf("\nDirectory: %s\n", dir)

					if _, err := exec.LookPath("gum"); err == nil {
						// Header
						header := runGum("join",
							"--horizontal",
							"--align", "left",
							runGum("style", "--width", "30", "--foreground", "212", "--bold", "Pattern"),
							runGum("style", "--width", "50", "--foreground", "212", "--bold", "Target Directory"))
						fmt.Println(header)

						// Rules
						for i, rule := range rules {
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
						for i, rule := range rules {
							fmt.Printf("%-30s %-50s\n", fmt.Sprintf("%d. %s", i+1, rule.Pattern), rule.Target)
						}
					}
				}
			}
		},
	}

	rulesAddCmd := &cobra.Command{
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

			// First, ask if the user wants to use a template
			createMethod := runGumChoose(
				"Create custom rule",
				"Use rule template",
				"Create file type rule",
				"Create directory-specific rule")

			switch createMethod {
			case "Use rule template":
				createRuleFromTemplate()
				return

			case "Create file type rule":
				createFileTypeRule()
				return

			case "Create directory-specific rule":
				createDirectorySpecificRule()
				return

				// For "Create custom rule", continue with the existing flow
			}

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

	rulesRemoveCmd := &cobra.Command{
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

			// First, ask whether to remove a global rule or a directory-specific rule
			if cfg.DirectoryRules != nil && len(cfg.DirectoryRules) > 0 {
				ruleType := runGumChoose("Remove global rule", "Remove directory-specific rule")

				if ruleType == "Remove directory-specific rule" {
					removeDirectorySpecificRule()
					return
				}
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

	rulesCmd.AddCommand(rulesListCmd)
	rulesCmd.AddCommand(rulesAddCmd)
	rulesCmd.AddCommand(rulesRemoveCmd)

	return rulesCmd
}

// createRuleFromTemplate creates a new rule from a template
func createRuleFromTemplate() {
	if cfg == nil {
		cfg = config.New()
	}

	// List available templates
	fmt.Println("📋 Choose a template:")

	var options []string
	for _, template := range cfg.Templates {
		options = append(options, fmt.Sprintf("%s - %s", template.Name, template.Description))
	}

	selection := runGumChoose(options...)
	if selection == "" {
		fmt.Println("🛑 Cancelled")
		return
	}

	// Extract template name from selection
	templateName := strings.SplitN(selection, " - ", 2)[0]

	// Find the selected template
	var selectedTemplate *struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Rules       []struct {
			Pattern string `yaml:"pattern"`
			Target  string `yaml:"target"`
		} `yaml:"rules"`
	}

	for i, template := range cfg.Templates {
		if template.Name == templateName {
			selectedTemplate = &cfg.Templates[i]
			break
		}
	}

	if selectedTemplate == nil {
		fmt.Println("❌ Template not found")
		return
	}

	// Show the rules in the template
	fmt.Printf("📋 Rules in template '%s':\n", selectedTemplate.Name)
	for _, rule := range selectedTemplate.Rules {
		fmt.Printf("  • %s -> %s\n", rule.Pattern, rule.Target)
	}

	// Choose base directory
	fmt.Println("\n📂 Choose base directory for targets:")

	baseDir := runGumFile("--directory")
	if baseDir == "" {
		baseDir = getHomeDir()
	}

	// Confirm
	fmt.Printf("📌 This will add %d rules from the '%s' template\n",
		len(selectedTemplate.Rules), selectedTemplate.Name)
	if !runGumConfirm("Add these rules?") {
		fmt.Println("🛑 Cancelled")
		return
	}

	// Add rules from template
	added := 0
	for _, rule := range selectedTemplate.Rules {
		// Build the target path by combining the base directory with the template target
		target := filepath.Join(baseDir, rule.Target)

		// Add the rule
		cfg.Rules = append(cfg.Rules, struct {
			Pattern string `yaml:"pattern"`
			Target  string `yaml:"target"`
		}{
			Pattern: rule.Pattern,
			Target:  target,
		})
		added++
	}

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

	fmt.Printf("✅ Added %d rules from template '%s'!\n", added, selectedTemplate.Name)
}

// createFileTypeRule creates a rule based on file type categories
func createFileTypeRule() {
	if cfg == nil {
		cfg = config.New()
	}

	// List file type categories
	fmt.Println("📋 Choose a file type category:")

	var categories []string
	for category := range fileTypeCategories {
		categories = append(categories, category)
	}

	selectedCategory := runGumChoose(categories...)
	if selectedCategory == "" {
		fmt.Println("🛑 Cancelled")
		return
	}

	// Show the patterns in the category
	fmt.Printf("📋 Patterns in category '%s':\n", selectedCategory)
	for _, pattern := range fileTypeCategories[selectedCategory] {
		fmt.Printf("  • %s\n", pattern)
	}

	// Let the user choose a specific pattern or use all
	fmt.Println("\n🔍 Choose patterns to include:")
	patternChoice := runGumChoose("Use all patterns", "Select specific pattern")

	var patterns []string
	if patternChoice == "Use all patterns" {
		patterns = fileTypeCategories[selectedCategory]
	} else {
		pattern := runGumChoose(fileTypeCategories[selectedCategory]...)
		if pattern == "" {
			fmt.Println("🛑 Cancelled")
			return
		}
		patterns = []string{pattern}
	}

	// Choose target directory
	fmt.Println("\n📂 Choose target directory:")

	target := runGumFile("--directory")
	if target == "" {
		fmt.Println("❌ Target directory cannot be empty")
		return
	}

	// Confirm
	if len(patterns) == 1 {
		fmt.Printf("📌 New rule: Files matching '%s' will be moved to '%s'\n",
			patterns[0], target)
	} else {
		fmt.Printf("📌 %d new rules for %s files will be added\n",
			len(patterns), selectedCategory)
	}

	if !runGumConfirm("Add these rules?") {
		fmt.Println("🛑 Cancelled")
		return
	}

	// Add rules
	added := 0
	for _, pattern := range patterns {
		cfg.Rules = append(cfg.Rules, struct {
			Pattern string `yaml:"pattern"`
			Target  string `yaml:"target"`
		}{
			Pattern: pattern,
			Target:  target,
		})
		added++
	}

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

	fmt.Printf("✅ Added %d rules for %s files!\n", added, selectedCategory)
}

// createDirectorySpecificRule creates rules that only apply to specific directories
func createDirectorySpecificRule() {
	if cfg == nil {
		cfg = config.New()
	}

	// Choose the directory to create rules for
	fmt.Println("📂 Choose directory to create rules for:")

	directory := runGumFile("--directory")
	if directory == "" {
		fmt.Println("❌ Directory cannot be empty")
		return
	}

	// Ensure the DirectoryRules map is initialized
	if cfg.DirectoryRules == nil {
		cfg.DirectoryRules = make(map[string][]struct {
			Pattern string `yaml:"pattern"`
			Target  string `yaml:"target"`
		})
	}

	// Choose how to create the rule
	createMethod := runGumChoose(
		"Create custom rule",
		"Use rule template",
		"Create file type rule")

	var newRules []struct {
		Pattern string `yaml:"pattern"`
		Target  string `yaml:"target"`
	}

	switch createMethod {
	case "Use rule template":
		// List available templates
		fmt.Println("📋 Choose a template:")

		var options []string
		for _, template := range cfg.Templates {
			options = append(options, fmt.Sprintf("%s - %s", template.Name, template.Description))
		}

		selection := runGumChoose(options...)
		if selection == "" {
			fmt.Println("🛑 Cancelled")
			return
		}

		// Extract template name from selection
		templateName := strings.SplitN(selection, " - ", 2)[0]

		// Find the selected template
		var selectedTemplate *struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
			Rules       []struct {
				Pattern string `yaml:"pattern"`
				Target  string `yaml:"target"`
			} `yaml:"rules"`
		}

		for i, template := range cfg.Templates {
			if template.Name == templateName {
				selectedTemplate = &cfg.Templates[i]
				break
			}
		}

		if selectedTemplate == nil {
			fmt.Println("❌ Template not found")
			return
		}

		// Show the rules in the template
		fmt.Printf("📋 Rules in template '%s':\n", selectedTemplate.Name)
		for _, rule := range selectedTemplate.Rules {
			fmt.Printf("  • %s -> %s\n", rule.Pattern, rule.Target)
		}

		// Choose base directory for targets
		fmt.Println("\n📂 Choose base directory for targets:")

		baseDir := runGumFile("--directory")
		if baseDir == "" {
			baseDir = getHomeDir()
		}

		// Add rules from template
		for _, rule := range selectedTemplate.Rules {
			// Build the target path by combining the base directory with the template target
			target := filepath.Join(baseDir, rule.Target)

			newRules = append(newRules, struct {
				Pattern string `yaml:"pattern"`
				Target  string `yaml:"target"`
			}{
				Pattern: rule.Pattern,
				Target:  target,
			})
		}

	case "Create file type rule":
		// List file type categories
		fmt.Println("📋 Choose a file type category:")

		var categories []string
		for category := range fileTypeCategories {
			categories = append(categories, category)
		}

		selectedCategory := runGumChoose(categories...)
		if selectedCategory == "" {
			fmt.Println("🛑 Cancelled")
			return
		}

		// Let the user choose a specific pattern or use all
		fmt.Println("\n🔍 Choose patterns to include:")
		patternChoice := runGumChoose("Use all patterns", "Select specific pattern")

		var patterns []string
		if patternChoice == "Use all patterns" {
			patterns = fileTypeCategories[selectedCategory]
		} else {
			pattern := runGumChoose(fileTypeCategories[selectedCategory]...)
			if pattern == "" {
				fmt.Println("🛑 Cancelled")
				return
			}
			patterns = []string{pattern}
		}

		// Choose target directory
		fmt.Println("\n📂 Choose target directory:")

		target := runGumFile("--directory")
		if target == "" {
			fmt.Println("❌ Target directory cannot be empty")
			return
		}

		// Add rules
		for _, pattern := range patterns {
			newRules = append(newRules, struct {
				Pattern string `yaml:"pattern"`
				Target  string `yaml:"target"`
			}{
				Pattern: pattern,
				Target:  target,
			})
		}

	default: // "Create custom rule"
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

		newRules = append(newRules, struct {
			Pattern string `yaml:"pattern"`
			Target  string `yaml:"target"`
		}{
			Pattern: pattern,
			Target:  target,
		})
	}

	// Confirm
	fmt.Printf("📌 %d new rules will be added for directory: %s\n",
		len(newRules), directory)
	if !runGumConfirm("Add these rules?") {
		fmt.Println("🛑 Cancelled")
		return
	}

	// Add rules to the directory
	cfg.DirectoryRules[directory] = append(cfg.DirectoryRules[directory], newRules...)

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

	fmt.Printf("✅ Added %d directory-specific rules for %s!\n", len(newRules), directory)
}

// removeDirectorySpecificRule removes a directory-specific rule
func removeDirectorySpecificRule() {
	if cfg == nil || cfg.DirectoryRules == nil || len(cfg.DirectoryRules) == 0 {
		fmt.Println("❓ No directory-specific rules defined to remove.")
		return
	}

	// List the directories
	fmt.Println("📂 Select a directory:")
	var dirOptions []string
	for dir := range cfg.DirectoryRules {
		dirOptions = append(dirOptions, dir)
	}

	selectedDir := runGumChoose(dirOptions...)
	if selectedDir == "" {
		fmt.Println("🛑 Cancelled")
		return
	}

	rules := cfg.DirectoryRules[selectedDir]
	if len(rules) == 0 {
		fmt.Printf("❓ No rules defined for directory: %s\n", selectedDir)
		return
	}

	// List the rules for the selected directory
	fmt.Printf("📝 Rules for directory: %s\n", selectedDir)
	var options []string
	for i, rule := range rules {
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
	if index >= 0 && index < len(rules) {
		if len(rules) == 1 {
			// If this is the last rule, remove the directory entry
			delete(cfg.DirectoryRules, selectedDir)
		} else {
			// Remove the rule
			cfg.DirectoryRules[selectedDir] = append(
				rules[:index],
				rules[index+1:]...)
		}

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
	} else {
		fmt.Println("❌ Invalid rule index")
	}
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

	cfg.Rules = append(cfg.Rules[:index], cfg.Rules[index+1:]...)
}

func saveConfigToFile(cfg *config.Config, path string) error {
	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save the config
	return config.SaveConfig(cfg, path)
}
