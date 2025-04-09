package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gobwas/glob"
	"gopkg.in/yaml.v3"

	"sortd/pkg/types"
)

// Manager handles the loading, evaluating, and executing of workflows
type Manager struct {
	workflows  []types.Workflow
	configPath string
	dryRun     bool
}

// NewManager creates a new workflow manager instance
func NewManager(configPath string) (*Manager, error) {
	manager := &Manager{
		configPath: configPath,
		dryRun:     false,
	}

	// Load workflows from config
	if err := manager.LoadWorkflows(); err != nil {
		return nil, err
	}

	return manager, nil
}

// LoadWorkflows loads workflow definitions from the config directory
func (m *Manager) LoadWorkflows() error {
	m.workflows = []types.Workflow{}

	// Ensure the config directory exists
	if err := os.MkdirAll(m.configPath, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// List all YAML files in the config directory
	entries, err := os.ReadDir(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}

		path := filepath.Join(m.configPath, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read workflow file %s: %w", path, err)
		}

		var workflow types.Workflow
		if err := yaml.Unmarshal(data, &workflow); err != nil {
			return fmt.Errorf("failed to parse workflow file %s: %w", path, err)
		}

		// Validate the workflow
		if err := validateWorkflow(&workflow); err != nil {
			return fmt.Errorf("invalid workflow in %s: %w", path, err)
		}

		m.workflows = append(m.workflows, workflow)
	}

	return nil
}

// validateWorkflow performs basic validation on a workflow definition
func validateWorkflow(workflow *types.Workflow) error {
	if workflow.ID == "" {
		return errors.New("workflow ID is required")
	}

	if workflow.Name == "" {
		return errors.New("workflow name is required")
	}

	if len(workflow.Actions) == 0 {
		return errors.New("workflow must have at least one action")
	}

	return nil
}

// ProcessEvent handles a single file system event received from an external watcher.
// It checks if any enabled workflows should be triggered by this event based on
// type, pattern, and conditions. If a matching workflow is found and executed,
// it returns processed=true. If execution fails, it returns processed=false and the error.
func (m *Manager) ProcessEvent(event fsnotify.Event) (processed bool, err error) {
	// Skip temporary and hidden files
	fileName := filepath.Base(event.Name)
	if strings.HasPrefix(fileName, ".") || strings.HasSuffix(fileName, "~") {
		return false, nil // Not processed, no error
	}

	// Basic check: ensure file exists before proceeding (might have been deleted quickly)
	fileInfo, statErr := os.Stat(event.Name)
	if statErr != nil {
		// Log? For now, just treat as not processed
		return false, nil
	}

	// Determine event type for trigger matching
	var triggerType types.TriggerType
	// Only consider Create or Write events for workflow triggers
	if event.Op&fsnotify.Create == fsnotify.Create {
		triggerType = types.FileCreated
	} else if event.Op&fsnotify.Write == fsnotify.Write {
		triggerType = types.FileModified
	} else {
		return false, nil // Skip other event types (like Chmod, Remove)
	}

	// Iterate through loaded workflows to find a match
	var workflowProcessed bool = false // Track if any workflow handled this

	for _, workflow := range m.workflows {
		if !workflow.Enabled {
			continue
		}

		// Check if trigger type matches
		// Allow FilePatternMatch to trigger on Create or Write events
		triggerMatches := (workflow.Trigger.Type == triggerType) ||
			(workflow.Trigger.Type == types.FilePatternMatch && (triggerType == types.FileCreated || triggerType == types.FileModified))

		if !triggerMatches {
			continue
		}

		// --- Trigger Type Matches ---
		// Now, always check the pattern if one is defined in the trigger
		if workflow.Trigger.Pattern != "" {
			patternMatcher, compileErr := glob.Compile(workflow.Trigger.Pattern)
			if compileErr != nil {
				fmt.Fprintf(os.Stderr, "Error compiling workflow pattern '%s' for %s: %v\n", workflow.Trigger.Pattern, workflow.ID, compileErr)
				continue // Skip workflow with invalid pattern
			}
			// Match against the full path of the event
			if !patternMatcher.Match(event.Name) {
				continue // Pattern doesn't match
			}
		}
		// At this point, the trigger type and pattern (if applicable) match

		// Evaluate conditions using the fileInfo we got earlier
		if !m.evaluateConditions(workflow.Conditions, event.Name, fileInfo) {
			continue // Conditions not met
		}

		// --- Trigger and Conditions Met ---
		// Execute the workflow actions
		result := m.executeWorkflow(workflow, event.Name)
		workflowProcessed = true // Mark that at least one workflow was triggered

		// Log the result
		fmt.Printf("Workflow %s (%s) execution: %v\n", workflow.Name, workflow.ID, result.Success)
		if !result.Success && result.Error != nil {
			fmt.Printf("  Error: %v\n", result.Error)
			// If a workflow fails, return processed=true (it was attempted) but also return the error
			return true, result.Error
		}

		// If we successfully executed *this* workflow, we consider the event processed by workflows.
		// We could add logic here to stop processing further workflows if needed (e.g., based on workflow priority or a 'stop processing' flag)
		// For now, we'll let subsequent workflows also trigger if they match.
	}

	// Return true if any workflow was triggered and executed (even if others didn't match)
	// Return nil error if all triggered workflows executed successfully
	return workflowProcessed, nil
}

