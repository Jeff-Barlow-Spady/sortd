package log

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"sortd/internal/errors"
)

// Log levels
const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
	LevelFatal = "FATAL"
)

var (
	isDebug  = false
	logger   = NewLogger()
	logMutex sync.Mutex
)

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// F creates a new log field
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Logger is the main logger structure
type Logger struct {
	out     io.Writer
	file    *os.File
	level   string
	fields  []Field
	useJSON bool
}

// LoggerOption defines a functional option for configuring the logger
type LoggerOption func(*Logger)

// WithOutput sets the output writer
func WithOutput(out io.Writer) LoggerOption {
	return func(l *Logger) {
		l.out = out
	}
}

// WithFile sets a file output for logging
func WithFile(path string) LoggerOption {
	return func(l *Logger) {
		// Create directory if it doesn't exist
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
			return
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			return
		}

		if l.file != nil {
			l.file.Close()
		}

		l.file = file
		// Use MultiWriter to write to both stdout and the file
		l.out = io.MultiWriter(os.Stdout, file)
	}
}

// WithLevel sets the minimum log level
func WithLevel(level string) LoggerOption {
	return func(l *Logger) {
		l.level = level
	}
}

// WithFields adds default fields to the logger
func WithFields(fields ...Field) LoggerOption {
	return func(l *Logger) {
		l.fields = append(l.fields, fields...)
	}
}

// WithJSON enables JSON-formatted logging
func WithJSON() LoggerOption {
	return func(l *Logger) {
		l.useJSON = true
	}
}

// NewLogger creates a new logger instance
func NewLogger(opts ...LoggerOption) *Logger {
	l := &Logger{
		out:   os.Stdout,
		level: LevelInfo,
	}

	// Apply options
	for _, opt := range opts {
		opt(l)
	}

	return l
}

// With creates a new logger with additional fields
func (l *Logger) With(fields ...Field) *Logger {
	newLogger := &Logger{
		out:     l.out,
		file:    l.file,
		level:   l.level,
		useJSON: l.useJSON,
	}

	// Copy existing fields
	newLogger.fields = make([]Field, len(l.fields))
	copy(newLogger.fields, l.fields)

	// Add new fields
	newLogger.fields = append(newLogger.fields, fields...)

	return newLogger
}

// WithContext creates a new logger with context information
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Here you can extract values from context and add them as fields
	// This is a placeholder for future context-aware logging
	return l.With()
}

// Close closes the logger and any open files
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// SetDebug enables or disables debug logging
func SetDebug(debug bool) {
	isDebug = debug
}

// Configure configures the global logger
func Configure(opts ...LoggerOption) {
	logMutex.Lock()
	defer logMutex.Unlock()

	newLogger := NewLogger(opts...)
	if logger.file != nil {
		logger.file.Close()
	}
	logger = newLogger
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	logger.log(LevelInfo, format, args...)
}

// Debug logs a debug message
func Debug(msg string, args ...interface{}) {
	if isDebug {
		logger.log(LevelDebug, msg+": %v", args...)
	}
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...interface{}) {
	if isDebug {
		logger.log(LevelDebug, format, args...)
	}
}

