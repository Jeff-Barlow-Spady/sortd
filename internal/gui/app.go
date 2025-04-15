package gui

import (
	"image/color"
	"os"
	"path/filepath"

	"sortd/internal/config"
	"sortd/internal/log"
	"sortd/internal/organize"
	"sortd/internal/watch"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// App is the GUI application
type App struct {
	fyneApp        fyne.App
	mainWindow     fyne.Window
	cfg            *config.Config
	organizeEngine *organize.Engine
	watchDaemon    *watch.Daemon
	pathLabel      *widget.Label // Reference to the path display label
	statusUpdater  func()        // Function to update system tray status

	// Track selected items in lists
	selectedPatternIndex  int // Index of the selected pattern in the organize tab list
	selectedWatchDirIndex int // Index of the selected watch directory in the settings tab list

	// Theme settings
	accentColor color.NRGBA
	bgColor     color.NRGBA

	// Active workflow wizard (if any)
	activeWizard *WorkflowWizard
}

// NewApp creates a new GUI application
func NewApp(cfg *config.Config, organizeEngine *organize.Engine) *App {
	// Create app with a unique ID for preferences storage
	fyneApp := app.NewWithID("io.github.sortd")

	// Load the app icon
	iconPath := "s.png"
	if _, err := os.Stat(iconPath); os.IsNotExist(err) {
		altPath := filepath.Join("internal", "gui", "s.png")
		if _, errStat := os.Stat(altPath); errStat == nil {
			iconPath = altPath
		}
	}

	if appIcon, err := fyne.LoadResourceFromPath(iconPath); err == nil {
		fyneApp.SetIcon(appIcon)
	} else {
		log.Warnf("Could not load app icon from %s: %v", iconPath, err)
	}

	// Create and start the watch daemon
	watchDaemon, err := watch.NewDaemon(cfg)
	if err != nil {
		log.Errorf("Failed to create watch daemon: %v", err)
		// Don't exit - GUI can still be used without daemon
		watchDaemon = nil
	}

	a := &App{
		fyneApp:               fyneApp,
		cfg:                   cfg,
		organizeEngine:        organizeEngine,
		watchDaemon:           watchDaemon,
		selectedPatternIndex:  -1, // Initialize to -1 (no selection)
		selectedWatchDirIndex: -1, // Initialize to -1 (no selection)
		accentColor:           color.NRGBA{R: 255, G: 165, B: 0, A: 255},
		bgColor:               color.NRGBA{R: 16, G: 16, B: 16, A: 255},
	}

	a.mainWindow = a.fyneApp.NewWindow("Sortd")

	if appIcon, err := fyne.LoadResourceFromPath(iconPath); err == nil {
		a.mainWindow.SetIcon(appIcon)
	} else {
		log.Warnf("Could not load window icon from %s: %v", iconPath, err)
	}

	a.setupSystemTray()

	return a
}

// GetMainWindow returns the main window instance
func (a *App) GetMainWindow() fyne.Window {
	return a.mainWindow
}

// IsDaemonRunning checks if the watch daemon is running
func (a *App) IsDaemonRunning() bool {
	if a.watchDaemon == nil {
		return false
	}
	return a.watchDaemon.Status().Running
}

// GetDaemonStatus returns the current daemon status
func (a *App) GetDaemonStatus() string {
	if a.watchDaemon == nil {
		return "Daemon not initialized"
	}
	status := a.watchDaemon.Status()
	if status.Running {
		return "Running"
	}
	return "Stopped"
}

// setupSystemTray sets up the system tray icon and menu
func (a *App) setupSystemTray() {
	if deskApp, ok := a.fyneApp.(desktop.App); ok {
		var items []*fyne.MenuItem
		var updateMenuFunc func() []*fyne.MenuItem // Declare ahead

		// Function to create/update the menu items
		updateMenuFunc = func() []*fyne.MenuItem {
			status := a.watchDaemon.Status()
			items := []*fyne.MenuItem{
				fyne.NewMenuItem("Show Sortd", func() {
					a.mainWindow.Show()
				}),
				fyne.NewMenuItemSeparator(),
			}
			if status.Running {
				items = append(items, fyne.NewMenuItem("Stop Watch Mode", func() {
					a.stopWatchMode()
					// Update the menu immediately after action
					deskApp.SetSystemTrayMenu(fyne.NewMenu("Sortd", updateMenuFunc()...))
				}))
			} else {
				items = append(items, fyne.NewMenuItem("Start Watch Mode", func() {
					a.startWatchMode()
					// Update the menu immediately after action
					deskApp.SetSystemTrayMenu(fyne.NewMenu("Sortd", updateMenuFunc()...))
				}))
			}
			items = append(items, fyne.NewMenuItemSeparator(), fyne.NewMenuItem("Exit", func() {
				a.stopWatchMode()
				a.fyneApp.Quit()
			}))
			return items
		}

		// Set the desktop tray menu
		items = updateMenuFunc()
		deskApp.SetSystemTrayMenu(fyne.NewMenu("Sortd", items...))

		// Store a reference to update status later if needed
		a.statusUpdater = func() {
			deskApp.SetSystemTrayMenu(fyne.NewMenu("Sortd", updateMenuFunc()...))
		}
	}
}

// Run starts the GUI application
func (a *App) Run() {
	a.setupMainWindow()

	a.mainWindow.Show()

	a.fyneApp.Run()
}

// setupMainWindow sets up the main window content
func (a *App) setupMainWindow() {
	background := canvas.NewRectangle(a.bgColor)
	background.Resize(fyne.NewSize(900, 700))

	logoText := `
  █████████     ███████    ███████████   ███████████ ██████████
 ███░░░░░███  ███░░░░░███ ░░███░░░░░███ ░█░░░███░░░█░░███░░░░███
░███    ░░░  ███     ░░███ ░███    ░███ ░   ░███  ░  ░███   ░░███
░░█████████ ░███      ░███ ░██████████      ░███     ░███    ░███
 ░░░░░░░░███░███      ░███ ░███░░░░░███     ░███     ░███    ░███
 ███    ░███░░███     ███  ░███    ░███     ░███     ░███    ███
░░█████████  ░░░███████░   █████   █████    █████    ██████████
 ░░░░░░░░░     ░░░░░░░    ░░░░░   ░░░░░    ░░░░░    ░░░░░░░░░░



`
	logoDisplay := canvas.NewText(logoText, a.accentColor)
	logoDisplay.TextStyle.Monospace = true
	logoDisplay.TextSize = 18
	logoDisplay.Alignment = fyne.TextAlignCenter

	// Set initial size before setting final content
	a.mainWindow.Resize(fyne.NewSize(900, 700))

	// Create the main toolbar
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.DocumentCreateIcon(), func() {
			NewWorkflowWizard(a).Show()
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			a.refreshContent()
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			dialog.ShowInformation("About Sortd",
				"Sortd is a file organization utility designed to help\n"+
					"automate the process of organizing files using rules\n"+
					"and workflows.",
				a.mainWindow)
		}),
	)

	// --- Tabs Setup ---
	tabs := container.NewAppTabs(
		container.NewTabItem("Organize", a.createOrganizeTab()),
		container.NewTabItem("Workflows", a.createWorkflowsTab()),
		container.NewTabItem("Cloud", a.createCloudTab()),
		container.NewTabItem("Settings", a.createSettingsTab()),
	)
	tabs.SetTabLocation(container.TabLocationTop) // Ensure tabs are at the top

	content := container.NewBorder(
		// Top content - logo, toolbar and a separation line
		container.NewVBox(
			logoDisplay,
			toolbar,
			canvas.NewLine(a.accentColor),
		),
		// Status bar at bottom
		a.createStatusBar(),
		nil,  // No left content
		nil,  // No right content
		tabs, // Center content is the tabs
	)

	a.mainWindow.SetContent(content)
}

