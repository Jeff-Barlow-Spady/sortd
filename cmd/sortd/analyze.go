package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// NewAnalyzeCmd creates the analyze command
func NewAnalyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Perform advanced file analysis",
		Long:  `Analyze files and directories using advanced semantic understanding and classification.`,
	}

	// Add subcommands
	cmd.AddCommand(NewAnalyzeContentCmd())
	cmd.AddCommand(NewAnalyzeDuplicatesCmd())
	cmd.AddCommand(NewAnalyzeGroupCmd())

	return cmd
}

// NewAnalyzeContentCmd creates the content analysis command
func NewAnalyzeContentCmd() *cobra.Command {
	var recursive bool

	cmd := &cobra.Command{
		Use:   "content [path]",
		Short: "Analyze file content semantically",
		Long:  `Use semantic analysis to understand and categorize file contents.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(primaryText("🔍 Content Analysis"))

			// Get the path to analyze
			path := "." // Default path
			var gumAvailable bool
			if _, err := exec.LookPath("gum"); err == nil {
				gumAvailable = true
			}

			if len(args) > 0 {
				path = args[0]
			} else if gumAvailable {
				// Use Gum to interactively select a path if none provided
				fmt.Println(infoText("Select a file or directory to analyze:"))
				pathChoice := runGumChoose("Enter path manually", "Browse filesystem")
				if pathChoice == "Browse filesystem" {
					path = runGumFile("") // Allows selecting file or directory
				} else {
					path = runGumInput("Enter path to analyze", ".")
				}
				if path == "" {
					fmt.Println(infoText("No path selected. Exiting."))
					return // Exit if no path was chosen
				}
			} else if len(args) == 0 { // No args and no gum
				fmt.Println(errorText("Cannot run analysis without a path or interactive selection."))
				fmt.Println(infoText("Please provide a path argument or install gum for interactive selection."))
				fmt.Println(infoText("Install Gum from https://github.com/charmbracelet/gum"))
				return
			}

			// Setup signal handling for clean exits
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Println(infoText("\nAnalysis cancelled."))
				os.Exit(0)
			}()

			// Clean and validate path
			path = filepath.Clean(path)
			info, err := os.Stat(path)
			if err != nil {
				fmt.Println(errorText("Error accessing path: " + err.Error()))
				return
			}

			// Show analysis options if interactive
			if gumAvailable {
				fmt.Println(infoText("Select analysis type:"))
				analysisType := runGumChoose(
					"Basic content detection",
					"Advanced semantic analysis",
					"Topic modeling",
					"Classification",
				)

				if analysisType == "" {
					fmt.Println(infoText("Analysis cancelled"))
					return
				}

				fmt.Printf(successText("Selected: %s\n"), analysisType)
			}

			// Show a message about the upcoming feature
			fmt.Println(infoText("Analyzing path: " + path))
			if info.IsDir() {
				fmt.Println(infoText("Directory analysis" + (map[bool]string{true: " (recursive)", false: ""})[recursive]))
			} else {
				fmt.Println(infoText("File analysis"))
			}

			// Display a placeholder for the upcoming functionality
			fmt.Println(warningText("Advanced semantic analysis is under development."))
			fmt.Println(infoText("This feature will be available in an upcoming release."))

			runGum("style",
				"--foreground", "212",
				"--border", "rounded",
				"--border-foreground", "212",
				"--padding", "1",
				"--width", "70",
				"Upcoming Analysis Features:\n\n"+
					"• Natural language understanding for documents\n"+
					"• Image content recognition\n"+
					"• Audio and video transcription\n"+
					"• Semantic grouping of related files\n"+
					"• Content-based organization rules")
		},
	}

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Analyze directories recursively")

	return cmd
}

// NewAnalyzeDuplicatesCmd creates the duplicate analysis command
func NewAnalyzeDuplicatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "duplicates [path]",
		Short: "Find duplicate files",
		Long:  `Scan for duplicate files using content hash comparison.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(primaryText("🔍 Duplicate File Detection"))

			// Get the path to analyze
			path := "." // Default path
			var gumAvailable bool
			if _, err := exec.LookPath("gum"); err == nil {
				gumAvailable = true
			}

			if len(args) > 0 {
				path = args[0]
			} else if gumAvailable {
				// Use Gum to interactively select a path if none provided
				fmt.Println(infoText("Select a directory to scan for duplicates:"))
				pathChoice := runGumChoose("Enter path manually", "Browse directories")
				if pathChoice == "Browse directories" {
					path = runGumFile("--directory") // Only allow directory selection
				} else {
					path = runGumInput("Enter directory path", ".")
				}
				if path == "" {
					fmt.Println(infoText("No path selected. Exiting."))
					return // Exit if no path was chosen
				}
			} else if len(args) == 0 { // No args and no gum
				fmt.Println(errorText("Cannot run duplicate detection without a path or interactive selection."))
				fmt.Println(infoText("Please provide a path argument or install gum for interactive selection."))
				fmt.Println(infoText("Install Gum from https://github.com/charmbracelet/gum"))
				return
			}

			// Add placeholder interactive options if gum is available
			var hashingAlgorithm, minSize string
			if gumAvailable {
				fmt.Println(infoText("Select duplicate detection options:"))
				hashingAlgorithm = runGumChoose("SHA256", "SHA1", "MD5")
				minSize = runGumInput("Minimum file size (e.g., 1KB, 1MB, leave blank for all)", "1KB")
				fmt.Printf(successText("Options set: Hash=%s, MinSize=%s\n"), hashingAlgorithm, minSize)
			}

			// Show a message about the upcoming feature
			fmt.Println(infoText("Scanning for duplicates in: " + path))
			fmt.Println(warningText("Duplicate detection is under development."))
			fmt.Println(infoText("This feature will be available in an upcoming release."))

			// If demo mode, show a simulated duplicate analysis
			if len(args) > 1 && (args[1] == "demo" || args[1] == "test") {
				fmt.Println(infoText("\nRunning demonstration of duplicate analysis..."))

				// Create a channel to handle interruption signals
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

				// Run the demo in a separate goroutine
				done := make(chan bool)
				go func() {
					// Simulate scanning
					fmt.Println(infoText("Scanning files..."))
					for i := 0; i < 5; i++ {
						select {
						case <-sigChan:
							fmt.Println(infoText("\nScan cancelled"))
							close(done)
							return
						default:
							time.Sleep(500 * time.Millisecond)
							fmt.Printf(".")
						}
					}

					// Show mock results
					fmt.Println("\n\n" + successText("Analysis complete!"))
					fmt.Println(infoText("Files scanned: 247"))
					fmt.Println(infoText("Duplicate sets found: 3"))

					// Display fake duplicate groups
					duplicateGroups := [][]string{
						{
							"Documents/report-v1.pdf",
							"Documents/archive/report.pdf",
							"Downloads/report_copy.pdf",
						},
						{
							"Pictures/vacation/beach.jpg",
							"Pictures/favorites/beach_edited.jpg",
						},
						{
							"Downloads/installer.exe",
							"Applications/installer.exe",
						},
					}

					for i, group := range duplicateGroups {
						fmt.Printf("\n%s Duplicate set #%d: %d files, %s\n",
							errorText("⚠"), i+1, len(group), primaryText("4.2 MB wasted"))

						for _, file := range group {
							fmt.Printf("  %s\n", file)
						}
					}

					// Show action options
					fmt.Println("\n" + infoText("In the full version, you'll be able to:"))
					fmt.Println("• Automatically delete duplicates")
					fmt.Println("• Keep files based on rules (newest, original location, etc.)")
					fmt.Println("• Create symbolic links to save space")

					close(done)
				}()

				<-done
			}
		},
	}

	return cmd
}

