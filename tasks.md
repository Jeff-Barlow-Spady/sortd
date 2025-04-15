# Sortd Implementation Plan

## Current Implementation Status

Before proceeding with new work, I've analyzed the current state of the codebase:

1. **Error Handling**: The errors package is well-designed with structured error types (FileError, ConfigError, RuleError) and context support, but usage is inconsistent across modules.

2. **Logging**: There's a structured logging implementation in the internal/log package with field support, but it's not consistently used throughout the codebase.

3. **Organization Methods**: The `OrganizeDir` method appears to be implemented but may need refinements.

4. **Workflow System**: Core workflow functionality exists but has placeholders like `executeCommandAction`.

## High Priority Tasks (Revised)

### 1. Standardize Error Handling Usage
**Objective**: Ensure consistent usage of the existing error handling system.

**Tasks**:
- [x] Audit current error handling usage across modules
- [x] Update inconsistent error handling in organize module
  - [x] Fixed raw error returns in createBackup
  - [x] Improved error context in OrganizeByPatterns
  - [x] Enhanced OrganizeDir with proper error checking and context
- [x] Update inconsistent error handling in watch module
  - [x] Improved NewDaemon error handling
  - [x] Enhanced Start method error reporting
  - [x] Added file existence check in OrganizeFile
  - [x] Improved error context in OrganizeFile
- [x] Update inconsistent error handling in workflow module
  - [x] Replaced fmt.Errorf with appropriate error types
  - [x] Added better error context
  - [x] Improved file error handling in ExecuteWorkflow
- [x] Ensure proper error context in all instances

**Implementation Steps**:
1. **Fix raw error returns**: In organize/engine.go and other files, several functions return raw errors from standard library calls instead of wrapping them in the appropriate custom error types.
   ```go
   // Example of problematic code in createBackup:
   srcFile, err := os.Open(dest)
   if err != nil {
       return err // Should use errors.NewFileError
   }
   ```
   ✅ Fixed in createBackup

2. **Standardize error kind usage**: Some error creations don't specify the correct error kind, using Unknown by default.
   ✅ Added appropriate error kinds in organize module
   ✅ Added appropriate error kinds in workflow module

3. **Improve error propagation**: Some functions like OrganizeByPatterns track the first error but don't provide enough context when propagating it.
   ✅ Added better error context in OrganizeByPatterns
   ✅ Added better error context in workflow module methods

4. **Fix direct fmt.Errorf usage**: Update GUI code that uses fmt.Errorf directly:
   ```go
   a.ShowError("No Directory Selected", fmt.Errorf("please select a directory to organize"))
   ```
   ✅ Replaced fmt.Errorf with appropriate error types in watch module
   ✅ Replaced fmt.Errorf with appropriate error types in workflow module

5. **Normalize error logging**: Ensure consistent patterns for error logging, using log.LogError or log.LogWithError for errors.
   ✅ Added proper error logging in organize module
   ✅ Maintained proper error logging in watch module

### 2. Complete Missing Workflow Functionality
**Objective**: Complete the placeholder implementations in the workflow system.

**Tasks**:
- [x] Complete the executeCommandAction implementation
  - [x] Added proper command execution with shell support
  - [x] Implemented variable replacement
  - [x] Added asynchronous execution option
  - [x] Implemented output capture and error handling
- [ ] Ensure workflow execution supports all action types properly
- [x] Implement proper dry run support for all workflow actions
- [x] Add proper error handling and reporting for workflow execution

**Implementation Steps**:
1. Complete the executeCommandAction in pkg/workflow/manager.go with proper command execution:
   ```go
   // Current placeholder:
   func (m *Manager) executeCommandAction(action types.Action, filePath string) error {
       // In dry run mode, just log what would happen
       if m.dryRun {
           fmt.Printf("[DRY RUN] Would execute command: %s (with file: %s)\n", action.Target, filePath)
           return nil
       }

       // This is a placeholder - a real implementation would need to safely execute commands
       fmt.Printf("Would execute command: %s (with file: %s)\n", action.Target, filePath)
       return nil
   }
   ```
   ✅ Implemented with full functionality

2. Add comprehensive error handling for command execution
   ✅ Implemented error handling with sortdErrors package

3. Implement dry run support for command actions
   ✅ Maintained existing dry run support

4. Add proper security safeguards for command execution
   ✅ Used shell for command execution
   ✅ Added environment variable for file path instead of direct command line insertion
   ✅ Implemented configurable shell selection

5. Create tests for the completed implementation
   ❌ Still needs implementation

### 3. Ensure Consistent Logging
**Objective**: Make logging usage consistent across all modules.

**Tasks**:
- [x] Audit how the existing log package is used across different modules
- [x] Standardize log usage patterns
- [x] Ensure proper field usage for structured logging
- [x] Update any direct uses of non-structured logging

**Implementation Steps**:
1. ✅ Document current logging patterns in each major module
2. ✅ Create a standardized logging approach document
3. ✅ Update inconsistent logging usage
   - ✅ Replaced fmt logging in workflow package with structured logging
   - ✅ Updated watch daemon to use structured logging consistently
   - ✅ Updated watcher to use structured logging consistently
