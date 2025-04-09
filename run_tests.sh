#!/bin/bash

# Define directories to exclude from testing
EXCLUDE_DIRS=(
  "./internal/.tui.bak"
  "./tests/.tui.bak"
)

# Build the exclude pattern for go test
EXCLUDE_PATTERN=""
for dir in "${EXCLUDE_DIRS[@]}"; do
  EXCLUDE_PATTERN="${EXCLUDE_PATTERN} -not -path \"${dir}/*\""
done

# Find all packages to test, excluding the defined directories
TEST_PACKAGES=$(eval "find ./ -type f -name '*_test.go' ${EXCLUDE_PATTERN}" | xargs -n1 dirname | sort -u)

echo "Running tests for the following packages:"
echo "${TEST_PACKAGES}"
echo ""

# Run go test on the packages
go test -v ${TEST_PACKAGES}