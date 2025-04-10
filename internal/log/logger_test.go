package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"sortd/internal/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicLogging(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	l := NewLogger(WithOutput(&buf))

	// Test basic logging methods
	l.Info("info message")
	assert.Contains(t, buf.String(), "INFO")
	assert.Contains(t, buf.String(), "info message")
	buf.Reset()

	l.Warn("warn message")
	assert.Contains(t, buf.String(), "WARN")
	assert.Contains(t, buf.String(), "warn message")
	buf.Reset()

	l.Error("error message")
	assert.Contains(t, buf.String(), "ERROR")
	assert.Contains(t, buf.String(), "error message")
	buf.Reset()

	// Test formatted logging
	l.Infof("formatted %s", "message")
	assert.Contains(t, buf.String(), "formatted message")
	buf.Reset()
}

func TestDebugLogging(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	l := NewLogger(WithOutput(&buf))

	// Test with debug off
	SetDebug(false)
	l.Debug("debug message")
	assert.Empty(t, buf.String())
	buf.Reset()

	// Test with debug on
	SetDebug(true)
	l.Debug("debug message")
	assert.Contains(t, buf.String(), "DEBUG")
	assert.Contains(t, buf.String(), "debug message")
	buf.Reset()

	l.Debugf("formatted %s", "debug")
	assert.Contains(t, buf.String(), "formatted debug")
	buf.Reset()

	// Reset debug for other tests
	SetDebug(false)
}

func TestStructuredLogging(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	l := NewLogger(WithOutput(&buf))

	// Test with fields
	l.With(F("key1", "value1"), F("key2", 123)).Info("structured message")
	output := buf.String()
	assert.Contains(t, output, "structured message")
	assert.Contains(t, output, "key1=value1")
	assert.Contains(t, output, "key2=123")
	buf.Reset()

	// Test chaining fields
	l.With(F("key1", "value1")).With(F("key2", 123)).Info("chained fields")
	output = buf.String()
	assert.Contains(t, output, "chained fields")
	assert.Contains(t, output, "key1=value1")
	assert.Contains(t, output, "key2=123")
	buf.Reset()

	// Test global WithFields
	logger := NewLogger(WithOutput(&buf))
	logger.With(F("globalkey", "globalvalue")).Info("global fields")
	output = buf.String()
	assert.Contains(t, output, "global fields")
	assert.Contains(t, output, "globalkey=globalvalue")
	buf.Reset()
}

func TestJSONLogging(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	l := NewLogger(WithOutput(&buf), WithJSON())

	// Test basic JSON logging
	l.Info("json message")
	output := buf.String()

	// Verify it's valid JSON
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	// Check fields
	assert.Equal(t, "INFO", logEntry["level"])
	assert.Equal(t, "json message", logEntry["message"])
	assert.Contains(t, logEntry, "timestamp")
	assert.Contains(t, logEntry, "caller")
	buf.Reset()

	// Test JSON with fields
	l.With(F("key1", "value1"), F("key2", 123)).Info("structured json")
	output = buf.String()

	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "value1", logEntry["key1"])
	assert.Equal(t, float64(123), logEntry["key2"]) // JSON numbers are float64
	buf.Reset()
}

func TestErrorLogging(t *testing.T) {
	// Capture output
	var buf bytes.Buffer

	// Save original logger and configure a new one with our buffer
	originalLogger := logger
	Configure(WithOutput(&buf))
	defer func() { logger = originalLogger }() // Restore when test completes

	// Test with standard error
	stdErr := fmt.Errorf("standard error")
	LogWithFields(F("error", stdErr.Error())).Error("error occurred")
	output := buf.String()
	assert.Contains(t, output, "error occurred")
	assert.Contains(t, output, "standard error")
	buf.Reset()

	// Test with ApplicationError
	appErr := errors.New("application error")
	LogWithError(appErr).Error("app error occurred")
	output = buf.String()
	assert.Contains(t, output, "app error occurred")
	assert.Contains(t, output, "application error")
	assert.Contains(t, output, "error_kind=0") // Unknown error kind
	buf.Reset()

	// Test with FileError
	fileErr := errors.NewFileError("file error", "/path/to/file", errors.FileNotFound, nil)
	LogWithError(fileErr).Error("file error occurred")
	output = buf.String()
	assert.Contains(t, output, "file error occurred")
	assert.Contains(t, output, "file error: /path/to/file")
	assert.Contains(t, output, "path=/path/to/file")
	assert.Contains(t, output, "error_kind=1") // FileNotFound kind
	buf.Reset()

	// Test with ConfigError
	configErr := errors.NewConfigError("config error", "timeout", errors.InvalidConfig, nil)
	LogWithError(configErr).Error("config error occurred")
	output = buf.String()
	assert.Contains(t, output, "config error occurred")
	assert.Contains(t, output, "config error: timeout")
	assert.Contains(t, output, "param=timeout")
	assert.Contains(t, output, "error_kind=7") // Updated InvalidConfig kind
	buf.Reset()

	// Test with RuleError
	ruleErr := errors.NewRuleError("rule error", "date-rule", errors.InvalidRule, nil)
	LogWithError(ruleErr).Error("rule error occurred")
	output = buf.String()
	assert.Contains(t, output, "rule error occurred")
	assert.Contains(t, output, "rule error: date-rule")
	assert.Contains(t, output, "rule_name=date-rule")
	assert.Contains(t, output, "error_kind=10") // Updated InvalidRule kind
	buf.Reset()

	// Test the convenience function
	LogError(fileErr, "convenient error log")
	output = buf.String()
	assert.Contains(t, output, "convenient error log")
	assert.Contains(t, output, "file error: /path/to/file")
	buf.Reset()
}