4. ✅ Ensure consistent log levels across modules
5. ✅ Add missing contextual information to log calls

**Documentation**:
- Created [`docs/logging_standards.md`](docs/logging_standards.md) with best practices and guidelines for logging in the project.

### 4. Smart Rule Learning (Standout Feature)
**Objective**: Implement a system that learns from user actions to suggest intelligent file organization patterns and file classifications.

**Tasks**:
- [x] Design the pattern learning module foundation with embedded SQLite database
- [ ] Implement the SQLite database setup with Go embed
- [ ] Implement file operation tracking and analysis
- [ ] Develop pattern detection, file classification, and rule generation
- [ ] Build the feedback mechanism for refining suggestions
- [ ] Integrate with the existing UI for a seamless experience

**Current Status**:
- Created design document for Smart Rule Learning system (`smart_rule_learning.md`)
- Defined database schema and data models
- Selected Go embed with SQLite for persistence mechanism
- Established file classification methodology

**Implementation Steps**:

#### 4.1 Pattern Learning Module Creation
1. Create module structure in `internal/patterns/learning`:
   ```
   internal/patterns/
     learning/
       engine.go       # Core learning functionality
       tracker.go      # File operation tracking
       detector.go     # Pattern detection algorithms
       classifier.go   # File classification logic
       suggestion.go   # Rule suggestion generation
       feedback.go     # User feedback processing
       model.go        # Data models
       persistence.go  # SQLite storage operations
       db/             # Embedded SQLite database
         schema.sql    # Database schema definition
         classifications.sql # Base classification definitions
       config.go       # Configuration options
   ```
2. Implement embedded SQLite database:
   ```go
   //go:embed db/schema.sql
   var schemaSQL string

   //go:embed db/classifications.sql
   var classificationsSQL string

   // Initialize embedded database
   func initDatabase() (*sql.DB, error) {
       // Create in-memory database for development or use file for production
       db, err := sql.Open("sqlite", "file:sortd.db?cache=shared")
       if err != nil {
           return nil, fmt.Errorf("failed to open database: %w", err)
       }

       // Initialize schema
       if _, err := db.Exec(schemaSQL); err != nil {
           return nil, fmt.Errorf("failed to initialize schema: %w", err)
       }

       // Load base classifications
       if _, err := db.Exec(classificationsSQL); err != nil {
           return nil, fmt.Errorf("failed to load classifications: %w", err)
       }

       return db, nil
   }
   ```

3. Define comprehensive data models for:
   - `OperationRecord`: Track file operations (move, copy, rename)
   - `DetectedPattern`: Store recognized patterns with confidence scores
   - `FileClassification`: Store file classification data and criteria
   - `RuleSuggestion`: Manage suggestions with user feedback status

4. Define database schema with tables for:
   ```sql
   -- Operations table stores file operations
   CREATE TABLE IF NOT EXISTS operations (
       id TEXT PRIMARY KEY,
       timestamp TEXT NOT NULL,
       operation_type TEXT NOT NULL,
       source_path TEXT NOT NULL,
       destination_path TEXT NOT NULL,
       file_name TEXT NOT NULL,
       file_ext TEXT NOT NULL,
       file_size INTEGER NOT NULL,
       manual BOOLEAN NOT NULL,
       success BOOLEAN NOT NULL
   );

   -- Patterns table stores detected patterns
   CREATE TABLE IF NOT EXISTS patterns (
       id TEXT PRIMARY KEY,
       pattern_type TEXT NOT NULL,
       pattern_value TEXT NOT NULL,
       destination_path TEXT NOT NULL,
       confidence REAL NOT NULL,
       occurrence_count INTEGER NOT NULL,
       first_seen TEXT NOT NULL,
       last_seen TEXT NOT NULL
   );

   -- Classifications table stores file classification data
   CREATE TABLE IF NOT EXISTS classifications (
       id TEXT PRIMARY KEY,
       name TEXT NOT NULL,
       description TEXT NOT NULL,
       criteria_json TEXT NOT NULL,
       confidence_threshold REAL NOT NULL,
       system_defined BOOLEAN NOT NULL
   );

   -- ClassificationMatches tracks which files match classifications
   CREATE TABLE IF NOT EXISTS classification_matches (
       file_path TEXT NOT NULL,
       classification_id TEXT NOT NULL,
       confidence REAL NOT NULL,
       timestamp TEXT NOT NULL,
       PRIMARY KEY (file_path, classification_id)
   );

   -- Suggestions table stores rule suggestions
   CREATE TABLE IF NOT EXISTS suggestions (
       id TEXT PRIMARY KEY,
       pattern_id TEXT,
       classification_id TEXT,
       suggested_rule_json TEXT NOT NULL,
       confidence REAL NOT NULL,
       status TEXT NOT NULL,
       created_at TEXT NOT NULL,
       responded_at TEXT,
       user_modifications_json TEXT,
       FOREIGN KEY (pattern_id) REFERENCES patterns(id),
       FOREIGN KEY (classification_id) REFERENCES classifications(id)
   );
   ```

