package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"strings"
)

type VimCommandResult struct {
	Success bool
	Message string
}

type VimCommand struct {
	Command string
	Time    time.Time
}

var commandHistory []VimCommand

func GetVimCommandHistory() []VimCommand {
	return commandHistory
}

func ExecuteVimCommand(cmd string) VimCommandResult {
	// Add command to history
	commandHistory = append(commandHistory, VimCommand{
		Command: cmd,
		Time:    time.Now(),
	})

	// Basic command validation and execution
	switch {
	case cmd == ":wq":
		return VimCommandResult{Success: true, Message: "File saved and closed"}
	case cmd == ":w":
		return VimCommandResult{Success: true, Message: "File saved"}
	case strings.HasPrefix(cmd, ":e "):
		filename := strings.TrimPrefix(cmd, ":e ")
		if filename == "" {
			return VimCommandResult{Success: false, Message: "No filename specified"}
		}
		return VimCommandResult{Success: true, Message: "Opened " + filename}
	default:
		return VimCommandResult{Success: false, Message: "Invalid command: " + cmd}
	}
}

func TestVimCommands(t *testing.T) {
	t.Run("basic command execution", func(t *testing.T) {
		result := ExecuteVimCommand(":wq")
		assert.True(t, result.Success, "Should execute write and quit")
	})

	t.Run("invalid command handling", func(t *testing.T) {
		result := ExecuteVimCommand(":invalid")
		assert.False(t, result.Success, "Should return error for invalid command")
	})

	t.Run("command history tracking", func(t *testing.T) {
		// Clear history before test
		commandHistory = nil

		ExecuteVimCommand(":e file.txt")
		ExecuteVimCommand(":w")
		history := GetVimCommandHistory()
		assert.Len(t, history, 2, "Should track command history")
		assert.Equal(t, ":e file.txt", history[0].Command)
		assert.Equal(t, ":w", history[1].Command)
	})
}
