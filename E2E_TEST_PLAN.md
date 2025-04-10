# End-to-End Test Plan for Sortd (Mock File System)

## Overview

This document outlines the plan for end-to-end testing of the sortd application using a mock file system. These tests serve as a full demo separate from unit tests and simulate real-world usage with automated workflows.

## Objectives

1. Validate the full sorting workflow on a controlled mock file system.
2. Ensure the configuration is correctly loaded from the sortd-config-workflow-test folder, simulating ~/.config/sortd.
3. Verify that workflow definitions (located in the workflows folder) are applied properly.
4. Bypass interactive selections (e.g., gum selection logic) in automated mode, eliminating manual file selection.
5. Confirm robust error handling and proper logging.

## Test Components

### A. Test Environment Setup

- **Mock File System:** Use the contents of `/home/toasty/projects/sortd/mock_filesystem_playground` as the test data source.
- **Shell Scripts:** Utilize provided scripts (e.g., `create_mock_fs.sh`, `simple_test_files.sh`, `create_realistic_files.sh`) to set up and seed the mock file system as needed.
- **Configuration:** Point to `@sortd-config-workflow-test` to simulate the user's `~/.config/sortd` configuration.
- **Workflows:** Use the workflow definitions from the `workflows` directory (e.g., in `/home/toasty/projects/sortd/examples/workflows`).

### B. Test Execution

- **Non-Interactive Mode:** Run the sortd application in a mode that bypasses gum selection logic to avoid interactive prompts. This might involve a flag (e.g., `--non-interactive`) or environment variable to force auto-selection.
- **Workflow Triggering:** Ensure that when running a workflow, files are automatically processed without manual selection.

### C. Test Cases

1. **Full Sorting Demo**
   - **Setup:** Copy the mock file system to a temporary location.
   - **Execution:** Run the sortd application with the test configuration and workflows.
   - **Validation:** Confirm that files are correctly sorted, renamed, or moved as specified by the workflow logic.

2. **Workflow Execution Test**
   - **Setup:** Ensure configuration and workflow files are in place.
   - **Execution:** Launch a complete workflow and verify that it runs from start to finish automatically.
   - **Validation:** Check that the expected output directories and file operations occur without any interactive file selection prompts.

3. **Gum Selection Override Test**
   - **Objective:** Confirm that interactive prompts (i.e., gum selection) are bypassed in non-interactive mode.
   - **Execution:** Run the CLI application in non-interactive mode and verify that it does not invoke the gum logic, instead using default or pre-determined file selections.

4. **Error Handling and Edge Cases**
   - **Testing:** Simulate scenarios such as missing configuration files, permission issues, and read-only files.
   - **Validation:** Ensure that proper error messages are logged and the application fails gracefully.

5. **Cleanup**
   - **Objective:** Ensure that any temporary resources (directories, files) created during testing are removed after test execution.

## Running the Tests

- **Setup Script:** Consider creating or reusing existing shell scripts to initialize the mock file system environment.
- **Execution Command:** Run the sortd application with test-specific configurations (for example, using a command like `sortd --config=./sortd-config-workflow-test --non-interactive <other-options>`).
- **Integration:** These tests could later be integrated into a CI/CD pipeline for automated execution.

## Future Considerations

- **Non-Interactive Flag:** If not already present, consider refactoring the CLI to add a `--non-interactive` flag to cleanly bypass gum selection.
- **Enhanced Logging:** Improve logging throughout the application to better trace workflow execution during end-to-end tests.
- **CI Integration:** Work towards integrating these tests into the continuous integration pipeline to catch regressions early.

## Conclusion

This end-to-end test plan provides a structured approach to validate the functionality of the sortd application using a mock file system, configuration files, and workflow definitions. By bypassing interactive selection, it aims to ensure that automated workflows run smoothly and reliably in a non-interactive environment.