5. Implement persistence layer with:
   - Efficient schema design with indexes for query performance
   - Transaction support for data integrity
   - Prepared statements for query efficiency
   - Connection pooling for concurrent access

6. Create configuration options for:
   - Learning sensitivity and thresholds
   - Analysis frequency and depth
   - Classification confidence thresholds
   - Suggestion presentation criteria

7. Create core database initialization with Go embed:
   ```go
   import (
       "database/sql"
       "embed"
       _ "github.com/mattn/go-sqlite3"
   )

   //go:embed db/schema.sql
   //go:embed db/classifications.sql
   var dbFS embed.FS

   // InitDatabase initializes the embedded SQLite database
   func InitDatabase(dbPath string) (*sql.DB, error) {
       // If dbPath is empty, use in-memory database
       connectionString := dbPath
       if connectionString == "" {
           connectionString = ":memory:"
       }

       db, err := sql.Open("sqlite3", connectionString)
       if err != nil {
           return nil, errors.NewDatabaseError("failed to open SQLite database", err).
               WithContext("connectionString", connectionString)
       }

       // Read and execute schema
       schemaSQL, err := dbFS.ReadFile("db/schema.sql")
       if err != nil {
           return nil, errors.NewDatabaseError("failed to read schema SQL", err)
       }

       // Execute schema initialization
       if _, err := db.Exec(string(schemaSQL)); err != nil {
           return nil, errors.NewDatabaseError("failed to initialize database schema", err)
       }

       // Initialize base classifications
       classificationSQL, err := dbFS.ReadFile("db/classifications.sql")
       if err != nil {
           return nil, errors.NewDatabaseError("failed to read classifications SQL", err)
       }

       if _, err := db.Exec(string(classificationSQL)); err != nil {
           return nil, errors.NewDatabaseError("failed to initialize base classifications", err)
       }

       return db, nil
   }
   ```

8. Implement the persistence layer interfaces for testability:
   ```go
   // Repository interface defines database operations
   type Repository interface {
       SaveOperationRecord(record *OperationRecord) error
       GetOperationsByType(opType string, limit int) ([]*OperationRecord, error)
       SavePattern(pattern *DetectedPattern) error
       GetPatternsByConfidence(minConfidence float64) ([]*DetectedPattern, error)
       SaveClassificationMatch(match *ClassificationMatch) error
       GetFileClassifications(filePath string) ([]*ClassificationMatch, error)
       SaveSuggestion(suggestion *RuleSuggestion) error
       GetPendingSuggestions(limit int) ([]*RuleSuggestion, error)
       Close() error
   }
   ```

#### 4.2 File Operation Tracking
1. Add tracking hooks in organize engine:
   ```go
   // Add to organize.Engine.MoveFile():
   if err == nil && e.learningEnabled {
       opRecord := &learning.OperationRecord{
           OperationType:   "move",
           SourcePath:      src,
           DestinationPath: dest,
           FileName:        filepath.Base(src),
           FileExt:         filepath.Ext(src),
           FileSize:        fileInfo.Size(),
           Timestamp:       time.Now(),
           Success:         true,
           Manual:          e.manualOperation,
       }
       e.learningEngine.TrackOperation(opRecord)
   }
   ```
2. Implement similar tracking in:
   - Workflow execution actions
   - Manual file organization operations
   - Rule-based organization (marked differently)
3. Create metrics collection for:
   - File metadata (type, size, extension, name patterns)
   - File content samples (for content-based classification)
   - Operation contexts (time, source, destination)
   - Operation success rates
   - User interactions with the system

4. Define operation tracking interface in the organize engine:
   ```go
   // LearningEngine interface for the main organize engine
   type LearningEngine interface {
       TrackOperation(record *learning.OperationRecord) error
       AnalyzeRecentOperations() error
       GetSuggestedRules(minConfidence float64) ([]*learning.RuleSuggestion, error)
       ClassifyFile(filePath string) ([]*learning.ClassificationMatch, error)
       ProcessFeedback(suggestionID string, accepted bool, modifications map[string]interface{}) error
   }
   ```

5. Create the operation tracking implementation:
   ```go
   // Tracker implements file operation tracking
   type Tracker struct {
       repo           Repository
       analysisConfig AnalysisConfig
       logger         log.Logger
   }

   // TrackOperation stores an operation record in the database
   func (t *Tracker) TrackOperation(record *OperationRecord) error {
       if record == nil {
           return errors.NewInvalidInputError("operation record cannot be nil", nil)
       }

       // Generate ID if not provided
       if record.ID == "" {
           record.ID = uuid.New().String()
       }

       // Set timestamp if not provided
       if record.Timestamp.IsZero() {
           record.Timestamp = time.Now()
       }

       // Save to database
       if err := t.repo.SaveOperationRecord(record); err != nil {
           return errors.NewDatabaseError("failed to save operation record", err).
               WithContext("operationType", record.OperationType).
               WithContext("sourceFile", record.SourcePath)
       }

       // Log operation tracking
       t.logger.Info("Tracked file operation",
           "operationType", record.OperationType,
           "sourceFile", record.SourcePath,
           "destinationPath", record.DestinationPath,
           "manual", record.Manual)

       return nil
   }
   ```