// createStatusBar creates a status bar to display app status information
func (a *App) createStatusBar() fyne.CanvasObject {
	daemonStatus := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{})

	// Update status text based on daemon state
	updateStatusText := func() {
		if a.watchDaemon == nil {
			daemonStatus.SetText("Watch Daemon: Not initialized")
			return
		}

		status := a.watchDaemon.Status()
		if status.Running {
			daemonStatus.SetText("Watch Daemon: Running")
		} else {
			daemonStatus.SetText("Watch Daemon: Stopped")
		}
	}

	// Initial update
	updateStatusText()

	// Create refresh button
	refreshButton := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		updateStatusText()
	})

	return container.NewHBox(
		daemonStatus,
		layout.NewSpacer(),
		refreshButton,
	)
}

// refreshContent refreshes all dynamic content in the UI
func (a *App) refreshContent() {
	// Refresh the main window content
	a.mainWindow.Content().Refresh()

	// Display notification
	a.ShowNotification("Refresh Complete", "UI has been refreshed with latest data")
}

// ShowError displays an error dialog
func (a *App) ShowError(title string, err error) {
	if err == nil {
		return
	}
	dialog.ShowError(err, a.mainWindow)

	// Also send to system notification
	a.ShowNotification("Error: "+title, err.Error())
}

// ShowInfo displays an information dialog
func (a *App) ShowInfo(message string) {
	dialog.ShowInformation("Information", message, a.mainWindow)
}

// ShowNotification displays a system notification if available
func (a *App) ShowNotification(title, content string) {
	// Check if notifications are enabled in settings
	if a.cfg.Settings.EnableNotifications {
		// Using dialog instead of system notification since the notification package is not available
		go func() {
			dialog.ShowInformation(title, content, a.mainWindow)
		}()
	}
}

// startWatchMode starts the watch mode
func (a *App) startWatchMode() {
	// Daemon uses config from initialization. Set dynamic options if needed.
	a.watchDaemon.SetDryRun(a.cfg.Settings.DryRun)

	err := a.watchDaemon.Start()
	if err != nil {
		a.ShowError("Failed to start watch mode", err)
	} else {
		a.ShowInfo("Watch mode started.")
		a.ShowNotification("Watch Mode", "Watch mode has been started")
		if a.statusUpdater != nil {
			a.statusUpdater()
		}
	}
}

