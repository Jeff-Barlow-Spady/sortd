# Organization Engine Specification

## Core Components

### 1. File Operations Manager
```go
// Handles actual file system operations with safety checks
type FileOps interface {
    // Core operations
    Move(src, dst string) error
    Copy(src, dst string) error
    Delete(path string) error
    
    // Safety operations
    ValidatePath(path string) error
    EnsureDirectoryExists(path string) error
    CheckCollisions(moves []Move) []Collision
    
    // Dry run support
    SimulateOperations(ops []Operation) []SimulatedResult
}

// Represents a single file operation
type Operation struct {
    Type      OperationType // Move, Copy, Delete
    Source    string
    Dest      string
    Checksum  string       // For verification
    Priority  int          // For resolving conflicts
}

// Collision detection
type Collision struct {
    Path      string
    Type      CollisionType // Exists, Permission, etc.
    Solutions []Solution    // Possible resolutions
}
```

### 2. Pattern Matcher
```go
// Handles pattern matching and rule application
type PatternMatcher interface {
    // Pattern matching
    MatchFile(file FileInfo, patterns []Pattern) []Match
    MatchContent(content []byte, patterns []Pattern) []Match
    
    // Rule application
    ApplyRules(file FileInfo, rules []Rule) []RuleMatch
    
    // Pattern compilation
    CompilePattern(pattern string) (Pattern, error)
    ValidatePattern(pattern string) error
}

// Pattern definition
type Pattern struct {
    Type      PatternType  // Glob, Regex, Content
    Expr      string       // Pattern expression
    Flags     uint32       // Pattern flags
    Compiled  interface{}  // Compiled pattern
}

// Match result
type Match struct {
    Pattern   Pattern
    Score     float64    // Match confidence
    Captures  []string   // Captured groups
    Context   MatchContext
}
```

### 3. Organization Planner
```go
// Plans file organization operations
type Planner interface {
    // Planning
    CreatePlan(files []FileInfo, rules []Rule) Plan
    ValidatePlan(plan Plan) []Issue
    OptimizePlan(plan Plan) Plan
    
    // Execution
    ExecutePlan(plan Plan) error
    RollbackPlan(plan Plan) error
}

// Organization plan
type Plan struct {
    Operations []Operation
    Conflicts  []Conflict
    Stats      PlanStats
    Rollback   []Operation
}

// Plan statistics
type PlanStats struct {
    TotalFiles      int
    TotalBytes      int64
    EstimatedTime   time.Duration
    RiskLevel       RiskLevel
}
```

## Core Logic

### 1. File Organization Process
```go
func (e *Engine) OrganizeFiles(paths []string, opts Options) error {
    // 1. Validate inputs
    if err := e.validatePaths(paths); err != nil {
        return fmt.Errorf("invalid paths: %w", err)
    }
    
    // 2. Gather file information
    files, err := e.gatherFileInfo(paths)
    if err != nil {
        return fmt.Errorf("failed to gather file info: %w", err)
    }
    
    // 3. Create organization plan
    plan, err := e.planner.CreatePlan(files, e.rules)
    if err != nil {
        return fmt.Errorf("failed to create plan: %w", err)
    }
    
    // 4. Validate plan
    if issues := e.planner.ValidatePlan(plan); len(issues) > 0 {
        return fmt.Errorf("plan validation failed: %v", issues)
    }
    
    // 5. Execute plan (with dry-run support)
    if opts.DryRun {
        return e.simulatePlan(plan)
    }
    
    return e.executePlan(plan, opts)
}
```

### 2. Pattern Matching Logic
```go
func (m *Matcher) MatchFile(file FileInfo, patterns []Pattern) []Match {
    matches := make([]Match, 0)
    
    // 1. Check filename patterns
    if nameMatches := m.matchFilename(file.Name, patterns); len(nameMatches) > 0 {
        matches = append(matches, nameMatches...)
    }
    
    // 2. Check content patterns if needed
    if contentPatterns := filterContentPatterns(patterns); len(contentPatterns) > 0 {
        if contentMatches := m.matchContent(file, contentPatterns); len(contentMatches) > 0 {
            matches = append(matches, contentMatches...)
        }
    }
    
    // 3. Score and sort matches
    scored := m.scoreMatches(matches, file)
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].Score > scored[j].Score
    })
    
    return scored
}
```

### 3. Rule Application
```go
func (e *Engine) ApplyRules(file FileInfo, rules []Rule) []RuleMatch {
    matches := make([]RuleMatch, 0)
    
    for _, rule := range rules {
        // 1. Check rule conditions
        if !e.checkRuleConditions(file, rule) {
            continue
        }
        
        // 2. Match patterns
        if patternMatches := e.matcher.MatchFile(file, rule.Patterns); len(patternMatches) > 0 {
            // 3. Calculate destination
            dest, err := e.calculateDestination(file, rule, patternMatches)
            if err != nil {
                continue
            }
            
            matches = append(matches, RuleMatch{
                Rule:        rule,
                Matches:    patternMatches,
                Destination: dest,
                Confidence:  calculateConfidence(patternMatches),
            })
        }
    }
    
    return matches
}
```

### 4. Safety Checks
```go
func (e *Engine) validateOperation(op Operation) error {
    // 1. Path validation
    if err := e.ops.ValidatePath(op.Source); err != nil {
        return fmt.Errorf("invalid source path: %w", err)
    }
    if err := e.ops.ValidatePath(op.Dest); err != nil {
        return fmt.Errorf("invalid destination path: %w", err)
    }
    
    // 2. Permission checks
    if err := e.checkPermissions(op); err != nil {
        return fmt.Errorf("permission denied: %w", err)
    }
    
    // 3. Space requirements
    if err := e.checkSpaceRequirements(op); err != nil {
        return fmt.Errorf("insufficient space: %w", err)
    }
    
    // 4. Collision detection
    if collisions := e.ops.CheckCollisions([]Operation{op}); len(collisions) > 0 {
        return fmt.Errorf("path collisions detected: %v", collisions)
    }
    
    return nil
}
```

## Configuration

```toml
[organization]
# Operation settings
concurrent_ops = 4
batch_size = 100
verify_moves = true

# Safety settings
collision_policy = "ask"  # ask, skip, rename
space_buffer = "10%"     # Required free space buffer
backup = true           # Create backups before moving

# Pattern matching
content_match_size = "4kb"
min_confidence = 0.8
```

## Error Handling

```go
// Error types
type OrganizationError struct {
    Op        string
    Path      string
    Err       error
    Recoverable bool
}

// Recovery actions
type RecoveryAction struct {
    Type      ActionType
    Steps     []Operation
    Rollback  []Operation
}

func (e *Engine) handleError(err error, op Operation) error {
    organizationErr, ok := err.(*OrganizationError)
    if !ok {
        return fmt.Errorf("operation failed: %w", err)
    }
    
    if organizationErr.Recoverable {
        if action := e.planRecovery(organizationErr); action != nil {
            return e.executeRecovery(action)
        }
    }
    
    return err
}
```

This specification defines the core organization logic with:
- Exact file operation handling
- Pattern matching algorithms
- Rule application process
- Safety checks and validation
- Error handling and recovery

Would you like me to expand on any of these components or move on to the learning system specification?
