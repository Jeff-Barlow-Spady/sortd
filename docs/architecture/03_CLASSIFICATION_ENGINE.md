# File Classification Engine

## Core Classification Pipeline

### 1. Multi-Level Analysis
```go
type ClassificationEngine struct {
    // Core components
    contentAnalyzer    *ContentAnalyzer
    signatureDetector  *SignatureDetector
    modelProcessor     *MultimodalProcessor
    projectDetector    *ProjectDetector

    // Configuration
    config            ClassificationConfig
}

// Classification result with confidence levels
type ClassificationResult struct {
    Category    string
    Confidence  float64
    Evidence    []Evidence
    Suggestions []Suggestion
    Context     *ProjectContext
}

// Evidence for classification decisions
type Evidence struct {
    Type      EvidenceType // Content, Signature, Context, Model
    Weight    float64
    Details   string
}
```
### 1.4 Project Fingerprinting
```go
// Project signature database
type ProjectSignature struct {
    MarkerFiles []string       // e.g. ["go.mod", "package.json"]
    ContentHooks []ContentHook // Regex patterns in key files
    StructureRules []GlobRule  // Directory structure patterns
}

// Example content hook for Node.js projects
var nodeProjectHook = ContentHook{
    FilePattern: "package.json",
    RequiredFields: []string{
        `"name":\s*".+"`,
        `"version":\s*".+"`,
        `"dependencies":\s*{`,
    },
}

### 2. Classification Pipeline
```go
func (e *Engine) Classify(file *File) (*ClassificationResult, error) {
    // 1. Quick signature check (fast path)
    if sig, conf := e.signatureDetector.Detect(file); conf > e.config.QuickConfidenceThreshold {
        return e.createResult(sig, conf, []Evidence{{Type: SignatureEvidence}})
    }

    // 2. Project context detection
    projectCtx := e.projectDetector.DetectContext(file.Path)

    // 3. Content analysis
    contentResult := e.contentAnalyzer.Analyze(file, AnalysisOptions{
        MaxSampleSize: e.config.ContentSampleSize,
        DeepScan:     shouldDeepScan(file, projectCtx),
    })

    // 4. Multimodal analysis if needed
    var modelResult *ModelAnalysis
    if needsModelAnalysis(contentResult, projectCtx) {
        modelResult = e.modelProcessor.Process(file)
    }

    // 5. Combine evidence and make decision
    return e.makeDecision(file, contentResult, modelResult, projectCtx)
}
```

### 3. Content Analysis
```go
type ContentAnalyzer struct {
    // Content type detection
    mimeDetector  *MIMEDetector
    textAnalyzer  *TextAnalyzer
    mediaAnalyzer *MediaAnalyzer

    // Pattern matching
    patterns      []Pattern
}

func (a *ContentAnalyzer) Analyze(file *File, opts AnalysisOptions) *ContentResult {
    // 1. Determine content type
    contentType := a.mimeDetector.Detect(file)

    // 2. Type-specific analysis
    var result *ContentResult
    switch contentType.Category {
    case "text":
        result = a.analyzeText(file, opts)
    case "image":
        result = a.analyzeImage(file, opts)
    case "audio":
        result = a.analyzeAudio(file, opts)
    case "video":
        result = a.analyzeVideo(file, opts)
    case "application":
        result = a.analyzeApplication(file, opts)
    }

    return result
}
```

### 4. Decision Making
```go
func (e *Engine) makeDecision(
    file *File,
    contentResult *ContentResult,
    modelResult *ModelAnalysis,
    projectCtx *ProjectContext,
) (*ClassificationResult, error) {
    // 1. Gather all evidence
    evidence := []Evidence{}

    // Content-based evidence
    if contentResult != nil {
        evidence = append(evidence, Evidence{
            Type:   ContentEvidence,
            Weight: calculateContentWeight(contentResult),
            Details: contentResult.Description,
        })
    }

    // Model-based evidence
    if modelResult != nil {
        evidence = append(evidence, Evidence{
            Type:   ModelEvidence,
            Weight: calculateModelWeight(modelResult),
            Details: modelResult.Description,
        })
    }

    // Project context evidence
    if projectCtx != nil {
        evidence = append(evidence, Evidence{
            Type:   ContextEvidence,
            Weight: calculateContextWeight(projectCtx),
            Details: projectCtx.Description,
        })
    }

    // 2. Weight and combine evidence
    weightedResults := e.weightEvidence(evidence)

    // 3. Make final decision
    decision := e.selectBestCategory(weightedResults)

    // 4. Generate alternative suggestions
    suggestions := e.generateSuggestions(weightedResults)

    return &ClassificationResult{
        Category:    decision.Category,
        Confidence:  decision.Confidence,
        Evidence:    evidence,
        Suggestions: suggestions,
        Context:     projectCtx,
    }, nil
}
```

## Classification Rules

### 1. Project-Based Rules
```go
type ProjectRule struct {
    // Project identification
    Markers     []string          // e.g., .git, package.json
    PathPattern string            // e.g., **/src/**

    // Classification rules
    Categories  map[string]Rule   // Category-specific rules
    Priorities  map[string]int    // Rule priorities
}

// Example project rules
var DefaultProjectRules = map[string]ProjectRule{
    "web_project": {
        Markers: []string{"package.json", "node_modules"},
        Categories: map[string]Rule{
            "source": {
                Pattern: "**/src/**/*.{js,ts,jsx,tsx}",
                Category: "source_code",
            },
            "assets": {
                Pattern: "**/assets/**/*.{png,jpg,svg}",
                Category: "project_assets",
            },
        },
    },
    "python_project": {
        Markers: []string{"setup.py", "requirements.txt"},
        Categories: map[string]Rule{
            "source": {
                Pattern: "**/*.py",
                Category: "source_code",
            },
            "tests": {
                Pattern: "**/tests/**/*.py",
                Category: "test_code",
            },
        },
    },
}
```

### 2. Content-Based Rules
```go
type ContentRule struct {
    // Content matching
    Patterns    []string    // Content patterns to match
    Keywords    []string    // Keywords to identify
    Metadata    []string    // Metadata fields to check

    // Classification
    Category    string
    Confidence  float64
}

// Example content rules
var DefaultContentRules = []ContentRule{
    {
        Patterns: []string{
            `invoice`, `receipt`, `order\s+number`,
            `total\s+amount`, `payment\s+due`,
        },
        Keywords: []string{"invoice", "receipt", "order"},
        Category: "financial_documents",
        Confidence: 0.8,
    },
    {
        Patterns: []string{
            `meeting\s+minutes`, `agenda`,
            `attendees:`, `action\s+items`,
        },
        Keywords: []string{"meeting", "minutes", "agenda"},
        Category: "meeting_notes",
        Confidence: 0.7,
    },
}
```

### 3. Model-Based Classification
```go
type ModelClassifier struct {
    model       *MultimodalModel
    tokenizer   *Tokenizer
    threshold   float64
}

func (c *ModelClassifier) Classify(file *File) (*ModelResult, error) {
    // 1. Prepare input
    input, err := c.prepareInput(file)
    if err != nil {
        return nil, err
    }

    // 2. Run model inference
    predictions := c.model.Predict(input)

    // 3. Filter and validate predictions
    validPredictions := c.filterPredictions(predictions)

    // 4. Return best matches
    return &ModelResult{
        Categories: validPredictions,
        Confidence: calculateConfidence(validPredictions),
    }, nil
}
```

## Configuration

```toml
[classification]
# Analysis settings
quick_confidence_threshold = 0.9
content_sample_size = "4kb"
deep_scan_threshold = 0.7

# Model settings
model_batch_size = 16
model_threshold = 0.8

# Project detection
project_markers = [".git", "package.json", "requirements.txt"]
max_marker_depth = 5

# Content analysis
text_analysis = true
media_analysis = true
metadata_analysis = true
```

This specification focuses on:
- Exact classification pipeline
- Evidence gathering and weighting
- Project context awareness
- Content analysis rules
- Model-based classification
- Decision making process

Would you like me to expand on any of these aspects or show more detailed examples of the classification rules?
