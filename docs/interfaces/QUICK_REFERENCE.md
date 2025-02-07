# Sortd Quick Command Reference

## Navigation Mode (Default)

### Movement
```
h       Move left/parent directory
j       Move down
k       Move up
l       Move right/enter directory
gg      Go to top
G       Go to bottom
ctrl+u  Page up
ctrl+d  Page down
```

### Selection
```
space   Toggle selection
v       Enter visual mode
V       Enter visual line mode
y       Yank (copy) selection
d       Delete selection
p       Paste
```

### Marks
```
m{a-z}  Set mark
'{a-z}  Go to mark
```

## Command Mode (Press ':')

### File Operations
```
:o [pattern] [dest]    Organize files (optional pattern and destination)
:w [dir]              Watch directory for changes
:m [dest]             Move selected files to destination
:find [pattern]       Find files matching pattern
```

### Organization
```
:r add [rule]         Add new rule
:r list               List all rules
:r edit [name]        Edit rule
:r remove [name]      Remove rule

:p add [pattern]      Add new pattern
:p list               List patterns
```

### Project Management
```
:project add [path]   Add project
:project detect       Detect project type
:project list         List known projects
```

### Configuration
```
:set [option]=[value] Set configuration option
:set list            List all options
```

### Help
```
:help                Show general help
:help [command]      Show help for specific command
```

## Visual Mode

### Selection
```
h,j,k,l  Extend selection
esc      Exit visual mode
y        Yank selection
d        Delete selection
```

## Common Workflows

### Quick Organize
```
1. Navigate to directory:    cd [path]
2. Select files:            space or v
3. Organize:               :o [destination]
```

### Watch Directory
```
1. Navigate to directory:    cd [path]
2. Start watching:          :w
3. Set rules:              :r add "rule description"
```

### Project Setup
```
1. Detect project:          :project detect
2. Add custom rules:        :r add "project specific rule"
3. Start organizing:        :o
```

## Tips

- Use tab completion in command mode
- Press ? for context-sensitive help
- Use marks to quickly jump between locations
- Command history available with up/down arrows in command mode
