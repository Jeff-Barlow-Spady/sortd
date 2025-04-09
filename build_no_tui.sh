#!/bin/bash

# Build the main executable
echo "Building sortd executable..."
go build -o sortd ./cmd/sortd

# If you want to build CLI separately
# echo "Building sortd-cli executable..."
# go build -o sortd-cli ./cmd/sortd-cli

echo "Build completed."