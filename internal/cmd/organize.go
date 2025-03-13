package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sortd/internal/organize"

	"github.com/spf13/cobra"
)

var (
	dryRun    bool
	directory string
)

var organizeCmd = &cobra.Command{
	Use:   "organize [directory]",
	Short: "Organize files in a directory",
	Long:  `Organize files according to your rules, with a fun interactive interface.`,
	Run: func(cmd *cobra.Command, args []string) {
		var targetDir string

		// Determine which directory to use
		if len(args) > 0 {
			targetDir = args[0]
		} else if directory != "" {
			targetDir = directory
		} else if cfg != nil && cfg.Directories.Default != "" {
			targetDir = cfg.Directories.Default
		} else {
			// If in test mode, use current directory
			if os.Getenv("TESTMODE") == "true" {
				targetDir, _ = os.Getwd()
			} else {
				// Use Gum to let the user choose a directory
				fmt.Println("📂 Choose a directory to organize:")
				targetDir = runGumFile("--directory")
				if targetDir == "" {
					fmt.Println("❌ No directory selected")
					return
				}
			}
		}

		// Validate directory
		info, err := os.Stat(targetDir)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		if !info.IsDir() {
			fmt.Printf("❌ Error: %s is not a directory\n", targetDir)
			return
		}

		// Setup engine
		organizeEngine := organize.New()

		// Configure the engine
		organizeEngine.SetConfig(cfg)

		// Override dry run if specified
		if dryRun {
			organizeEngine.SetDryRun(true)
		}

		// Scan for files
		fmt.Printf("🔍 Scanning %s for files...\n", targetDir)
		files, err := findFiles(targetDir)
		if err != nil {
			fmt.Printf("❌ Error scanning directory: %v\n", err)
			return
		}

		// Analyze files
		fmt.Println("🧠 Analyzing files...")
		var filesToOrganize []string

		// If we're in test mode, just organize everything
		if os.Getenv("TESTMODE") == "true" {
			filesToOrganize = files
		} else {
			// Use Gum to let the user select files
			if len(files) > 0 {
				// Show a summary
				runGum("style",
					"--foreground", "212",
					"--border", "rounded",
					"--border-foreground", "212",
					"--padding", "1",
					fmt.Sprintf("Found %d files to organize", len(files)))

				// If there are too many files, first show a category filter
				if len(files) > 20 {
					fmt.Println("🔍 That's a lot of files! Let's filter by type first:")
					categories := analyzeFileTypes(files)
					category := runGumChoose(categories...)

					// Filter files by the selected category
					var filteredFiles []string
					for _, file := range files {
						ext := filepath.Ext(file)
						if ext == "" {
							ext = "(no extension)"
						}
						if ext == category || category == "All files" {
							filteredFiles = append(filteredFiles, file)
						}
					}
					files = filteredFiles
				}

				// Now let the user select specific files
				fmt.Println("📋 Select files to organize (space to select, enter to confirm):")
				// Fix paths to be relative to make them more readable
				var displayFiles []string
				for _, file := range files {
					rel, _ := filepath.Rel(targetDir, file)
					displayFiles = append(displayFiles, rel)
				}

				// Convert displayFiles to a variadic argument
				args := append([]string{"choose", "--no-limit"}, displayFiles...)
				selections := runGum(args...)

				// Find the actual paths from the selections
				for _, rel := range strings.Split(selections, "\n") {
					if rel == "" {
						continue
					}
					fullPath := filepath.Join(targetDir, rel)
					filesToOrganize = append(filesToOrganize, fullPath)
				}
			} else {
				fmt.Println("❓ No files found to organize")
				return
			}
		}

		// Confirm organization
		if len(filesToOrganize) == 0 {
			fmt.Println("❓ No files selected for organization")
			return
		}

		fmt.Printf("🗃️ Ready to organize %d files\n", len(filesToOrganize))

		// Give a preview of what will happen
		preview := fmt.Sprintf("Preview: %d files will be organized", len(filesToOrganize))
		if dryRun {
			preview += " (DRY RUN - no changes will be made)"
		}
		runGum("style", preview)

		// Ask for confirmation unless in dry run mode
		if !dryRun {
			fmt.Println("⚠️ This will move files. Continue?")
			if !runGumConfirm("Proceed with organization") {
				fmt.Println("🛑 Operation cancelled")
				return
			}
		}

		// Organize files
		if dryRun {
			fmt.Println("🔄 Simulating organization (dry run)...")
		} else {
			fmt.Println("🔄 Organizing files...")
		}

		// Organize the files
		err = organizeEngine.OrganizeByPatterns(filesToOrganize)
		if err != nil {
			fmt.Printf("❌ Error during organization: %v\n", err)
			return
		}

		// Show results
		fmt.Println("\n✨ Organization Results:")
		if dryRun {
			fmt.Println("🧪 DRY RUN - No files were actually moved")
		} else {
			fmt.Println("🎉 Files have been organized successfully!")
		}
	},
}

func init() {
	rootCmd.AddCommand(organizeCmd)
	organizeCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate organization without moving files")
	organizeCmd.Flags().StringVarP(&directory, "directory", "d", "", "Directory to organize")
}

func runGumFile(args ...string) string {
	args = append([]string{"file"}, args...)
	return runGum(args...)
}

func analyzeFileTypes(files []string) []string {
	extensions := make(map[string]bool)
	extensions["All files"] = true

	for _, file := range files {
		ext := filepath.Ext(file)
		if ext == "" {
			ext = "(no extension)"
		}
		extensions[ext] = true
	}

	result := []string{"All files"}
	for ext := range extensions {
		if ext != "All files" {
			result = append(result, ext)
		}
	}

	return result
}

// findFiles recursively finds all files in a directory
func findFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
