#!/bin/bash

# Sortd Test Script for Linux
# This script creates a sandboxed environment to test sortd functionality

# Enable strict mode
set -euo pipefail

# Color definitions for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_ROOT="${PROJECT_ROOT}/test_sandbox"
MOCK_FS="${TEST_ROOT}/mock_filesystem"
CONFIG_DIR="${TEST_ROOT}/sortd-config"
WORKFLOWS_DIR="${CONFIG_DIR}/workflows"
LOG_DIR="${TEST_ROOT}/logs"
LOG_FILE="${LOG_DIR}/test_$(date +%Y%m%d_%H%M%S).log"
SORTD_BINARY="${PROJECT_ROOT}/sortd"

# Print banner
print_banner() {
    echo -e "${MAGENTA}${BOLD}"
    echo "╔═════════════════════════════════════════════════════════╗"
    echo "║                                                         ║"
    echo "║   SORTD TEST SUITE - Let Chaos Sort Itself Out! 🗂️      ║"
    echo "║                                                         ║"
    echo "╚═════════════════════════════════════════════════════════╝"
    echo -e "${RESET}"
}

# Log function with timestamp and color
log() {
    local level=$1
    local message=$2
    local color=$RESET
    local symbol=""

    case $level in
        "INFO")
            color=$BLUE
            symbol="ℹ️"
            ;;
        "SUCCESS")
            color=$GREEN
            symbol="✅"
            ;;
        "WARNING")
            color=$YELLOW
            symbol="⚠️"
            ;;
        "ERROR")
            color=$RED
            symbol="❌"
            ;;
        "STEP")
            color=$CYAN
            symbol="🔷"
            ;;
    esac

    echo -e "${color}${symbol} [$(date '+%Y-%m-%d %H:%M:%S')] [${level}] ${message}${RESET}" | tee -a "$LOG_FILE"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to run a test case
run_test() {
    local test_name="$1"
    local test_cmd="$2"

    log "STEP" "Running test: ${BOLD}${test_name}${RESET}"
    log "INFO" "Command: $test_cmd"

    echo "---------------------------------------------" | tee -a "$LOG_FILE"
    if eval "$test_cmd"; then
        log "SUCCESS" "Test passed: ${test_name}"
        return 0
    else
        local exit_code=$?
        log "ERROR" "Test failed: ${test_name} (Exit code: ${exit_code})"
        return $exit_code
    fi
}

# Function to create test files
create_test_files() {
    local dir="$1"

    # Create directories
    mkdir -p "${dir}/Downloads/Documents"
    mkdir -p "${dir}/Downloads/Images"
    mkdir -p "${dir}/Downloads/Music"
    mkdir -p "${dir}/Downloads/Videos"
    mkdir -p "${dir}/Downloads/Archives"

    log "INFO" "Creating test document files..."
    # Create document files
    for i in {1..5}; do
        echo "This is test document $i" > "${dir}/Downloads/test_doc_${i}.txt"
        touch "${dir}/Downloads/test_doc_${i}.pdf"
        touch "${dir}/Downloads/invoice_${i}.pdf"
    done

    log "INFO" "Creating test image files..."
    # Create image files
    for i in {1..5}; do
        touch "${dir}/Downloads/test_image_${i}.jpg"
        touch "${dir}/Downloads/test_image_${i}.png"
        touch "${dir}/Downloads/vacation_pic_${i}.jpg"
    done

    log "INFO" "Creating test audio files..."
    # Create audio files
    for i in {1..3}; do
        touch "${dir}/Downloads/test_song_${i}.mp3"
        touch "${dir}/Downloads/test_audio_${i}.wav"
    done

    log "INFO" "Creating test video files..."
    # Create video files
    for i in {1..3}; do
        touch "${dir}/Downloads/test_video_${i}.mp4"
        touch "${dir}/Downloads/movie_${i}.mkv"
    done

    log "INFO" "Creating test archive files..."
    # Create archive files
    for i in {1..3}; do
        touch "${dir}/Downloads/test_archive_${i}.zip"
        touch "${dir}/Downloads/backup_${i}.tar.gz"
    done

    # Create a large file
    log "INFO" "Creating a large test file..."
    dd if=/dev/urandom of="${dir}/Downloads/large_file.bin" bs=1M count=5 2>/dev/null

    # Create a readonly file
    log "INFO" "Creating a readonly file..."
    echo "This file is readonly" > "${dir}/Downloads/readonly_file.txt"
    chmod 444 "${dir}/Downloads/readonly_file.txt"

    log "SUCCESS" "Test files created successfully"
}

