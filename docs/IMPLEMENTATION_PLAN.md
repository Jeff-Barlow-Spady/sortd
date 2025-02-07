# Implementation Plan

## Phase 1: Core Pattern Matching and Organization

### 1. Basic File Operations
```go
// Core operations with safety checks
type FileOps struct {
    // Safety settings
    DryRun    bool
    Backup    bool
    
    // Operation tracking
    history   []Operation
    undoLog   *UndoLog
}

func (ops *FileOps) Move(src, dest string) error {
    // 1. Safety checks
    if err := ops.validatePaths(src, dest); err != nil {
        return err
    }
    
    // 2. Ensure destination directory exists
    if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
        return err
    }
    
    // 3. Handle existing files
    if exists, err := fileExists(dest); err != nil {
        return err
    } else if exists {
        dest = ops.handleCollision(dest)
    }
    
    // 4. Backup if enabled
    if ops.Backup {
        if err := ops.createBackup(src); err != nil {
            return err
        }
    }
    
    // 5. Move file
    if !ops.DryRun {
        if err := os.Rename(src, dest); err != nil {
            return err
        }
    }
    
    // 6. Record operation
    ops.recordOperation(src, dest)
    return nil
}
```

### 2. Pattern Matching Engine
```go
type Pattern struct {
    Glob     string   // e.g., "*.pdf"
    Prefixes []string // e.g., ["invoice", "receipt"]
    Suffixes []string // e.g., ["2024", "final"]
}

type Matcher struct {
    patterns map[string][]Pattern // dest -> patterns
}

func (m *Matcher) FindDestination(file string) (string, float64) {
    name := filepath.Base(file)
    
    for dest, patterns := range m.patterns {
        for _, pat := range patterns {
            // Check glob pattern
            if match, err := filepath.Match(pat.Glob, name); err == nil && match {
                confidence := 1.0
                
                // Check prefixes/suffixes for better confidence
                if len(pat.Prefixes) > 0 || len(pat.Suffixes) > 0 {
                    confidence = m.checkAffixes(name, pat)
                }
                
                if confidence > 0.5 {
                    return dest, confidence
                }
            }
        }
    }
    
    return "", 0.0
}
```

### 3. Configuration Management
```yaml
# ~/.config/sortd/config.yaml
organize:
  "~/Documents/Invoices":
    - pattern: "*.pdf"
      prefixes: ["invoice", "receipt"]
      
  "~/Pictures/Screenshots":
    - pattern: "Screenshot*.png"
    - pattern: "*_screenshot.png"
    
  "~/Downloads/Installers":
    - pattern: "*.{exe,msi,deb}"
      suffixes: ["setup", "install"]

settings:
  dry_run: true        # Safe by default
  create_dirs: true    # Create destination dirs
  backup: false        # Backup before moving
  collision: "rename"  # rename, skip, or ask
```

### 4. Watch Mode (Basic)
```go
type Watcher struct {
    matcher  *Matcher
    ops      *FileOps
    events   chan Event
    done     chan bool
}

func (w *Watcher) Watch(dir string) error {
    fsWatcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    defer fsWatcher.Close()
    
    // Start watching
    if err := fsWatcher.Add(dir); err != nil {
        return err
    }
    
    // Event loop
    for {
        select {
        case event := <-fsWatcher.Events:
            if event.Op == fsnotify.Create {
                w.handleNewFile(event.Name)
            }
        case err := <-fsWatcher.Errors:
            log.Printf("Watch error: %v", err)
        case <-w.done:
            return nil
        }
    }
}

func (w *Watcher) handleNewFile(path string) {
    // Let the file settle (avoid partial writes)
    time.Sleep(100 * time.Millisecond)
    
    // Find destination
    if dest, conf := w.matcher.FindDestination(path); dest != "" {
        w.ops.Move(path, filepath.Join(dest, filepath.Base(path)))
    }
}
```

### 5. Simple CLI
```go
func main() {
    app := &cli.App{
        Name: "sortd",
        Commands: []*cli.Command{
            {
                Name:  "watch",
                Usage: "Watch directory for files to organize",
                Action: watchCommand,
            },
            {
                Name:  "organize",
                Usage: "Organize files in directory",
                Action: organizeCommand,
            },
            {
                Name:  "test",
                Usage: "Test patterns against files (dry run)",
                Action: testCommand,
            },
        },
    }
    
    app.Run(os.Args)
}
```

## Phase 2: UI and Interaction

### 1. Terminal UI
- File list with selection
- Basic commands (organize, undo, quit)
- Status updates
- Progress indication

### 2. Interactive Mode
- Pattern testing
- Rule creation
- Destination browsing

## Phase 3: Advanced Features

### 1. Enhanced Pattern Matching
- Content-based patterns
- Size/date conditions
- Project awareness

### 2. Batch Operations
- Group related files
- Bulk moves
- Operation queuing

### 3. Statistics and Reporting
- Operation history
- Pattern effectiveness
- Space usage

## Safety Features

1. **Operation Safety**
```go
type SafetyChecks struct {
    // Pre-operation checks
    validatePaths()
    checkPermissions()
    ensureSpace()
    detectCollisions()
    
    // Post-operation checks
    verifyMove()
    updateHistory()
    
    // Recovery
    createBackup()
    restoreBackup()
}
```

2. **Error Recovery**
```go
type Recovery struct {
    // Operation logging
    LogOperation(op Operation)
    
    // Undo support
    CreateUndoPoint()
    Undo()
    
    // Backup management
    BackupFile(path string)
    RestoreBackup(path string)
}
```

## Testing Plan

1. **Pattern Matching**
- Glob pattern matching
- Prefix/suffix matching
- Edge cases (special characters, long names)

2. **File Operations**
- Move operations
- Collision handling
- Directory creation
- Permission handling

3. **Watch Mode**
- File detection
- Concurrent operations
- Error handling

## Initial Milestones

1. **Core Functionality** (Week 1)
- Basic pattern matching
- Safe file operations
- Configuration loading

2. **Watch Mode** (Week 2)
- Directory watching
- Event handling
- Basic UI

3. **Safety Features** (Week 3)
- Operation validation
- Backup system
- Undo support

4. **Testing & Polish** (Week 4)
- Comprehensive testing
- Error handling
- Documentation

Would you like me to:
1. Detail any specific component further?
2. Add more test cases?
3. Expand the safety features?
4. Show more UI examples?
