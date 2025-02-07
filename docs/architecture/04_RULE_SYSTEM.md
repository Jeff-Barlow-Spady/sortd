# Rule System

## User-Facing Rules

### 1. Simple Text Rules
Users can define rules in plain text, using natural language-like patterns:

```yaml
# Simple rules in rules.yaml
rules:
  - "move screenshots to ~/Pictures/Screenshots"
  - "put invoices in ~/Documents/Finance"
  - "organize downloads older than 30 days into ~/Archive"
  - "keep source code in ~/Code"
```

### 2. Pattern Definitions
```yaml
# patterns.yaml - Users can define what constitutes each type
patterns:
  screenshots:
    - "Screenshot*.png"
    - "*.screenshot.png"
    - "*_screenshot.*"
  
  invoices:
    - "*invoice*.pdf"
    - "*receipt*.pdf"
    - matches: ["total:", "invoice number", "amount due"]
  
  source_code:
    - "*.{py,js,go,cpp,h}"
    - folders: ["src", "lib", "pkg"]
```

### 3. Rule Implementation
```go
// Simple rule structure that maps to user definitions
type Rule struct {
    Description string            // Original user text
    Patterns    []string         // File patterns to match
    Destination string           // Where to put matching files
    Conditions  []string         // Optional conditions (age, size, etc)
}

// Parser converts user text to rules
func ParseRule(text string) Rule {
    // Example: "move screenshots to ~/Pictures/Screenshots"
    parts := strings.Fields(text)
    
    return Rule{
        Description: text,
        Patterns:    lookupPatterns(parts[1]),     // "screenshots"
        Destination: expandPath(parts[3]),         // "~/Pictures/Screenshots"
    }
}

// Pattern matcher
func (r *Rule) Matches(file *File) bool {
    // Check each pattern
    for _, pattern := range r.Patterns {
        if matchesPattern(file.Name, pattern) {
            return true
        }
    }
    
    // Check conditions if any
    for _, condition := range r.Conditions {
        if !matchesCondition(file, condition) {
            return false
        }
    }
    
    return false
}
```

## Common Use Cases

### 1. Downloads Organization
```yaml
rules:
  - "organize downloads by type"  # Expands to multiple rules based on file types
  - "clean old downloads after 30 days"
  - "keep installers in ~/Downloads/Software"

patterns:
  installers:
    - "*.{exe,msi,deb,rpm,dmg}"
    - "*setup*.exe"
    - "*install*.exe"
```

### 2. Document Management
```yaml
rules:
  - "sort documents by year"  # Creates YYYY folders
  - "keep tax documents in ~/Documents/Tax/{YYYY}"
  - "organize receipts by month"  # Creates YYYY/MM folders

patterns:
  tax_docs:
    - "*tax*.pdf"
    - "*1099*.pdf"
    - "*w2*.pdf"
```

### 3. Media Organization
```yaml
rules:
  - "sort photos by date taken"  # Uses EXIF data when available
  - "organize screenshots by app"  # Groups by application name in filename
  - "keep wallpapers in ~/Pictures/Wallpapers"

patterns:
  wallpapers:
    - "*.{jpg,png}" # Plus size check for typical wallpaper dimensions
    - folders: ["wallpaper", "background"]
```

## Rule Processing

```go
type RuleEngine struct {
    rules    []Rule
    patterns map[string][]string
}

// Process a single file
func (e *RuleEngine) Process(file *File) *Action {
    for _, rule := range e.rules {
        if rule.Matches(file) {
            return &Action{
                Rule:        rule,
                Source:     file.Path,
                Dest:       rule.GetDestination(file),
                Confidence: 1.0,
            }
        }
    }
    
    // No explicit rule match, use default organization
    return e.getDefaultAction(file)
}

// Default organization by file type
func (e *RuleEngine) getDefaultAction(file *File) *Action {
    category := categorizeFile(file)
    return &Action{
        Rule:        e.defaultRules[category],
        Source:     file.Path,
        Dest:       e.getDefaultPath(category, file),
        Confidence: 0.8,
    }
}
```

## Configuration File
```toml
# config.toml - User preferences
[organization]
default_mode = "ask"        # ask, auto, dry-run
create_folders = true       # Create destination folders if missing
use_date_folders = false    # Organize by date
backup = false             # Keep backups of moved files

[paths]
documents = "~/Documents"
pictures = "~/Pictures"
downloads = "~/Downloads"
archive = "~/Archive"

[rules]
rule_file = "~/.config/sortd/rules.yaml"
pattern_file = "~/.config/sortd/patterns.yaml"
```

This approach:
1. Uses natural language-like rules that are easy to understand
2. Separates patterns from rules for better reuse
3. Has sensible defaults but is easily customizable
4. Supports both simple and complex use cases
5. Keeps the implementation straightforward

The focus is on making it easy for users to:
- Define what they want in plain language
- Reuse common patterns
- Override defaults when needed
- Understand what the system is doing

Would you like me to:
1. Add more example patterns for specific use cases?
2. Show how to handle more complex organization needs?
3. Add more details about the pattern matching implementation?