// evaluateConditions checks if a file meets all the conditions
func (m *Manager) evaluateConditions(conditions []types.Condition, filePath string, fileInfo os.FileInfo) bool {
	if len(conditions) == 0 {
		return true // No conditions means always match
	}

	for _, condition := range conditions {
		if !m.evaluateCondition(condition, filePath, fileInfo) {
			return false
		}
	}

	return true
}

// evaluateCondition checks if a file meets a specific condition
func (m *Manager) evaluateCondition(condition types.Condition, filePath string, fileInfo os.FileInfo) bool {
	switch condition.Type {
	case types.FileSizeCondition:
		return m.evaluateFileSizeCondition(condition, fileInfo)
	case types.FileNameCondition:
		return m.evaluateFileNameCondition(condition, filePath)
	case types.FileTypeCondition:
		return m.evaluateFileTypeCondition(condition, filePath)
	case types.FileAgeCondition:
		return m.evaluateFileAgeCondition(condition, fileInfo)
	default:
		return false
	}
}

// evaluateFileSizeCondition checks if a file's size meets the condition
func (m *Manager) evaluateFileSizeCondition(condition types.Condition, fileInfo os.FileInfo) bool {
	size := fileInfo.Size()
	targetSize, err := strconv.ParseInt(condition.Value, 10, 64)
	if err != nil {
		return false
	}

	// Apply unit multiplier if specified
	switch strings.ToUpper(condition.ValueUnit) {
	case "KB":
		targetSize *= 1024
	case "MB":
		targetSize *= 1024 * 1024
	case "GB":
		targetSize *= 1024 * 1024 * 1024
	}

	switch condition.Operator {
	case types.Equals:
		return size == targetSize
	case types.NotEquals:
		return size != targetSize
	case types.GreaterThan:
		return size > targetSize
	case types.LessThan:
		return size < targetSize
	default:
		return false
	}
}

// evaluateFileNameCondition checks if a file's name meets the condition
func (m *Manager) evaluateFileNameCondition(condition types.Condition, filePath string) bool {
	fileName := filepath.Base(filePath)

	switch condition.Operator {
	case types.Equals:
		return fileName == condition.Value
	case types.NotEquals:
		return fileName != condition.Value
	case types.Contains:
		return strings.Contains(fileName, condition.Value)
	case types.StartsWith:
		return strings.HasPrefix(fileName, condition.Value)
	case types.EndsWith:
		return strings.HasSuffix(fileName, condition.Value)
	case types.MatchesRegex:
		matched, err := regexp.MatchString(condition.Value, fileName)
		return err == nil && matched
	default:
		return false
	}
}

