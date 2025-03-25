# Sortd

Sortd is a Go-based project to build a context aware file sorting and organizing tool.

## Installation

1. Clone the repository:
   ```sh
   git clone https://github.com/Jeff-Barlow-Spady/sortd.git
   ```
2. Navigate to the project directory:
   ```sh
   cd sortd
   ```
3. Build the project:
   ```sh
   go build
   ```

## Usage

To use Sortd, run the executable generated after building the project:
```sh
./sortd [options]
```

### Key Features

- **Enhanced Rule Management**: Create and manage custom organization rules
- **Rule Templates**: Use predefined templates for common file organization patterns
- **Interactive Wizard**: Configure your setup through an easy-to-use wizard
- **Watch Mode**: Automatically organize files as they arrive in watched directories
- **Confirmation Phase**: Optionally review and confirm file movements before execution
- **Directory-Specific Rules**: Create different rule sets for different directories
- **Smart File Categorization**: Intelligent categorization of files by type

### Available Commands

- `sortd setup` - Run the interactive setup wizard
- `sortd organize` - Organize files according to rules
- `sortd watch` - Watch directories for new files and organize automatically
- `sortd rules` - Manage organization rules
- `sortd confirm` - Confirm pending file operations in watch mode
- `sortd theme` - Manage the application's visual theme
- `sortd scan` - Scan directories without organizing
- `sortd analyze` - Analyze files and suggest organization rules

### New Features

#### Rule Templates
Quickly create common rule sets using predefined templates:
```sh
sortd rules add
# Then select "Use rule template" from the menu
```

#### Watch Confirmation Phase
Enable confirmation before executing file operations:
```sh
sortd watch --require-confirmation
# or
sortd watch --confirmation-period 60
```

Then confirm pending operations:
```sh
sortd confirm
```

#### Directory-Specific Rules
Create rules that only apply to specific directories:
```sh
sortd rules add
# Then select "Create directory-specific rule" from the menu
```

#### Improved File Type Categorization
Create rules for entire categories of files:
```sh
sortd rules add
# Then select "Create file type rule" from the menu
```

## Contributing

Contributions are welcome! Please fork the repository and create a pull request.

## License

This project is licensed under the MIT License. See the LICENSE file for details.

## Contact

For more information, please contact Jeff Barlow-Spady.

---
