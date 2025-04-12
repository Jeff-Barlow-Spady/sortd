# Smart Rule Learning - Technical Design Document

## Overview

The Smart Rule Learning system is designed to analyze user file organization behaviors and suggest automated rules that match observed patterns. This feature aims to reduce the manual effort required to set up organization rules by learning directly from user actions.

## Architecture

### Core Components

1. **Learning Engine** - Central component responsible for:
   * Collecting and processing file operation data
   * Analyzing patterns
   * Generating rule suggestions
   * Processing feedback

2. **Storage Layer** - Persists:
   * Historical file operations
   * Learned patterns
   * Rule suggestions
   * User feedback

3. **Pattern Detection** - Algorithms for:
   * Statistical pattern recognition
   * Confidence scoring
   * Rule generation

4. **Integration Points** - Hooks into:
   * Organize Engine
   * Workflow System
   * GUI

### Module Structure

```
internal/
  patterns/
    learning/
      engine.go       # Core learning functionality
      tracker.go      # File operation tracking
      detector.go     # Pattern detection algorithms
      suggestion.go   # Rule suggestion generation
      feedback.go     # User feedback processing
      model.go        # Data models
      persistence.go  # Storage operations
      config.go       # Configuration options
```

## Data Models

### OperationRecord

```go
type OperationRecord struct {
    ID              string    `json:"id"`
    Timestamp       time.Time `json:"timestamp"`
    OperationType   string    `json:"operation_type"` // "move", "copy", "rename", etc.
    SourcePath      string    `json:"source_path"`
    DestinationPath string    `json:"destination_path"`
    FileName        string    `json:"file_name"`
    FileExt         string    `json:"file_ext"`
    FileSize        int64     `json:"file_size"`
    Success         bool      `json:"success"`
    Manual          bool      `json:"manual"` // User-initiated vs. rule-initiated
}
```

### DetectedPattern

```go
type PatternType string

const (
    ExtensionPattern PatternType = "extension"
    NamePrefixPattern PatternType = "name_prefix"
    NameSuffixPattern PatternType = "name_suffix"
    ContentTypePattern PatternType = "content_type"
    TimeBasedPattern PatternType = "time_based"
    SizeBasedPattern PatternType = "size_based"
)

type DetectedPattern struct {
    ID               string      `json:"id"`
    PatternType      PatternType `json:"pattern_type"`
    PatternValue     string      `json:"pattern_value"` // e.g., "*.jpg", "Invoice_*"
    DestinationPath  string      `json:"destination_path"`
    Confidence       float64     `json:"confidence"` // 0.0 to 1.0
    OccurrenceCount  int         `json:"occurrence_count"`
    FirstSeen        time.Time   `json:"first_seen"`
    LastSeen         time.Time   `json:"last_seen"`
    SampleOperations []string    `json:"sample_operations"` // IDs of sample operations
}
```

### RuleSuggestion

```go
type SuggestionStatus string

const (
    StatusPending SuggestionStatus = "pending"
    StatusAccepted SuggestionStatus = "accepted"
    StatusRejected SuggestionStatus = "rejected"
    StatusModified SuggestionStatus = "modified"
)

type RuleSuggestion struct {
    ID             string          `json:"id"`
    PatternID      string          `json:"pattern_id"`
    SuggestedRule  types.Pattern   `json:"suggested_rule"`
    Confidence     float64         `json:"confidence"`
    Status         SuggestionStatus `json:"status"`
    CreatedAt      time.Time       `json:"created_at"`
    RespondedAt    time.Time       `json:"responded_at,omitempty"`
    UserModifications map[string]interface{} `json:"user_modifications,omitempty"`
}
```

## Core Algorithms

### Pattern Detection

1. **Extension-based Pattern Detection**
   * Group file operations by extension
   * Calculate frequency of destination paths for each extension
   * Generate patterns for extensions with consistent destinations

2. **Name-based Pattern Detection**
   * Analyze common prefixes and suffixes in filenames
   * Identify correlations between name patterns and destinations
   * Generate patterns for consistent name-destination mappings

3. **Time-based Pattern Detection**
   * Group operations by time periods (day of week, time of day)
   * Identify correlations between time periods and destinations
   * Generate time-sensitive organization suggestions

4. **Content-based Pattern Detection**
   * Perform basic content analysis (text detection, image analysis)
   * Correlate content types with destinations
   * Generate content-based organization suggestions

