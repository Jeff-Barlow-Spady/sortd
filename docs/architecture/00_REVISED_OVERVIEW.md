# Sortd: Simplified File Organization

## Core Philosophy
1. Default to simplicity
2. Progressive disclosure of features
3. Work out of the box
4. Clear, predictable behavior

## Components

### 1. Core Engine
```go
type Sortd struct {
    // Simple, flat configuration
    config struct {
        rules    map[string][]string  // dest -> patterns
        defaults map[string]string    // ext -> dest
    }
    
    // Core functions
    Organize(paths []string) error
    Watch(dir string) error
    AddRule(dest, pattern string) error
}
```

### 2. File Classification
- Start with extension-based classification
- Fall back to filename patterns
- Only use content inspection as last resort
- No ML/AI unless explicitly needed

### 3. Organization Rules
- Direct mapping from patterns to destinations
- Simple glob patterns by default
- Optional regex for advanced users
- Clear, single-purpose rules

### 4. User Interface
- Simple Mode:
  ```
  Arrow keys: Navigate
  Space: Select
  Enter: Organize
  ?: Help
  ```

- Advanced Mode (opt-in):
  ```
  Vim-style navigation
  Command palette
  Custom keybindings
  ```

### 5. Default Behavior
```go
// Sensible defaults that work immediately
var DefaultRules = map[string]string{
    "*.png":  "~/Pictures",
    "*.jpg":  "~/Pictures",
    "*.pdf":  "~/Documents",
    "*.doc*": "~/Documents",
    "*.mp3":  "~/Music",
    "*.mp4":  "~/Videos",
}

// Simple project detection
var ProjectMarkers = []string{
    ".git",
    "package.json",
    "requirements.txt",
}
```

## Usage Examples

### 1. Basic Usage
```bash
# Organize current directory
sortd .

# Watch downloads folder
sortd watch ~/Downloads

# Add simple rule
sortd rule "~/Pictures/Screenshots" "Screenshot*.png"
```

### 2. Configuration
```yaml
# ~/.config/sortd/config.yaml
organize:
  "~/Pictures/Screenshots":
    - "Screenshot*.png"
  "~/Documents/Finance":
    - "*invoice*.pdf"

# Optional advanced settings
advanced:
  enabled: false
  vim_mode: false
  watch_dirs:
    - "~/Downloads"
```

## Implementation Priority

1. Core Organization
   - Basic file moves
   - Extension-based sorting
   - Simple patterns

2. User Interface
   - Basic navigation
   - File selection
   - Clear feedback

3. Rules System
   - Simple pattern matching
   - Directory mapping
   - Basic configuration

4. Advanced Features (Optional)
   - Vim commands
   - Content inspection
   - Project awareness

## Safety Features
- Dry run by default
- No destructive operations
- Clear undo capability
- Backup important files

This revised approach:
1. Starts simple and grows only when needed
2. Has clear, predictable behavior
3. Works well out of the box
4. Keeps advanced features optional