#### 4.3 Pattern Detection and Classification System
1. Implement multiple detection algorithms:
   - Extension-based pattern detection
   - Name-based pattern detection (prefixes, suffixes, keywords)
   - Time-based pattern detection
   - Content-based pattern detection
2. Implement file classification system:
   ```go
   type ClassifierCriteria struct {
       ExtensionPatterns []string  `json:"extension_patterns"`
       NamePatterns      []string  `json:"name_patterns"`
       ContentSignatures []string  `json:"content_signatures"`
       MinFileSize       int64     `json:"min_file_size"`
       MaxFileSize       int64     `json:"max_file_size"`
       MimeTypes         []string  `json:"mime_types"`
   }

   type FileClassification struct {
       ID                 string    `json:"id"`
       Name               string    `json:"name"`
       Description        string    `json:"description"`
       Criteria           ClassifierCriteria `json:"criteria"`
       ConfidenceThreshold float64  `json:"confidence_threshold"`
       SystemDefined      bool      `json:"system_defined"`
   }

   func (c *Classifier) ClassifyFile(filePath string) ([]ClassificationMatch, error) {
       // Read file info and potentially sample content
       fileInfo, err := os.Stat(filePath)
       if err != nil {
           return nil, fmt.Errorf("failed to stat file: %w", err)
       }

       // Analyze file properties against all classification criteria
       var matches []ClassificationMatch

       // Query classifications from database
       rows, err := c.db.Query("SELECT id, name, criteria_json, confidence_threshold FROM classifications")
       if err != nil {
           return nil, fmt.Errorf("failed to query classifications: %w", err)
       }
       defer rows.Close()

       // For each classification, check if the file matches
       for rows.Next() {
           var id, name, criteriaJSON string
           var threshold float64
           if err := rows.Scan(&id, &name, &criteriaJSON, &threshold); err != nil {
               return nil, fmt.Errorf("failed to scan classification row: %w", err)
           }

           var criteria ClassifierCriteria
           if err := json.Unmarshal([]byte(criteriaJSON), &criteria); err != nil {
               return nil, fmt.Errorf("failed to unmarshal criteria: %w", err)
           }

           confidence := c.calculateMatchConfidence(filePath, fileInfo, criteria)
           if confidence >= threshold {
               matches = append(matches, ClassificationMatch{
                   FilePath:         filePath,
                   ClassificationID: id,
                   Confidence:       confidence,
                   Timestamp:        time.Now(),
               })
           }
       }

       return matches, nil
   }
   ```
3. Create advanced confidence scoring:
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
4. Implement pattern consolidation to merge similar patterns
5. Build rule generation that converts patterns and classifications to actionable rules

6. Create base classification definitions:
   ```sql
   -- Base file classifications
   INSERT INTO classifications (id, name, description, criteria_json, confidence_threshold, system_defined)
   VALUES
   ('doc', 'Documents', 'Common document file types',
   '{"extension_patterns":[".pdf",".doc",".docx",".txt",".rtf",".odt"],
     "name_patterns":["report","document","letter","memo"],
     "mime_types":["application/pdf","application/msword",
                  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
                  "text/plain"],
     "min_file_size":0,
     "max_file_size":0}',
   0.7, 1),

   ('img', 'Images', 'Common image file types',
   '{"extension_patterns":[".jpg",".jpeg",".png",".gif",".bmp",".tiff",".svg"],
     "name_patterns":["image","photo","picture","screenshot","img"],
     "mime_types":["image/jpeg","image/png","image/gif","image/bmp","image/tiff","image/svg+xml"],
     "min_file_size":0,
     "max_file_size":0}',
   0.7, 1),

   ('vid', 'Videos', 'Common video file types',
   '{"extension_patterns":[".mp4",".avi",".mkv",".mov",".wmv",".flv",".webm"],
     "name_patterns":["video","movie","clip","recording"],
     "mime_types":["video/mp4","video/x-msvideo","video/x-matroska","video/quicktime","video/x-ms-wmv"],
     "min_file_size":0,
     "max_file_size":0}',
   0.7, 1),

   ('aud', 'Audio', 'Common audio file types',
   '{"extension_patterns":[".mp3",".wav",".ogg",".flac",".aac",".m4a"],
     "name_patterns":["audio","sound","music","song","recording"],
     "mime_types":["audio/mpeg","audio/wav","audio/ogg","audio/flac","audio/aac","audio/mp4"],
     "min_file_size":0,
     "max_file_size":0}',
   0.7, 1),

   ('code', 'Source Code', 'Programming source code files',
   '{"extension_patterns":[".go",".java",".py",".js",".ts",".c",".cpp",".cs",".php",".rb",".swift"],
     "name_patterns":["main","index","app","utils","helpers","test"],
     "mime_types":["text/x-go","text/x-java","text/x-python","text/javascript","text/x-c"],
     "min_file_size":0,
     "max_file_size":0}',
   0.7, 1),

   ('arch', 'Archives', 'Compressed archive files',
   '{"extension_patterns":[".zip",".rar",".tar",".gz",".7z",".bz2"],
     "name_patterns":["archive","backup","compressed"],
     "mime_types":["application/zip","application/x-rar-compressed","application/x-tar","application/gzip"],
     "min_file_size":0,
     "max_file_size":0}',
   0.7, 1),

   ('data', 'Data Files', 'Structured data files',
   '{"extension_patterns":[".csv",".json",".xml",".yaml",".yml",".sql",".db",".sqlite"],
     "name_patterns":["data","export","backup","config"],
     "mime_types":["text/csv","application/json","application/xml","application/x-yaml"],
     "min_file_size":0,
     "max_file_size":0}',
   0.7, 1)
   ';
   ```

