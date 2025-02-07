# Watch Mode Commands

## Core Concept
Watch mode provides a simple, interactive way to monitor directories and organize files as they arrive. The interface is minimal but effective.

## Basic Commands

```
Navigation:
j/↓    Move down
k/↑    Move up
space  Select/Deselect file
enter  Confirm action

Quick Actions:
o      Organize selected files
u      Undo last action
r      Refresh view
q      Quit watch mode

Watch Controls:
p      Pause/Resume watching
c      Clear processed items
?      Show/hide help
```

## Implementation

```go
type WatchMode struct {
    active    bool
    paused    bool
    files     []FileInfo
    selected  map[string]bool
    cursor    int
}

// Simple command handler
func (w *WatchMode) handleKey(key Key) {
    switch key.String() {
    case "j", "down":
        w.moveCursor(1)
    case "k", "up":
        w.moveCursor(-1)
    case "space":
        w.toggleSelection()
    case "o":
        w.organizeSelected()
    case "u":
        w.undoLast()
    case "p":
        w.togglePause()
    case "c":
        w.clearProcessed()
    case "q":
        w.quit()
    }
}

// Simple UI rendering
func (w *WatchMode) render() string {
    var s strings.Builder

    // Status line
    status := "Watching"
    if w.paused {
        status = "Paused"
    }
    s.WriteString(fmt.Sprintf("Status: %s | Files: %d | Selected: %d\n\n",
        status, len(w.files), len(w.selected)))

    // File list
    for i, file := range w.files {
        prefix := "  "
        if w.selected[file.Path] {
            prefix = "* "
        }
        if i == w.cursor {
            prefix = "> " + prefix
        }
        s.WriteString(fmt.Sprintf("%s%s\n", prefix, file.Name))
    }

    // Help line
    s.WriteString("\n[o]rganize [u]ndo [p]ause [c]lear [q]uit [?]help")

    return s.String()
}
```

## Watch Mode Features

### 1. Auto-Organization
```go
type WatchConfig struct {
    AutoOrganize  bool              // Organize without confirmation
    Patterns      map[string]string // pattern -> destination
    IgnoreHidden  bool              // Skip hidden files
    BatchDelay    time.Duration     // Wait for related files
}

func (w *WatchMode) handleNewFile(file FileInfo) {
    if w.config.AutoOrganize {
        if dest := w.findDestination(file); dest != "" {
            w.organizeFile(file, dest)
            return
        }
    }

    // Add to list for manual organization
    w.files = append(w.files, file)
    w.render()
}
```

### 2. Batch Operations
```go
func (w *WatchMode) organizeSelected() {
    if len(w.selected) == 0 {
        return
    }

    // Group by likely destination
    groups := w.groupByPattern(w.getSelectedFiles())

    // Organize each group
    for dest, files := range groups {
        w.organizeFiles(files, dest)
    }

    // Clear selection
    w.selected = make(map[string]bool)
    w.render()
}
```

### 3. Undo Support
```go
type Operation struct {
    Files    []FileInfo
    OldPath  string
    NewPath  string
    Time     time.Time
}

func (w *WatchMode) undoLast() {
    if len(w.history) == 0 {
        return
    }

    // Get last operation
    op := w.history[len(w.history)-1]

    // Reverse the move
    for _, file := range op.Files {
        os.Rename(file.NewPath, file.OldPath)
    }

    // Remove from history
    w.history = w.history[:len(w.history)-1]
    w.render()
}
```
## Collision Resolution Protocol
```go
// Strategy implementations
type CollisionStrategy interface {
    Handle(src, dest string) (string, error)
}

type VersionStrategy struct {
    Pattern string // e.g. "{base}_v{num}{ext}"
    Counter int
}

func (s *VersionStrategy) Handle(src, dest string) (string, error) {
    base := filepath.Base(dest)
    ext := filepath.Ext(dest)
    name := strings.TrimSuffix(base, ext)

    newName := fmt.Sprintf(s.Pattern,
        name,
        s.Counter+1,
        ext,
    )
    s.Counter++
    return newName, nil
}

func (w *WatchMode) resolveCollisions() {
    // Resolve collisions
    collisions := w.ops.CheckCollisions(w.history)
    if len(collisions) == 0 {
        return
    }

    // Get unique destinations
}
## Usage Example

```bash
# Start watching downloads
$ sortd watch ~/Downloads

# Interactive mode starts:
Status: Watching | Files: 3 | Selected: 0

  screenshot.png
> invoice.pdf
  document.docx

[o]rganize [u]ndo [p]ause [c]lear [q]uit [?]help

# After selecting and pressing 'o':
Moving invoice.pdf to ~/Documents/Finance
```

This simplified approach:
1. Keeps essential commands for file organization
2. Makes common operations quick and easy
3. Provides clear visual feedback
4. Maintains undo capability for safety
5. Supports both automatic and manual organization

Would you like me to:
1. Add more specific watch mode features?
2. Show how to handle specific file patterns?
3. Detail the auto-organization logic?