// NewAnalyzeGroupCmd creates the group analysis command
func NewAnalyzeGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group [path]",
		Short: "Group files by similarity",
		Long:  `Group files using semantic similarity and content analysis.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(primaryText("🔍 Semantic Grouping"))

			// Get the path to analyze
			path := "." // Default path
			var gumAvailable bool
			if _, err := exec.LookPath("gum"); err == nil {
				gumAvailable = true
			}

			if len(args) > 0 {
				path = args[0]
			} else if gumAvailable {
				// Use Gum to interactively select a path if none provided
				fmt.Println(infoText("Select a directory to group files in:"))
				pathChoice := runGumChoose("Enter path manually", "Browse directories")
				if pathChoice == "Browse directories" {
					path = runGumFile("--directory") // Only allow directory selection
				} else {
					path = runGumInput("Enter directory path", ".")
				}
				if path == "" {
					fmt.Println(infoText("No path selected. Exiting."))
					return // Exit if no path was chosen
				}
			} else if len(args) == 0 { // No args and no gum
				fmt.Println(errorText("Cannot run semantic grouping without a path or interactive selection."))
				fmt.Println(infoText("Please provide a path argument or install gum for interactive selection."))
				fmt.Println(infoText("Install Gum from https://github.com/charmbracelet/gum"))
				return
			}

			// Add placeholder interactive options if gum is available
			var similarityThreshold, groupingMethod string
			if gumAvailable {
				fmt.Println(infoText("Select semantic grouping options:"))
				similarityThreshold = runGumInput("Similarity threshold (0.0-1.0)", "0.8")
				groupingMethod = runGumChoose("Content", "Filename", "Metadata")
				fmt.Printf(successText("Options set: Threshold=%s, Method=%s\n"), similarityThreshold, groupingMethod)
			}

			// Show a message about the upcoming feature
			fmt.Println(infoText("Semantic grouping of files in: " + path))
			fmt.Println(warningText("Semantic grouping is under development."))
			fmt.Println(infoText("This feature will be available in an upcoming release."))

			runGum("style",
				"--foreground", "212",
				"--border", "rounded",
				"--border-foreground", "212",
				"--padding", "1",
				"--width", "70",
				"Semantic Grouping Features:\n\n"+
					"• Group files by content similarity\n"+
					"• Cluster documents by topic\n"+
					"• Find related media files\n"+
					"• Create smart collections\n"+
					"• Generate organization suggestions")
		},
	}

	return cmd
}