#### 4.3.1 Content-Based Analysis System

##### 4.3.1.1 Semantic File Relationship Detection

The content-based analysis system will be enhanced to provide semantic understanding of relationships between files. This will involve deeper analysis of content beyond basic signatures to identify meaningful connections between files.

**Requirements:**

1. **Advanced Content Understanding**
   - Text Analysis: Implement NLP techniques to extract semantic meaning from text files
   - Image Analysis: Implement perceptual hashing and feature extraction for visual similarity
   - Document Analysis: Extract and analyze content from PDFs and Office documents
   - Audio/Video Analysis: Implement acoustic fingerprinting and feature extraction
   - Binary Analysis: Implement fuzzy hashing for binary similarity detection

2. **Relationship Types to Detect**
   - Content-based relationships: Files with similar content but different formats
   - Semantic relationships: Files about the same topic or concept
   - Derivative relationships: Files created from other files (e.g., compressed versions)
   - Sequential relationships: Files that are part of a sequence (e.g., chapters, episodes)
   - Versioning relationships: Files that are different versions of the same document

3. **Semantic Grouping**
   - Topic clustering: Group files by subject matter or topic
   - Project detection: Identify files likely related to the same project
   - Temporal sequences: Detect files that form a chronological sequence
   - Version history: Track different versions of the same content

**Implementation:**

1. **Advanced Content Analysis Libraries**
   - Text Analysis:
     ```go
     // Implement NLP analysis using advanced libraries
     import (
         "github.com/jdkato/prose/tokenize"
         "github.com/bbalet/stopwords"
         "github.com/james-bowman/nlp"
     )

     func analyzeTextSemantics(content string) (map[string]float64, error) {
         // Remove stopwords
         cleanText := stopwords.CleanString(content, "en", true)

         // Tokenize
         tokenizer := tokenize.NewTreebankWordTokenizer()
         tokens := tokenizer.Tokenize(cleanText)

         // Create vector representation
         // ...

         return semanticMap, nil
     }
     ```

   - Image Analysis:
     ```go
     // Implement perceptual hashing for image similarity
     import (
         "github.com/corona10/goimagehash"
         "image"
     )

     func calculateImageHash(img image.Image) (string, error) {
         hash, err := goimagehash.PerceptionHash(img)
         if err != nil {
             return "", err
         }
         return hash.ToString(), nil
     }
     ```

   - Document Analysis:
     ```go
     // Extract text from PDF documents
     import (
         "github.com/unidoc/unipdf/model"
     )

     func extractPDFText(pdfPath string) (string, error) {
         // Extract text from PDF
         // ...

         return extractedText, nil
     }
     ```

2. **Relationship Detection Engine**
   ```go
   // Define a relationship detector interface
   type RelationshipDetector interface {
       DetectRelationships(sourceFile string, candidateFiles []string) ([]*FileRelationship, error)
       CalculateRelationshipStrength(file1, file2 string) (float64, string, error)
       GroupRelatedFiles(files []string, minSimilarity float64) ([]*ContentGroup, error)
   }
   ```

3. **Integration with Classification System**
   - Enhance the `ClassifyFile` method to incorporate semantic relationship data
   - Update rule suggestions to leverage file relationship information
   - Create new classification criteria based on file relationships

**Database Schema Extension:**
```sql
-- Semantic content features table for advanced analysis results
CREATE TABLE IF NOT EXISTS semantic_features (
    signature_id TEXT PRIMARY KEY,
    feature_type TEXT NOT NULL,
    feature_data BLOB NOT NULL,
    extracted_at TEXT NOT NULL,
    FOREIGN KEY (signature_id) REFERENCES content_signatures(id)
);

-- Semantic relationships between files
CREATE TABLE IF NOT EXISTS semantic_relationships (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    relationship_type TEXT NOT NULL,
    confidence REAL NOT NULL,
    metadata_json TEXT,
    created_at TEXT NOT NULL,
    FOREIGN KEY (source_id) REFERENCES content_signatures(id),
    FOREIGN KEY (target_id) REFERENCES content_signatures(id)
);

-- Topic clusters for semantic grouping
CREATE TABLE IF NOT EXISTS topic_clusters (
    id TEXT PRIMARY KEY,
    topic_name TEXT NOT NULL,
    confidence REAL NOT NULL,
    keywords_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- File membership in topic clusters
CREATE TABLE IF NOT EXISTS topic_members (
    cluster_id TEXT NOT NULL,
    signature_id TEXT NOT NULL,
    relevance_score REAL NOT NULL,
    PRIMARY KEY (cluster_id, signature_id),
    FOREIGN KEY (cluster_id) REFERENCES topic_clusters(id),
    FOREIGN KEY (signature_id) REFERENCES content_signatures(id)
);
```

