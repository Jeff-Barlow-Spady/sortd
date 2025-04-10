# Workflows in Sortd

Workflows are a powerful way to automate file organization in Sortd. Each workflow consists of a trigger, optional conditions, and actions that are executed when the trigger fires and conditions are met.

## Workflow Components

### Triggers

Triggers define when a workflow should be executed:

- **File Created**: Triggered when a file is created in a watched directory
- **File Modified**: Triggered when a file is modified in a watched directory
- **File Pattern Match**: Triggered when a file matching a pattern is created or modified
- **Manual**: Triggered only when explicitly executed through the CLI or GUI
- **Scheduled**: Triggered based on a schedule (cron format)

### Conditions

Conditions determine if the workflow actions should be executed. A workflow can have multiple conditions, and all must be satisfied for the actions to run:

- **File Size**: Check if the file size meets a criterion (e.g., greater than 1MB)
- **File Type**: Check the file extension or content type
- **File Name**: Check the file name using various operators (contains, starts with, etc.)
- **File Age**: Check how old the file is

### Actions

Actions are executed when the trigger fires and all conditions are met:

- **Move**: Move the file to a target location
- **Copy**: Copy the file to a target location
- **Rename**: Change the file name
- **Tag**: Add a tag to the file
- **Delete**: Remove the file
- **Command**: Execute a custom command with the file

## Creating Workflows

### Using the GUI

1. Open the Sortd application
2. Navigate to the "Organize" tab
3. Click on "Create New Workflow" in the Advanced Workflows section
4. Follow the step-by-step wizard:
   - Step 1: Enter basic workflow information
   - Step 2: Set up the trigger
   - Step 3: Define conditions (optional)
   - Step 4: Add actions
   - Step 5: Review and save

### Using the CLI

The CLI offers several commands for managing workflows:

#### Creating a Workflow with the Wizard

```bash
sortd workflow wizard
```

This starts an interactive wizard that guides you through the workflow creation process.

#### Creating a Workflow from a Template File

```bash
sortd workflow create my-workflow.yaml
```

This creates a workflow from a YAML or JSON file.

#### Workflow File Format

```yaml
id: "document-processor"
name: "Document Processor"
description: "Process documents based on content and size"
enabled: true
priority: 5

trigger:
  type: "FileCreated"
  pattern: "*.{doc,docx,pdf,txt}"

conditions:
  - type: "FileCondition"
    field: "size"
    operator: "GreaterThan"
    value: "1"
    valueUnit: "MB"
  - type: "FileCondition"
    field: "name"
    operator: "Contains"
    value: "invoice"
    caseSensitive: false

actions:
  - type: "CopyAction"
    target: "/home/user/Documents/Invoices"
    options:
      createTargetDir: "true"

  - type: "TagAction"
    target: "invoice"
    options:
      addToMetadata: "true"
```

## Managing Workflows

### Listing Workflows

```bash
sortd workflow list
```

This displays all configured workflows with their basic information.

### Testing Workflows (Dry Run)

You can test a workflow without making any actual changes using the dry run mode:

```bash
sortd workflow test workflow-id /path/to/file.txt
```

This simulates running the workflow on the specified file.

### Running Workflows

To execute a workflow on a specific file:

```bash
sortd workflow run workflow-id /path/to/file.txt
```

### Deleting Workflows

To delete a workflow:

```bash
sortd workflow delete workflow-id
```

## Best Practices

1. **Start with Dry Runs**: Always test workflows with dry runs before applying them to actual files.

2. **Build Incrementally**: Start with simple workflows and gradually add more complexity.

3. **Use Descriptive IDs and Names**: Give your workflows clear, descriptive names and IDs to help you remember their purpose.

4. **Mind Your Conditions**: Be careful with conditions - too restrictive and the workflow won't execute; too loose and it might execute on unintended files.

5. **Order Matters**: Workflows are executed in order of priority. Higher priority workflows run first.

## Troubleshooting

### Workflow Not Triggering

- Check if the trigger pattern matches your files
- Verify the watched directories include where your files are being created/modified
- Ensure the workflow is enabled

### Actions Not Executing

- Check if all conditions are being met
- Review the action configuration for errors
- Try running the workflow manually with the test command

### Conflicts Between Workflows

- If multiple workflows might apply to the same file, use priorities to control execution order
- Design workflows to be complementary rather than conflicting

## Examples

### Image Sorter Workflow

```yaml
id: "image-sorter"
name: "Image File Sorter"
description: "Automatically sorts image files into folders based on type"
enabled: true
priority: 10

trigger:
  type: "FileCreated"
  pattern: "*.{jpg,jpeg,png,gif,svg,webp}"

conditions:
  - type: "FileSizeCondition"
    field: "size"
    operator: "LessThan"
    value: "10"
    valueUnit: "MB"

actions:
  - type: "MoveAction"
    target: "/home/user/Pictures/Sorted"
    options:
      overwrite: "false"
      createTargetDir: "true"

  - type: "TagAction"
    target: "image"
    options:
      addToMetadata: "true"
```

### Invoice Processor Workflow

```yaml
id: "invoice-processor"
name: "Invoice Processor"
description: "Move invoices to a specific folder and notify"
enabled: true
priority: 5

trigger:
  type: "FilePatternMatch"
  pattern: "*.pdf"

conditions:
  - type: "FileNameCondition"
    field: "name"
    operator: "Contains"
    value: "invoice"
    caseSensitive: false

actions:
  - type: "MoveAction"
    target: "/home/user/Documents/Invoices"
    options:
      createTargetDir: "true"

  - type: "CommandAction"
    command: "python3 /home/user/scripts/notify.py --file \"{{ .FilePath }}\" --type invoice"
    options:
      runAsynchronously: true
      shell: "/bin/bash"
```