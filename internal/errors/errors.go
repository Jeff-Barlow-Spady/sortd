// Package errors provides standardized error handling for the Sortd application.
// It defines common error types, constants, and helper functions for consistent
// error creation, wrapping, and handling across the application.
package errors

import (
	"errors"
	"fmt"
)

// Standard errors package errors that we re-export for convenience
var (
	// Unwrap unwraps an error to access the underlying error
	Unwrap = errors.Unwrap
	// Is reports whether any error in err's chain matches target
	Is = errors.Is
	// As finds the first error in err's chain that matches target
	As = errors.As
)

// Common error constants for frequently occurring errors
var (
	ErrFileNotFound  = NewFileError("file not found", "", FileNotFound, nil)
	ErrFileAccess    = NewFileError("file access denied", "", FileAccessDenied, nil)
	ErrInvalidPath   = NewFileError("invalid file path", "", InvalidPath, nil)
	ErrInvalidConfig = NewConfigError("invalid configuration", "", InvalidConfig, nil)
	ErrInvalidRule   = NewRuleError("invalid rule", "", InvalidRule, nil)
)

// ErrorKind represents the kind of error
type ErrorKind int

// Error kinds
const (
	Unknown ErrorKind = iota
	// File error kinds
	FileNotFound
	FileAccessDenied
	InvalidPath
	FileCreateFailed
	FileOperationFailed
	InvalidOperation
	// Config error kinds
	InvalidConfig
	ConfigNotFound
	ConfigNotSet
	// Rule error kinds
	InvalidRule
	RuleNotFound
	// Database error kinds
	DatabaseConnectionFailed
	DatabaseQueryFailed
	DatabaseOperationFailed
	InvalidInputData
)

// Additional error checks for database errors
var (
	ErrDatabaseOperation = NewDatabaseError("database operation failed", nil)
	ErrInvalidInput      = NewInvalidInputError("invalid input data", nil)
)

// ApplicationError is the base error type for all application errors
type ApplicationError struct {
	msg  string
	err  error
	kind ErrorKind
}

// Error returns the error message
func (e *ApplicationError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.msg, e.err)
	}
	return e.msg
}

// Unwrap returns the wrapped error
func (e *ApplicationError) Unwrap() error {
	return e.err
}

// Kind returns the kind of error
func (e *ApplicationError) Kind() ErrorKind {
	return e.kind
}

// FileError represents errors related to file operations
type FileError struct {
	ApplicationError
	path string
}

// NewFileError creates a new file error
func NewFileError(msg string, path string, kind ErrorKind, err error) *FileError {
	return &FileError{
		ApplicationError: ApplicationError{
			msg:  msg,
			err:  err,
			kind: kind,
		},
		path: path,
	}
}

// Error returns the file error message
func (e *FileError) Error() string {
	if e.path != "" {
		if e.err != nil {
			return fmt.Sprintf("%s: %s: %v", e.msg, e.path, e.err)
		}
		return fmt.Sprintf("%s: %s", e.msg, e.path)
	}
	return e.ApplicationError.Error()
}

// Path returns the file path associated with the error
func (e *FileError) Path() string {
	return e.path
}

// ConfigError represents errors related to configuration
type ConfigError struct {
	ApplicationError
	param string
}

// NewConfigError creates a new configuration error
func NewConfigError(msg string, param string, kind ErrorKind, err error) *ConfigError {
	return &ConfigError{
		ApplicationError: ApplicationError{
			msg:  msg,
			err:  err,
			kind: kind,
		},
		param: param,
	}
}

// Error returns the config error message
func (e *ConfigError) Error() string {
	if e.param != "" {
		if e.err != nil {
			return fmt.Sprintf("%s: %s: %v", e.msg, e.param, e.err)
		}
		return fmt.Sprintf("%s: %s", e.msg, e.param)
	}
	return e.ApplicationError.Error()
}

// Param returns the configuration parameter associated with the error
func (e *ConfigError) Param() string {
	return e.param
}

// RuleError represents errors related to sorting rules
type RuleError struct {
	ApplicationError
	ruleName string
}

// NewRuleError creates a new rule error
func NewRuleError(msg string, ruleName string, kind ErrorKind, err error) *RuleError {
	return &RuleError{
		ApplicationError: ApplicationError{
			msg:  msg,
			err:  err,
			kind: kind,
		},
		ruleName: ruleName,
	}
}

