# Logging Standards for Sortd

This document outlines the standardized logging approach for the Sortd project to ensure consistent, structured logging across all components.

## Core Principles

1. **Use the internal/log package** - Always use the structured logging implementation from `internal/log` instead of ad-hoc logging solutions.
2. **Structured over unstructured** - Prefer structured logging with fields over string interpolation.
3. **Consistent field names** - Use the same field names for the same types of data across the codebase.
4. **Appropriate log levels** - Use the correct log level for each message (debug, info, warn, error).
5. **Context-rich logging** - Include relevant context information in logs to aid debugging.

## Log Levels

- **Debug**: Low-level, detailed information only useful for debugging
- **Info**: General information highlighting progress of the application
- **Warning**: Non-critical issues that might require attention
- **Error**: Error events that might still allow the application to continue running
- **Fatal**: Very severe error events that will lead to application termination

## Standard Field Names

Use these standard field names for consistency:

| Field Name    | Description                        | Example                      |
|--------------|------------------------------------|------------------------------|
| `file`        | Path to the file being processed    | `/home/user/documents/file.txt` |
| `directory`   | Path to a directory                | `/home/user/downloads/`     |
| `error`       | Error object or message            | `"file not found"`          |
| `source`      | Source file path for operations    | `/source/path/file.txt`     |
| `destination` | Destination file path for operations | `/dest/path/file.txt`    |
| `command`     | Command being executed             | `"ls -la"`                  |
| `output`      | Output from a command              | `"total 12\ndrwxr-xr-x..."` |
| `pattern`     | File matching pattern              | `"*.jpg"`                   |
| `target`      | Target location for operations     | `"Images/"`                 |
| `workflow_id` | ID of a workflow                   | `"photo-organizer"`         |
| `workflow_name` | Name of a workflow               | `"Photo Organizer"`         |
| `event`       | Event information                  | `"CREATE /path/file.txt"`   |

## Code Examples

### Basic Logging

```go
// Simple logging with a message
log.Info("Service started")
log.Error("Service failed to start")
```

### Structured Logging with Fields

```go
// Log with fields for better context
log.LogWithFields(
    log.F("file", filePath),
    log.F("destination", destPath),
).Info("Moving file")

// Log with error information
if err != nil {
    log.LogWithFields(
        log.F("file", filePath),
        log.F("error", err),
    ).Error("Failed to process file")
}
```

### Error Logging with Error Object

```go
// When handling errors from internal/errors package:
if err != nil {
    log.LogError(err, "Failed to organize file")
    return err
}
```

### Dry Run Mode Logging

```go
// For operations in dry run mode
if dryRun {
    log.LogWithFields(
        log.F("file", filePath),
        log.F("destination", destPath),
    ).Info("[DRY RUN] Would move file")
    return nil
}
```

## Best Practices

1. **Keep messages concise and clear** - Log messages should be readable and convey what happened.

2. **Include relevant context** - Add fields that help understand the context (file paths, error details, etc.).

3. **Be consistent with message formats** - Use consistent tenses and phrasing for similar events.

4. **Consider log consumption** - Remember that logs will be read by both humans and potentially log analysis tools.

5. **Don't log sensitive information** - Avoid logging passwords, keys, or personal data.

6. **Log at appropriate levels** - Don't log routine operations at ERROR level and don't log errors at INFO level.

7. **Group related fields** - Keep related information together for clarity.

8. **Log entry and exit points** - For complex operations, log when starting and completing (successful or not).

## Implementation

### Module-specific Logging Guidelines

- **organize**: Focus on file operations, movements, and pattern matching
- **watch**: Focus on file system events and daemon status
- **workflow**: Focus on triggered workflows, conditions, and actions
- **gui**: Focus on user interactions and application state

## Migration Guide

When migrating code to use the standardized logging:

1. Replace direct `fmt` printing with appropriate log calls.
2. Replace unstructured `log.Debugf/Infof/...` calls with structured `log.LogWithFields().Debug/Info/...` calls.
3. Include relevant context fields using `log.F()`.
4. Use `log.LogError()` for error contexts.
5. Ensure the log level matches the intent of the message.