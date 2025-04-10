package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sortd/internal/organize"

	"github.com/spf13/cobra"
)

// filterFilesByCategory filters a list of files by category (file extension)
func filterFilesByCategory(files []string, category string) []string {
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
	return filteredFiles
}

// selectFilesInteractive shows an interactive file selection UI
func selectFilesInteractive(files []string) []string {
	// Show UI for selecting files
	fmt.Println(" Select files to organize (space to select, enter to confirm):")

	// Calculate relative paths to make them more readable
	var displayFiles []string
	for _, file := range files {
		// Get base name to keep it simple
		displayFiles = append(displayFiles, filepath.Base(file))
	}

	// Use the dedicated multi-choice function
	selections := runGumMultiChoose(displayFiles...)

	// Find the actual paths from the selections
	var selectedFiles []string
	for _, selection := range strings.Split(selections, "\n") {
		if selection == "" {
			continue
		}

		// Find the full path matching this selection
		for _, file := range files {
			if filepath.Base(file) == selection {
				selectedFiles = append(selectedFiles, file)
				break
			}
		}
	}

	return selectedFiles
}

// printOrganizePlan prints the organization plan for dry run mode
func printOrganizePlan(organizer *organize.Engine, files []string) {
	fmt.Printf("Would organize %d files:\n", len(files))
	for _, file := range files {
		fmt.Printf("  %s would be moved\n", file)
	}
}

