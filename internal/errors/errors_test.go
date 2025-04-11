package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Test creating a new error
	err := New("test error")
	assert.NotNil(t, err)
	assert.Equal(t, "test error", err.Error())

	// Test creating a new formatted error
	err = Newf("formatted %s", "error")
	assert.NotNil(t, err)
	assert.Equal(t, "formatted error", err.Error())

	// Check that the error is an ApplicationError
	var appErr *ApplicationError
	assert.True(t, As(err, &appErr))
	assert.Equal(t, "formatted error", appErr.Error())
	assert.Equal(t, Unknown, appErr.Kind())
}

func TestWrapping(t *testing.T) {
	// Test wrapping an error
	origErr := New("original error")
	wrappedErr := Wrap(origErr, "wrapped")
	assert.NotNil(t, wrappedErr)
	assert.Equal(t, "wrapped: original error", wrappedErr.Error())

	// Test unwrapping
	unwrappedErr := Unwrap(wrappedErr)
	assert.Equal(t, origErr, unwrappedErr)

	// Test wrapped formatted error
	wrappedFormatted := Wrapf(origErr, "formatted %s", "wrapper")
	assert.NotNil(t, wrappedFormatted)
	assert.Equal(t, "formatted wrapper: original error", wrappedFormatted.Error())

	// Test wrapping nil returns nil
	assert.Nil(t, Wrap(nil, "wrapper"))
	assert.Nil(t, Wrapf(nil, "formatted %s", "wrapper"))

	// Test deeper wrapping
	deepWrapped := Wrap(wrappedErr, "deeper")
	assert.Equal(t, "deeper: wrapped: original error", deepWrapped.Error())

	// Test Is function
	assert.True(t, Is(wrappedErr, origErr))
	assert.True(t, Is(deepWrapped, origErr))
}

func TestFileError(t *testing.T) {
	// Test creating a file error
	fileErr := NewFileError("cannot access", "/path/to/file", FileAccessDenied, nil)
	assert.NotNil(t, fileErr)
	assert.Equal(t, "cannot access: /path/to/file", fileErr.Error())
	assert.Equal(t, "/path/to/file", fileErr.Path())
	assert.Equal(t, FileAccessDenied, fileErr.Kind())

	// Test with wrapped error
	origErr := fmt.Errorf("permission denied")
	fileErr = NewFileError("cannot access", "/path/to/file", FileAccessDenied, origErr)
	assert.Equal(t, "cannot access: /path/to/file: permission denied", fileErr.Error())
	assert.Equal(t, origErr, Unwrap(fileErr))

	// Test predefined errors
	assert.Equal(t, "file not found", ErrFileNotFound.Error())
	assert.Equal(t, FileNotFound, ErrFileNotFound.Kind())

	// Test IsFileNotFound predicate
	notFoundErr := NewFileError("file not found", "/missing/file", FileNotFound, nil)
	assert.True(t, IsFileNotFound(notFoundErr))
	assert.False(t, IsFileNotFound(fileErr)) // This is FileAccessDenied

	// Test IsFileAccessDenied predicate
	assert.True(t, IsFileAccessDenied(fileErr))
	assert.False(t, IsFileAccessDenied(notFoundErr))

	// Test As for FileError
	var fe *FileError
	assert.True(t, As(fileErr, &fe))
	assert.Equal(t, "/path/to/file", fe.Path())
}

func TestConfigError(t *testing.T) {
	// Test creating a config error
	configErr := NewConfigError("invalid value", "timeout", InvalidConfig, nil)
	assert.NotNil(t, configErr)
	assert.Equal(t, "invalid value: timeout", configErr.Error())
	assert.Equal(t, "timeout", configErr.Param())
	assert.Equal(t, InvalidConfig, configErr.Kind())

	// Test with wrapped error
	origErr := fmt.Errorf("value out of range")
	configErr = NewConfigError("invalid value", "timeout", InvalidConfig, origErr)
	assert.Equal(t, "invalid value: timeout: value out of range", configErr.Error())
	assert.Equal(t, origErr, Unwrap(configErr))

	// Test predefined errors
	assert.Equal(t, "invalid configuration", ErrInvalidConfig.Error())
	assert.Equal(t, InvalidConfig, ErrInvalidConfig.Kind())

	// Test IsInvalidConfig predicate
	assert.True(t, IsInvalidConfig(configErr))
	assert.False(t, IsInvalidConfig(New("some other error")))

	// Test As for ConfigError
	var ce *ConfigError
	assert.True(t, As(configErr, &ce))
	assert.Equal(t, "timeout", ce.Param())
}

func TestRuleError(t *testing.T) {
	// Test creating a rule error
	ruleErr := NewRuleError("invalid rule", "date-pattern", InvalidRule, nil)
	assert.NotNil(t, ruleErr)
	assert.Equal(t, "invalid rule: date-pattern", ruleErr.Error())
	assert.Equal(t, "date-pattern", ruleErr.RuleName())
	assert.Equal(t, InvalidRule, ruleErr.Kind())

	// Test with wrapped error
	origErr := fmt.Errorf("pattern syntax error")
	ruleErr = NewRuleError("invalid rule", "date-pattern", InvalidRule, origErr)
	assert.Equal(t, "invalid rule: date-pattern: pattern syntax error", ruleErr.Error())
	assert.Equal(t, origErr, Unwrap(ruleErr))

	// Test predefined errors
	assert.Equal(t, "invalid rule", ErrInvalidRule.Error())
	assert.Equal(t, InvalidRule, ErrInvalidRule.Kind())

	// Test IsInvalidRule predicate
	assert.True(t, IsInvalidRule(ruleErr))
	assert.False(t, IsInvalidRule(New("some other error")))

	// Test As for RuleError
	var re *RuleError
	assert.True(t, As(ruleErr, &re))
	assert.Equal(t, "date-pattern", re.RuleName())
}

func TestErrorChains(t *testing.T) {
	// Create a chain of errors
	baseErr := errors.New("base error")
	fileErr := NewFileError("file error", "/path/to/file", FileNotFound, baseErr)
	configErr := NewConfigError("config error", "pattern", InvalidConfig, fileErr)
	ruleErr := NewRuleError("rule error", "date-pattern", InvalidRule, configErr)

	// Test complete error message
	assert.Equal(t, "rule error: date-pattern: config error: pattern: file error: /path/to/file: base error", ruleErr.Error())

	// Test Is function through the chain
	assert.True(t, Is(ruleErr, baseErr))
	assert.True(t, Is(ruleErr, fileErr))
	assert.True(t, Is(ruleErr, configErr))

	// Test As function through the chain
	var fe *FileError
	assert.True(t, As(ruleErr, &fe))
	assert.Equal(t, "/path/to/file", fe.Path())

	var ce *ConfigError
	assert.True(t, As(ruleErr, &ce))
	assert.Equal(t, "pattern", ce.Param())

	// Test error predicates through the chain
	assert.True(t, IsFileNotFound(ruleErr))
	assert.True(t, IsInvalidConfig(ruleErr))
	assert.True(t, IsInvalidRule(ruleErr))
}
