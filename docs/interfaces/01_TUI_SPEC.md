# Terminal User Interface Specification

## Overview
The TUI is built using Bubble Tea and Lipgloss, providing a modern, responsive interface that scales from simple to advanced usage patterns.

## Core Components

### 1. Base Styles (Lipgloss)
```go
var Styles = struct {
    App         lipgloss.Style
    Title       lipgloss.Style
    Subtle      lipgloss.Style
    Interactive lipgloss.Style
    Success     lipgloss.Style
    Error       lipgloss.Style
}{
    App: lipgloss.NewStyle().
        Padding(1, 2),
    
    Title: lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#7B61FF")).
        MarginBottom(1),
    
    Subtle: lipgloss.NewStyle().
        Foreground(lipgloss.Color("#666666")),
    
    Interactive: lipgloss.NewStyle().
        Foreground(lipgloss.Color("#5A9")).
        Bold(true),
    
    Success: lipgloss.NewStyle().
        Foreground(lipgloss.Color("#73F59F")),
    
    Error: lipgloss.NewStyle().
        Foreground(lipgloss.Color("#F14E32")),
}
```

### 2. Main Application Model
```go
type Model struct {
    // Core state
    state        State
    activeView   View
    
    // Components
    spinner      spinner.Model
    input        textinput.Model
    viewport     viewport.Model
    help         help.Model
    
    // Data
    files        []FileInfo
    progress     Progress
    
    // Flags
    ready        bool
    quitting     bool
}

func (m Model) Init() tea.Cmd {
    return tea.Batch(
        spinner.Tick,
        m.checkInitialState,
    )
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            m.quitting = true
            return m, tea.Quit
        }
    }
    return m, nil
}

func (m Model) View() string {
    if m.quitting {
        return "Thanks for using Sortd!\n"
    }
    
    return lipgloss.JoinVertical(
        lipgloss.Left,
        m.headerView(),
        m.contentView(),
        m.footerView(),
    )
}
```

### 3. Views

#### Dashboard View
```go
type DashboardView struct {
    stats    Stats
    activity []Activity
    help     help.Model
}

func (v DashboardView) View() string {
    return lipgloss.JoinVertical(
        lipgloss.Left,
        Styles.Title.Render("Sortd Dashboard"),
        v.statsSection(),
        v.activitySection(),
        v.helpSection(),
    )
}
```

#### Organization View
```go
type OrganizeView struct {
    files    []FileInfo
    selected map[string]bool
    cursor   int
    viewport viewport.Model
}

func (v OrganizeView) View() string {
    var s strings.Builder
    
    // Render file list with selection
    for i, file := range v.files {
        style := Styles.Subtle
        if i == v.cursor {
            style = Styles.Interactive
        }
        if v.selected[file.Path] {
            style = style.Copy().Bold(true)
        }
        
        s.WriteString(style.Render(file.DisplayName()))
        s.WriteRune('\n')
    }
    
    return lipgloss.JoinVertical(
        lipgloss.Left,
        Styles.Title.Render("Organize Files"),
        s.String(),
        v.helpSection(),
    )
}
```

### 4. Interactive Components

#### Command Palette
```go
type CommandPalette struct {
    input    textinput.Model
    commands map[string]Command
    matches  []string
    visible  bool
}

func (c *CommandPalette) Update(msg tea.Msg) tea.Cmd {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "esc":
            c.visible = false
            return nil
        case "enter":
            return c.executeCommand
        }
    }
    
    var cmd tea.Cmd
    c.input, cmd = c.input.Update(msg)
    return cmd
}
```

#### Progress Indicator
```go
type Progress struct {
    spinner    spinner.Model
    bar        progress.Model
    status     string
    percentage float64
}

func (p Progress) View() string {
    return lipgloss.JoinHorizontal(
        lipgloss.Left,
        p.spinner.View(),
        p.bar.ViewAs(p.percentage),
        Styles.Subtle.Render(p.status),
    )
}
```

### 5. Keyboard Shortcuts
```go
type KeyMap struct {
    Up        key.Binding
    Down      key.Binding
    Select    key.Binding
    Quit      key.Binding
    Help      key.Binding
    Command   key.Binding
}

var DefaultKeyMap = KeyMap{
    Up: key.NewBinding(
        key.WithKeys("up", "k"),
        key.WithHelp("↑/k", "move up"),
    ),
    Down: key.NewBinding(
        key.WithKeys("down", "j"),
        key.WithHelp("↓/j", "move down"),
    ),
    Select: key.NewBinding(
        key.WithKeys("enter", "space"),
        key.WithHelp("enter", "select"),
    ),
    Quit: key.NewBinding(
        key.WithKeys("q", "ctrl+c"),
        key.WithHelp("q", "quit"),
    ),
    Help: key.NewBinding(
        key.WithKeys("?"),
        key.WithHelp("?", "toggle help"),
    ),
    Command: key.NewBinding(
        key.WithKeys(":"),
        key.WithHelp(":", "command mode"),
    ),
}
```

## Usage Examples

### 1. Basic Application Setup
```go
func main() {
    p := tea.NewProgram(
        NewModel(),
        tea.WithAltScreen(),
        tea.WithMouseCellMotion(),
    )
    
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error running program: %v", err)
        os.Exit(1)
    }
}
```

### 2. Interactive File Selection
```go
func (m Model) handleFileSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            m.cursor--
        case "down", "j":
            m.cursor++
        case "space":
            m.toggleSelection()
        case "enter":
            return m, m.processSelected
        }
    }
    
    // Ensure cursor stays in bounds
    m.cursor = clamp(m.cursor, 0, len(m.files)-1)
    
    return m, nil
}
```

### 3. Progress Updates
```go
func (m Model) updateProgress(msg ProgressMsg) (tea.Model, tea.Cmd) {
    m.progress.percentage = msg.Percentage
    m.progress.status = msg.Status
    
    if msg.Done {
        return m, m.showResults
    }
    
    return m, nil
}
```

This TUI specification provides:
- Clean, modern interface using Bubble Tea and Lipgloss
- Responsive and interactive components
- Support for both mouse and keyboard input
- Progressive disclosure of features
- Clear feedback and progress indication
- Power user features (vim-like bindings, command palette)

Would you like me to continue with more specifications or expand on any particular aspect?