// evaluateFileTypeCondition checks if a file's type meets the condition
func (m *Manager) evaluateFileTypeCondition(condition types.Condition, filePath string) bool {
	// For simplicity, we're just checking file extension here
	// A more comprehensive implementation would use MIME type detection
	fileExt := strings.ToLower(filepath.Ext(filePath))
	if fileExt != "" && fileExt[0] == '.' {
		fileExt = fileExt[1:] // Remove leading dot
	}

	switch condition.Operator {
	case types.Equals:
		return fileExt == condition.Value
	case types.NotEquals:
		return fileExt != condition.Value
	case types.Contains:
		return strings.Contains(fileExt, condition.Value)
	default:
		return false
	}
}

// evaluateFileAgeCondition checks if a file's age meets the condition
func (m *Manager) evaluateFileAgeCondition(condition types.Condition, fileInfo os.FileInfo) bool {
	modTime := fileInfo.ModTime()
	ageInSeconds := time.Since(modTime).Seconds()

	targetAge, err := strconv.ParseFloat(condition.Value, 64)
	if err != nil {
		return false
	}

	// Apply unit multiplier if specified
	switch strings.ToLower(condition.ValueUnit) {
	case "minutes":
		targetAge *= 60
	case "hours":
		targetAge *= 3600
	case "days":
		targetAge *= 86400
	}

	switch condition.Operator {
	case types.Equals:
		return ageInSeconds == targetAge
	case types.NotEquals:
		return ageInSeconds != targetAge
	case types.GreaterThan:
		return ageInSeconds > targetAge
	case types.LessThan:
		return ageInSeconds < targetAge
	default:
		return false
	}
}

// executeWorkflow performs the actions defined in a workflow
func (m *Manager) executeWorkflow(workflow types.Workflow, filePath string) types.WorkflowResult {
	result := types.WorkflowResult{
		WorkflowID:   workflow.ID,
		WorkflowName: workflow.Name,
		FilePath:     filePath,
		Success:      true,
	}

	for _, action := range workflow.Actions {
		if err := m.executeAction(action, filePath); err != nil {
			result.Success = false
			result.Error = err
			result.Message = fmt.Sprintf("Failed to execute action: %v", err)
			return result
		}
	}

	result.Message = "All actions completed successfully"
	return result
}

// executeAction performs a single action
func (m *Manager) executeAction(action types.Action, filePath string) error {
	switch action.Type {
	case types.MoveAction:
		return m.executeMoveAction(action, filePath)
	case types.CopyAction:
		return m.executeCopyAction(action, filePath)
	case types.RenameAction:
		return m.executeRenameAction(action, filePath)
	case types.TagAction:
		return m.executeTagAction(action, filePath)
	case types.DeleteAction:
		return m.executeDeleteAction(action, filePath)
	case types.ExecuteAction:
		return m.executeCommandAction(action, filePath)
	default:
		return fmt.Errorf("unsupported action type: %s", action.Type)
	}
}

