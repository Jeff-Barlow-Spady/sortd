#!/bin/bash
# Test script to validate release builds locally
# This mimics what the GitHub Actions workflow does

set -e  # Exit on any error

# Set a test version
TEST_VERSION="v0.0.0-test"
echo "🔧 Testing release build for version: $TEST_VERSION"

# Create a build directory
BUILD_DIR="./test-release-build"
mkdir -p "$BUILD_DIR"
echo "📁 Created build directory: $BUILD_DIR"

# Build Linux binary
echo "🐧 Building Linux binary..."
go build -o "$BUILD_DIR/sortd-$TEST_VERSION-linux" ./cmd/sortd
chmod +x "$BUILD_DIR/sortd-$TEST_VERSION-linux"
tar -czf "$BUILD_DIR/sortd-$TEST_VERSION-linux.tar.gz" -C "$BUILD_DIR" "sortd-$TEST_VERSION-linux"
echo "✅ Linux build complete"

# Build Windows binary
echo "🪟 Building Windows binary..."
echo "   (Using build tag 'nogui' to disable GUI components for cross-compilation)"
GOOS=windows GOARCH=amd64 go build -tags nogui -o "$BUILD_DIR/sortd-$TEST_VERSION-windows.exe" ./cmd/sortd
(cd "$BUILD_DIR" && zip "sortd-$TEST_VERSION-windows.zip" "sortd-$TEST_VERSION-windows.exe")
echo "✅ Windows build complete"

# Create a validation file to simulate a GitHub release
echo "📝 Creating validation file..."
cat > "$BUILD_DIR/release-validation.txt" << EOF
This would be a GitHub release with:
- Name: Release $TEST_VERSION
- Files:
  - sortd-$TEST_VERSION-linux.tar.gz
  - sortd-$TEST_VERSION-windows.zip
EOF

echo "
✅✅ VALIDATION COMPLETE ✅✅

The release build has been tested successfully. The following files were created:
  - $BUILD_DIR/sortd-$TEST_VERSION-linux.tar.gz
  - $BUILD_DIR/sortd-$TEST_VERSION-windows.zip

You can verify these files by:
  - Extracting and running the Linux binary (on Linux)
  - Extracting and running the Windows binary (on Windows)

NOTE: The Windows binary is built with GUI disabled for cross-compilation.
      If you need GUI support on Windows, you'll need to build on Windows or
      set up proper cross-compilation toolchain with GUI libraries.

If everything looks good, you can create a real release by:
  1. git tag v1.0.0 (or your desired version)
  2. git push origin v1.0.0
"