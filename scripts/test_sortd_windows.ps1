# Sortd Test Script for Windows
# This script creates a sandboxed environment to test sortd functionality

# Configuration
$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir
$testRoot = Join-Path $projectRoot "test_sandbox"
$mockFS = Join-Path $testRoot "mock_filesystem"
$configDir = Join-Path $testRoot "sortd-config"
$workflowsDir = Join-Path $configDir "workflows"
$logDir = Join-Path $testRoot "logs"
$logFile = Join-Path $logDir "test_$(Get-Date -Format 'yyyyMMdd_HHmmss').log"
$sortdBinary = Join-Path $projectRoot "sortd.exe"

# Color definitions
function Write-ColorOutput {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Message,

        [Parameter(Mandatory=$false)]
        [string]$ForegroundColor = "White"
    )

    $originalColor = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    Write-Output $Message
    $host.UI.RawUI.ForegroundColor = $originalColor
}

# Print banner
function Print-Banner {
    Write-ColorOutput "╔═════════════════════════════════════════════════════════╗" -ForegroundColor Magenta
    Write-ColorOutput "║                                                         ║" -ForegroundColor Magenta
    Write-ColorOutput "║   SORTD TEST SUITE - Let Chaos Sort Itself Out! 🗂️      ║" -ForegroundColor Magenta
    Write-ColorOutput "║                                                         ║" -ForegroundColor Magenta
    Write-ColorOutput "╚═════════════════════════════════════════════════════════╝" -ForegroundColor Magenta
}

# Log function with timestamp and color
function Log {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Level,

        [Parameter(Mandatory=$true)]
        [string]$Message
    )

    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $symbol = ""
    $color = "White"

    switch ($Level) {
        "INFO" {
            $color = "Cyan"
            $symbol = "ℹ️"
        }
        "SUCCESS" {
            $color = "Green"
            $symbol = "✅"
        }
        "WARNING" {
            $color = "Yellow"
            $symbol = "⚠️"
        }
        "ERROR" {
            $color = "Red"
            $symbol = "❌"
        }
        "STEP" {
            $color = "Blue"
            $symbol = "🔷"
        }
    }

    $logMessage = "$symbol [$timestamp] [$Level] $Message"
    Write-ColorOutput $logMessage -ForegroundColor $color
    Add-Content -Path $logFile -Value $logMessage
}

# Function to run a test case
function Run-Test {
    param(
        [Parameter(Mandatory=$true)]
        [string]$TestName,

        [Parameter(Mandatory=$true)]
        [string]$TestCommand
    )

    Log -Level "STEP" -Message "Running test: $TestName"
    Log -Level "INFO" -Message "Command: $TestCommand"

    Add-Content -Path $logFile -Value "---------------------------------------------"

    try {
        $output = Invoke-Expression $TestCommand
        Log -Level "SUCCESS" -Message "Test passed: $TestName"
        return $true
    } catch {
        Log -Level "ERROR" -Message "Test failed: $TestName (Error: $($_.Exception.Message))"
        return $false
    }
}

