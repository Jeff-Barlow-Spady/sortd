package workflow

import (
	"os"
	"testing"

	"sortd/pkg/types"
)

func TestValidateWorkflow(t *testing.T) {
	tests := []struct {
		name      string
		workflow  types.Workflow
		wantError bool
	}{
		{
			name: "Valid workflow",
			workflow: types.Workflow{
				ID:      "test-workflow",
				Name:    "Test Workflow",
				Enabled: true,
				Actions: []types.Action{
					{Type: types.MoveAction, Target: "/tmp"},
				},
			},
			wantError: false,
		},
		{
			name: "Missing ID",
			workflow: types.Workflow{
				Name:    "Test Workflow",
				Enabled: true,
				Actions: []types.Action{
					{Type: types.MoveAction, Target: "/tmp"},
				},
			},
			wantError: true,
		},
		{
			name: "Missing name",
			workflow: types.Workflow{
				ID:      "test-workflow",
				Enabled: true,
				Actions: []types.Action{
					{Type: types.MoveAction, Target: "/tmp"},
				},
			},
			wantError: true,
		},
		{
			name: "No actions",
			workflow: types.Workflow{
				ID:      "test-workflow",
				Name:    "Test Workflow",
				Enabled: true,
				Actions: []types.Action{},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkflow(&tt.workflow)
			if (err != nil) != tt.wantError {
				t.Errorf("validateWorkflow() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestEvaluateFileSizeCondition(t *testing.T) {
	// Create a temporary file for testing
	tmpfile, err := os.CreateTemp("", "test-file-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write 1KB of data
	data := make([]byte, 1024)
	if _, err := tmpfile.Write(data); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Get file info
	fileInfo, err := os.Stat(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	manager := &Manager{}

	tests := []struct {
		name      string
		condition types.Condition
		want      bool
	}{
		{
			name: "Equal to 1KB",
			condition: types.Condition{
				Type:      types.FileSizeCondition,
				Field:     "size",
				Operator:  types.Equals,
				Value:     "1",
				ValueUnit: "KB",
			},
			want: true,
		},
		{
			name: "Less than 2KB",
			condition: types.Condition{
				Type:      types.FileSizeCondition,
				Field:     "size",
				Operator:  types.LessThan,
				Value:     "2",
				ValueUnit: "KB",
			},
			want: true,
		},
		{
			name: "Greater than 2KB",
			condition: types.Condition{
				Type:      types.FileSizeCondition,
				Field:     "size",
				Operator:  types.GreaterThan,
				Value:     "2",
				ValueUnit: "KB",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.evaluateFileSizeCondition(tt.condition, fileInfo)
			if got != tt.want {
				t.Errorf("evaluateFileSizeCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateFileNameCondition(t *testing.T) {
	manager := &Manager{}
	testFilePath := "/path/to/test-file.txt"

	tests := []struct {
		name      string
		condition types.Condition
		want      bool
	}{
		{
			name: "Equals",
			condition: types.Condition{
				Type:     types.FileNameCondition,
				Field:    "name",
				Operator: types.Equals,
				Value:    "test-file.txt",
			},
			want: true,
		},
		{
			name: "Contains",
			condition: types.Condition{
				Type:     types.FileNameCondition,
				Field:    "name",
				Operator: types.Contains,
				Value:    "file",
			},
			want: true,
		},
		{
			name: "StartsWith",
			condition: types.Condition{
				Type:     types.FileNameCondition,
				Field:    "name",
				Operator: types.StartsWith,
				Value:    "test",
			},
			want: true,
		},
		{
			name: "EndsWith",
			condition: types.Condition{
				Type:     types.FileNameCondition,
				Field:    "name",
				Operator: types.EndsWith,
				Value:    ".txt",
			},
			want: true,
		},
		{
			name: "NotEquals",
			condition: types.Condition{
				Type:     types.FileNameCondition,
				Field:    "name",
				Operator: types.NotEquals,
				Value:    "other-file.txt",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.evaluateFileNameCondition(tt.condition, testFilePath)
			if got != tt.want {
				t.Errorf("evaluateFileNameCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDryRunExecution tests workflow execution in dry run mode
func TestDryRunExecution(t *testing.T) {
	// This will be implemented once we add dry run capability
}
