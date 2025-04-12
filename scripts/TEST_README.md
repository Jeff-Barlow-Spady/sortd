# Sortd Testing Suite

This directory contains scripts to test Sortd in a sandboxed environment on both Linux and Windows platforms. The scripts create a controlled file system with test files, set up configuration, and run various test cases to validate the functionality of Sortd.

## Requirements

### Linux
- Bash shell
- `dd` command (usually pre-installed on Linux)
- `find` command (usually pre-installed on Linux)
- `diff` command (usually pre-installed on Linux)

### Windows
- PowerShell 5.1 or higher
- Administrator privileges may be required for certain file operations

## Getting Started

### Build Sortd First

Before running the tests, make sure to build the Sortd binary:

**Linux:**
```bash
cd /path/to/sortd
make build
# or
./build_no_tui.sh
```

**Windows:**
```powershell
cd C:\path\to\sortd
go build -o sortd.exe .\cmd\sortd\
```

### Running the Tests

**Linux:**
```bash
cd /path/to/sortd
./scripts/test_sortd_linux.sh
```

**Windows:**
```powershell
cd C:\path\to\sortd
.\scripts\test_sortd_windows.ps1
```

## What the Tests Do

Both test scripts perform the following operations:

1. **Create a Sandbox Environment**: Sets up a separate test directory structure to avoid interfering with any existing files.

2. **Generate Test Files**: Creates a variety of test files with different extensions to simulate a real file system:
   - Documents (txt, pdf) including some named with "invoice"
   - Images (jpg, png) including some named with "vacation"
   - Audio files (mp3, wav)
   - Video files (mp4, mkv)
   - Archive files (zip, tar.gz)
   - A large file to test size-based conditions
   - A read-only file to test permission handling

3. **Set Up Configuration**: Creates a configuration directory with `config.yaml` and workflow definitions:
   - Basic file patterns for sorting different file types
   - A document processor workflow for handling invoices
   - An image sorter workflow for vacation photos

4. **Take Before Snapshot**: Records the initial state of the file system for comparison.

5. **Run Test Cases**:
   - Basic sorting using patterns
   - Document processor workflow execution
   - Image sorter workflow execution
   - Error handling with invalid configuration

6. **Take After Snapshot**: Records the final state of the file system.

7. **Analyze Results**: Compares before and after snapshots to verify changes.

8. **Generate Report**: Creates a detailed test report with logs and directory structure.

## Test Output

The tests generate comprehensive logs with colored output for easy readability. All logs are saved to the `test_sandbox/logs` directory with timestamps.

After running the tests, you can find:
- Test logs: `test_sandbox/logs/test_YYYYMMDD_HHMMSS.log`
- File difference report: `test_sandbox/logs/file_diff.txt`
- Before and after snapshots: `test_sandbox/snapshot_before` and `test_sandbox/snapshot_after`

## Customizing Tests

You can modify the scripts to:
- Add more test cases by extending the test runner functions
- Change the test file patterns or contents
- Modify the workflow configurations
- Add more complex test scenarios

## Troubleshooting

### Linux
- If you get permission errors, you may need to run the script with `sudo`
- Ensure the Sortd binary is executable: `chmod +x /path/to/sortd`

### Windows
- If PowerShell script execution is restricted, you may need to run: `Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass`
- Ensure you have the correct path to the Sortd executable in the script

## Note on Non-Interactive Mode

These tests rely on the `--non-interactive` flag being properly implemented in Sortd to bypass any interactive prompts. If you encounter issues with interactive prompts during testing, verify that the non-interactive mode is working correctly.