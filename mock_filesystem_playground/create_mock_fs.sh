#!/bin/bash

# Script to create a mock file system for testing directory organization tools
# Creates both Linux-style and Windows-style mock file systems

set -e

BASE_DIR="$HOME/mock_filesystem_playground"
LINUX_DIR="$BASE_DIR/linux"
WINDOWS_DIR="$BASE_DIR/windows"

# Make sure directories exist
mkdir -p "$LINUX_DIR"
mkdir -p "$WINDOWS_DIR"

echo "🚀 Creating mock filesystem playgrounds for testing..."

# Function to create common directories
create_common_dirs() {
    local base_dir=$1
    
    # Create typical directories
    mkdir -p "$base_dir/Documents/work"
    mkdir -p "$base_dir/Documents/personal"
    mkdir -p "$base_dir/Documents/receipts"
    mkdir -p "$base_dir/Pictures/vacation"
    mkdir -p "$base_dir/Pictures/screenshots"
    mkdir -p "$base_dir/Pictures/memes"
    mkdir -p "$base_dir/Music/playlists"
    mkdir -p "$base_dir/Music/albums"
    mkdir -p "$base_dir/Videos/movies"
    mkdir -p "$base_dir/Videos/tutorials"
    mkdir -p "$base_dir/Downloads"
    mkdir -p "$base_dir/Desktop"

    # Create a messy directory structure to simulate real-world chaos
    mkdir -p "$base_dir/projects/old"
    mkdir -p "$base_dir/projects/archive"
    mkdir -p "$base_dir/projects/incomplete"
    mkdir -p "$base_dir/old_stuff"
    mkdir -p "$base_dir/stuff_to_sort"
    mkdir -p "$base_dir/temp"
    mkdir -p "$base_dir/backup"
    mkdir -p "$base_dir/misc"
}

# Function to create random content for a file
create_file_content() {
    local filename=$1
    local size_kb=$2
    
    dd if=/dev/urandom of="$filename" bs=1024 count=$size_kb 2>/dev/null
}

# Function to create various file types in Downloads folder (chaos)
create_messy_downloads() {
    local downloads_dir=$1
    local is_windows=$2
    
    cd "$downloads_dir"
    
    # Create document files
    for i in {1..5}; do
        create_file_content "report_$i.docx" 5
        create_file_content "presentation_$i.pptx" 10
        create_file_content "spreadsheet_$i.xlsx" 3
        create_file_content "document_$i.pdf" 15
        create_file_content "notes_$i.txt" 1
        create_file_content "specification_$i.md" 2
    done
    
    # Create image files
    for i in {1..10}; do
        create_file_content "photo_$i.jpg" 20
        create_file_content "screenshot_$i.png" 15
        create_file_content "image_$i.gif" 5
        create_file_content "graphic_$i.svg" 3
        create_file_content "design_$i.webp" 8
    done
    
    # Create archive files
    for i in {1..3}; do
        create_file_content "backup_$i.zip" 50
        create_file_content "archive_$i.tar.gz" 30
        create_file_content "files_$i.rar" 25
        create_file_content "old_project_$i.7z" 40
    done
    
    # Create code files
    for i in {1..5}; do
        create_file_content "script_$i.py" 2
        create_file_content "program_$i.js" 3
        create_file_content "app_$i.html" 4
        create_file_content "styles_$i.css" 2
        create_file_content "server_$i.go" 3
        create_file_content "code_$i.c" 2
        create_file_content "library_$i.cpp" 4
        create_file_content "module_$i.ts" 3
        create_file_content "component_$i.tsx" 5
    done
    
    # Create config files
    create_file_content ".config" 1
    create_file_content ".env" 1
    create_file_content "settings.json" 2
    create_file_content "config.yaml" 2
    create_file_content ".gitignore" 1
    
    # Create media files
    for i in {1..3}; do
        create_file_content "song_$i.mp3" 30
        create_file_content "video_$i.mp4" 100
        create_file_content "podcast_$i.wav" 50
        create_file_content "movie_trailer_$i.mkv" 75
    done
    
    # Create random files with duplicate content but different names
    create_file_content "final.docx" 5
    create_file_content "final_v2.docx" 5
    create_file_content "final_FINAL.docx" 5
    create_file_content "final_FINAL_REALLY.docx" 5
    
    # Add some files with spaces and special characters
    create_file_content "my important document.pdf" 10
    create_file_content "tax return (2024).pdf" 12
    create_file_content "vacation pics - summer 2024.zip" 45
    create_file_content "meeting notes - project X.txt" 2
    
    # Create some Windows-specific files if this is the Windows mock
    if [ "$is_windows" = true ]; then
        create_file_content "Untitled.exe" 10
        create_file_content "setup.msi" 30
        create_file_content "program.dll" 15
        create_file_content "driver.sys" 5
        create_file_content "installer.bat" 1
        create_file_content "Desktop.ini" 1
    else
        # Create some Linux-specific files
        create_file_content "install.sh" 1
        create_file_content "backup.AppImage" 40
        create_file_content "program.deb" 35
        create_file_content "app.bin" 12
        create_file_content ".bash_profile" 1
    fi
    
    # Create files with no extensions
    create_file_content "README" 2
    create_file_content "INSTALL" 1
    create_file_content "data" 5
    create_file_content "backup_data" 10
    
    # Create some hidden files and folders
    mkdir -p .cache
    create_file_content ".hidden_file" 1
    create_file_content ".secret" 1
    
    # Add some "temp" files that should be cleaned up
    create_file_content "temp1234.tmp" 2
    create_file_content "download.part" 10
    create_file_content "~$report.docx" 1
    
    # Files with unusual extensions
    create_file_content "database.sqlite" 25
    create_file_content "data.json" 5
    create_file_content "coordinates.geojson" 10
    create_file_content "diagram.drawio" 3
    create_file_content "book.epub" 20
    create_file_content "firmware.bin" 15
    
    echo "Created messy Downloads directory at: $downloads_dir"
}

