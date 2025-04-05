package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sortd/internal/config"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// NewRulesCmd creates the rules command
func NewRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Manage organization rules",
		Long:  `View, add, edit, and remove file organization rules.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Default to listing rules when no subcommand is provided
			listRules()
		},
	}

	// Add subcommands
	cmd.AddCommand(newRulesAddCmd())
	cmd.AddCommand(newRulesListCmd())
	cmd.AddCommand(newRulesRemoveCmd())
	cmd.AddCommand(newRulesTestCmd())

	return cmd
}

// newRulesAddCmd creates the 'rules add' command
func newRulesAddCmd() *cobra.Command {
	var (
		pattern string
		target  string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new organization rule",
		Long:  `Add a new rule for organizing files.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Skip interactive mode in tests
			if os.Getenv("TESTMODE") == "true" {
				if pattern == "" || target == "" {
					fmt.Println(errorText("Pattern and target are required in test mode"))
					return
				}

				// Add the rule to config
				if cfg != nil {
					cfg.Rules = append(cfg.Rules, struct {
						Pattern string `yaml:"pattern"`
						Target  string `yaml:"target"`
					}{
						Pattern: pattern,
						Target:  target,
					})

					if err := config.SaveConfig(cfg); err != nil {
						fmt.Println(errorText(fmt.Sprintf("Error saving rule: %v", err)))
						return
					}

					fmt.Println(successText("Rule added successfully"))
				} else {
					fmt.Println(errorText("No configuration available"))
				}
				return
			}

			// Interactive mode
			if pattern == "" || target == "" {
				fmt.Println(primaryText("📝 New Organization Rule"))
				fmt.Println(infoText("Rules help organize files automatically"))

				// Select rule type
				ruleType := runGumChoose("Simple file pattern", "File extension", "Directory rule")

				switch ruleType {
				case "Simple file pattern":
					pattern = runGumInput("File pattern (e.g., *.jpg, screenshot*.png)", "")
					if pattern == "" {
						fmt.Println(warningText("Empty pattern, cannot add rule"))
						return
					}

				case "File extension":
					ext := runGumInput("File extension (without dot, e.g., pdf, jpg)", "")
					if ext == "" {
						fmt.Println(warningText("Empty extension, cannot add rule"))
						return
					}
					pattern = "*." + strings.TrimPrefix(ext, ".")

				case "Directory rule":
					dirPath := runGumInput("Directory path to organize", "")
					if dirPath == "" {
						fmt.Println(warningText("Empty directory path, cannot add rule"))
						return
					}
					pattern = filepath.Join(dirPath, "*")
				}

				// Get target directory
				fmt.Println(infoText("\nSelect target directory:"))
				targetChoice := runGumChoose("Enter path manually", "Browse directories")

				if targetChoice == "Browse directories" {
					target = runGumFile("--directory")
					if target == "" {
						fmt.Println(warningText("No target directory selected, cannot add rule"))
						return
					}
				} else {
					target = runGumInput("Target directory", "")
					if target == "" {
						fmt.Println(warningText("Empty target directory, cannot add rule"))
						return
					}
				}
			}

			// Confirm the rule
			fmt.Println(infoText("\nNew rule summary:"))
			fmt.Println("  Pattern: " + primaryText(pattern))
			fmt.Println("  Target:  " + primaryText(target))

			if !runGumConfirm("Add this rule?") {
				fmt.Println(infoText("Rule creation cancelled"))
				return
			}

			// Add the rule to config
			if cfg != nil {
				cfg.Rules = append(cfg.Rules, struct {
					Pattern string `yaml:"pattern"`
					Target  string `yaml:"target"`
				}{
					Pattern: pattern,
					Target:  target,
				})

				if err := config.SaveConfig(cfg); err != nil {
					fmt.Println(errorText(fmt.Sprintf("Error saving rule: %v", err)))
					return
				}

				fmt.Println(successText("Rule added successfully"))
			} else {
				fmt.Println(errorText("No configuration available"))
			}
		},
	}

	cmd.Flags().StringVarP(&pattern, "pattern", "p", "", "File pattern (e.g. *.jpg, document*.pdf)")
	cmd.Flags().StringVarP(&target, "target", "t", "", "Target directory for files matching the pattern")

	return cmd
}

// newRulesListCmd creates the 'rules list' command
func newRulesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all organization rules",
		Long:  `Display all configured file organization rules.`,
		Run: func(cmd *cobra.Command, args []string) {
			listRules()
		},
	}
}

