# Test Suite Refactoring Plan

## Current State
The current test suite contains tests for the TUI (Text User Interface) implementation which has been replaced with a CLI (Command Line Interface). The tests need to be refactored to:

1. Remove TUI-specific tests
2. Update integration tests to use the new CLI commands
3. Maintain core functionality tests for config, analysis, and organization

## Files to Remove or Refactor

### Files to Remove
- `tests/tui_test.go` - Contains TUI-specific tests that are no longer relevant
- `tests/main_view_test.go` - Tests the TUI views which no longer exist

### Files to Refactor
- `tests/config_test.go` - Keep and update to work with the new CLI implementation
- `tests/analysis_engine_test.go` - Keep core functionality tests, update any UI references
- `tests/organization_engine_test.go` - Keep core functionality tests, update any UI references
- `tests/integration_test.go` - Update to test the new CLI commands
- `tests/watch_commands_test.go` - Update to test the new watch functionality
- `tests/vim_commands_test.go` - Likely remove as vim-style commands are not part of the CLI

## New Tests to Add
- `tests/cli_commands_test.go` - Test the new CLI commands (setup, organize, rules, watch, daemon)
- `tests/gum_interaction_test.go` - Test the interactive components using Gum

## Refactoring Approach
1. First, remove the TUI-specific test files
2. Create new test fixtures for CLI commands
3. Update integration tests to use the CLI commands
4. Ensure core functionality tests continue to work
5. Add new tests for CLI-specific functionality

## Test Coverage Goals
- Maintain or improve test coverage for core functionality
- Ensure all CLI commands are properly tested
- Focus on integration tests that validate the end-to-end functionality