package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/toasty/sortd/internal/analysis"
	"github.com/toasty/sortd/internal/config"
	"github.com/toasty/sortd/internal/log"
	"github.com/toasty/sortd/internal/organize"
)

func main() {
	// Command line flags
	watchMode := flag.Bool("watch", false, "Watch directory for changes")
	dryRun := flag.Bool("dry-run", false, "Show what would be done without making changes")
	configFile := flag.String("config", "", "Path to config file (default: ~/.config/sortd/config.yaml)")
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	// Setup logging
	log.SetDebug(*debug)

	// Load configuration
	var cfg *config.Config
	var err error
	if *configFile != "" {
		cfg, err = config.LoadConfigFile(*configFile)
	} else {
		cfg, err = config.LoadConfig()
	}
	if err != nil {
		log.Error("Failed to load config: %v", err)
		os.Exit(1)
	}

	// Override dry-run from command line if specified
	if *dryRun {
		cfg.Settings.DryRun = true
	}

	// Create engines
	analysisEngine := analysis.New()
	organizeEngine := organize.New()

	// Add patterns from config
	for _, pattern := range cfg.Organize.Patterns {
		organizeEngine.AddPattern(pattern)
	}

	// Get target directory
	var targetDir string
	if args := flag.Args(); len(args) > 0 {
		targetDir = args[0]
	} else {
		targetDir = "."
	}

	targetDir, err = filepath.Abs(targetDir)
	if err != nil {
		log.Error("Invalid directory path: %v", err)
		os.Exit(1)
	}

	if cfg.Settings.DryRun {
		log.Info("Running in dry-run mode - no changes will be made")
	}

	if *watchMode {
		log.Info("Watching directory: %s", targetDir)
		// TODO: Implement watch mode
		fmt.Println("Watch mode not implemented yet")
		os.Exit(1)
	} else {
		// Scan directory
		log.Info("Scanning directory: %s", targetDir)
		files, err := analysisEngine.ScanDirectory(targetDir)
		if err != nil {
			log.Error("Failed to scan directory: %v", err)
			os.Exit(1)
		}

		// Display file information
		for _, file := range files {
			fmt.Println(file.String())
		}

		// Convert to string slice for organization
		filePaths := make([]string, len(files))
		for i, f := range files {
			filePaths[i] = f.Path
		}

		// Organize files
		if err := organizeEngine.OrganizeByPatterns(filePaths); err != nil {
			log.Error("Failed to organize files: %v", err)
			os.Exit(1)
		}

		log.Info("Organization complete")
	}
}
