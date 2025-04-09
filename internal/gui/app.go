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
	bgColor := color.NRGBA{R: 16, G: 16, B: 16, A: 255}
	accentColor := color.NRGBA{R: 255, G: 165, B: 0, A: 255}

	background := canvas.NewRectangle(bgColor)
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
	logoDisplay := canvas.NewText(logoText, accentColor)
	logoDisplay.TextStyle.Monospace = true
	logoDisplay.TextSize = 18
	logoDisplay.Alignment = fyne.TextAlignCenter

	// Set initial size before setting final content
	a.mainWindow.Resize(fyne.NewSize(900, 700))

	// --- Tabs Setup ---
	tabs := container.NewAppTabs(
		container.NewTabItem("Organize", a.createOrganizeTab()),
		container.NewTabItem("Cloud", a.createCloudTab()),
		container.NewTabItem("Settings", a.createSettingsTab()),
	)
	tabs.SetTabLocation(container.TabLocationTop) // Ensure tabs are at the top

	content := container.NewBorder(
		// Top content - logo and a separation line
		container.NewVBox(
			logoDisplay,
			canvas.NewLine(accentColor),
		),
		nil,  // No bottom content
		nil,  // No left content
		nil,  // No right content
		tabs, // Center content is the tabs
	)

	a.mainWindow.SetContent(content)
}

// ShowError displays an error dialog
func (a *App) ShowError(title string, err error) {
	if err == nil {
		return
	}
	dialog.ShowError(err, a.mainWindow)
}

// ShowInfo displays an information dialog
func (a *App) ShowInfo(message string) {
	dialog.ShowInformation("Information", message, a.mainWindow)
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
	if a.statusUpdater != nil {
		a.statusUpdater()
	}
}

// saveConfig saves the configuration to file
func (a *App) saveConfig() {
	if err := config.SaveConfig(a.cfg); err != nil {
		a.ShowError("Failed to save configuration", err)
	}
}