// Error returns the rule error message
func (e *RuleError) Error() string {
	if e.ruleName != "" {
		if e.err != nil {
			return fmt.Sprintf("%s: %s: %v", e.msg, e.ruleName, e.err)
		}
		return fmt.Sprintf("%s: %s", e.msg, e.ruleName)
	}
	return e.ApplicationError.Error()
}

// RuleName returns the rule name associated with the error
func (e *RuleError) RuleName() string {
	return e.ruleName
}

// New creates a new error with a message
func New(msg string) error {
	return &ApplicationError{
		msg:  msg,
		kind: Unknown,
	}
}

// Newf creates a new error with a formatted message
func Newf(format string, args ...interface{}) error {
	return &ApplicationError{
		msg:  fmt.Sprintf(format, args...),
		kind: Unknown,
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return &ApplicationError{
		msg:  msg,
		err:  err,
		kind: Unknown,
	}
}

// Wrapf wraps an existing error with additional formatted context
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return &ApplicationError{
		msg:  fmt.Sprintf(format, args...),
		err:  err,
		kind: Unknown,
	}
}

// IsFileNotFound checks if the error is a file not found error
func IsFileNotFound(err error) bool {
	var fileErr *FileError
	if errors.As(err, &fileErr) {
		return fileErr.Kind() == FileNotFound
	}
	return false
}

// IsFileAccessDenied checks if the error is a file access denied error
func IsFileAccessDenied(err error) bool {
	var fileErr *FileError
	if errors.As(err, &fileErr) {
		return fileErr.Kind() == FileAccessDenied
	}
	return false
}

// IsInvalidConfig checks if the error is an invalid configuration error
func IsInvalidConfig(err error) bool {
	var configErr *ConfigError
	if errors.As(err, &configErr) {
		return configErr.Kind() == InvalidConfig
	}
	return false
}

// IsInvalidRule checks if the error is an invalid rule error
func IsInvalidRule(err error) bool {
	var ruleErr *RuleError
	if errors.As(err, &ruleErr) {
		return ruleErr.Kind() == InvalidRule
	}
	return false
}

// DatabaseError represents errors related to database operations
type DatabaseError struct {
	ApplicationError
	operation string
	context   map[string]interface{}
}

// NewDatabaseError creates a new database error
func NewDatabaseError(msg string, err error) *DatabaseError {
	return &DatabaseError{
		ApplicationError: ApplicationError{
			msg:  msg,
			err:  err,
			kind: DatabaseOperationFailed,
		},
		operation: "",
		context:   make(map[string]interface{}),
	}
}

// WithOperation adds operation information to the database error
func (e *DatabaseError) WithOperation(operation string) *DatabaseError {
	e.operation = operation
	return e
}

// WithContext adds context information to the database error
func (e *DatabaseError) WithContext(key string, value interface{}) *DatabaseError {
	e.context[key] = value
	return e
}

// Error returns the database error message
func (e *DatabaseError) Error() string {
	if e.operation != "" {
		if e.err != nil {
			return fmt.Sprintf("%s: operation=%s: %v", e.msg, e.operation, e.err)
		}
		return fmt.Sprintf("%s: operation=%s", e.msg, e.operation)
	}
	return e.ApplicationError.Error()
}

// Operation returns the database operation associated with the error
func (e *DatabaseError) Operation() string {
	return e.operation
}

// Context returns the context information associated with the error
func (e *DatabaseError) Context() map[string]interface{} {
	return e.context
}

// InvalidInputError represents errors related to invalid input data
type InvalidInputError struct {
	ApplicationError
	context map[string]interface{}
}

// NewInvalidInputError creates a new invalid input error
func NewInvalidInputError(msg string, err error) *InvalidInputError {
	return &InvalidInputError{
		ApplicationError: ApplicationError{
			msg:  msg,
			err:  err,
			kind: InvalidInputData,
		},
		context: make(map[string]interface{}),
	}
}

// WithContext adds context information to the invalid input error
func (e *InvalidInputError) WithContext(key string, value interface{}) *InvalidInputError {
	e.context[key] = value
	return e
}

// Error returns the invalid input error message
func (e *InvalidInputError) Error() string {
	return e.ApplicationError.Error()
}

// Context returns the context information associated with the error
func (e *InvalidInputError) Context() map[string]interface{} {
	return e.context
}

// IsDatabaseError checks if the error is a database error
func IsDatabaseError(err error) bool {
	var dbErr *DatabaseError
	return errors.As(err, &dbErr)
}

// IsInvalidInputError checks if the error is an invalid input error
func IsInvalidInputError(err error) bool {
	var inputErr *InvalidInputError
	return errors.As(err, &inputErr)
}