// stopWatchMode stops the watch mode
func (a *App) stopWatchMode() {
	if !a.watchDaemon.Status().Running {
		return
	}

	a.watchDaemon.Stop()
	a.ShowInfo("Watch mode stopped.")
	a.ShowNotification("Watch Mode", "Watch mode has been stopped")
	if a.statusUpdater != nil {
		a.statusUpdater()
	}
}

// saveConfig saves the current configuration
func (a *App) saveConfig() {
	err := a.cfg.Save()
	if err != nil {
		a.ShowError("Failed to save configuration", err)
	}
}

// createWorkflowsTab creates a new tab for managing workflows
func (a *App) createWorkflowsTab() fyne.CanvasObject {
	// Create a list to display existing workflows
	workflowList := widget.NewList(
		func() int {
			return len(a.cfg.Workflows)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.DocumentIcon()),
				widget.NewLabel("Template workflow name"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(a.cfg.Workflows) {
				return
			}

			workflow := a.cfg.Workflows[id]
			icon := obj.(*fyne.Container).Objects[0].(*widget.Icon)
			label := obj.(*fyne.Container).Objects[1].(*widget.Label)

			// Set different icon based on enabled state
			if workflow.Enabled {
				icon.SetResource(theme.DocumentIcon())
			} else {
				icon.SetResource(theme.DocumentCreateIcon())
			}

			label.SetText(workflow.Name)
		},
	)

	// Track selected index
	var selectedWorkflowIndex = -1
	workflowList.OnSelected = func(id widget.ListItemID) {
		selectedWorkflowIndex = int(id)
	}
	workflowList.OnUnselected = func(id widget.ListItemID) {
		if selectedWorkflowIndex == int(id) {
			selectedWorkflowIndex = -1
		}
	}

	// Create button actions
	newButton := widget.NewButtonWithIcon("New Workflow", theme.ContentAddIcon(), func() {
		// Close existing wizard if one is already open
		if a.activeWizard != nil {
			if a.activeWizard.window != nil {
				a.activeWizard.window.Close()
			}
			a.activeWizard = nil
		}

		// Create and store the wizard
		a.activeWizard = NewWorkflowWizard(a)
		a.activeWizard.Show()
	})

	editButton := widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
		if selectedWorkflowIndex < 0 || selectedWorkflowIndex >= len(a.cfg.Workflows) {
			a.ShowInfo("Please select a workflow to edit.")
			return
		}

		// TODO: Open the workflow wizard with the selected workflow
		a.ShowInfo("Edit workflow functionality coming soon")
	})

	deleteButton := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		if selectedWorkflowIndex < 0 || selectedWorkflowIndex >= len(a.cfg.Workflows) {
			a.ShowInfo("Please select a workflow to delete.")
			return
		}

		// Confirm deletion
		dialog.ShowConfirm("Delete Workflow",
			"Are you sure you want to delete this workflow?",
			func(confirmed bool) {
				if confirmed {
					// Remove the workflow
					a.cfg.Workflows = append(a.cfg.Workflows[:selectedWorkflowIndex], a.cfg.Workflows[selectedWorkflowIndex+1:]...)

					// Save changes
					a.saveConfig()

					// Refresh the list
					workflowList.Refresh()

					// Show notification
					a.ShowNotification("Workflow Deleted", "The workflow has been deleted successfully")
				}
			},
			a.mainWindow)
	})

	toggleButton := widget.NewButtonWithIcon("Enable/Disable", theme.MediaPlayIcon(), func() {
		if selectedWorkflowIndex < 0 || selectedWorkflowIndex >= len(a.cfg.Workflows) {
			a.ShowInfo("Please select a workflow to toggle.")
			return
		}

		// Toggle the enabled state
		a.cfg.Workflows[selectedWorkflowIndex].Enabled = !a.cfg.Workflows[selectedWorkflowIndex].Enabled

		// Save changes
		a.saveConfig()

		// Refresh the list
		workflowList.Refresh()

		// Show notification
		state := "enabled"
		if !a.cfg.Workflows[selectedWorkflowIndex].Enabled {
			state = "disabled"
		}
		a.ShowNotification("Workflow Updated",
			"The workflow has been "+state)
	})

	// Create button container
	buttonContainer := container.NewHBox(
		newButton,
		layout.NewSpacer(),
		editButton,
		toggleButton,
		deleteButton,
	)

	// Create help text
	helpText := widget.NewRichTextFromMarkdown("# Working with Workflows\n\nWorkflows allow you to automate file organization based on triggers and conditions.\n\n- **Create a new workflow** with the New Workflow button\n- **Edit a workflow** by selecting it and clicking Edit\n- **Enable/Disable a workflow** to control when it runs\n\nWorkflows are processed in order of priority.")

	helpCard := widget.NewCard("Help", "", helpText)

	// Main container with toolbar at top, list in middle, info at bottom
	return container.NewBorder(
		widget.NewLabelWithStyle("Manage Workflows", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		container.NewVBox(
			buttonContainer,
			helpCard,
		),
		nil,
		nil,
		container.NewScroll(workflowList),
	)
}