This semantic relationship system will significantly enhance the Smart Rule Learning by providing deeper understanding of file relationships, leading to more intelligent organization suggestions and pattern detection.

#### 4.4 Feedback Mechanism
1. Design comprehensive feedback system:
   - Acceptance/rejection tracking
   - Modification analysis (what users change)
   - Feedback-based confidence adjustment
   - Classification accuracy feedback
2. Implement learning from rejections:
   - Identify pattern attributes that lead to rejections
   - Adjust detection algorithms based on rejection patterns
   - Update classification criteria based on feedback
3. Create modification tracking:
   - Analyze differences between suggested and accepted rules
   - Learn user preferences from modifications
   - Refine classification criteria based on user corrections
4. Develop a reinforcement learning approach:
   - Increase confidence for patterns similar to accepted suggestions
   - Decrease confidence for patterns similar to rejected suggestions
   - Adjust classification criteria based on confirmed matches

#### 4.5 UI Integration
1. Add "Smart Suggestions" panel to the Organize tab:
   - Display rule suggestions with confidence indicators
   - Show file classification results with confidence
   - Provide accept/modify/reject buttons
   - Show explanation of why each rule was suggested
2. Create rule testing interface:
   - Allow users to test suggestions on sample files
   - Preview classification results on files
   - Show preview of organization results
   - Provide fine-tuning controls
3. Add classifications management UI:
   - View and edit existing classifications
   - Create new custom classifications
   - Test classifications against sample files
4. Add advanced settings panel:
   - Learning sensitivity controls
   - Classification threshold settings
   - History management options
   - System reset capabilities
5. Implement notification system for high-confidence suggestions and classifications

#### 4.6 Implementation Strategy
1. **Phase 1: Core Infrastructure (Current Focus)**
   - Setup module structure and database schema ✅
   - Implement SQLite with Go embed integration 🔄
   - Create baseline classifications ✅
   - Implement persistence layer interfaces ✅

2. **Phase 2: Operation Tracking & Analysis**
   - Integrate tracking hooks in organize engine
   - Add operation metrics collection
   - Implement file classification engine
   - Create basic pattern detection

3. **Phase 3: Rule Generation & Learning**
   - Implement rule generation from patterns
   - Create confidence scoring system
   - Build pattern consolidation logic
   - Implement feedback-based refinement

4. **Phase 4: UI & User Experience**
   - Add suggestions panel to UI
   - Create classification management interface
   - Implement feedback collection
   - Add settings and configuration options

5. **Dependencies & Requirements**
   - Go embed for database embedding
   - SQLite for structured storage
   - UUID generation for unique identifiers
   - JSON for criteria serialization/deserialization

### 5. Comprehensive Observability System
**Objective**: Implement a robust observability system throughout the entire Sortd application to provide users with clear visibility into all operations and progress.

**Tasks**:
- [ ] Design a comprehensive observability architecture
- [x] Implement simple progress logging for all file operations
- [x] Create detailed activity logging with structured data
- [ ] Develop visual indicators for operation status
- [ ] Build a dashboard for operation monitoring

**Implementation Steps**:

#### 5.1 Core Observability Architecture
1. Design the observability framework:
   ```go
   // Observability provides tracking for all operations
   type Observability interface {
       // StartOperation begins tracking a named operation with expected total items
       StartOperation(name string, totalItems int) OperationTracker

       // LogAction records a specific action with context
       LogAction(category string, message string, fields ...Field)

       // GetRecentActivity returns recent actions
       GetRecentActivity(limit int) []Activity
   }

   // OperationTracker tracks progress of a single operation
   type OperationTracker interface {
       // Update increments the progress counter and returns completion percentage
       Update(increment int) (completed int, total int, percentage float64)

       // AddDetail adds contextual information to the operation
       AddDetail(key string, value interface{})

       // Complete marks the operation as finished
       Complete()

       // Failed marks the operation as failed with error
       Failed(err error)
   }
   ```

2. Create implementations for different contexts:
   - CLIObservability: Provides progress bars and console logging
   - WebObservability: Enables real-time updates via SSE for web interfaces
   - APIObservability: Returns structured data for API clients
   - FileObservability: Logs all activities to rotating log files

3. Implement observability provider:
   ```go
   // New creates an observability provider appropriate for the context
   func NewObservability(ctx context.Context, config *ObservabilityConfig) Observability {
       // Determine context (CLI, Web, API, etc.)
       if isWebContext(ctx) {
           return NewWebObservability(config)
       } else if isAPIContext(ctx) {
           return NewAPIObservability(config)
       } else {
           return NewCLIObservability(config)
       }
   }
   ```