# Function to create test files
function Create-TestFiles {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Directory
    )

    # Create directories
    $dirs = @(
        "$Directory\Downloads\Documents",
        "$Directory\Downloads\Images",
        "$Directory\Downloads\Music",
        "$Directory\Downloads\Videos",
        "$Directory\Downloads\Archives"
    )

    foreach ($dir in $dirs) {
        New-Item -Path $dir -ItemType Directory -Force | Out-Null
    }

    Log -Level "INFO" -Message "Creating test document files..."
    # Create document files
    1..5 | ForEach-Object {
        $i = $_
        "This is test document $i" | Out-File -FilePath "$Directory\Downloads\test_doc_$i.txt"
        New-Item -Path "$Directory\Downloads\test_doc_$i.pdf" -ItemType File -Force | Out-Null
        New-Item -Path "$Directory\Downloads\invoice_$i.pdf" -ItemType File -Force | Out-Null
    }

    Log -Level "INFO" -Message "Creating test image files..."
    # Create image files
    1..5 | ForEach-Object {
        $i = $_
        New-Item -Path "$Directory\Downloads\test_image_$i.jpg" -ItemType File -Force | Out-Null
        New-Item -Path "$Directory\Downloads\test_image_$i.png" -ItemType File -Force | Out-Null
        New-Item -Path "$Directory\Downloads\vacation_pic_$i.jpg" -ItemType File -Force | Out-Null
    }

    Log -Level "INFO" -Message "Creating test audio files..."
    # Create audio files
    1..3 | ForEach-Object {
        $i = $_
        New-Item -Path "$Directory\Downloads\test_song_$i.mp3" -ItemType File -Force | Out-Null
        New-Item -Path "$Directory\Downloads\test_audio_$i.wav" -ItemType File -Force | Out-Null
    }

    Log -Level "INFO" -Message "Creating test video files..."
    # Create video files
    1..3 | ForEach-Object {
        $i = $_
        New-Item -Path "$Directory\Downloads\test_video_$i.mp4" -ItemType File -Force | Out-Null
        New-Item -Path "$Directory\Downloads\movie_$i.mkv" -ItemType File -Force | Out-Null
    }

    Log -Level "INFO" -Message "Creating test archive files..."
    # Create archive files
    1..3 | ForEach-Object {
        $i = $_
        New-Item -Path "$Directory\Downloads\test_archive_$i.zip" -ItemType File -Force | Out-Null
        New-Item -Path "$Directory\Downloads\backup_$i.tar.gz" -ItemType File -Force | Out-Null
    }

    # Create a large file
    Log -Level "INFO" -Message "Creating a large test file..."
    $buffer = New-Object byte[] (5MB)
    $rng = New-Object System.Security.Cryptography.RNGCryptoServiceProvider
    $rng.GetBytes($buffer)
    [System.IO.File]::WriteAllBytes("$Directory\Downloads\large_file.bin", $buffer)

    # Create a readonly file
    Log -Level "INFO" -Message "Creating a readonly file..."
    "This file is readonly" | Out-File -FilePath "$Directory\Downloads\readonly_file.txt"
    Set-ItemProperty -Path "$Directory\Downloads\readonly_file.txt" -Name IsReadOnly -Value $true

    Log -Level "SUCCESS" -Message "Test files created successfully"
}

# Function to set up configuration
function Setup-Config {
    param(
        [Parameter(Mandatory=$true)]
        [string]$ConfigDir,

        [Parameter(Mandatory=$true)]
        [string]$WorkflowsDir
    )

    # Create config directory
    New-Item -Path $WorkflowsDir -ItemType Directory -Force | Out-Null

    # Create main config.yaml
    $configYaml = @"
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
  - "$($mockFS -replace '\\', '/')/Downloads"
"@
    Set-Content -Path "$ConfigDir\config.yaml" -Value $configYaml

    # Create document processor workflow
    $docProcessorYaml = @"
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
    target: "$($mockFS -replace '\\', '/')/Downloads/Documents/Invoices"
    options:
      createTargetDir: "true"
"@
    Set-Content -Path "$WorkflowsDir\document_processor.yaml" -Value $docProcessorYaml

    # Create image sorter workflow
    $imageSorterYaml = @"
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
    target: "$($mockFS -replace '\\', '/')/Downloads/Images/Vacation"
    options:
      createTargetDir: "true"
"@
    Set-Content -Path "$WorkflowsDir\image_sorter.yaml" -Value $imageSorterYaml

    Log -Level "SUCCESS" -Message "Configuration files created successfully"
}

