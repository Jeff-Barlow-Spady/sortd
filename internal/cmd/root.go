package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"sortd/internal/config"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
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

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/sortd/config.yaml)")
}
