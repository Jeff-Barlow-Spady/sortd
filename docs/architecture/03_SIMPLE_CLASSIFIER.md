# Simple File Classification System

## Core Concept
The classifier uses a simple, layered approach:
1. File extensions and names first (fast and reliable)
2. Basic content checks if needed (for ambiguous files)
3. Project context as a final layer (when in project directories)

## Implementation

### 1. Basic Classifier
```go
type Classifier struct {
    // Simple maps for quick lookups
    extensionRules map[string]string    // .ext -> category
    nameRules     map[string]string     // filename pattern -> category
    projectRules  map[string][]string   // project marker -> paths to check
}

// Classification result
type Result struct {
    Category    string
    Confidence  float64    // Higher for exact matches, lower for guesses
    Source      string     // What determined this classification
}

// Simple classification function
func (c *Classifier) Classify(file *File) Result {
    // 1. Check extension first (fastest)
    if category, ok := c.extensionRules[file.Extension()]; ok {
        return Result{
            Category:   category,
            Confidence: 1.0,
            Source:    "extension",
        }
    }
    
    // 2. Check filename patterns
    if category := c.checkNameRules(file.Name()); category != "" {
        return Result{
            Category:   category,
            Confidence: 0.9,
            Source:    "filename",
        }
    }
    
    // 3. Check project context if applicable
    if category := c.checkProjectContext(file.Path); category != "" {
        return Result{
            Category:   category,
            Confidence: 0.8,
            Source:    "project",
        }
    }
    
    // 4. Basic content check as last resort
    return c.checkContent(file)
}
```

### 2. Simple Rules
```go
// Default extension rules - easy to maintain and modify
var DefaultExtensionRules = map[string]string{
    // Documents
    ".pdf":  "documents",
    ".doc":  "documents",
    ".docx": "documents",
    ".txt":  "documents",
    
    // Images
    ".jpg":  "images",
    ".jpeg": "images",
    ".png":  "images",
    ".gif":  "images",
    
    // Code
    ".py":   "code",
    ".js":   "code",
    ".go":   "code",
    ".cpp":  "code",
    
    // Archives
    ".zip":  "archives",
    ".tar":  "archives",
    ".gz":   "archives",
    
    // Media
    ".mp3":  "audio",
    ".mp4":  "video",
    ".mov":  "video",
}

// Simple filename patterns
var DefaultNameRules = map[string]string{
    "README*":     "documentation",
    "Dockerfile":  "docker",
    "LICENSE*":    "legal",
    "*.test.*":    "tests",
    "*.spec.*":    "tests",
}

// Basic project markers
var ProjectMarkers = map[string][]string{
    ".git":            {"source"},
    "package.json":    {"javascript"},
    "requirements.txt": {"python"},
    "go.mod":         {"golang"},
}
```

### 3. Content Checker
```go
func (c *Classifier) checkContent(file *File) Result {
    // Only check first few KB of text files
    content, err := file.ReadSample(4096)
    if err != nil {
        return Result{Category: "unknown", Confidence: 0.0}
    }
    
    // Simple content checks
    switch {
    case containsAny(content, []string{"SELECT", "INSERT", "UPDATE", "DELETE"}):
        return Result{Category: "database", Confidence: 0.7}
        
    case containsAny(content, []string{"<!DOCTYPE", "<html>", "<head>"}):
        return Result{Category: "web", Confidence: 0.7}
        
    case containsAny(content, []string{"invoice", "receipt", "order number"}):
        return Result{Category: "financial", Confidence: 0.7}
    }
    
    return Result{Category: "misc", Confidence: 0.5}
}
```

### 4. Organization Rules
```go
type OrganizationRules struct {
    // Simple category to path mapping
    Destinations map[string]string
    
    // Optional date-based organization
    UseDateFolders bool
    DateFormat     string
}

// Default organization rules
var DefaultRules = OrganizationRules{
    Destinations: map[string]string{
        "documents": "~/Documents",
        "images":    "~/Pictures",
        "code":      "~/Code",
        "archives":  "~/Archives",
        "audio":     "~/Music",
        "video":     "~/Videos",
        "downloads": "~/Downloads",
        "misc":      "~/Other",
    },
    UseDateFolders: false,
    DateFormat:     "2006-01",  // YYYY-MM
}
```

## Usage Example
```go
func ExampleUsage() {
    // Initialize with default rules
    classifier := NewClassifier(
        DefaultExtensionRules,
        DefaultNameRules,
        ProjectMarkers,
    )
    
    // Classify a file
    file := &File{
        Path: "/downloads/document.pdf",
        Size: 1024,
        ModTime: time.Now(),
    }
    
    result := classifier.Classify(file)
    
    // Get destination based on classification
    rules := DefaultRules
    destPath := rules.Destinations[result.Category]
    
    if rules.UseDateFolders {
        destPath = filepath.Join(destPath, 
            file.ModTime.Format(rules.DateFormat))
    }
    
    // Move file
    err := MoveFile(file.Path, filepath.Join(destPath, file.Name))
    if err != nil {
        log.Printf("Failed to move file: %v", err)
    }
}
```

This simplified approach:
- Uses straightforward maps for quick lookups
- Relies primarily on file extensions and names
- Minimizes content analysis
- Has clear, maintainable rules
- Is easy to modify and extend

The system can still handle most common use cases while being much simpler to understand and maintain. Would you like me to expand on any part or show more examples?
