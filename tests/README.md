# Sortd Test Suite

This directory contains tests for the Sortd file organization tool.

## Test Structure

- `tests/fixtures/` - Contains common test files used across tests
- `tests/cli_commands_test.go` - Tests for CLI commands
- `tests/gum_interaction_test.go` - Tests for Gum interactive interfaces
- `tests/integration_test.go` - Integration tests for core functionality
- `tests/config_test.go` - Tests for configuration handling
- `tests/analysis_engine_test.go` - Tests for file analysis
- `tests/organization_engine_test.go` - Tests for file organization
- `tests/watch_commands_test.go` - Tests for watch functionality

## Running Tests

To run all tests:
```bash
go test ./tests/...
```

To run specific tests:
```bash
go test ./tests/ -run TestCliCommands
```

## Test Data

The test fixtures in `tests/fixtures/` include sample files of different types for testing. These include:
- Text files (`.txt`)
- Image files (`.jpg`)
- Symbolic links

## Environment Variables

Some tests check for environment variables:

- `CI=true` - Will skip interactive tests when running in CI environments
- `INTERACTIVE_TESTS=true` - Enable tests that require interactive input
- `VERBOSE_TESTS=true` - Enable more verbose output in some tests

## Creating New Tests

When creating new tests:

1. Use the helper functions defined in the test files
2. Consider test isolation - each test should be independent
3. For file-related tests, use `t.TempDir()` to get a clean directory
4. Use test fixtures from `tests/fixtures/` for consistent file testing

## Skipping Tests

Some tests may need to be skipped in certain environments:

```go
if os.Getenv("CI") == "true" {
    t.Skip("Skipping interactive test in CI environment")
}
```