# Function to create a desktop mess
create_messy_desktop() {
    local desktop_dir=$1
    
    cd "$desktop_dir"
    
    # Create a typical "messy desktop" with various shortcuts and files
    create_file_content "Untitled Document.docx" 5
    create_file_content "Screenshot_20250410_123045.png" 15
    create_file_content "IMG_20250321.jpg" 20
    create_file_content "Notes.txt" 1
    create_file_content "To-Do List.txt" 1
    create_file_content "Important!.pdf" 10
    create_file_content "Resume.pdf" 8
    create_file_content "Budget_2025.xlsx" 3
    create_file_content "Project Plan.pptx" 12
    
    # Create shortcut files
    echo "[InternetShortcut]" > "Google.url"
    echo "URL=https://www.google.com" >> "Google.url"
    
    echo "[InternetShortcut]" > "YouTube.url"
    echo "URL=https://www.youtube.com" >> "YouTube.url"
    
    # Create some folders on desktop too
    mkdir -p "New Folder"
    mkdir -p "Stuff"
    mkdir -p "Work Files"
    
    echo "Created messy Desktop at: $desktop_dir"
}

# Create Linux-style home directory
echo "Setting up Linux mock filesystem..."
create_common_dirs "$LINUX_DIR"
create_messy_downloads "$LINUX_DIR/Downloads" false
create_messy_desktop "$LINUX_DIR/Desktop"

# Create similar structure for Windows mock
echo "Setting up Windows mock filesystem..."
create_common_dirs "$WINDOWS_DIR"
create_messy_downloads "$WINDOWS_DIR/Downloads" true
create_messy_desktop "$WINDOWS_DIR/Desktop"

# Add some specific Windows-style directories and files
mkdir -p "$WINDOWS_DIR/Program Files/Common Files"
mkdir -p "$WINDOWS_DIR/Program Files (x86)"
mkdir -p "$WINDOWS_DIR/Windows"
mkdir -p "$WINDOWS_DIR/Users/Default"

# Add Linux-specific directories
mkdir -p "$LINUX_DIR/.config"
mkdir -p "$LINUX_DIR/.local/share"
mkdir -p "$LINUX_DIR/.cache"
mkdir -p "$LINUX_DIR/bin"

echo "✅ Mock filesystem playgrounds created successfully!"
echo "Linux mock home: $LINUX_DIR"
echo "Windows mock home: $WINDOWS_DIR"
echo ""
echo "These environments are designed to test directory organization tools."
echo "They contain a variety of file types in an intentionally disorganized structure."