// Error logs an error message
func Error(msg string, args ...interface{}) {
	logger.log(LevelError, msg+": %v", args...)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	logger.log(LevelError, format, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...interface{}) {
	logger.log(LevelWarn, msg+": %v", args...)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	logger.log(LevelWarn, format, args...)
}

// WithFields creates a new entry with specified fields
func LogWithFields(fields ...Field) *Logger {
	return logger.With(fields...)
}

// WithError creates a new entry with error fields
func LogWithError(err error) *Logger {
	fields := extractErrorFields(err)
	return logger.With(fields...)
}

// extractErrorFields extracts fields from our custom error types
func extractErrorFields(err error) []Field {
	if err == nil {
		return []Field{
			{Key: "error", Value: "<nil>"},
		}
	}

	fields := []Field{
		{Key: "error", Value: err.Error()},
	}

	// Extract kind and other details from our custom error types
	var appErr *errors.ApplicationError
	if errors.As(err, &appErr) {
		fields = append(fields, Field{Key: "error_kind", Value: appErr.Kind()})
	}

	var fileErr *errors.FileError
	if errors.As(err, &fileErr) {
		fields = append(fields,
			Field{Key: "error_kind", Value: fileErr.Kind()},
			Field{Key: "path", Value: fileErr.Path()},
		)
	}

	var configErr *errors.ConfigError
	if errors.As(err, &configErr) {
		fields = append(fields,
			Field{Key: "error_kind", Value: configErr.Kind()},
			Field{Key: "param", Value: configErr.Param()},
		)
	}

	var ruleErr *errors.RuleError
	if errors.As(err, &ruleErr) {
		fields = append(fields,
			Field{Key: "error_kind", Value: ruleErr.Kind()},
			Field{Key: "rule_name", Value: ruleErr.RuleName()},
		)
	}

	return fields
}

// log implements the core logging functionality
func (l *Logger) log(level, format string, args ...interface{}) {
	logMutex.Lock()
	defer logMutex.Unlock()

	// Format the message
	msg := fmt.Sprintf(format, args...)

	if l.useJSON {
		l.logJSON(level, msg)
	} else {
		l.logText(level, msg)
	}
}

// logText logs a message in text format
func (l *Logger) logText(level, msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Get caller information for context
	_, file, line, ok := runtime.Caller(3) // Skip through the call stack to find the original caller
	caller := "unknown"
	if ok {
		// Get just the base filename
		caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	// Build the base message
	logMsg := fmt.Sprintf("[%s] %s [%s]: %s", timestamp, level, caller, msg)

	// Add fields if any
	if len(l.fields) > 0 {
		fields := make([]string, 0, len(l.fields))
		for _, field := range l.fields {
			fields = append(fields, fmt.Sprintf("%s=%v", field.Key, field.Value))
		}
		logMsg = fmt.Sprintf("%s | %s", logMsg, strings.Join(fields, ", "))
	}

	fmt.Fprintln(l.out, logMsg)
}

// logJSON logs a message in JSON format
func (l *Logger) logJSON(level, msg string) {
	// Create a map for JSON logging
	logMap := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level,
		"message":   msg,
	}

	// Get caller information
	_, file, line, ok := runtime.Caller(3)
	if ok {
		logMap["caller"] = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	// Add fields
	for _, field := range l.fields {
		logMap[field.Key] = field.Value
	}

	// Convert to JSON
	jsonBytes, err := json.Marshal(logMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
		return
	}

	fmt.Fprintln(l.out, string(jsonBytes))
}

// Info logs an info message with fields
func (l *Logger) Info(msg string) {
	l.log(LevelInfo, msg)
}

// Infof logs a formatted info message with fields
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Debug logs a debug message with fields
func (l *Logger) Debug(msg string) {
	if isDebug {
		l.log(LevelDebug, msg)
	}
}

// Debugf logs a formatted debug message with fields
func (l *Logger) Debugf(format string, args ...interface{}) {
	if isDebug {
		l.log(LevelDebug, format, args...)
	}
}

// Warn logs a warning message with fields
func (l *Logger) Warn(msg string) {
	l.log(LevelWarn, msg)
}

// Warnf logs a formatted warning message with fields
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message with fields
func (l *Logger) Error(msg string) {
	l.log(LevelError, msg)
}

// Errorf logs a formatted error message with fields
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// ErrorWithStack logs an error with its stack trace
func (l *Logger) ErrorWithStack(err error, msg string) {
	l.With(F("error", err.Error())).log(LevelError, msg)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string) {
	l.log(LevelFatal, msg)
	os.Exit(1)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(LevelFatal, format, args...)
	os.Exit(1)
}

// LogError logs an error with appropriate context
func LogError(err error, msg string) {
	LogWithError(err).Error(msg)
}

// LogErrorf logs a formatted error message with appropriate context
func LogErrorf(err error, format string, args ...interface{}) {
	LogWithError(err).Errorf(format, args...)
}