// executeMoveAction moves a file to a target directory
func (m *Manager) executeMoveAction(action types.Action, filePath string) error {
	// Create target directory if it doesn't exist
	if action.Options["createTargetDir"] == "true" {
		if err := os.MkdirAll(action.Target, 0755); err != nil {
			return fmt.Errorf("failed to create target directory: %w", err)
		}
	}

	// Construct target path
	fileName := filepath.Base(filePath)
	targetPath := filepath.Join(action.Target, fileName)

	// Handle existing files at the destination
	targetExists := false
	if _, err := os.Stat(targetPath); err == nil {
		targetExists = true
		if action.Options["overwrite"] == "true" {
			// Remove existing file (in non-dry run mode)
			if !m.dryRun {
				if err := os.Remove(targetPath); err != nil {
					return fmt.Errorf("failed to remove existing file: %w", err)
				}
			}
		} else {
			// Rename with a unique suffix
			targetPath = m.generateUniqueFilePath(targetPath)
		}
	}

	// In dry run mode, just log what would happen
	if m.dryRun {
		if targetExists && action.Options["overwrite"] == "true" {
			fmt.Printf("[DRY RUN] Would overwrite existing file: %s\n", targetPath)
		}
		fmt.Printf("[DRY RUN] Would move file from %s to %s\n", filePath, targetPath)
		return nil
	}

	// Move the file
	if err := os.Rename(filePath, targetPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

// executeCopyAction copies a file to a target directory
func (m *Manager) executeCopyAction(action types.Action, filePath string) error {
	// Create target directory if it doesn't exist
	if action.Options["createTargetDir"] == "true" {
		if err := os.MkdirAll(action.Target, 0755); err != nil {
			return fmt.Errorf("failed to create target directory: %w", err)
		}
	}

	// Construct target path
	fileName := filepath.Base(filePath)
	targetPath := filepath.Join(action.Target, fileName)

	// Handle existing files at the destination
	targetExists := false
	if _, err := os.Stat(targetPath); err == nil {
		targetExists = true
		if action.Options["overwrite"] == "true" {
			// Remove existing file (in non-dry run mode)
			if !m.dryRun {
				if err := os.Remove(targetPath); err != nil {
					return fmt.Errorf("failed to remove existing file: %w", err)
				}
			}
		} else {
			// Rename with a unique suffix
			targetPath = m.generateUniqueFilePath(targetPath)
		}
	}

	// In dry run mode, just log what would happen
	if m.dryRun {
		if targetExists && action.Options["overwrite"] == "true" {
			fmt.Printf("[DRY RUN] Would overwrite existing file: %s\n", targetPath)
		}
		fmt.Printf("[DRY RUN] Would copy file from %s to %s\n", filePath, targetPath)
		return nil
	}

	// Copy the file
	sourceFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer targetFile.Close()

	if _, err := targetFile.ReadFrom(sourceFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}

// executeRenameAction renames a file
func (m *Manager) executeRenameAction(action types.Action, filePath string) error {
	// Get directory and new file name
	dir := filepath.Dir(filePath)
	newName := action.Target
	targetPath := filepath.Join(dir, newName)

	// Handle existing files at the destination
	targetExists := false
	if _, err := os.Stat(targetPath); err == nil {
		targetExists = true
		if action.Options["overwrite"] == "true" {
			// Remove existing file (in non-dry run mode)
			if !m.dryRun {
				if err := os.Remove(targetPath); err != nil {
					return fmt.Errorf("failed to remove existing file: %w", err)
				}
			}
		} else {
			// Rename with a unique suffix
			targetPath = m.generateUniqueFilePath(targetPath)
		}
	}

	// In dry run mode, just log what would happen
	if m.dryRun {
		if targetExists && action.Options["overwrite"] == "true" {
			fmt.Printf("[DRY RUN] Would overwrite existing file: %s\n", targetPath)
		}
		fmt.Printf("[DRY RUN] Would rename file from %s to %s\n", filePath, targetPath)
		return nil
	}

	// Rename the file
	if err := os.Rename(filePath, targetPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// executeTagAction adds tags to a file
func (m *Manager) executeTagAction(action types.Action, filePath string) error {
	// In dry run mode, just log what would happen
	if m.dryRun {
		fmt.Printf("[DRY RUN] Would add tag '%s' to file %s\n", action.Target, filePath)
		return nil
	}

	// Placeholder for tag implementation
	fmt.Printf("Added tag '%s' to file %s\n", action.Target, filePath)
	return nil
}

// executeDeleteAction deletes a file
func (m *Manager) executeDeleteAction(action types.Action, filePath string) error {
	// In dry run mode, just log what would happen
	if m.dryRun {
		fmt.Printf("[DRY RUN] Would delete file %s\n", filePath)
		return nil
	}

	return os.Remove(filePath)
}

// executeCommandAction executes a command
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

// generateUniqueFilePath creates a unique file path by adding a timestamp
func (m *Manager) generateUniqueFilePath(filePath string) string {
	ext := filepath.Ext(filePath)
	basePath := filePath[:len(filePath)-len(ext)]
	timestamp := time.Now().Format("_20060102_150405")
	return basePath + timestamp + ext
}

// GetWorkflows returns the currently loaded workflows
func (m *Manager) GetWorkflows() []types.Workflow {
	return m.workflows
}

// AddWorkflow adds a new workflow to the configuration
func (m *Manager) AddWorkflow(workflow types.Workflow) error {
	// Validate the workflow
	if err := validateWorkflow(&workflow); err != nil {
		return err
	}

	// Check for ID collision
	for _, existing := range m.workflows {
		if existing.ID == workflow.ID {
			return fmt.Errorf("workflow with ID %s already exists", workflow.ID)
		}
	}

	// Add to in-memory collection
	m.workflows = append(m.workflows, workflow)

	// Save to file
	return m.saveWorkflow(workflow)
}

// UpdateWorkflow updates an existing workflow
func (m *Manager) UpdateWorkflow(workflow types.Workflow) error {
	// Validate the workflow
	if err := validateWorkflow(&workflow); err != nil {
		return err
	}

	// Find and update the workflow
	found := false
	for i, existing := range m.workflows {
		if existing.ID == workflow.ID {
			m.workflows[i] = workflow
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("workflow with ID %s not found", workflow.ID)
	}

	// Save to file
	return m.saveWorkflow(workflow)
}

// DeleteWorkflow removes a workflow by ID
func (m *Manager) DeleteWorkflow(id string) error {
	// Find the workflow
	for i, workflow := range m.workflows {
		if workflow.ID == id {
			// Remove from in-memory collection
			m.workflows = append(m.workflows[:i], m.workflows[i+1:]...)

			// Delete the file
			filePath := filepath.Join(m.configPath, id+".yaml")
			if err := os.Remove(filePath); err != nil {
				return fmt.Errorf("failed to delete workflow file: %w", err)
			}

			return nil
		}
	}

	return fmt.Errorf("workflow with ID %s not found", id)
}

// saveWorkflow saves a workflow to its configuration file
func (m *Manager) saveWorkflow(workflow types.Workflow) error {
	// Marshal to YAML
	data, err := yaml.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	// Save to file
	filePath := filepath.Join(m.configPath, workflow.ID+".yaml")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	return nil
}

// ExecuteWorkflow manually executes a workflow on a specific file
func (m *Manager) ExecuteWorkflow(workflowID, filePath string) (*types.WorkflowResult, error) {
	// Find the workflow
	var targetWorkflow *types.Workflow
	for i := range m.workflows {
		if m.workflows[i].ID == workflowID {
			targetWorkflow = &m.workflows[i]
			break
		}
	}

	if targetWorkflow == nil {
		return nil, fmt.Errorf("workflow with ID %s not found", workflowID)
	}

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// For manual execution, we skip the trigger check but still evaluate conditions
	if !m.evaluateConditions(targetWorkflow.Conditions, filePath, fileInfo) {
		return nil, fmt.Errorf("file does not meet workflow conditions")
	}

	// Execute the workflow
	result := m.executeWorkflow(*targetWorkflow, filePath)
	return &result, nil
}

// SetDryRun enables or disables dry run mode
func (m *Manager) SetDryRun(enabled bool) {
	m.dryRun = enabled
}

// IsDryRun returns the current dry run status
func (m *Manager) IsDryRun() bool {
	return m.dryRun
}
