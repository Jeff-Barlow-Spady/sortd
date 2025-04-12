#!/bin/bash
#
# Script to create release binaries for Sortd
# This uses the Docker test environment to build binaries for Linux and Windows

# Strict mode
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
RELEASE_DIR="${PROJECT_ROOT}/release"

# Print banner
print_banner() {
    echo -e "${MAGENTA}${BOLD}"
    echo "╔═════════════════════════════════════════════════════════╗"
    echo "║                                                         ║"
    echo "║   SORTD RELEASE BUILDER                                 ║"
    echo "║   Let Chaos Sort Itself Out! 🗂️                         ║"
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

    echo -e "${color}${symbol} ${message}${RESET}"
}

# Get the version from git
get_version() {
    local version
    if git describe --exact-match --tags HEAD 2>/dev/null; then
        version=$(git describe --exact-match --tags HEAD)
    else
        # If no exact tag, use commit hash
        version="dev-$(git rev-parse --short HEAD)"
        log "WARNING" "No tag found, using commit hash: ${version}"
    fi
    echo "${version}"
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

# Create release directory
create_release_dir() {
    local dir=$1
    if [ ! -d "$dir" ]; then
        mkdir -p "$dir"
        log "INFO" "Created release directory: $dir"
    else
        log "INFO" "Using existing release directory: $dir"
    fi
}

# Build the Docker image if needed
ensure_docker_image() {
    if ! docker image inspect sortd-builder &> /dev/null; then
        log "STEP" "Building Docker image for sortd..."
        docker build -t sortd-builder -f "${SCRIPT_DIR}/Dockerfile" "${PROJECT_ROOT}"
        log "SUCCESS" "Docker image built successfully"
    else
        log "INFO" "Using existing Docker image for sortd"
    fi
}

# Build Linux binary
build_linux() {
    local version=$1
    local output_dir=$2
    local binary_name="sortd-${version}-linux-amd64"
    local tarball_name="${binary_name}.tar.gz"

    log "STEP" "Building Linux binary..."

    # Build the binary
    docker run --rm -v "${PROJECT_ROOT}:/app/src" -v "${output_dir}:/app/bin" \
        sortd-builder build_linux

    # Rename the binary
    cp "${output_dir}/sortd" "${output_dir}/${binary_name}"
    chmod +x "${output_dir}/${binary_name}"

    # Create tarball
    tar -czf "${output_dir}/${tarball_name}" -C "${output_dir}" "${binary_name}"

    log "SUCCESS" "Linux binary built: ${output_dir}/${tarball_name}"
}

# Build Windows binary
build_windows() {
    local version=$1
    local output_dir=$2
    local binary_name="sortd-${version}-windows-amd64.exe"
    local zip_name="sortd-${version}-windows-amd64.zip"

    log "STEP" "Building Windows binary..."

    # Build the binary
    docker run --rm -v "${PROJECT_ROOT}:/app/src" -v "${output_dir}:/app/bin" \
        sortd-builder build_windows

    # Rename the binary
    cp "${output_dir}/sortd.exe" "${output_dir}/${binary_name}"

    # Create zip
    (cd "${output_dir}" && zip "${zip_name}" "${binary_name}")

    log "SUCCESS" "Windows binary built: ${output_dir}/${zip_name}"
}

# Generate checksums
generate_checksums() {
    local version=$1
    local output_dir=$2
    local linux_tarball="sortd-${version}-linux-amd64.tar.gz"
    local windows_zip="sortd-${version}-windows-amd64.zip"

    log "STEP" "Generating checksums..."

    # Generate checksums file
    (cd "${output_dir}" && sha256sum "${linux_tarball}" "${windows_zip}" > SHA256SUMS.txt)

    # Generate markdown version
    echo "## Checksums" > "${output_dir}/CHECKSUMS.md"
    echo "### Linux (AMD64)" >> "${output_dir}/CHECKSUMS.md"
    (cd "${output_dir}" && sha256sum "${linux_tarball}" | tee -a CHECKSUMS.md)

    echo "" >> "${output_dir}/CHECKSUMS.md"
    echo "### Windows (AMD64)" >> "${output_dir}/CHECKSUMS.md"
    (cd "${output_dir}" && sha256sum "${windows_zip}" | tee -a CHECKSUMS.md)

    log "SUCCESS" "Checksums generated: ${output_dir}/SHA256SUMS.txt"
}

# Main function
main() {
    print_banner

    # Check Docker
    check_docker

    # Get version
    VERSION=$(get_version)
    log "INFO" "Building release for version: ${VERSION}"

    # Create output directory
    VERSION_DIR="${RELEASE_DIR}/${VERSION}"
    create_release_dir "${VERSION_DIR}"

    # Ensure Docker image exists
    ensure_docker_image

    # Build binaries
    build_linux "${VERSION}" "${VERSION_DIR}"
    build_windows "${VERSION}" "${VERSION_DIR}"

    # Generate checksums
    generate_checksums "${VERSION}" "${VERSION_DIR}"

    log "SUCCESS" "Release files created in ${VERSION_DIR}"
    log "INFO" "You can now create a GitHub release with these files"
}

# Run the main function
main