// NewOrganizeCmd creates the organize command
func NewOrganizeCmd() *cobra.Command {
	var (
		dryRun         bool
		directory      string
		verbose        bool
		recursive      bool
		nonInteractive bool
	)

	cmd := &cobra.Command{
		Use:   "organize [directory|file]",
		Short: "Organize files in a directory",
		Long:  `Organize files according to your rules, with a fun interactive interface.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Set non-interactive mode in environment for consistent access across functions
			if nonInteractive {
				os.Setenv("SORTD_NON_INTERACTIVE", "true")
			}

			// Determine target path
			targetPath, err := determineTargetPath(args, directory)
			if err != nil {
				return err
			}

			// Set verbose mode in test mode
			if os.Getenv("TESTMODE") == "true" {
				verbose = true
			}

			// Validate path exists
			info, err := os.Stat(targetPath)
			if err != nil {
				return fmt.Errorf("error accessing path: %w", err)
			}

			// Setup engine using the organize package directly
			organizeEngine := organize.NewWithConfig(cfg)

			// Override dry run if specified
			if dryRun {
				organizeEngine.SetDryRun(true)
			}

			// Handle organization based on whether the target is a file or directory
			if !info.IsDir() {
				return organizeSingleFile(ctx, organizeEngine, targetPath, verbose)
			}

			return organizeDirectory(ctx, organizeEngine, targetPath, recursive, verbose)
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be organized without making changes")
	cmd.Flags().StringVarP(&directory, "directory", "d", "", "Directory to organize (overrides positional argument)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively organize subdirectories")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Run in non-interactive mode (no user prompts)")

	return cmd
}

// determineTargetPath decides which path to use for organization
func determineTargetPath(args []string, flagDirectory string) (string, error) {
	// Check command line arguments first
	if len(args) > 0 {
		return args[0], nil
	}

	// Check if directory flag is provided
	if flagDirectory != "" {
		return flagDirectory, nil
	}

	// Check config default directory
	if cfg != nil && cfg.Directories.Default != "" {
		return cfg.Directories.Default, nil
	}

	// If in test mode, use current directory
	if os.Getenv("TESTMODE") == "true" {
		dir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("error getting current directory: %w", err)
		}
		return dir, nil
	}

	// Use Gum to let the user choose a directory
	fmt.Println(" Choose a directory to organize:")
	targetPath := runGumFile("--directory")
	if targetPath == "" {
		return "", fmt.Errorf("no directory selected")
	}

	return targetPath, nil
}

// organizeSingleFile organizes a single file according to configured patterns
func organizeSingleFile(ctx context.Context, engine *organize.Engine, filePath string, verbose bool) error {
	if verbose {
		fmt.Printf(" Processing single file: %s\n", filePath)

		// Ensure absolute path for better clarity
		absPath, err := filepath.Abs(filePath)
		if err == nil {
			fmt.Printf(" Absolute path: %s\n", absPath)
			filePath = absPath
		}

		// Print configuration info
		if cfg != nil {
			fmt.Printf(" Collision strategy: %s\n", cfg.Settings.Collision)
			for i, pattern := range cfg.Organize.Patterns {
				fmt.Printf(" Pattern %d: %s -> %s\n", i+1, pattern.Match, pattern.Target)
			}
		}
	} else {
		fmt.Printf(" Note: %s is a file, not a directory\n", filePath)
	}

	// Find matching pattern
	destDir, matched := findMatchingPattern(filePath)

	// If no pattern matched, inform the user
	if !matched {
		return fmt.Errorf("no pattern matched for file: %s", filePath)
	}

	// Build destination path
	var fullDestDir string
	if filepath.IsAbs(destDir) {
		fullDestDir = destDir
	} else {
		// If relative, make it relative to parent directory of the file
		parentDir := filepath.Dir(filePath)
		fullDestDir = filepath.Join(parentDir, destDir)
	}

	destPath := filepath.Join(fullDestDir, filepath.Base(filePath))

	// Check for dry run
	if engine.IsDryRun() {
		fmt.Printf(" Would move: %s -> %s\n", filePath, destPath)
		return nil
	}

	// Perform the move
	fmt.Printf(" Moving: %s -> %s\n", filePath, destPath)
	if err := engine.MoveFile(filePath, destPath); err != nil {
		return fmt.Errorf("error moving file: %w", err)
	}

	fmt.Println(successText(" File organized successfully"))
	return nil
}

// organizeDirectory organizes all files in a directory
func organizeDirectory(ctx context.Context, engine *organize.Engine, dirPath string, recursive bool, verbose bool) error {
	fmt.Printf(" Organizing directory: %s\n", dirPath)

	// Find files to organize
	var files []string
	var err error

	if recursive {
		files, err = findFilesRecursive(dirPath)
	} else {
		files, err = findFiles(dirPath)
	}

	if err != nil {
		return fmt.Errorf("error finding files: %w", err)
	}

	fmt.Printf(" Found %d files to organize\n", len(files))

	// Allow interactive selection if not in test mode or non-interactive mode
	if os.Getenv("TESTMODE") != "true" && !isNonInteractive() && !recursive {
		files = selectFilesInteractive(files)
		fmt.Printf(" Selected %d files to organize\n", len(files))
	} else if isNonInteractive() {
		fmt.Println(" Running in non-interactive mode, processing all files")
	}

	// Check for dry run
	if engine.IsDryRun() {
		printOrganizePlan(engine, files)
		return nil
	}

	// Perform organization
	err = engine.OrganizeByPatterns(files)
	if err != nil {
		return fmt.Errorf("error organizing files: %w", err)
	}

	// Print results
	fmt.Printf(" Organized %d files\n", len(files))
	if verbose {
		for i, file := range files {
			fmt.Printf(" %d. Organized: %s\n", i+1, file)
		}
	}

	return nil
}

// findMatchingPattern finds a pattern that matches the given file
func findMatchingPattern(filePath string) (string, bool) {
	if cfg == nil || len(cfg.Organize.Patterns) == 0 {
		return "", false
	}

	for _, pattern := range cfg.Organize.Patterns {
		isMatch, err := filepath.Match(pattern.Match, filepath.Base(filePath))
		if err == nil && isMatch {
			return pattern.Target, true
		}
	}

	return "", false
}

// findFilesRecursive finds all files in a directory and its subdirectories
func findFilesRecursive(root string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// Skip the root directory itself
		if path == root {
			return nil
		}

		// Skip errors and directories
		if err != nil || info.IsDir() {
			return nil
		}

		// Add the file to our list
		files = append(files, path)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return files, nil
}

// analyzeFileTypes returns a list of unique file extensions found in the files
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
