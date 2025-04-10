package types

// TriggerType defines what causes a workflow to be executed
type TriggerType string

const (
	// FileCreated trigger runs when a new file is created
	FileCreated TriggerType = "file_created"
	// FileModified trigger runs when an existing file is modified
	FileModified TriggerType = "file_modified"
	// FilePatternMatch trigger runs when a file matches a specific pattern
	FilePatternMatch TriggerType = "file_pattern_match"
	// ManualTrigger runs when explicitly requested by the user
	ManualTrigger TriggerType = "manual"
	// ScheduledTrigger runs on a defined schedule
	ScheduledTrigger TriggerType = "scheduled"
)

// ActionType defines what kind of operation the workflow performs
type ActionType string

const (
	// MoveAction moves a file to a target location
	MoveAction ActionType = "move"
	// CopyAction copies a file to a target location
	CopyAction ActionType = "copy"
	// RenameAction renames a file
	RenameAction ActionType = "rename"
	// TagAction adds tags to a file
	TagAction ActionType = "tag"
	// DeleteAction deletes a file
	DeleteAction ActionType = "delete"
	// ExecuteAction runs a specified command
	ExecuteAction ActionType = "execute"
)

// ConditionType defines what type of condition to evaluate
type ConditionType string

const (
	// FileSizeCondition evaluates based on file size
	FileSizeCondition ConditionType = "file_size"
	// FileTypeCondition evaluates based on file content type
	FileTypeCondition ConditionType = "file_type"
	// FileNameCondition evaluates based on file name properties
	FileNameCondition ConditionType = "file_name"
	// FileAgeCondition evaluates based on file creation/modification time
	FileAgeCondition ConditionType = "file_age"
	// CustomCondition evaluates a custom expression
	CustomCondition ConditionType = "custom"
)

// OperatorType defines comparison operators for conditions
type OperatorType string

const (
	// Equals check for equality
	Equals OperatorType = "equals"
	// NotEquals check for inequality
	NotEquals OperatorType = "not_equals"
	// Contains check if a string contains another
	Contains OperatorType = "contains"
	// StartsWith check if a string starts with another
	StartsWith OperatorType = "starts_with"
	// EndsWith check if a string ends with another
	EndsWith OperatorType = "ends_with"
	// GreaterThan check if a value is greater than another
	GreaterThan OperatorType = "greater_than"
	// LessThan check if a value is less than another
	LessThan OperatorType = "less_than"
	// MatchesRegex check if a string matches a regex pattern
	MatchesRegex OperatorType = "matches_regex"
)

// Condition defines a single condition to be evaluated
type Condition struct {
	Type      ConditionType `yaml:"type" json:"type"`                                 // Type of condition
	Field     string        `yaml:"field" json:"field"`                               // Field to evaluate (e.g., "name", "size", "content_type")
	Operator  OperatorType  `yaml:"operator" json:"operator"`                         // Comparison operator
	Value     string        `yaml:"value" json:"value"`                               // Value to compare against
	ValueUnit string        `yaml:"value_unit,omitempty" json:"value_unit,omitempty"` // Optional unit for values (e.g., "KB", "MB", "days")
}

// Action defines a single action to be executed
type Action struct {
	Type    ActionType        `yaml:"type" json:"type"`                           // Type of action
	Target  string            `yaml:"target" json:"target"`                       // Target path/location/value for the action
	Options map[string]string `yaml:"options,omitempty" json:"options,omitempty"` // Additional options for the action
}

// Trigger defines what causes a workflow to run
type Trigger struct {
	Type     TriggerType `yaml:"type" json:"type"`                             // Type of trigger
	Pattern  string      `yaml:"pattern,omitempty" json:"pattern,omitempty"`   // File pattern for pattern-based triggers
	Schedule string      `yaml:"schedule,omitempty" json:"schedule,omitempty"` // Cron-like schedule for scheduled triggers
}

// Workflow defines a complete workflow with trigger, conditions, and actions
type Workflow struct {
	ID          string      `yaml:"id" json:"id"`                                       // Unique identifier for the workflow
	Name        string      `yaml:"name" json:"name"`                                   // Human-readable name
	Description string      `yaml:"description,omitempty" json:"description,omitempty"` // Optional description
	Enabled     bool        `yaml:"enabled" json:"enabled"`                             // Whether the workflow is active
	Trigger     Trigger     `yaml:"trigger" json:"trigger"`                             // What activates this workflow
	Conditions  []Condition `yaml:"conditions,omitempty" json:"conditions,omitempty"`   // Optional conditions that must be met
	Actions     []Action    `yaml:"actions" json:"actions"`                             // Actions to perform
	Priority    int         `yaml:"priority,omitempty" json:"priority,omitempty"`       // Optional execution priority (higher runs first)
}

// WorkflowResult represents the result of executing a workflow
type WorkflowResult struct {
	WorkflowID   string `json:"workflow_id"`
	WorkflowName string `json:"workflow_name"`
	Success      bool   `json:"success"`
	FilePath     string `json:"file_path,omitempty"`
	Message      string `json:"message,omitempty"`
	Error        error  `json:"error,omitempty"`
}