### Confidence Scoring

Confidence is calculated based on:
* Consistency of the pattern (% of matching files going to same destination)
* Frequency of observations (more observations = higher confidence)
* Recency of observations (more recent = higher confidence)
* User feedback on similar suggestions (positive feedback increases confidence)

```go
func calculateConfidence(pattern DetectedPattern) float64 {
    // Base confidence from consistency
    baseConfidence := float64(pattern.MatchCount) / float64(pattern.TotalPossibleMatches)

    // Adjust for frequency
    frequencyFactor := math.Min(1.0, float64(pattern.OccurrenceCount)/10.0)

    // Adjust for recency (higher weight for recent patterns)
    daysSinceLastSeen := time.Since(pattern.LastSeen).Hours() / 24
    recencyFactor := math.Max(0.5, 1.0 - (daysSinceLastSeen / 30)) // Decay over 30 days

    // Calculate final confidence
    confidence := baseConfidence * 0.6 + frequencyFactor * 0.2 + recencyFactor * 0.2

    return math.Min(1.0, confidence)
}
```

## Implementation Details

### Operation Tracking

Operation tracking is implemented by inserting hooks into key file operation methods:

```go
// Hooks in Engine.MoveFile
func (e *Engine) MoveFile(src, dest string) error {
    // Existing functionality
    // ...

    // After successful move
    if err == nil && e.learningEnabled {
        opRecord := &learning.OperationRecord{
            OperationType:   "move",
            SourcePath:      src,
            DestinationPath: dest,
            FileName:        filepath.Base(src),
            FileExt:         filepath.Ext(src),
            Timestamp:       time.Now(),
            Success:         true,
            Manual:          e.manualOperation,
        }
        e.learningEngine.TrackOperation(opRecord)
    }

    return err
}
```

### Learning Process Flow

1. **Data Collection Phase**
   * Track file operations
   * Store operation records
   * Maintain statistical counters

2. **Analysis Phase**
   * Periodically analyze operation history
   * Run pattern detection algorithms
   * Generate and score patterns
   * Create rule suggestions

3. **Suggestion Phase**
   * Present high-confidence suggestions to user
   * Collect feedback on suggestions
   * Apply accepted suggestions as rules
   * Learn from rejections and modifications

4. **Refinement Phase**
   * Update pattern confidence based on feedback
   * Adjust detection algorithms
   * Remove irrelevant patterns
   * Consolidate similar patterns

### GUI Integration

The GUI integration includes:

1. **Suggestions Panel**
   * Displays rule suggestions with confidence scores
   * Provides accept/modify/reject buttons
   * Shows explanation of why the rule was suggested

2. **Rule Testing**
   * Allows users to test suggested rules on sample files
   * Shows preview of organization results before accepting

3. **Advanced Settings**
   * Controls learning sensitivity
   * Manages learning history
   * Provides options to reset or tune the learning system

## Challenges and Mitigations

### Privacy Concerns
* **Challenge**: Tracking user file operations raises privacy concerns
* **Mitigation**:
  * Store only metadata, not file contents
  * Provide clear opt-out options
  * Keep all data local to user's machine

### False Positives
* **Challenge**: Incorrect pattern detection leading to unhelpful suggestions
* **Mitigation**:
  * Conservative confidence thresholds for suggestions
  * Learning from rejection feedback
  * Clear explanation of pattern reasoning

### Performance Impact
* **Challenge**: Pattern analysis could impact system performance
* **Mitigation**:
  * Background processing of pattern analysis
  * Limited history retention
  * Configurable analysis frequency

## Testing Plan

### Unit Tests
* Pattern detection algorithms
* Confidence scoring accuracy
* Feedback processing logic

### Integration Tests
* End-to-end learning flow
* UI interaction with learning system
* Persistence mechanisms

### User Scenario Tests
* Multi-week simulated usage patterns
* Validation of suggestion quality

## Success Metrics

The success of the Smart Rule Learning feature will be measured by:

1. **Suggestion Acceptance Rate**
   * Target: >60% of suggestions accepted without modification

2. **Time-to-Value**
   * Target: Useful suggestions within 2 weeks of usage

3. **Rule Coverage**
   * Target: >75% of manual file operations captured by learned rules

4. **User Satisfaction**
   * Target: >80% of users report saving time with the feature