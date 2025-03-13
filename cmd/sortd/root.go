package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"sortd/internal/config"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "sortd",
		Short: "A fun and powerful file organization tool",
		Long: `
	::######:::'#######::'########::'########:'########::
	'##... ##:'##.... ##: ##.... ##:... ##..:: ##.... ##:
	'##:::..:: ##:::: ##: ##:::: ##:::: ##:::: ##:::: ##:
	. ######:: ##:::: ##: ########::::: ##:::: ##:::: ##:
	:..... ##: ##:::: ##: ##.. ##:::::: ##:::: ##:::: ##:
	'##::: ##: ##:::: ##: ##::. ##::::: ##:::: ##:::: ##:
	. ######::. #######:: ##:::. ##:::: ##:::: ########::
	:......::::.......:::..:::::..:::::..:::::........:::

Sortd helps you organize files in a smart, fun way!
		`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Check if we're in a test environment
			if os.Getenv("TESTMODE") == "true" {
				return
			}

			// Check if gum is installed
			_, err := exec.LookPath("gum")
			if err != nil {
				fmt.Println("⚠️ Gum is not installed! Some interactive features won't work.")
				fmt.Println("💡 Install Gum from https://github.com/charmbracelet/gum")
			}

			// Load config
			var configErr error
			if cfgFile != "" {
				cfg, configErr = config.LoadConfigFile(cfgFile)
			} else {
				cfg, configErr = config.LoadConfig()
			}

			if configErr != nil {
				fmt.Printf("⚠️ Warning: %v\n", configErr)
				fmt.Println("💡 Using default settings. Run 'sortd setup' to configure.")
				cfg = config.New()
			}
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/sortd/config.yaml)")

	// Add subcommands
	rootCmd.AddCommand(NewSetupCmd())
	rootCmd.AddCommand(NewOrganizeCmd())
	rootCmd.AddCommand(NewRulesCmd())
	rootCmd.AddCommand(NewWatchCmd())
	rootCmd.AddCommand(NewDaemonCmd())

	return rootCmd
}

// Helper functions to run Gum commands
func runGum(args ...string) string {
	cmd := exec.Command("gum", args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		// Fallback for errors
		return ""
	}
	return strings.TrimSpace(string(output))
}

func runGumInput(prompt, defaultValue string) string {
	return runGum("input", "--placeholder", prompt, "--value", defaultValue)
}

func runGumConfirm(prompt string) bool {
	result := runGum("confirm", prompt)
	return result == ""
}

func runGumChoose(options ...string) string {
	args := []string{"choose"}
	args = append(args, options...)
	return runGum(args...)
}

func runGumFile(args ...string) string {
	args = append([]string{"file"}, args...)
	return runGum(args...)
}

func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

func parseInterval(input string) int {
	var interval int
	fmt.Sscanf(input, "%d", &interval)
	if interval <= 0 {
		return 300 // Default to 5 minutes
	}
	return interval
}