# Main function
function Main {
    Print-Banner

    # Check if sortd exists
    if (-not (Test-Path $sortdBinary)) {
        Log -Level "ERROR" -Message "sortd binary not found at $sortdBinary"
        Log -Level "INFO" -Message "Please build sortd first with: go build -o sortd.exe .\cmd\sortd\"
        exit 1
    }

    # Create directories
    Log -Level "STEP" -Message "Creating test directories"
    New-Item -Path $testRoot, $mockFS, $configDir, $logDir -ItemType Directory -Force | Out-Null

    # Create test files
    Log -Level "STEP" -Message "Setting up mock file system"
    Create-TestFiles -Directory $mockFS

    # Create configuration
    Log -Level "STEP" -Message "Setting up configuration"
    Setup-Config -ConfigDir $configDir -WorkflowsDir $workflowsDir

    # Create snapshot for comparison
    Log -Level "STEP" -Message "Creating pre-test snapshot"
    $snapshotBefore = Join-Path $testRoot "snapshot_before"
    New-Item -Path $snapshotBefore -ItemType Directory -Force | Out-Null
    Get-ChildItem -Path "$mockFS\Downloads" -File -Recurse | ForEach-Object {
        Copy-Item -Path $_.FullName -Destination $snapshotBefore -Force
    }

    # Test cases
    Log -Level "STEP" -Message "Running test cases"

    # Test Case 1: Basic sorting with patterns
    $testCmd1 = "& '$sortdBinary' organize --config='$configDir' --dir='$mockFS\Downloads' --non-interactive"
    Run-Test -TestName "Basic sorting with patterns" -TestCommand $testCmd1

    # Test Case 2: Document processor workflow
    $testCmd2 = "& '$sortdBinary' workflow run --config='$configDir' --id=document-processor --non-interactive"
    Run-Test -TestName "Document processor workflow" -TestCommand $testCmd2

    # Test Case 3: Image sorter workflow
    $testCmd3 = "& '$sortdBinary' workflow run --config='$configDir' --id=image-sorter --non-interactive"
    Run-Test -TestName "Image sorter workflow" -TestCommand $testCmd3

    # Test Case 4: Error handling test
    $testCmd4 = "try { & '$sortdBinary' --config='C:\path\that\does\not\exist' } catch { `$_.Exception.Message | Select-String -Pattern 'error|fail|invalid' }"
    Run-Test -TestName "Error handling test" -TestCommand $testCmd4

    # Create snapshot after tests
    Log -Level "STEP" -Message "Creating post-test snapshot"
    $snapshotAfter = Join-Path $testRoot "snapshot_after"
    New-Item -Path $snapshotAfter -ItemType Directory -Force | Out-Null
    Get-ChildItem -Path $mockFS -File -Recurse -Exclude "snapshot_*" | ForEach-Object {
        Copy-Item -Path $_.FullName -Destination $snapshotAfter -Force
    }

    # Analyze changes
    Log -Level "STEP" -Message "Analyzing results"
    $diffFile = Join-Path $logDir "file_diff.txt"

    $beforeFiles = Get-ChildItem -Path $snapshotBefore -File | ForEach-Object { $_.Name }
    $afterFiles = Get-ChildItem -Path $snapshotAfter -File | ForEach-Object { $_.Name }

    $differences = Compare-Object -ReferenceObject $beforeFiles -DifferenceObject $afterFiles

    if ($differences) {
        Log -Level "INFO" -Message "Changes detected in the file system:"
        $differences | ForEach-Object {
            $indicator = if ($_.SideIndicator -eq "<=") { "Removed" } else { "Added" }
            "$indicator: $($_.InputObject)" | Tee-Object -FilePath $diffFile -Append
        }
    } else {
        Log -Level "WARNING" -Message "No changes detected in the file system"
    }

    # Show final directory structure
    Log -Level "STEP" -Message "Final directory structure"
    Get-ChildItem -Path $mockFS -Directory -Recurse | Select-Object FullName | ForEach-Object {
        $_.FullName | Add-Content -Path $logFile
    }

    # Test summary
    Log -Level "STEP" -Message "Test summary"
    Write-ColorOutput "Test Summary" -ForegroundColor Cyan
    "=============================================" | Add-Content -Path $logFile
    "Environment: $mockFS" | Add-Content -Path $logFile
    "Configuration: $configDir" | Add-Content -Path $logFile
    "Log file: $logFile" | Add-Content -Path $logFile

    Log -Level "SUCCESS" -Message "End-to-end tests completed successfully"
}

# Run the main function
Main