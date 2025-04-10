package main

import (
	"fmt"
	"os"

	"sortd/internal/analysis"
	"sortd/internal/config"

	"github.com/spf13/cobra"
)

// NewScanCmd creates the scan command
func NewScanCmd() *cobra.Command {
	var jsonOutput bool
	var detailedScan bool

	cmd := &cobra.Command{
		Use:   "scan [file]",
		Short: "Scan a file for basic information",
		Long:  `Scan a file to get its type, size, and other basic metadata.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path := args[0]

			// Check if file exists
			_, err := os.Stat(path)
			if err != nil {
				fmt.Println(errorText(fmt.Sprintf("Error: %v", err)))
				return
			}

			// Create a new analysis engine
			engine := analysis.New()

			// If we have a configuration, set it
			if cfg != nil {
				engine.SetConfig(cfg)
			} else {
				// Create a default config if none is available
				defaultCfg := config.New()
				engine.SetConfig(defaultCfg)
			}

			// Scan the file
			result, err := engine.Scan(path)
			if err != nil {
				fmt.Println(errorText(fmt.Sprintf("Error scanning file: %v", err)))
				return
			}

			// Output the results
			if jsonOutput {
				fmt.Println(result.ToJSON())
			} else {
				fmt.Println(primaryText("File Analysis:"))
				fmt.Println(result.String())
			}
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output results in JSON format")
	cmd.Flags().BoolVarP(&detailedScan, "detailed", "d", false, "Perform a more detailed scan")

	return cmd
}
