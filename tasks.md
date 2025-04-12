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

1. Extend database schema for content analysis:
   ```sql
   -- Content signatures table for storing file content analysis
   CREATE TABLE IF NOT EXISTS content_signatures (
       file_path TEXT PRIMARY KEY,
       mime_type TEXT NOT NULL,
       content_hash TEXT NOT NULL,
       keywords_json TEXT,
       metadata_json TEXT,
       signature_data_json TEXT NOT NULL,
       created_at TEXT NOT NULL,
       updated_at TEXT NOT NULL
   );

   -- Content similarities table for tracking relationships
   CREATE TABLE IF NOT EXISTS content_similarities (
       file_path1 TEXT NOT NULL,
       file_path2 TEXT NOT NULL,
       similarity_score REAL NOT NULL,
       compared_at TEXT NOT NULL,
       PRIMARY KEY (file_path1, file_path2)
   );

   -- Content relationship groups
   CREATE TABLE IF NOT EXISTS content_groups (
       id TEXT PRIMARY KEY,
       name TEXT NOT NULL,
       description TEXT NOT NULL,
       primary_keywords_json TEXT NOT NULL,
       confidence REAL NOT NULL,
       created_at TEXT NOT NULL
   );

   -- Files in content groups
   CREATE TABLE IF NOT EXISTS content_group_members (
       group_id TEXT NOT NULL,
       file_path TEXT NOT NULL,
       confidence REAL NOT NULL,
       PRIMARY KEY (group_id, file_path),
       FOREIGN KEY (group_id) REFERENCES content_groups(id)
   );
   ```

2. Create file content analyzer infrastructure:
   ```go
   // ContentSignature represents a file's content metadata
   type ContentSignature struct {
       FilePath     string                 `json:"file_path"`
       MimeType     string                 `json:"mime_type"`
       ContentHash  string                 `json:"content_hash"`
       Keywords     []string               `json:"keywords"`
       Metadata     map[string]string      `json:"metadata"`
       FrequencyMap map[string]int         `json:"frequency_map"`
       SignatureData map[string]interface{} `json:"signature_data"`
       CreatedAt    time.Time              `json:"created_at"`
       UpdatedAt    time.Time              `json:"updated_at"`
   }

   // ContentAnalyzer generates signatures from file content
   type ContentAnalyzer interface {
       AnalyzeFile(filePath string) (*ContentSignature, error)
       CompareSignatures(sig1, sig2 *ContentSignature) (float64, error)
       ExtractKeywords(content []byte) ([]string, error)
       GenerateContentHash(content []byte) (string, error)
   }
   ```

3. Implement MIME type detection with appropriate Go libraries:
   ```go
   import "github.com/gabriel-vasile/mimetype"

   // MIMETypeDetector detects file MIME types
   type MIMETypeDetector struct{}

   // DetectFile determines the MIME type of a file
   func (d *MIMETypeDetector) DetectFile(filePath string) (string, error) {
       mime, err := mimetype.DetectFile(filePath)
       if err != nil {
           return "", errors.NewFileError("failed to detect MIME type", filePath, errors.FileOperationFailed, err)
       }
       return mime.String(), nil
   }
   ```

4. Implement content analysis for different file types:
   - Text files (using NLP techniques)
   - Image files (using metadata and potentially color analysis)
   - Document files (PDF, Office docs)
   - Audio/video files (metadata extraction)
   - Binary files (header analysis, basic signatures)

5. Implement content similarity engine:
   ```go
   // SimilarityEngine calculates relationships between files based on content
   type SimilarityEngine struct {
       repo Repository
       analyzer ContentAnalyzer
       thresholds map[string]float64
   }

   // FindRelatedFiles identifies files related to a given file
   func (e *SimilarityEngine) FindRelatedFiles(filePath string, minSimilarity float64) ([]*ContentSimilarity, error) {
       // Get signature for source file
       sourceSig, err := e.getOrCreateSignature(filePath)
       if err != nil {
           return nil, err
       }

       // Query existing signatures
       allSigs, err := e.repo.GetAllContentSignatures(1000)
       if err != nil {
           return nil, err
       }

       var similarities []*ContentSimilarity

       // Calculate similarities
       for _, targetSig := range allSigs {
           if targetSig.FilePath == filePath {
               continue
           }

           score, err := e.analyzer.CompareSignatures(sourceSig, targetSig)
           if err != nil {
               continue
           }

           if score >= minSimilarity {
               similarities = append(similarities, &ContentSimilarity{
                   FilePath1: sourceSig.FilePath,
                   FilePath2: targetSig.FilePath,
                   Score: score,
                   ComparedAt: time.Now(),
               })
           }
       }

       return similarities, nil
   }
   ```