#### 5.2 Progress Tracking System
1. Implement operation trackers for different contexts:
   - **CLI Progress Bar**: Using libraries like progressbar or go-pretty
   - **Web Progress Updates**: Using Server-Sent Events for real-time updates
   - **Status Files**: For tracking long-running operations

2. Create a generic progress tracking framework:
   ```go
   // Operation represents a tracked operation
   type Operation struct {
       ID          string
       Name        string
       Started     time.Time
       Updated     time.Time
       Completed   time.Time
       Total       int
       Current     int
       Percentage  float64
       Status      string // "running", "completed", "failed"
       Details     map[string]interface{}
       Error       string
   }

   // TrackOperation wraps any operation with progress tracking
   func (o *StandardObservability) TrackOperation(name string, totalItems int) OperationTracker {
       op := &Operation{
           ID:      uuid.New().String(),
           Name:    name,
           Started: time.Now(),
           Total:   totalItems,
           Status:  "running",
           Details: make(map[string]interface{}),
       }

       o.activeOperations.Store(op.ID, op)
       o.notifyListeners(op)

       return &operationTracker{
           operation: op,
           parent:    o,
       }
   }
   ```

3. Implement hooks throughout the codebase:
   - Add to file operations in organize package
   - Add to workflow execution
   - Add to learning system operations
   - Add to batch processing

#### 5.3 Activity Logging System
1. Define structured activity logs:
   ```go
   // Activity represents a logged action
   type Activity struct {
       Timestamp time.Time
       Category  string
       Message   string
       Fields    map[string]interface{}
   }
   ```

2. Implement logging with context:
   ```go
   // LogAction records a specific action with context
   func (o *StandardObservability) LogAction(category string, message string, fields ...Field) {
       activity := Activity{
           Timestamp: time.Now(),
           Category:  category,
           Message:   message,
           Fields:    make(map[string]interface{}),
       }

       // Add fields to the activity
       for _, field := range fields {
           activity.Fields[field.Key] = field.Value
       }

       // Store in recent activities buffer
       o.recentActivities.Add(activity)

       // Notify listeners
       o.notifyActivityListeners(activity)

       // Log to file/console depending on context
       o.logger.With(
          log.F("category", category),
          transformFields(fields)...,
       ).Info(message)
   }
   ```

3. Create activity categories for different operations:
   - `FILE_OPERATION`: File moves, copies, and deletions
   - `ORGANIZATION`: Pattern matching and organization actions
   - `CLASSIFICATION`: File classification operations
   - `WORKFLOW`: Workflow execution steps
   - `SYSTEM`: System-level operations
   - `ERROR`: Error conditions
   - `SECURITY`: Security-related events

#### 5.4 Visual Indicators
1. Implement CLI visual indicators:
   - Progress bars for file operations
   - Spinners for indeterminate operations
   - Color-coded status messages

2. Design web UI components:
   - Real-time progress bars
   - Activity feed with filtering
   - Operation dashboard
   - Status indicators

3. Create notification system for important events:
   - Operation completion notifications
   - Error alerts
   - System status changes

#### 5.5 Usage Examples
1. Tracking file organization:
   ```go
   // In organize.Engine.OrganizeByPatterns:
   tracker := o.observability.TrackOperation("File Organization", len(files))
   defer tracker.Complete()

   for i, file := range files {
       // Process file...

       // Update progress after each file
       tracker.Update(1)

       // Log specific action
       o.observability.LogAction("FILE_OPERATION", "Organized file",
           log.F("source", file),
           log.F("destination", destPath),
           log.F("pattern", matchedPattern),
       )
   }
   ```

2. Tracking workflow execution:
   ```go
   // In workflow.Manager.ExecuteWorkflow:
   tracker := o.observability.TrackOperation("Workflow: " + workflow.Name, len(workflow.Actions))
   defer tracker.Complete()

   for i, action := range workflow.Actions {
       // Log workflow step
       o.observability.LogAction("WORKFLOW", "Executing workflow step",
           log.F("workflow", workflow.Name),
           log.F("step", i+1),
           log.F("action", action.Type),
       )

       // Execute action...

       // Update progress
       tracker.Update(1)
   }
   ```

3. Tracking learning system operations:
   ```go
   // In learning.Engine.PerformAnalysis:
   tracker := o.observability.TrackOperation("Pattern Analysis", estimatedOperations)
   defer tracker.Complete()

   // Get operations to analyze
   operations, err := e.repo.GetRecentOperations(30, 1000)
   if err != nil {
       tracker.Failed(err)
       return err
   }

   // Update with actual count
   tracker.AddDetail("actualOperations", len(operations))

   // Process operations...
   for i, op := range operations {
       // Analyze operation...

       // Update progress periodically (e.g., every 10 operations)
       if i%10 == 0 {
           tracker.Update(10)
       }
   }
   ```

