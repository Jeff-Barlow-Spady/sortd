# Sortd Docker Test Environment

This directory contains scripts to set up a persistent Docker-based testing environment for Sortd. The environment allows for interactive testing, cloud storage integration, and cross-platform compilation.

## Overview

The Docker test environment provides:

1. **Persistent Test Sandbox**: All test files and configurations persist across container restarts
2. **Cross-Platform Compilation**: Build Sortd for both Linux and Windows
3. **Cloud Storage Integration**: Test S3 bucket organization (AWS credentials required)
4. **Interactive Testing**: Shell into the container for manual testing and experimentation
5. **Automated Test Workflows**: Run predefined workflows against the test data

## Prerequisites

- Docker
- Docker Compose
- Bash shell (for Linux/macOS) or Git Bash/WSL (for Windows)

## Getting Started

### 1. Start the Docker Environment

```bash
./scripts/docker_test_env.sh start
```

This will build the Docker image if it doesn't exist and start the container.

### 2. Prepare the Test Environment

```bash
./scripts/docker_test_env.sh prepare
```

This creates test files, directories, and configuration files in the persistent test sandbox.

### 3. Build Sortd

For Linux:
```bash
./scripts/docker_test_env.sh build-linux
```

For Windows:
```bash
./scripts/docker_test_env.sh build-win
```

### 4. Run Sortd

```bash
./scripts/docker_test_env.sh run organize --config=/app/test_sandbox/sortd-config --dir=/app/test_sandbox/mock_filesystem/Downloads --non-interactive
```

### 5. Open a Shell for Interactive Testing

```bash
./scripts/docker_test_env.sh shell
```

Once inside the shell, you can run Sortd directly:

```bash
# Run Sortd with organize command
run_organize

# Run a specific workflow
run_workflow document-processor

# Run any Sortd command
run workflow list
```

## Test Sandbox Structure

The test sandbox is a Docker volume that persists across container restarts:

- `/app/test_sandbox/mock_filesystem/` - Mock file system with Downloads directory and test files
- `/app/test_sandbox/sortd-config/` - Configuration for Sortd
- `/app/test_sandbox/sortd-config/workflows/` - Test workflow definitions
- `/app/test_sandbox/logs/` - Log files

## Cloud Storage Integration

To test S3 integration:

1. Set AWS credentials as environment variables before starting the container:

```bash
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_DEFAULT_REGION=your_region
./scripts/docker_test_env.sh start
```

2. Use the provided S3 upload workflow:

```bash
./scripts/docker_test_env.sh run_workflow s3-upload
```

## Cross-Platform Testing

To test Windows compatibility:

1. Build the Windows binary:

```bash
./scripts/docker_test_env.sh build-win
```

2. Copy the binary to your host:

```bash
docker cp sortd-test:/app/bin/sortd.exe .
```

3. Run the Windows binary on a Windows machine with your test files

## Path Handling

The Docker environment handles path differences between platforms:

- Inside the container, paths use the Linux format (e.g., `/app/test_sandbox/...`)
- For Windows builds, the binary handles path conversion automatically
- Workflow definitions use path templates that work on both platforms

## Available Commands

Run `./scripts/docker_test_env.sh` with no arguments to see all available commands:

- `start` - Start the container
- `stop` - Stop the container
- `shell` - Open a shell in the container
- `prepare` - Set up the test environment
- `build-linux` - Build Sortd for Linux
- `build-win` - Build Sortd for Windows
- `run [args]` - Run Sortd with arguments
- `status` - Check if the container is running
- `clean` - Remove the container and volume (WARNING: deletes all test data)

## Cleaning Up

To stop the container:

```bash
./scripts/docker_test_env.sh stop
```

To remove the container and all test data:

```bash
./scripts/docker_test_env.sh clean
```