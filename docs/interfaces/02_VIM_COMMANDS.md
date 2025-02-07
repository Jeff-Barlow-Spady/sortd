# Vim-Style Command Interface

## Overview
Sortd implements a vim-inspired command system that allows power users to quickly navigate, select, and organize files using keyboard shortcuts and a command palette.

## Command Mode

### 1. Command Palette
```go
type CommandPalette struct {
    active    bool
    input     textinput.Model
    commands  map[string]Command
    history   []string
    matches   []string
}

type Command struct {
    Name        string
    Aliases     []string
    Description string
    Handler     func(args []string) tea.Cmd
}
```

### 2. Core Commands
```go
var Commands = map[string]Command{
    // File Operations
    ":o":      {"organize", []string{":organize"}, "Organize current selection", handleOrganize},
    ":w":      {"watch", []string{":watch"}, "Watch directory for changes", handleWatch},
    ":m":      {"move", []string{":mv"}, "Move selected files", handleMove},
    
    // Navigation
    ":cd":     {"cd", nil, "Change working directory", handleCD},
    ":find":   {"find", []string{":f"}, "Find files by pattern", handleFind},
    ":b":      {"back", []string{":back"}, "Go back to previous directory", handleBack},
    
    // Rules and Patterns
    ":r":      {"rule", []string{":rules"}, "Manage organization rules", handleRules},
    ":p":      {"pattern", []string{":pat"}, "Define new pattern", handlePattern},
    
    // Project Management
    ":project": {"project", []string{":proj"}, "Project management", handleProject},
    
    // Configuration
    ":set":    {"set", nil, "Set configuration option", handleSet},
    
    // Help and Documentation
    ":help":   {"help", []string{":h"}, "Show help for command", handleHelp},
}
```

## Navigation Mode

### 1. Movement Keys
```go
var NavigationKeys = map[string]Action{
    // Basic Movement
    "h":        {Desc: "Left/Parent directory", Fn: moveLeft},
    "j":        {Desc: "Down", Fn: moveDown},
    "k":        {Desc: "Up", Fn: moveUp},
    "l":        {Desc: "Right/Enter directory", Fn: moveRight},
    
    // Quick Movement
    "gg":       {Desc: "Go to top", Fn: moveTop},
    "G":        {Desc: "Go to bottom", Fn: moveBottom},
    "ctrl+d":   {Desc: "Page down", Fn: pageDown},
    "ctrl+u":   {Desc: "Page up", Fn: pageUp},
    
    // Marks
    "m":        {Desc: "Set mark", Fn: setMark},
    "'":        {Desc: "Go to mark", Fn: gotoMark},
}
```

### 2. Selection Keys
```go
var SelectionKeys = map[string]Action{
    // Selection
    "space":    {Desc: "Toggle selection", Fn: toggleSelection},
    "v":        {Desc: "Visual selection mode", Fn: visualMode},
    "V":        {Desc: "Visual line mode", Fn: visualLineMode},
    
    // Actions on Selection
    "y":        {Desc: "Yank (copy) selection", Fn: yankSelection},
    "d":        {Desc: "Delete selection", Fn: deleteSelection},
    "p":        {Desc: "Paste", Fn: paste},
}
```

## Command Implementation

### 1. Command Parser
```go
type CommandParser struct {
    buffer    string
    mode      Mode
}

func (p *CommandParser) Parse(input string) (Command, []string, error) {
    // Parse command and arguments
    parts := strings.Fields(input)
    if len(parts) == 0 {
        return Command{}, nil, errors.New("empty command")
    }
    
    // Look up command
    cmd, exists := Commands[parts[0]]
    if !exists {
        return Command{}, nil, fmt.Errorf("unknown command: %s", parts[0])
    }
    
    return cmd, parts[1:], nil
}
```

### 2. Command Examples
```go
// Organize files matching pattern
:o *.pdf ~/Documents/PDFs

// Watch directory for changes
:w ~/Downloads

// Define new rule
:r add "move screenshots to ~/Pictures/Screenshots"

// Set configuration
:set deep_scan=true

// Project management
:project add ~/code/myproject
:project detect
```

## Visual Feedback

### 1. Command Line UI
```go
func (m Model) renderCommandLine() string {
    if !m.commandMode {
        return ""
    }
    
    return lipgloss.JoinHorizontal(
        lipgloss.Left,
        Styles.Command.Render(":"),
        m.input.View(),
    )
}
```

### 2. Status Line
```go
func (m Model) renderStatus() string {
    mode := "NORMAL"
    if m.visualMode {
        mode = "VISUAL"
    } else if m.commandMode {
        mode = "COMMAND"
    }
    
    return lipgloss.JoinHorizontal(
        lipgloss.Left,
        Styles.Mode.Render(mode),
        Styles.Status.Render(m.currentPath),
        Styles.Info.Render(fmt.Sprintf("%d selected", len(m.selected))),
    )
}
```

## Command History

```go
type History struct {
    entries  []string
    position int
    max      int
}

func (h *History) Add(cmd string) {
    if len(h.entries) >= h.max {
        h.entries = h.entries[1:]
    }
    h.entries = append(h.entries, cmd)
    h.position = len(h.entries)
}

func (h *History) Previous() string {
    if h.position > 0 {
        h.position--
        return h.entries[h.position]
    }
    return ""
}

func (h *History) Next() string {
    if h.position < len(h.entries)-1 {
        h.position++
        return h.entries[h.position]
    }
    return ""
}
```

## Keyboard Shortcuts Reference

```
Navigation:
  h,j,k,l     Basic movement
  gg/G        Top/Bottom
  ctrl+u/d    Page up/down
  
Selection:
  space       Toggle selection
  v           Visual mode
  V           Visual line mode
  
File Operations:
  :o          Organize files
  :w          Watch directory
  :m          Move files
  
Management:
  :r          Rules
  :p          Patterns
  :project    Project management
  
Configuration:
  :set        Set options
  :help       Show help
```

This vim-style command system provides:
- Familiar interface for power users
- Quick access to all functionality
- Command history and completion
- Visual feedback
- Extensible command system

Would you like me to expand on any aspect of the command system or show more implementation details?