#### 5.6 Integration with Sandbox Environment
1. Extend the Docker sandbox environment to showcase observability features:
   - Display real-time progress in sandbox CLI
   - Provide web UI access to operation dashboard
   - Demonstrate activity logging
   - Show notification system

2. Create sandbox examples demonstrating different observability features:
   - Batch file organization with progress tracking
   - Long-running workflow execution with status updates
   - Error handling and notification
   - Classification and learning visualization

This comprehensive observability system will ensure users always have full visibility into what Sortd is doing, enhancing trust and providing clear feedback for all operations.

## Implementation Timeline (Refined)

### Week 1: Foundation and Core Architecture
1. Complete repository interface and SQLite implementation
   - [x] Operation record storage and retrieval
   - [x] Learning settings operations
   - [ ] File classification methods
   - [ ] Pattern detection methods
   - [ ] Classification match methods
   - [ ] Rule suggestion methods
2. Set up core engine framework
   - [x] Operation tracking
   - [x] Analysis scheduling
   - [x] Learning toggle functionality
   - [ ] Classification engine

#### Week 2: Pattern Detection and Classification
1. Implement basic pattern detection algorithms
   - [ ] Extension-based pattern detection
   - [ ] Name-based pattern detection
   - [ ] Time-based pattern detection
   - [ ] Size-based pattern detection
2. Build file classification system
   - [ ] Implement `ClassifyFile` method
   - [ ] Create confidence scoring system
   - [ ] Build classification matching algorithm
3. Implement content-based analysis system
   - [x] Create content signature generation
   - [x] Implement MIME type detection
   - [x] Build text content analysis (keywords, frequency maps)
   - [x] Develop image content analysis (metadata, color histograms)
   - [x] Set up document content extraction
   - [x] Create content signature storage

#### Week 3: Content Analysis and Learning Engine
1. Extend content analysis capabilities
   - [x] Implement content similarity engine
   - [x] Create content relationship discovery
   - [x] Build content group detection
   - [x] Integrate content analysis with classification
2. Complete pattern learning system
   - [ ] Implement pattern consolidation
   - [ ] Build pattern weighting system
   - [ ] Develop confidence threshold calibration
3. Implement rule generation
   - [ ] Create rule suggestion engine
   - [ ] Build rule template system
   - [ ] Implement rule confidence scoring
   - [x] Develop content-based rule generation

#### Week 4: Integration and Testing
1. Implement engine integration
   - [x] Create integration between learning engine and organize engine
   - [x] Implement EngineAdapter for learning system
   - [x] Add content-based classification to file organization
   - [x] Track file operations for learning
2. Set up testing framework
   - [ ] Create unit tests for learning engine
   - [ ] Implement integration tests for engine adapter
   - [ ] Create Docker-based testing environment
   - [ ] Set up performance benchmarks
3. Comprehensive testing
   - [ ] Test with various file types (text, images, documents, etc.)
   - [ ] Test content relationship detection
   - [ ] Test classification-based organization
   - [ ] Test performance with large file sets

## Next Immediate Tasks

To continue making progress on the Smart Rule Learning feature, we should focus on these immediate tasks:

1. **Create Directory Structure**
   - Create the `internal/patterns/learning` directory structure ✅
   - Set up the `db` subdirectory for SQL files ✅

2. **Database Schema Implementation**
   - Implement the `schema.sql` file with the defined schema ✅
   - Create the `classifications.sql` file with base classifications ✅

3. **Core Package Implementation**
   - Implement the data models in `model.go` ✅
   - Create the repository interface in `persistence.go` ✅
   - Implement the SQLite repository that implements the interface ✅

4. **Configuration Management**
   - Create the configuration structure in `config.go` ✅
   - Implement configuration validation and defaults ✅

5. **Database Initialization**
   - Implement the database initialization with Go embed ✅
   - Add proper error handling with the errors package ✅
   - Implement connection management for concurrency ✅

6. **Core Engine Implementation**
   - Create the main learning engine in `engine.go` ✅
   - Implement the operation tracking functionality ✅
   - Add periodic analysis scheduling ✅
   - Implement the ClassifyFile functionality ✅
   - Integrate content analysis with classification ✅

7. **Core Implementation Updates**
   - Update EngineAdapter to support content-based classification ✅
   - Implement operation tracking in file move operations ✅
   - Add daemon support for learning engine initialization ✅
   - Implement direct progress logging for all operations ✅

8. **Testing Implementation**
   - Create comprehensive unit tests for all components
   - Set up Docker-based testing environment
   - Test with various real-world file types
   - Benchmark performance with large datasets

## Progress Tracking

We'll track progress on the Smart Rule Learning feature implementation with weekly updates, focusing on the following metrics:

- **Code completion percentage**: Track completion of each component
- **Test coverage**: Ensure proper test coverage for all components
- **Integration status**: Track integration with the main application
- **Performance metrics**: Monitor database operation performance
- **User feedback**: Once implemented, collect user feedback on the feature

Each completed task will be marked in this document, and any issues or new requirements that emerge during implementation will be documented and addressed.