func TestCallerInfo(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	l := NewLogger(WithOutput(&buf))

	// Log message and check that caller info is included
	l.Info("caller test")
	output := buf.String()
	assert.Contains(t, output, "logger_test.go:")
	buf.Reset()
}

func TestFileOutput(t *testing.T) {
	// Create a temporary log file
	tmpFile, err := os.CreateTemp("", "logtest*.log")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// We need to save the original stdout
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Save original logger and configure a new one with our file
	originalLogger := logger
	Configure(WithFile(tmpFile.Name()))

	// Set cleanup
	defer func() {
		// Restore stdout
		w.Close()
		os.Stdout = originalStdout

		// Close file before restoring original logger
		if logger.file != nil {
			logger.file.Close()
		}
		logger = originalLogger
	}() // Restore when test completes

	// Log a message
	Info("file test message")
	w.Close() // Close the writer to flush output

	// Capture stdout output
	var stdoutBuf bytes.Buffer
	io.Copy(&stdoutBuf, r)

	// Check stdout
	assert.Contains(t, stdoutBuf.String(), "file test message")

	// Check file
	fileContent, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Contains(t, string(fileContent), "file test message")
}

func TestWithContext(t *testing.T) {
	// This is currently a placeholder for future context-aware logging
	var buf bytes.Buffer
	l := NewLogger(WithOutput(&buf))

	// Should work but not add any additional context yet
	l.WithContext(nil).Info("context message")
	assert.Contains(t, buf.String(), "context message")
}

func TestNestedErrors(t *testing.T) {
	var buf bytes.Buffer
	// Setup a new global logger with our buffer
	originalLogger := logger // Save original
	Configure(WithOutput(&buf))
	defer func() { logger = originalLogger }() // Restore when test completes

	// Create nested errors
	baseErr := fmt.Errorf("base error")
	fileErr := errors.NewFileError("file error", "/path/file", errors.FileNotFound, baseErr)
	configErr := errors.NewConfigError("config error", "setting", errors.InvalidConfig, fileErr)

	// Log the nested error
	LogWithError(configErr).Error("nested error occurred")
	output := buf.String()

	// Should contain info from all error levels
	assert.Contains(t, output, "nested error occurred")
	assert.Contains(t, output, "config error: setting: file error: /path/file: base error")
	assert.Contains(t, output, "error_kind=7") // Updated InvalidConfig
	assert.Contains(t, output, "param=setting")
}

// Test global configuration
func TestConfigure(t *testing.T) {
	// Save the original logger to restore later
	originalLogger := logger

	// Capture output
	var buf bytes.Buffer

	// Configure global logger
	Configure(WithOutput(&buf), WithJSON())

	// Use global functions
	Info("global config test")

	// Verify it used JSON format
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "global config test", logEntry["message"])

	// Restore original logger
	logger = originalLogger
}

// Test that we correctly handle nil errors
func TestNilErrorHandling(t *testing.T) {
	var buf bytes.Buffer
	// Setup a new global logger with our buffer
	originalLogger := logger // Save original
	Configure(WithOutput(&buf))
	defer func() { logger = originalLogger }() // Restore when test completes

	// Should not panic
	LogWithError(nil).Error("nil error test")
	output := buf.String()
	assert.Contains(t, output, "nil error test")
	assert.Contains(t, output, "error=<nil>")
}