// newRulesRemoveCmd creates the 'rules remove' command
func newRulesRemoveCmd() *cobra.Command {
	var index int

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove an organization rule",
		Long:  `Remove a file organization rule by its index.`,
		Run: func(cmd *cobra.Command, args []string) {
			if cfg == nil || len(cfg.Rules) == 0 {
				fmt.Println(infoText("No rules to remove"))
				return
			}

			// First list all rules
			listRules()

			// Get rule index to remove
			var ruleToRemove int
			if index >= 0 {
				ruleToRemove = index
			} else {
				// Interactive selection
				if os.Getenv("TESTMODE") != "true" {
					indexStr := runGumInput("Enter rule index to remove", "")
					var err error
					ruleToRemove, err = strconv.Atoi(indexStr)
					if err != nil || ruleToRemove < 0 || ruleToRemove >= len(cfg.Rules) {
						fmt.Println(errorText(fmt.Sprintf("Invalid rule index: %s", indexStr)))
						return
					}
				} else {
					fmt.Println(errorText("No rule index specified in test mode"))
					return
				}
			}

			// Validate index
			if ruleToRemove < 0 || ruleToRemove >= len(cfg.Rules) {
				fmt.Println(errorText(fmt.Sprintf("Invalid rule index: %d", ruleToRemove)))
				return
			}

			// Get the rule to remove for confirmation
			rulePattern := cfg.Rules[ruleToRemove].Pattern
			ruleTarget := cfg.Rules[ruleToRemove].Target

			// Confirm removal
			if os.Getenv("TESTMODE") != "true" {
				fmt.Println(infoText("\nRule to remove:"))
				fmt.Println("  Pattern: " + primaryText(rulePattern))
				fmt.Println("  Target:  " + primaryText(ruleTarget))

				if !runGumConfirm("Remove this rule?") {
					fmt.Println(infoText("Rule removal cancelled"))
					return
				}
			}

			// Remove the rule
			cfg.Rules = append(cfg.Rules[:ruleToRemove], cfg.Rules[ruleToRemove+1:]...)

			// Save config
			if err := config.SaveConfig(cfg); err != nil {
				fmt.Println(errorText(fmt.Sprintf("Error saving config: %v", err)))
				return
			}

			fmt.Println(successText("Rule removed successfully"))
		},
	}

	cmd.Flags().IntVarP(&index, "index", "i", -1, "Index of the rule to remove")

	return cmd
}

// newRulesTestCmd creates the 'rules test' command
func newRulesTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test [file]",
		Short: "Test rules against a file",
		Long:  `Test which organization rules would apply to a specific file.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println(errorText("No file specified"))
				return
			}

			filePath := args[0]

			// Check if file exists
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				fmt.Println(errorText(fmt.Sprintf("File not found: %s", filePath)))
				return
			}

			// Get just the filename
			fileName := filepath.Base(filePath)

			fmt.Println(primaryText(fmt.Sprintf("Testing rules against file: %s", fileName)))

			// No rules defined
			if cfg == nil || len(cfg.Rules) == 0 {
				fmt.Println(warningText("No rules defined"))
				return
			}

			// Check each rule
			matchFound := false
			for i, rule := range cfg.Rules {
				matched, err := filepath.Match(rule.Pattern, fileName)
				if err != nil {
					fmt.Println(errorText(fmt.Sprintf("Error in pattern %q: %v", rule.Pattern, err)))
					continue
				}

				if matched {
					matchFound = true
					fmt.Println(successText(fmt.Sprintf("✓ Rule %d matched:", i)))
					fmt.Println("  Pattern: " + infoText(rule.Pattern))
					fmt.Println("  Target:  " + infoText(rule.Target))
					fmt.Println("  Result:  " + primaryText(fmt.Sprintf("Would move to %s", filepath.Join(rule.Target, fileName))))
				}
			}

			if !matchFound {
				fmt.Println(warningText("No rules matched this file"))
			}
		},
	}
}

// listRules displays all configured rules
func listRules() {
	if cfg == nil || len(cfg.Rules) == 0 {
		fmt.Println(infoText("No organization rules defined"))
		fmt.Println(infoText("Use 'sortd rules add' to create a rule"))
		return
	}

	fmt.Println(primaryText("📋 Organization Rules"))
	fmt.Println(infoText(fmt.Sprintf("Found %d rules:", len(cfg.Rules))))

	for i, rule := range cfg.Rules {
		fmt.Println("")
		fmt.Println(emphasisText(fmt.Sprintf("Rule %d:", i)))
		fmt.Println("  Pattern: " + infoText(rule.Pattern))
		fmt.Println("  Target:  " + infoText(rule.Target))
	}
}
