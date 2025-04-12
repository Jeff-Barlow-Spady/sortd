#!/bin/bash

# Sortd Docker Test Environment Helper
# This script provides commands to manage the Sortd Docker test environment

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

# Print banner
print_banner() {
    echo -e "${MAGENTA}${BOLD}"
    echo "╔═════════════════════════════════════════════════════════╗"
    echo "║                                                         ║"
    echo "║         SORTD DOCKER TEST ENVIRONMENT                   ║"
    echo "║          Let Chaos Sort Itself Out! 🗂️                 ║"
    echo "║                                                         ║"
    echo "╚═════════════════════════════════════════════════════════╝"
    echo -e "${RESET}"
}

# Print usage
print_usage() {
    echo -e "${BOLD}Usage:${RESET} $0 [command]"
    echo
    echo "Commands:"
    echo -e "  ${CYAN}start${RESET}       Start the Docker test environment"
    echo -e "  ${CYAN}stop${RESET}        Stop the Docker test environment"
    echo -e "  ${CYAN}shell${RESET}       Open a shell in the running container"
    echo -e "  ${CYAN}prepare${RESET}     Prepare the test environment (create test files and configs)"
    echo -e "  ${CYAN}build-linux${RESET} Build Sortd for Linux"
    echo -e "  ${CYAN}build-win${RESET}   Build Sortd for Windows"
    echo -e "  ${CYAN}run${RESET}         Run Sortd with arguments (e.g. '$0 run organize --help')"
    echo -e "  ${CYAN}status${RESET}      Check if the container is running"
    echo -e "  ${CYAN}clean${RESET}       Remove the container and volume (WARNING: Deletes all test data)"
    echo
    echo "Examples:"
    echo -e "  $0 start"
    echo -e "  $0 prepare"
    echo -e "  $0 build-linux"
    echo -e "  $0 run organize --config=/app/test_sandbox/sortd-config --dir=/app/test_sandbox/mock_filesystem/Downloads --non-interactive"
    echo -e "  $0 shell"
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

    echo -e "${color}${symbol} ${message}${RESET}"
}

# Check if Docker is installed and running
check_docker() {
    if ! command -v docker &> /dev/null; then
        log "ERROR" "Docker is not installed. Please install Docker first."
        exit 1
    fi

    if ! docker info &> /dev/null; then
        log "ERROR" "Docker is not running. Please start Docker first."
        exit 1
    fi
}

# Check if docker compose is installed
check_docker_compose() {
    if ! docker compose version &> /dev/null; then
        log "ERROR" "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
}

# Start the container
start_container() {
    cd "$SCRIPT_DIR"
    log "STEP" "Starting Sortd Docker test environment..."
    docker compose up -d
    log "SUCCESS" "Sortd Docker test environment started"
    log "INFO" "Use '$0 shell' to open a shell in the container"
}

# Stop the container
stop_container() {
    cd "$SCRIPT_DIR"
    log "STEP" "Stopping Sortd Docker test environment..."
    docker compose down
    log "SUCCESS" "Sortd Docker test environment stopped"
}

# Clean up everything
clean_container() {
    cd "$SCRIPT_DIR"
    log "WARNING" "This will remove the container and all test data!"
    read -p "Are you sure you want to continue? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log "STEP" "Removing Sortd Docker test environment..."
        docker compose down -v
        log "SUCCESS" "Sortd Docker test environment removed"
    else
        log "INFO" "Operation cancelled"
    fi
}

# Open a shell in the container
open_shell() {
    if [[ "$(docker ps -q -f name=sortd-test)" ]]; then
        log "STEP" "Opening shell in Sortd Docker test environment..."
        docker exec -it sortd-test /bin/bash
    else
        log "ERROR" "Sortd Docker test environment is not running"
        log "INFO" "Use '$0 start' to start it first"
    fi
}

# Run a command in the container
run_command() {
    if [[ "$(docker ps -q -f name=sortd-test)" ]]; then
        log "STEP" "Running command in Sortd Docker test environment..."
        docker exec -it sortd-test "$@"
    else
        log "ERROR" "Sortd Docker test environment is not running"
        log "INFO" "Use '$0 start' to start it first"
    fi
}

# Check container status
check_status() {
    if [[ "$(docker ps -q -f name=sortd-test)" ]]; then
        log "SUCCESS" "Sortd Docker test environment is running"
    else
        log "INFO" "Sortd Docker test environment is not running"
    fi
}

# Prepare the test environment in the container
prepare_env() {
    if [[ "$(docker ps -q -f name=sortd-test)" ]]; then
        log "STEP" "Preparing test environment in Sortd Docker test environment..."
        docker exec -it sortd-test prepare
        log "SUCCESS" "Test environment prepared"
    else
        log "ERROR" "Sortd Docker test environment is not running"
        log "INFO" "Use '$0 start' to start it first"
    fi
}

# Build Sortd for Linux
build_linux() {
    if [[ "$(docker ps -q -f name=sortd-test)" ]]; then
        log "STEP" "Building Sortd for Linux..."
        docker exec -it sortd-test build_linux
        log "SUCCESS" "Sortd built for Linux"
        log "INFO" "Binary is available at /app/bin/sortd in the container"
    else
        log "ERROR" "Sortd Docker test environment is not running"
        log "INFO" "Use '$0 start' to start it first"
    fi
}

# Build Sortd for Windows
build_windows() {
    if [[ "$(docker ps -q -f name=sortd-test)" ]]; then
        log "STEP" "Building Sortd for Windows..."
        docker exec -it sortd-test build_windows
        log "SUCCESS" "Sortd built for Windows"
        log "INFO" "Binary is available at /app/bin/sortd.exe in the container"
        log "INFO" "You can copy it to your host with: docker cp sortd-test:/app/bin/sortd.exe ."
    else
        log "ERROR" "Sortd Docker test environment is not running"
        log "INFO" "Use '$0 start' to start it first"
    fi
}

# Run Sortd with arguments
run_sortd() {
    if [[ "$(docker ps -q -f name=sortd-test)" ]]; then
        log "STEP" "Running Sortd with arguments: $*"
        docker exec -it sortd-test run "$@"
    else
        log "ERROR" "Sortd Docker test environment is not running"
        log "INFO" "Use '$0 start' to start it first"
    fi
}

# Main command dispatcher
main() {
    print_banner

    check_docker
    check_docker_compose

    case "$1" in
        "start")
            start_container
            ;;
        "stop")
            stop_container
            ;;
        "shell")
            open_shell
            ;;
        "prepare")
            prepare_env
            ;;
        "build-linux")
            build_linux
            ;;
        "build-win")
            build_windows
            ;;
        "run")
            shift
            run_sortd "$@"
            ;;
        "status")
            check_status
            ;;
        "clean")
            clean_container
            ;;
        *)
            print_usage
            ;;
    esac
}

# Run the main function with all arguments
main "$@"