6. Create content relationship discovery process:
   ```go
   // ContentRelationshipDiscovery finds groups of related files
   type ContentRelationshipDiscovery struct {
       repo Repository
       similarityEngine *SimilarityEngine
       minGroupSize int
       minSimilarityScore float64
   }

   // DiscoverRelationships identifies content relationships across the system
   func (d *ContentRelationshipDiscovery) DiscoverRelationships() ([]*ContentGroup, error) {
       // Get all content similarities above threshold
       similarities, err := d.repo.GetContentSimilarities(d.minSimilarityScore)
       if err != nil {
           return nil, err
       }

       // Build graph of relationships
       graph := d.buildRelationshipGraph(similarities)

       // Use clustering algorithm to find groups
       groups := d.findContentGroups(graph)

       // Save discovered groups
       for _, group := range groups {
           if err := d.repo.SaveContentGroup(group); err != nil {
               continue
           }
       }

       return groups, nil
   }
   ```

7. Integrate content analysis into the file classification process:
   ```go
   // UpdateClassifyFile to include content-based classification
   func (e *Engine) ClassifyFile(filePath string) ([]*ClassificationMatch, error) {
       // ... existing code ...

       // Add content-based analysis
       contentSignature, _ := e.contentAnalyzer.AnalyzeFile(filePath)
       if contentSignature != nil {
           // Store the signature for future reference
           e.repo.SaveContentSignature(contentSignature)

           // Match content signature against classification criteria
           for _, classification := range classifications {
               // Calculate content match confidence
               contentConfidence := e.calculateContentMatch(contentSignature, classification.Criteria)

               // Add content confidence to overall score (50% weight)
               confidence += 0.5 * contentConfidence
           }
       }

       // ... existing code ...
   }
   ```

8. Leverage existing Go libraries:
   - `github.com/gabriel-vasile/mimetype` for MIME detection
   - `github.com/jdkato/prose` for NLP and text analysis
   - `github.com/disintegration/imaging` for image analysis
   - `github.com/dhowden/tag` for audio file metadata
   - `gonum.org/v1/gonum/stat` for statistical analysis of content
   - `github.com/sajari/docconv` for document conversion/text extraction

9. Implement periodic content analysis for enhanced pattern detection:
   ```go
   // Add to Engine.PerformAnalysis
   func (e *Engine) PerformAnalysis() error {
       // ... existing code ...

       // Perform content relationship discovery
       if e.config.ContentAnalysisEnabled {
           contentGroups, err := e.contentDiscovery.DiscoverRelationships()
           if err != nil {
               e.logger.With(log.F("error", err)).Warn("Content relationship discovery failed")
           } else {
               e.logger.With(log.F("groupCount", len(contentGroups))).Info("Discovered content relationships")

               // Generate content-based rule suggestions
               if err := e.generateContentBasedRules(contentGroups); err != nil {
                   e.logger.With(log.F("error", err)).Warn("Failed to generate content-based rules")
               }
           }
       }

       // ... existing code ...
   }
   ```

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
   - [ ] Create content signature generation
   - [ ] Implement MIME type detection
   - [ ] Build text content analysis (keywords, frequency maps)
   - [ ] Develop image content analysis (metadata, color histograms)
   - [ ] Set up document content extraction
   - [ ] Create content signature storage

#### Week 3: Content Analysis and Learning Engine
1. Extend content analysis capabilities
   - [ ] Implement content similarity engine
   - [ ] Create content relationship discovery
   - [ ] Build content group detection
   - [ ] Integrate content analysis with classification
2. Complete pattern learning system
   - [ ] Implement pattern consolidation
   - [ ] Build pattern weighting system
   - [ ] Develop confidence threshold calibration
3. Implement rule generation
   - [ ] Create rule suggestion engine
   - [ ] Build rule template system
   - [ ] Implement rule confidence scoring
   - [ ] Develop content-based rule generation

#### Week 4: Feedback System and Integration
1. Implement feedback mechanism
   - [ ] Create feedback collection system
   - [ ] Build reinforcement learning for patterns
   - [ ] Implement user preference tracking
2. Integrate with main application
   - [ ] Add hooks in organize engine
   - [ ] Create analysis triggers on file operations
   - [ ] Build UI elements for rule suggestions
   - [ ] Implement workflow suggestion feature
3. Test and refine
   - [ ] Create comprehensive test suite
   - [ ] Perform performance optimization
   - [ ] Refine confidence scoring algorithm
   - [ ] Polish user experience for suggestions

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

7. **Fix Dependencies and Error Handling**
   - Add missing dependencies (go-sqlite3, uuid) ✅
   - Fix error handling implementation in repository ✅
   - Fix logger parameter handling ✅

8. **Next Steps for Implementation**
   - Implement the pattern detector component
   - Create the file classifier component
   - Implement the suggestion generator
   - Add hooks in the organize engine to track operations

## Progress Tracking

We'll track progress on the Smart Rule Learning feature implementation with weekly updates, focusing on the following metrics:

- **Code completion percentage**: Track completion of each component
- **Test coverage**: Ensure proper test coverage for all components
- **Integration status**: Track integration with the main application
- **Performance metrics**: Monitor database operation performance
- **User feedback**: Once implemented, collect user feedback on the feature

Each completed task will be marked in this document, and any issues or new requirements that emerge during implementation will be documented and addressed.