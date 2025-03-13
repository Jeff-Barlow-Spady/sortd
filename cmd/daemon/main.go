package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"sortd/internal/config"
	"sortd/internal/watch"
)

func main() {
	fmt.Println("🚀 Starting sortd daemon...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("⚠️ Warning: %v\n", err)
		fmt.Println("💡 Using default settings. Run 'sortd setup' to configure.")
		cfg = config.New()
	}

	// Check if watch mode is enabled
	if !cfg.WatchMode.Enabled {
		fmt.Println("❌ Watch mode is not enabled in configuration.")
		fmt.Println("💡 Run 'sortd setup' to enable watch mode.")
		os.Exit(1)
	}

	// Get directories to watch
	var dirsToWatch []string
	if len(cfg.Directories.Watch) > 0 {
		dirsToWatch = cfg.Directories.Watch
	} else if cfg.Directories.Default != "" {
		dirsToWatch = []string{cfg.Directories.Default}
	} else {
		fmt.Println("❌ No directories configured to watch.")
		fmt.Println("💡 Run 'sortd setup' to configure watch directories.")
		os.Exit(1)
	}

	// Validate directories
	var validDirs []string
	for _, dir := range dirsToWatch {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			fmt.Printf("⚠️ Warning: Cannot resolve path %s: %v\n", dir, err)
			continue
		}

		info, err := os.Stat(absDir)
		if err != nil {
			fmt.Printf("⚠️ Warning: Cannot access %s: %v\n", absDir, err)
			continue
		}
		if !info.IsDir() {
			fmt.Printf("⚠️ Warning: %s is not a directory\n", absDir)
			continue
		}
		validDirs = append(validDirs, absDir)
	}

	if len(validDirs) == 0 {
		fmt.Println("❌ No valid directories to watch.")
		os.Exit(1)
	}

	// Determine interval
	interval := 300 // Default 5 minutes
	if cfg.WatchMode.Interval > 0 {
		interval = cfg.WatchMode.Interval
	}

	// Create watcher
	watcher, err := watch.NewWatcher(cfg, validDirs, time.Duration(interval)*time.Second)
	if err != nil {
		fmt.Printf("❌ Error creating watcher: %v\n", err)
		os.Exit(1)
	}

	// Start watching
	fmt.Printf("👀 Watching %d directories (checking every %d seconds)\n", len(validDirs), interval)
	go watcher.Start()

	// Setup signal catching for clean shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a signal
	<-sigChan

	fmt.Println("\n🛑 Stopping daemon...")
	watcher.Stop()
	fmt.Println("✅ Daemon stopped")
}
