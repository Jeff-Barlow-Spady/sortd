# Sortd Codebase Simplification Plan

## Core Engine (`internal/core`)

Consolidate the organization engine for use by both CLI and GUI:

```go
// internal/core/engine.go
package core

// Engine provides the core file organization functionality
type Engine struct {
    // Configuration
    Config    *Config
    DryRun    bool
    CreateDirs bool

    // Pattern matching
    Patterns  []Pattern

    // Safety
    BackupEnabled bool
    CollisionStrategy string
}

// OrganizeFile organizes a single file based on patterns
func (e *Engine) OrganizeFile(path string) (string, error) {
    // 1. Find matching pattern
    // 2. Calculate destination
    // 3. Create directories if needed
    // 4. Handle collisions
    // 5. Move the file
    // 6. Return new location
}

// OrganizeDirectory organizes all files in a directory
func (e *Engine) OrganizeDirectory(dir string) ([]Result, error) {
    // 1. Find all files
    // 2. Apply patterns to each
    // 3. Return results
}
```

Pattern matching should be consolidated into a single approach:

```go
// internal/core/pattern.go
package core

type Pattern struct {
    // Basic matching
    Glob      string    // e.g., "*.pdf"
    Target    string    // Destination

    // Enhanced matching (optional)
    Prefixes  []string  // Match these prefixes
    Suffixes  []string  // Match these suffixes
    Keywords  []string  // Content keywords

    // Metadata
    Name      string    // Human-readable name
    Priority  int       // Higher priority patterns applied first
}
```

## 2. Breaking Down Model.go

Split the massive 1344-line `internal/tui/model.go` into:

- `model.go` - Core data structure (< 200 lines)
- `view.go` - UI rendering logic (< 300 lines)
- `controller.go` - User input handling (< 300 lines)
- `file_ops.go` - File operations (< 200 lines)
- `commands.go` - Command implementations (< 200 lines)

## 3. UI Simplification

Create a clean abstraction between business logic and UI:

```go
// internal/ui/common/interface.go
package common

// UI defines common methods that all UI implementations must provide
type UI interface {
    // Initialize the UI
    Initialize() error

    // Run the main UI loop
    Run() error

    // Show error message
    ShowError(message string)

    // Show success message
    ShowSuccess(message string)

    // Ask for confirmation
    Confirm(message string) (bool, error)

    // Show progress
    ShowProgress(operation string, current, total int)
}
```

Implement both TUI and GUI with this interface.

## 4. Shared Configuration

Ensure consistent configuration management across CLI, TUI, and GUI:

```go
// internal/config/config.go
package config

type Config struct {
    // Directories
    Directories struct {
        Default string   `yaml:"default"`
        Watch   []string `yaml:"watch"`
    } `yaml:"directories"`

    // Rules for organization
    Rules []Rule `yaml:"rules"`

    // Settings
    Settings struct {
        DryRun    bool   `yaml:"dry_run"`
        CreateDirs bool  `yaml:"create_dirs"`
        Backup    bool   `yaml:"backup"`
        Collision string `yaml:"collision"` // rename, skip, overwrite
    } `yaml:"settings"`
}
```

## 5. Command Line Interface Simplification

Consolidate the many command files into a cleaner structure:

```go
// cmd/sortd/commands/organize.go
package commands

func OrganizeCmd() *cobra.Command {
    // Implementation
}

// cmd/sortd/commands/watch.go
package commands

func WatchCmd() *cobra.Command {
    // Implementation
}
```

Simplified main.go:

```go
// cmd/sortd/main.go
package main

func main() {
    // Create root command
    rootCmd := &cobra.Command{
        Use:     "sortd",
        Version: version,
    }

    // Add subcommands
    rootCmd.AddCommand(commands.OrganizeCmd())
    rootCmd.AddCommand(commands.WatchCmd())
    rootCmd.AddCommand(commands.TuiCmd())
    rootCmd.AddCommand(commands.GuiCmd())

    // Execute
    rootCmd.Execute()
}
```

## 6. File Tree Component Simplification

Simplify the 573-line file tree component:

```go
// internal/tui/components/file_tree.go
package components

// Separate rendering from data handling
func (f *FileTree) buildVisibleNodes() {
    // Logic to build visible nodes list
}

func (f *FileTree) renderNode(node *TreeNode, isSelected bool) string {
    // Logic to render a single node
}
```

## 7. Style Consolidation

Use a single style definition:

```go
// internal/ui/style/style.go
package style

// Create a consistent theme usable by both TUI and GUI
var Theme = struct {
    PrimaryColor   string
    SecondaryColor string
    ErrorColor     string
    SuccessColor   string

    // Font settings
    HeaderFont     FontSettings
    ContentFont    FontSettings

    // Sizes and spacing
    Padding        int
    Margin         int
}{
    PrimaryColor:   "#5A9",
    SecondaryColor: "#7B61FF",
    ErrorColor:     "#F14E32",
    SuccessColor:   "#73F59F",

    // Other settings...
}
```

This simplified structure preserves all functionality while making the code more maintainable.