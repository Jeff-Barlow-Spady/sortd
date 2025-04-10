#!/bin/bash

# Build the application
echo -e "\e[1;34mBuilding Sortd...\e[0m"
go build -o sortd -v cmd/sortd/main.go

if [ $? -ne 0 ]; then
    echo -e "\e[1;31mBuild failed!\e[0m"
    exit 1
fi

echo -e "\e[1;32mBuild successful!\e[0m"

# Check command line arguments
if [ $# -eq 0 ]; then
    echo -e "\e[1;33mAvailable commands:\e[0m"
    echo "  ./run.sh tui       - Run the TUI interface"
    echo "  ./run.sh gui       - Run the GUI interface"
    echo "  ./run.sh analyze   - Analyze the current directory"
    echo "  ./run.sh organize  - Organize files in the current directory"
    echo "  ./run.sh watch     - Watch the current directory for changes"
    echo "  ./run.sh help      - Show help"
    exit 0
fi

# Run the requested command
echo -e "\e[1;34mRunning: ./sortd $@\e[0m"
./sortd "$@"