# Function to set up configuration
setup_config() {
    local config_dir="$1"
    local workflows_dir="$2"

    # Create config directory
    mkdir -p "$workflows_dir"

    # Create main config.yaml
    cat > "${config_dir}/config.yaml" << EOF
# Sortd Configuration for Test Environment
version: 1

# Global Settings
settings:
  dry_run: false
  create_dirs: true
  collision_strategy: "rename"
  confirm_operations: false

# Sorting patterns
patterns:
  - match: "*.{jpg,jpeg,png,gif,bmp}"
    target: "Images/"
  - match: "*.{doc,docx,pdf,txt,md,rtf}"
    target: "Documents/"
  - match: "*.{mp3,wav,flac,ogg,m4a}"
    target: "Music/"
  - match: "*.{mp4,mkv,avi,mov,wmv}"
    target: "Videos/"
  - match: "*.{zip,tar,gz,rar,7z}"
    target: "Archives/"
  - match: "invoice_*.pdf"
    target: "Documents/Invoices/"

# Watch directories
watch_directories:
  - "${MOCK_FS}/Downloads"
EOF

    # Create document processor workflow
    cat > "${workflows_dir}/document_processor.yaml" << EOF
id: "document-processor"
name: "Document Processor"
description: "Process documents based on content and type"
enabled: true
priority: 5

trigger:
  type: "FileCreated"
  pattern: "*.{pdf,txt,doc,docx}"

conditions:
  - type: "FileCondition"
    field: "name"
    operator: "Contains"
    value: "invoice"
    caseSensitive: false

actions:
  - type: "MoveAction"
    target: "${MOCK_FS}/Downloads/Documents/Invoices"
    options:
      createTargetDir: "true"
EOF

    # Create image sorter workflow
    cat > "${workflows_dir}/image_sorter.yaml" << EOF
id: "image-sorter"
name: "Image Sorter"
description: "Sort images into appropriate folders"
enabled: true
priority: 4

trigger:
  type: "FileCreated"
  pattern: "*.{jpg,jpeg,png,gif}"

conditions:
  - type: "FileCondition"
    field: "name"
    operator: "Contains"
    value: "vacation"
    caseSensitive: false

actions:
  - type: "MoveAction"
    target: "${MOCK_FS}/Downloads/Images/Vacation"
    options:
      createTargetDir: "true"
EOF

    log "SUCCESS" "Configuration files created successfully"
}

# Main function
main() {
    print_banner

    # Check if sortd exists
    if ! command_exists "$SORTD_BINARY"; then
        log "ERROR" "sortd binary not found at $SORTD_BINARY"
        log "INFO" "Please build sortd first with: make build"
        exit 1
    fi

    # Create directories
    log "STEP" "Creating test directories"
    mkdir -p "$TEST_ROOT" "$MOCK_FS" "$CONFIG_DIR" "$LOG_DIR"

    # Create test files
    log "STEP" "Setting up mock file system"
    create_test_files "$MOCK_FS"

    # Create configuration
    log "STEP" "Setting up configuration"
    setup_config "$CONFIG_DIR" "$WORKFLOWS_DIR"

    # Create snapshot for comparison
    log "STEP" "Creating pre-test snapshot"
    mkdir -p "${TEST_ROOT}/snapshot_before"
    find "${MOCK_FS}/Downloads" -type f -exec cp --preserve=all {} "${TEST_ROOT}/snapshot_before" \;

    # Test cases
    log "STEP" "Running test cases"

    # Test Case 1: Basic sorting with patterns
    run_test "Basic sorting with patterns" "$SORTD_BINARY organize --config=$CONFIG_DIR --dir=${MOCK_FS}/Downloads --non-interactive"

    # Test Case 2: Document processor workflow
    run_test "Document processor workflow" "$SORTD_BINARY workflow run --config=$CONFIG_DIR --id=document-processor --non-interactive"

    # Test Case 3: Image sorter workflow
    run_test "Image sorter workflow" "$SORTD_BINARY workflow run --config=$CONFIG_DIR --id=image-sorter --non-interactive"

    # Test Case 4: Error handling test
    run_test "Error handling test" "! $SORTD_BINARY --config=/path/that/does/not/exist 2>&1 | grep -i 'error\\|fail\\|invalid'"

    # Create snapshot after tests
    log "STEP" "Creating post-test snapshot"
    mkdir -p "${TEST_ROOT}/snapshot_after"
    find "${MOCK_FS}" -type f -not -path "*/snapshot_*/*" -exec cp --preserve=all {} "${TEST_ROOT}/snapshot_after" \;

    # Analyze changes
    log "STEP" "Analyzing results"
    diff_file="${LOG_DIR}/file_diff.txt"
    diff -r "${TEST_ROOT}/snapshot_before" "${TEST_ROOT}/snapshot_after" > "$diff_file" || true

    if [ -s "$diff_file" ]; then
        log "INFO" "Changes detected in the file system:"
        cat "$diff_file" | tee -a "$LOG_FILE"
    else
        log "WARNING" "No changes detected in the file system"
    fi

    # Show final directory structure
    log "STEP" "Final directory structure"
    find "${MOCK_FS}" -type d | sort | tee -a "$LOG_FILE"

    # Test summary
    log "STEP" "Test summary"
    echo -e "${CYAN}${BOLD}Test Summary${RESET}" | tee -a "$LOG_FILE"
    echo "=============================================" | tee -a "$LOG_FILE"
    echo "Environment: $MOCK_FS" | tee -a "$LOG_FILE"
    echo "Configuration: $CONFIG_DIR" | tee -a "$LOG_FILE"
    echo "Log file: $LOG_FILE" | tee -a "$LOG_FILE"

    log "SUCCESS" "End-to-end tests completed successfully"
}

# Run the main function
main "$@"