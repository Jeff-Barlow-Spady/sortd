#!/bin/bash
# Simple script to build Sortd for both Linux and Windows

echo "🔧 Building Sortd for Linux and Windows..."

# Create output directory
mkdir -p bin

# Build for Linux
echo "🐧 Building Linux binary..."
go build -o bin/sortd-linux ./cmd/sortd
chmod +x bin/sortd-linux

# Build for Windows
echo "🪟 Building Windows binary..."
GOOS=windows GOARCH=amd64 go build -o bin/sortd-windows.exe ./cmd/sortd

echo "✅ Build complete! Binaries are in the bin/ directory:"
echo "   - bin/sortd-linux (Linux)"
echo "   - bin/sortd-windows.exe (Windows)"