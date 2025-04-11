#!/bin/bash

# End-to-End Test Runner for sortd
# Executes tests according to the E2E test plan

set -e

# Base directories
REPO_DIR="/home/toasty/projects/sortd"
BASE_DIR="$REPO_DIR/e2e_tests"
MOCK_FS="$BASE_DIR/mock_filesystem_playground"
CONFIG_DIR="$BASE_DIR/sortd-config-workflow-test"
LOG_DIR="$BASE_DIR/logs"

# Path to sortd executable
SORTD_CMD="$REPO_DIR/sortd"

# Log file
LOG_FILE="$LOG_DIR/e2e_test_$(date +%Y%m%d_%H%M%S).log"

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Function to run a test case
run_test() {
    local test_name="$1"
    local test_cmd="$2"

    log "▶️ Running test: $test_name"
    log "Command: $test_cmd"

    if eval "$test_cmd"; then
        log "✅ Test passed: $test_name"
        return 0
    else
        log "❌ Test failed: $test_name (Exit code: $?)"
        return 1
    fi
}

# Verify the test environment exists
if [ ! -d "$MOCK_FS" ] || [ ! -d "$CONFIG_DIR" ]; then
    log "❌ Test environment not found. Please run setup_test_env.sh first."
    exit 1
fi

# Verify sortd executable exists
if [ ! -x "$SORTD_CMD" ]; then
    log "❌ sortd executable not found at $SORTD_CMD or not executable."
    exit 1
fi

log "🚀 Starting sortd end-to-end tests"

# Create a test file snapshot for comparison
mkdir -p "$MOCK_FS/snapshot_before"
find "$MOCK_FS/Downloads" -type f -exec cp {} "$MOCK_FS/snapshot_before" \;
log "📸 Created snapshot of test files"

# Test Case 1: Full Sorting Demo
log "📋 Test Case 1: Full Sorting Demo"
# Run sortd with the test configuration in non-interactive mode
run_test "Full sorting demo" "$SORTD_CMD --config=$CONFIG_DIR --non-interactive"

# Test Case 2: Workflow Execution Test
log "📋 Test Case 2: Workflow Execution Test"
# Run a specific workflow in non-interactive mode
run_test "Document processor workflow" "$SORTD_CMD workflow run --config=$CONFIG_DIR --id=document-processor --non-interactive"
run_test "Image sorter workflow" "$SORTD_CMD workflow run --config=$CONFIG_DIR --id=image-sorter --non-interactive"

# Test Case 3: Gum Selection Override Test
log "📋 Test Case 3: Gum Selection Override Test"
# Ensure non-interactive mode bypasses gum selection
log "Running sortd with explicit bypass of gum selection..."
run_test "Gum bypass test" "$SORTD_CMD --config=$CONFIG_DIR --non-interactive --debug | grep -i 'bypassing gum selection'"

# Test Case 4: Error Handling Test
log "📋 Test Case 4: Error Handling Test"
# Test with invalid configuration
run_test "Invalid config test" "$SORTD_CMD --config=/path/that/does/not/exist 2>&1 | grep -i 'error\|fail\|invalid'"

# Create a snapshot after tests for comparison
mkdir -p "$MOCK_FS/snapshot_after"
find "$MOCK_FS" -type f -not -path "*/snapshot_*/*" -exec cp {} "$MOCK_FS/snapshot_after" \;

# Analyze changes
log "📊 Analyzing test results..."
diff_file="$LOG_DIR/file_diff.txt"
diff -r "$MOCK_FS/snapshot_before" "$MOCK_FS/snapshot_after" > "$diff_file" || true

if [ -s "$diff_file" ]; then
    log "🔄 Changes detected in the file system:"
    cat "$diff_file" | tee -a "$LOG_FILE"
else
    log "❓ No changes detected in the file system"
fi

# Output test summary
log "📝 Test Summary"
log "=============="
log "Environment: $MOCK_FS"
log "Configuration: $CONFIG_DIR"
log "Log file: $LOG_FILE"

log "✅ End-to-end tests completed"