# Contributing to Sortd

Thank you for considering contributing to Sortd! This document provides some guidelines to help you get started.

## Project Structure

The Sortd project is structured as follows:

- `cmd/`: Command-line entry points
  - `sortd/`: Main application entry point
- `internal/`: Internal packages not meant for external use
  - `config/`: Configuration management
  - `gui/`: GUI implementation using Fyne
  - `organize/`: File organization logic
  - `watch/`: File watching and daemon functionality
  - `.tui.bak/`: Deprecated TUI implementation (not currently used)
- `pkg/`: Public packages that can be imported by other projects
  - `types/`: Common type definitions
  - `workflow/`: Workflow management
- `tests/`: Integration and end-to-end tests
  - `.tui.bak/`: Deprecated TUI tests

## TUI Deprecation Note

The terminal user interface (TUI) has been deprecated in favor of the GUI implementation. The TUI code is still available in the `.tui.bak` directories but is not actively maintained or built. If you need to work with the TUI:

1. Rename the `.tui.bak` directories to `tui`
2. Update the `.gitignore` and `.nobuild` files accordingly
3. Uncomment the `tuiCmd()` function in `cmd/sortd/main.go`

## Development Workflow

### Building the Project

To build the project, use:

```bash
./build_no_tui.sh
```

### Running Tests

To run tests (excluding the deprecated TUI tests):

```bash
./run_tests.sh
```

### Code Style

We follow standard Go code style. Before submitting changes, please run:

```bash
go fmt ./...
go vet ./...
```

## Pull Request Process

1. Update the README.md or documentation with details of your changes if appropriate
2. Make sure all tests pass
3. Update the version numbers in any examples files and the README.md to the new version
4. Submit a pull request with a clear description of the changes

## License

By contributing to this project, you agree that your contributions will be licensed under the project's license.