# Enhanced Features

## Advanced Pattern Matching

### 1. Smart Pattern Groups
```go
type PatternGroup struct {
    // Basic patterns
    Include     []string          // Must match any
    Exclude     []string          // Must not match any

    // Smart matching
    Related     []string          // Related files to move together
    Sequential  bool              // Part of a sequence (e.g., part1, part2)

    // Time-based grouping
    TimeWindow  time.Duration     // Group files within time window
    DateFormat  string           // Extract date from filename
}

// Example: Group related downloads
patterns:
  "~/Downloads/Installers":
    - include: ["*setup*.exe", "*install*.exe"]
      exclude: ["*temp*", "*partial*"]
      related: ["*.md5", "*.sha256", "README*"]
      timeWindow: "5m"  # Group files downloaded within 5 minutes

  "~/Pictures/PhotoSets":
    - include: ["DSC*.jpg"]
      sequential: true  # Group sequential photos
      timeWindow: "1h"  # Same photo session
```

### 2. Content-Aware Matching
```go
type ContentMatcher struct {
    // Content signatures
    Headers    []byte            // File headers to match
    Markers    []string          // Content markers

    // Smart text matching
    Keywords   []string          // Must contain any
    Phrases    []string          // Must contain all
    Format     string            // Expected format (JSON, YAML, etc.)
}

// Example: Smart document organization
patterns:
  "~/Documents/Invoices":
    - include: ["*.pdf", "*.txt"]
      content:
        keywords: ["invoice", "bill", "receipt"]
        phrases: ["amount due", "payment terms"]

  "~/Documents/Code":
    - include: ["*.{js,py,go}"]
      content:
        format: "source_code"
        markers: ["#!/usr/bin", "package ", "import "]
```

### 3. Relationship Tracking
```go
type FileRelations struct {
    // File relationships
    Dependencies  []string    // Files this depends on
    References    []string    // Files that reference this
    Sequence     *Sequence   // Part of numbered sequence
    Group        string      // Group identifier
}

// Example: Handle related files
func (m *Matcher) handleRelatedFiles(file string, relations FileRelations) []string {
    toMove := []string{file}

    // Check dependencies
    for _, dep := range relations.Dependencies {
        if exists(dep) {
            toMove = append(toMove, dep)
        }
    }

    // Check sequence
    if relations.Sequence != nil {
        sequenceFiles := findSequence(file, relations.Sequence)
        toMove = append(toMove, sequenceFiles...)
    }

    return toMove
}
```

## Enhanced Safety Features

### 1. Transaction-like Operations
```go
type Operation struct {
    // Operation details
    Files      []string
    Action     string
    Timestamp  time.Time

    // Transaction support
    State      OpState
    Rollback   []RollbackStep
    Checksum   map[string]string
}

func (ops *FileOps) ExecuteWithRollback(op Operation) error {
    // 1. Create restore point
    if err := ops.createRestorePoint(op.Files); err != nil {
        return err
    }

    // 2. Verify all files exist and are unchanged
    if err := ops.verifyFiles(op.Files, op.Checksum); err != nil {
        return err
    }

    // 3. Execute in phases
    phases := []Phase{
        {Name: "prepare", Fn: ops.prepareOperation},
        {Name: "backup", Fn: ops.createBackups},
        {Name: "execute", Fn: ops.executeMove},
        {Name: "verify", Fn: ops.verifyResult},
    }

    for _, phase := range phases {
        if err := phase.Fn(op); err != nil {
            ops.rollback(op)
            return err
        }
    }

    return nil
}
```

### 2. File Integrity
```go
type IntegrityCheck struct {
    // File verification
    Size      int64
    ModTime   time.Time
    Checksum  string
    LockState FileLock
}

func (ops *FileOps) verifyIntegrity(path string, check IntegrityCheck) error {
    // 1. Check if file is being modified
    if ops.isFileChanging(path) {
        return ErrFileChanging
    }

    // 2. Verify size and modtime
    info, err := os.Stat(path)
    if err != nil {
        return err
    }
    if info.Size() != check.Size || info.ModTime() != check.ModTime {
        return ErrFileChanged
    }

    // 3. Verify checksum
    if currentSum, err := ops.calculateChecksum(path); err != nil {
        return err
    } else if currentSum != check.Checksum {
        return ErrChecksumMismatch
    }

    return nil
}
```

### 3. Conflict Resolution
```go
type ConflictResolver struct {
    // Conflict detection
    ExistingFile   FileInfo
    IncomingFile   FileInfo
    ConflictType   ConflictType

    // Resolution strategies
    Strategy       ResolutionStrategy
    CustomRename   func(string) string
}

func (r *ConflictResolver) Resolve(conflict Conflict) (Resolution, error) {
    switch r.Strategy {
    case StrategyRename:
        return r.generateUniqueName(conflict)

    case StrategyVersion:
        return r.createVersion(conflict)

    case StrategyMerge:
        return r.mergeFiles(conflict)

    case StrategySkip:
        if r.isNewer(conflict.IncomingFile) {
            return r.replaceExisting(conflict)
        }
        return Resolution{Action: ActionSkip}, nil
    }

    return Resolution{}, ErrNoResolution
}
```

### 4. Operation History
```go
type History struct {
    // Operation tracking
    Operations []Operation
    Timeline   map[string][]TimelineEntry
    Stats      OperationStats

    // Recovery
    RestorePoints map[string]RestorePoint
    Backups      map[string][]BackupInfo
}

func (h *History) TrackOperation(op Operation) {
    // 1. Record operation
    h.Operations = append(h.Operations, op)

    // 2. Update timeline
    for _, file := range op.Files {
        h.Timeline[file] = append(h.Timeline[file], TimelineEntry{
            Op:        op,
            Timestamp: time.Now(),
        })
    }

    // 3. Update stats
    h.Stats.Update(op)

    // 4. Cleanup old entries
    h.cleanup()
}
```

This enhanced version provides:
1. Smart pattern matching that understands file relationships
2. Transaction-like safety for file operations
3. Robust conflict resolution
4. Detailed operation history

The key advantages over a simple script:
1. Handles complex file relationships
2. Provides robust safety guarantees
3. Maintains operation history
4. Resolves conflicts intelligently
5. Verifies file integrity

Would you like me to:
1. Add more pattern matching examples?
2. Show more safety features?
3. Detail the conflict resolution strategies?
4. Expand the file relationship tracking?
## Atomic File Operations
```mermaid
sequenceDiagram
    participant User
    participant FileOps
    participant TransactionLog
    participant OS

    User->>FileOps: Move(src, dest)
    FileOps->>TransactionLog: Begin()
    TransactionLog-->>FileOps: TX ID
    FileOps->>TransactionLog: StageMove(src, dest, RENAME_NOREPLACE)
    TransactionLog->>OS: renameat2()
    alt Success
        OS-->>TransactionLog: Confirmation
        FileOps->>TransactionLog: Commit()
    else Collision
        OS--x TransactionLog: EEXIST
        FileOps->>CollisionHandler: Resolve()
        CollisionHandler-->>FileOps: newDest
        FileOps->>TransactionLog: UpdateStage(newDest)
        FileOps->>TransactionLog: Commit()
    end