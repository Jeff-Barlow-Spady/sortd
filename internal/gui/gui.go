package gui

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"sortd/internal/config"
	"strconv"
	"strings"

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
	fyneApp       fyne.App
	mainWindow    fyne.Window
	cfg           *config.Config
	cmdRunner     *CommandRunner
	watchRunning  bool
	statusUpdater func()
}

// NewApp creates a new GUI application
func NewApp(cfg *config.Config) *App {
	// Create app with a unique ID for preferences storage
	fyneApp := app.NewWithID("io.github.sortd")

	// Load the app icon
	if appIcon, err := fyne.LoadResourceFromPath(filepath.Join("internal", "gui", "s.png")); err == nil {
		fyneApp.SetIcon(appIcon)
	}

	// Find the binary path
	binaryPath := GetBinaryPath()

	// Get the config path
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "sortd", "config.yaml")

	a := &App{
		fyneApp:      fyneApp,
		cfg:          cfg,
		cmdRunner:    NewCommandRunner(binaryPath, configPath),
		watchRunning: false,
	}

	a.mainWindow = a.fyneApp.NewWindow("Sortd")

	// Set window icon (same as app icon)
	if appIcon, err := fyne.LoadResourceFromPath(filepath.Join("internal", "gui", "s.png")); err == nil {
		a.mainWindow.SetIcon(appIcon)
	}

	// Set up menu, icons, etc.
	a.setupSystemTray()

	// Check if watch mode is already running
	if running, _ := a.cmdRunner.CheckStatus(); running {
		a.watchRunning = true
	}

	return a
}

// setupSystemTray sets up the system tray icon and menu
func (a *App) setupSystemTray() {
	// Handle system tray features if we're on desktop
	if deskApp, ok := a.fyneApp.(desktop.App); ok {
		// Create menu items with different actions
		openItem := fyne.NewMenuItem("Show Sortd", func() {
			a.mainWindow.Show()
		})

		startWatchItem := fyne.NewMenuItem("Start Watch Mode", func() {
			a.startWatchMode()
		})

		stopWatchItem := fyne.NewMenuItem("Stop Watch Mode", func() {
			a.stopWatchMode()
		})

		// Add a separator
		separator := fyne.NewMenuItemSeparator()

		// Add exit item
		exitItem := fyne.NewMenuItem("Exit", func() {
			a.fyneApp.Quit()
		})

		// Create the tray menu
		items := []*fyne.MenuItem{openItem, separator}

		// Add correct watch control based on current status
		if a.watchRunning {
			items = append(items, stopWatchItem)
		} else {
			items = append(items, startWatchItem)
		}

		// Add exit section
		items = append(items, separator, exitItem)

		// Set the desktop tray menu
		deskApp.SetSystemTrayMenu(fyne.NewMenu("Sortd", items...))
	}
}

// Run starts the GUI application
func (a *App) Run() {
	// Set up main content
	a.setupMainWindow()

	// Show the window
	a.mainWindow.Show()

	// Start the application
	a.fyneApp.Run()
}

// setupMainWindow sets up the main window content
func (a *App) setupMainWindow() {
	// Set dark theme colors for a terminal-like appearance
	bgColor := color.NRGBA{R: 16, G: 16, B: 16, A: 255}      // Almost black
	accentColor := color.NRGBA{R: 255, G: 165, B: 0, A: 255} // Orange like in the screenshot
	borderColor := color.NRGBA{R: 255, G: 165, B: 0, A: 200} // Slightly transparent orange

	// Create main background
	background := canvas.NewRectangle(bgColor)
	background.Resize(fyne.NewSize(900, 700))

	// Create the logo text in orange, using a more TUI-like font
	logoText := `
   ███████   ██████   ██████    ████████  ██████
  ██░░░░░██ ░░██░░█  ░░██░░█   ░░███░░██░░██░░█
 ██     ░░██ ░██ ░    ░██ ░     ░███ ░██ ░██ ░
░██      ░██ ░██      ░██       ░███ ░██ ░██
░██      ░██ ░██      ░██       ░███ ░██ ░██
░░██     ██  ░██      ░██       ░███ ░██ ░██
 ░░███████  ░███     ░███      ████ ███ ░███
  ░░░░░░░   ░░░      ░░░      ░░░░ ░░░  ░░░
`
	logoDisplay := canvas.NewText(logoText, accentColor)
	logoDisplay.TextStyle.Monospace = true
	logoDisplay.TextSize = 18
	logoDisplay.Alignment = fyne.TextAlignCenter

	// Create terminal-like file browser (left panel)
	fileListLabel := widget.NewLabelWithStyle("File Browser", fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: true})

	// Create file list using a custom widget to look more terminal-like
	fileList := widget.NewList(
		func() int { return 10 },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.FolderIcon()),
				widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			hbox := obj.(*fyne.Container)
			iconObj := hbox.Objects[0].(*widget.Icon)
			label := hbox.Objects[1].(*widget.Label)

			// Sample folders
			folders := []string{
				".pytest_cache",
				".venv",
				".vscode",
				"__pycache__",
				"dist",
				"refs",
				"src",
				"tests",
				".gitignore",
				".python-version",
			}

			if id < len(folders) {
				// Set icon based on whether it's a file or folder
				if strings.HasPrefix(folders[id], ".") && !strings.Contains(folders[id], "/") {
					iconObj.SetResource(theme.DocumentIcon())
					label.SetText(folders[id])
				} else {
					// Check if it looks like a directory name
					if !strings.Contains(folders[id], ".") {
						iconObj.SetResource(theme.FolderIcon())
						label.SetText(folders[id])
					} else {
						iconObj.SetResource(theme.DocumentIcon())
						label.SetText(folders[id])
					}
				}
			}
		},
	)

	// Create right panel (selection panel)
	selectedFilesLabel := widget.NewLabelWithStyle("Selected Files:", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true, Bold: true})

	// Add a selection entry
	selectionEntry := widget.NewEntry()
	selectionEntry.SetPlaceHolder("No files selected")
	selectionEntry.Disable() // Just for display

	// Create organization button with terminal-like appearance
	organizeButton := widget.NewButton("Organize Files", func() {
		err := a.cmdRunner.OrganizeDirectory(a.cfg.Directories.Default, a.cfg.Settings.DryRun)
		if err != nil {
			a.showError("Failed to organize directory", err)
		} else {
			a.showInfo("Organization completed")
		}
	})
	organizeButton.Importance = widget.HighImportance

	// Create terminal-like "command bar" at the bottom
	commandBar := widget.NewEntry()
	commandBar.SetPlaceHolder("Type a command (e.g., 'organize photos', 'help')")
	commandBar.OnSubmitted = func(command string) {
		a.handleNaturalLanguageCommand(command)
		commandBar.SetText("")
	}

	// Create keyboard shortcuts label
	shortcutsText := "q Quit  ? Help  d Toggle Dry Run  u U ^p palette"
	shortcuts := widget.NewLabelWithStyle(shortcutsText, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})

	// Create borders for left panel (file browser)
	leftBorder := canvas.NewRectangle(color.Transparent)
	leftBorder.StrokeColor = borderColor
	leftBorder.StrokeWidth = 1

	// Create borders for right panel (selection)
	rightBorder := canvas.NewRectangle(color.Transparent)
	rightBorder.StrokeColor = borderColor
	rightBorder.StrokeWidth = 1

	// Create the file browser content
	fileListContainer := container.NewBorder(
		fileListLabel,
		nil,
		nil,
		nil,
		container.NewPadded(fileList),
	)

	// Create the selection content
	selectionContainer := container.NewBorder(
		selectedFilesLabel,
		container.NewPadded(organizeButton),
		nil,
		nil,
		container.NewPadded(selectionEntry),
	)

	// Stack borders with content
	leftPanel := container.NewMax(leftBorder, fileListContainer)
	rightPanel := container.NewMax(rightBorder, selectionContainer)

	// Create split container for the two panels
	mainContent := container.NewHSplit(leftPanel, rightPanel)
	mainContent.Offset = 0.4

	// Main content with logo, panels, and command bar
	content := container.NewBorder(
		logoDisplay,
		container.NewVBox(commandBar, shortcuts),
		nil,
		nil,
		container.NewMax(background, mainContent),
	)

	// Add menu for additional functionality
	menu := fyne.NewMainMenu(
		fyne.NewMenu("File",
			fyne.NewMenuItem("Settings", func() {
				settingsContent := a.createSettingsTab()
				dialog := dialog.NewCustom("Settings", "Close", settingsContent, a.mainWindow)
				dialog.Resize(fyne.NewSize(600, 400))
				dialog.Show()
			}),
			fyne.NewMenuItem("Exit", func() {
				a.fyneApp.Quit()
			}),
		),
		fyne.NewMenu("Rules",
			fyne.NewMenuItem("Manage Rules", func() {
				rulesContent := a.createRulesTab()
				dialog := dialog.NewCustom("Rules", "Close", rulesContent, a.mainWindow)
				dialog.Resize(fyne.NewSize(600, 400))
				dialog.Show()
			}),
		),
		fyne.NewMenu("Watch",
			fyne.NewMenuItem("Start Watch Mode", func() {
				a.startWatchMode()
			}),
			fyne.NewMenuItem("Stop Watch Mode", func() {
				a.stopWatchMode()
			}),
			fyne.NewMenuItem("Configure Watch", func() {
				watchContent := a.createWatchModeTab()
				dialog := dialog.NewCustom("Watch Configuration", "Close", watchContent, a.mainWindow)
				dialog.Resize(fyne.NewSize(600, 400))
				dialog.Show()
			}),
		),
		fyne.NewMenu("Cloud",
			fyne.NewMenuItem("Configure", func() {
				cloudContent := a.createCloudTab()
				dialog := dialog.NewCustom("Cloud Configuration", "Close", cloudContent, a.mainWindow)
				dialog.Resize(fyne.NewSize(600, 400))
				dialog.Show()
			}),
		),
		fyne.NewMenu("Help",
			fyne.NewMenuItem("About", func() {
				aboutText := "Sortd - File Organization Made Easy\n\nVersion 0.1.0\n\nA terminal-inspired file organization tool."
				dialog.ShowInformation("About Sortd", aboutText, a.mainWindow)
			}),
		),
	)
	a.mainWindow.SetMainMenu(menu)

	a.mainWindow.SetContent(content)
	a.mainWindow.Resize(fyne.NewSize(900, 700))
	a.mainWindow.SetCloseIntercept(func() {
		// Just hide the window, don't exit the app
		a.mainWindow.Hide()

		// Show notification if watch mode is running
		if a.watchRunning {
			a.showNotification("Sortd is still running in the background", "Watch mode is active")
		}
	})
}

// showSettingsDialog displays the settings in a dialog
func (a *App) showSettingsDialog() {
	settingsContent := a.createSettingsTab()
	dialog := dialog.NewCustom("Settings", "Close", settingsContent, a.mainWindow)
	dialog.Resize(fyne.NewSize(600, 400))
	dialog.Show()
}

// showRulesDialog displays the rules management dialog
func (a *App) showRulesDialog() {
	rulesContent := a.createRulesTab()
	dialog := dialog.NewCustom("Manage Rules", "Close", rulesContent, a.mainWindow)
	dialog.Resize(fyne.NewSize(600, 400))
	dialog.Show()
}

// showWatchConfigDialog displays the watch configuration dialog
func (a *App) showWatchConfigDialog() {
	watchContent := a.createWatchModeTab()
	dialog := dialog.NewCustom("Watch Configuration", "Close", watchContent, a.mainWindow)
	dialog.Resize(fyne.NewSize(600, 400))
	dialog.Show()
}

// showCloudDialog displays the cloud configuration dialog
func (a *App) showCloudDialog() {
	cloudContent := a.createCloudTab()
	dialog := dialog.NewCustom("Cloud Configuration", "Close", cloudContent, a.mainWindow)
	dialog.Resize(fyne.NewSize(600, 400))
	dialog.Show()
}

// showAboutDialog displays information about the application
func (a *App) showAboutDialog() {
	aboutText := "Sortd - File Organization Made Easy\n\nVersion 0.1.0\n\nA terminal-inspired file organization tool."
	dialog.ShowInformation("About Sortd", aboutText, a.mainWindow)
}

// showCommandsDialog displays available commands
func (a *App) showCommandsDialog() {
	commandsText := "Available Commands:\n\n" +
		"organize [path] - Organize files in the specified path\n" +
		"watch - Start watching directories for changes\n" +
		"rules - Manage organization rules\n" +
		"cloud - Configure cloud storage\n" +
		"exit - Exit the application"

	dialog.ShowInformation("Available Commands", commandsText, a.mainWindow)
}

// showPresetDialog shows a dialog with preset rules
func (a *App) showPresetDialog() {
	// Create buttons for each preset
	photoButton := widget.NewButton("Photos by Date", func() {
		a.applyPresetRule("photos_by_date")
	})

	docButton := widget.NewButton("Documents by Type", func() {
		a.applyPresetRule("documents_by_type")
	})

	musicButton := widget.NewButton("Music by Artist", func() {
		a.applyPresetRule("music_by_artist")
	})

	downloadButton := widget.NewButton("Clean Downloads", func() {
		a.applyPresetRule("clean_downloads")
	})

	content := container.NewVBox(
		widget.NewLabel("Choose a preset organization rule:"),
		photoButton,
		docButton,
		musicButton,
		downloadButton,
	)

	dialog := dialog.NewCustom("Preset Rules", "Cancel", content, a.mainWindow)
	dialog.Show()
}

// createFilesTab creates the main files tab that replaces the original settings tab
func (a *App) createFilesTab() fyne.CanvasObject {
	// Welcome message and description
	welcomeText := "Welcome to Sortd! Organizing files is now easy and fun."
	welcomeLabel := widget.NewLabelWithStyle(welcomeText, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	descriptionText := "Tell Sortd what you want to do with your files using plain language.\nYou can organize photos, documents, downloads, and more with just a few clicks!"
	descriptionLabel := widget.NewLabel(descriptionText)
	descriptionLabel.Alignment = fyne.TextAlignCenter

	// Quick action buttons
	organizeButton := widget.NewButton("Organize Files Now", func() {
		// Run organize command for default directory
		err := a.cmdRunner.OrganizeDirectory(a.cfg.Directories.Default, a.cfg.Settings.DryRun)
		if err != nil {
			a.showError("Failed to organize directory", err)
		} else {
			a.showInfo("Organization completed")
		}
	})
	organizeButton.Importance = widget.HighImportance

	watchButton := widget.NewButton("Start Watching Folders", func() {
		a.startWatchMode()
	})

	// Common tasks (with plain language prompts)
	taskPrompt := widget.NewLabel("What would you like to do?")
	taskPrompt.Alignment = fyne.TextAlignCenter
	taskPrompt.TextStyle = fyne.TextStyle{Bold: true}

	photoTask := widget.NewButton("Organize my photos by date", func() {
		a.applyPresetRule("photos_by_date")
	})

	docTask := widget.NewButton("Sort my documents by type", func() {
		a.applyPresetRule("documents_by_type")
	})

	musicTask := widget.NewButton("Arrange music files by artist", func() {
		a.applyPresetRule("music_by_artist")
	})

	downloadTask := widget.NewButton("Clean up my downloads folder", func() {
		a.applyPresetRule("clean_downloads")
	})

	customTask := widget.NewEntry()
	customTask.SetPlaceHolder("Tell Sortd what you want to do with your files...")

	customButton := widget.NewButton("Apply", func() {
		if customTask.Text != "" {
			a.handleNaturalLanguageCommand(customTask.Text)
		}
	})

	// Recent activity
	activityLabel := widget.NewLabelWithStyle("Recent Activity", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	activityList := widget.NewList(
		func() int { return 3 }, // Show last 3 activities
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			activities := []string{
				"Files organized in Downloads",
				"Added rule for photos",
				"Started watching Documents folder",
			}
			if id < len(activities) {
				label.SetText(activities[id])
			}
		},
	)

	// Directory selector
	dirLabel := widget.NewLabel("Current directory:")
	dirEntry := widget.NewEntry()
	dirEntry.SetText(a.cfg.Directories.Default)
	dirEntry.OnChanged = func(text string) {
		a.cfg.Directories.Default = text
	}

	browseButton := widget.NewButton("Browse", func() {
		// Open directory picker (not implemented here)
		a.showInfo("Directory picker would open here")
	})

	// Layout everything
	header := container.NewVBox(
		welcomeLabel,
		descriptionLabel,
	)

	actions := container.NewGridWithColumns(2,
		organizeButton,
		watchButton,
	)

	tasks := container.NewVBox(
		taskPrompt,
		photoTask,
		docTask,
		musicTask,
		downloadTask,
		container.NewBorder(nil, nil, nil, customButton, customTask),
	)

	directory := container.NewBorder(
		nil, nil, nil, browseButton,
		container.NewHBox(dirLabel, dirEntry),
	)

	activity := container.NewVBox(
		activityLabel,
		container.NewBorder(nil, nil, nil, nil, activityList),
	)

	// Final layout
	return container.NewVBox(
		header,
		widget.NewSeparator(),
		actions,
		widget.NewSeparator(),
		tasks,
		widget.NewSeparator(),
		directory,
		widget.NewSeparator(),
		activity,
	)
}

// createRulesTab creates the rules tab
func (a *App) createRulesTab() fyne.CanvasObject {
	// Rules list
	rulesList := widget.NewList(
		func() int { return len(a.cfg.Rules) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Pattern:"),
				widget.NewLabel("Target:"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			container := obj.(*fyne.Container)
			patternLabel := container.Objects[0].(*widget.Label)
			targetLabel := container.Objects[1].(*widget.Label)

			patternLabel.SetText(fmt.Sprintf("Pattern: %s", a.cfg.Rules[id].Pattern))
			targetLabel.SetText(fmt.Sprintf("Target: %s", a.cfg.Rules[id].Target))
		},
	)

	// Add rule section
	patternEntry := widget.NewEntry()
	patternEntry.SetPlaceHolder("Pattern (e.g., *.pdf)")

	targetEntry := widget.NewEntry()
	targetEntry.SetPlaceHolder("Target Directory")

	addButton := widget.NewButton("Add Rule", func() {
		if patternEntry.Text != "" && targetEntry.Text != "" {
			a.cfg.Rules = append(a.cfg.Rules, struct {
				Pattern string `yaml:"pattern"`
				Target  string `yaml:"target"`
			}{
				Pattern: patternEntry.Text,
				Target:  targetEntry.Text,
			})
			rulesList.Refresh()
			patternEntry.SetText("")
			targetEntry.SetText("")
		}
	})

	// Delete button (uses a separate tracking variable since we don't have direct access to list selection)
	var selectedRule int = -1
	rulesList.OnSelected = func(id widget.ListItemID) {
		selectedRule = id
	}

	deleteButton := widget.NewButton("Delete Selected Rule", func() {
		if selectedRule < 0 || selectedRule >= len(a.cfg.Rules) {
			return
		}

		// Remove the rule
		a.cfg.Rules = append(a.cfg.Rules[:selectedRule], a.cfg.Rules[selectedRule+1:]...)
		rulesList.UnselectAll()
		selectedRule = -1
		rulesList.Refresh()
	})

	// Save button
	saveButton := widget.NewButton("Save Rules", func() {
		a.saveConfig()
	})

	// Organize button
	organizeButton := widget.NewButton("Organize Now", func() {
		// Run organize command for default directory
		err := a.cmdRunner.OrganizeDirectory(a.cfg.Directories.Default, a.cfg.Settings.DryRun)
		if err != nil {
			a.showError("Failed to organize directory", err)
		} else {
			a.showInfo("Organization completed")
		}
	})

	return container.NewVBox(
		widget.NewCard("Current Rules", "", container.NewVBox(
			rulesList,
			deleteButton,
		)),
		widget.NewCard("Add New Rule", "", container.NewVBox(
			patternEntry,
			targetEntry,
			addButton,
		)),
		container.NewHBox(
			organizeButton,
			layout.NewSpacer(),
			saveButton,
		),
	)
}

// createWatchModeTab creates the watch mode tab
func (a *App) createWatchModeTab() fyne.CanvasObject {
	// Create a status label to show current status
	statusLabel := canvas.NewText("", nil)
	a.updateWatchStatus(statusLabel)

	// Store the status updater function
	a.statusUpdater = func() {
		a.updateWatchStatus(statusLabel)
	}

	// Watch mode settings
	enabledCheck := widget.NewCheck("Enable Watch Mode", func(value bool) {
		a.cfg.WatchMode.Enabled = value
	})
	enabledCheck.SetChecked(a.cfg.WatchMode.Enabled)

	requireConfirmCheck := widget.NewCheck("Require Confirmation", func(value bool) {
		a.cfg.WatchMode.RequireConfirmation = value
	})
	requireConfirmCheck.SetChecked(a.cfg.WatchMode.RequireConfirmation)

	intervalLabel := widget.NewLabel("Check Interval (seconds):")
	intervalEntry := widget.NewEntry()
	intervalEntry.SetText(fmt.Sprintf("%d", a.cfg.WatchMode.Interval))
	intervalEntry.OnChanged = func(text string) {
		if val, err := strconv.Atoi(text); err == nil && val > 0 {
			a.cfg.WatchMode.Interval = val
		}
	}

	confirmPeriodLabel := widget.NewLabel("Confirmation Period (seconds, 0 to disable):")
	confirmPeriodEntry := widget.NewEntry()
	confirmPeriodEntry.SetText(fmt.Sprintf("%d", a.cfg.WatchMode.ConfirmationPeriod))
	confirmPeriodEntry.OnChanged = func(text string) {
		if val, err := strconv.Atoi(text); err == nil && val >= 0 {
			a.cfg.WatchMode.ConfirmationPeriod = val
		}
	}

	// Watch directories
	watchDirsList := widget.NewList(
		func() int { return len(a.cfg.Directories.Watch) },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			label.SetText(a.cfg.Directories.Watch[id])
		},
	)

	// Add directory
	newDirEntry := widget.NewEntry()
	newDirEntry.SetPlaceHolder("New directory to watch")

	addDirButton := widget.NewButton("Add Directory", func() {
		if newDirEntry.Text != "" {
			a.cfg.Directories.Watch = append(a.cfg.Directories.Watch, newDirEntry.Text)
			watchDirsList.Refresh()
			newDirEntry.SetText("")
		}
	})

	// Remove directory button (with selection tracking)
	var selectedDir int = -1
	watchDirsList.OnSelected = func(id widget.ListItemID) {
		selectedDir = id
	}

	removeDirButton := widget.NewButton("Remove Selected Directory", func() {
		if selectedDir < 0 || selectedDir >= len(a.cfg.Directories.Watch) {
			return
		}

		// Remove the directory
		a.cfg.Directories.Watch = append(a.cfg.Directories.Watch[:selectedDir], a.cfg.Directories.Watch[selectedDir+1:]...)
		watchDirsList.UnselectAll()
		selectedDir = -1
		watchDirsList.Refresh()
	})

	// Control buttons
	startButton := widget.NewButton("Start Watch Mode", func() {
		a.startWatchMode()
	})

	stopButton := widget.NewButton("Stop Watch Mode", func() {
		a.stopWatchMode()
	})

	// Save button
	saveButton := widget.NewButton("Save Watch Settings", func() {
		a.saveConfig()
	})

	// Update button states based on current state
	a.updateWatchButtons(startButton, stopButton)

	return container.NewVBox(
		statusLabel,
		widget.NewCard("Watch Mode Settings", "", container.NewVBox(
			enabledCheck,
			requireConfirmCheck,
			container.NewHBox(intervalLabel, intervalEntry),
			container.NewHBox(confirmPeriodLabel, confirmPeriodEntry),
		)),
		widget.NewCard("Watch Directories", "", container.NewVBox(
			watchDirsList,
			container.NewHBox(
				newDirEntry,
				addDirButton,
			),
			removeDirButton,
		)),
		container.NewHBox(
			startButton,
			stopButton,
		),
		container.NewHBox(
			layout.NewSpacer(),
			saveButton,
		),
	)
}

// updateWatchStatus updates the watch mode status text
func (a *App) updateWatchStatus(statusLabel *canvas.Text) {
	if a.watchRunning {
		statusLabel.Text = "Watch Mode: Running"
		statusLabel.Color = color.NRGBA{R: 0, G: 200, B: 0, A: 255} // Green
	} else {
		statusLabel.Text = "Watch Mode: Stopped"
		statusLabel.Color = color.NRGBA{R: 200, G: 0, B: 0, A: 255} // Red
	}
	statusLabel.Refresh()
}

// updateWatchButtons updates the watch mode buttons based on current state
func (a *App) updateWatchButtons(startButton, stopButton *widget.Button) {
	if a.watchRunning {
		startButton.Disable()
		stopButton.Enable()
	} else {
		startButton.Enable()
		stopButton.Disable()
	}
}

// saveConfig saves the current configuration
func (a *App) saveConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		a.showError("Failed to get home directory", err)
		return
	}

	configPath := filepath.Join(home, ".config", "sortd", "config.yaml")
	err = config.SaveConfig(a.cfg, configPath)
	if err != nil {
		a.showError("Failed to save configuration", err)
		return
	}

	a.showInfo("Configuration saved successfully")
}

// showError displays an error message
func (a *App) showError(title string, err error) {
	dialog.ShowError(err, a.mainWindow)
}

// showInfo displays an information message
func (a *App) showInfo(message string) {
	dialog.ShowInformation("Information", message, a.mainWindow)
}

// showNotification shows a notification
func (a *App) showNotification(title, message string) {
	if desktop, ok := a.fyneApp.(fyne.App); ok {
		desktop.SendNotification(fyne.NewNotification(title, message))
	}
}

// startWatchMode starts the watch mode
func (a *App) startWatchMode() {
	// Ensure we have at least one directory to watch
	if len(a.cfg.Directories.Watch) == 0 {
		a.showError("Watch Mode Error", fmt.Errorf("no directories configured to watch"))
		return
	}

	// Start watch mode
	err := a.cmdRunner.StartWatchMode(
		a.cfg.Directories.Watch,
		a.cfg.WatchMode.Interval,
		a.cfg.WatchMode.RequireConfirmation,
		a.cfg.WatchMode.ConfirmationPeriod,
	)

	if err != nil {
		a.showError("Failed to start watch mode", err)
		return
	}

	a.watchRunning = true
	a.showInfo("Watch mode started")
	a.showNotification("Watch Mode", "Sortd watch mode has started")

	// Update UI
	if a.statusUpdater != nil {
		a.statusUpdater()
	}

	// Update tray menu
	a.setupSystemTray()
}

// stopWatchMode stops the watch mode
func (a *App) stopWatchMode() {
	err := a.cmdRunner.StopWatchMode()
	if err != nil {
		a.showError("Failed to stop watch mode", err)
		return
	}

	a.watchRunning = false
	a.showInfo("Watch mode stopped")
	a.showNotification("Watch Mode", "Sortd watch mode has stopped")

	// Update UI
	if a.statusUpdater != nil {
		a.statusUpdater()
	}

	// Update tray menu
	a.setupSystemTray()
}

// createCloudTab creates the cloud storage tab
func (a *App) createCloudTab() fyne.CanvasObject {
	// Cloud storage title
	titleLabel := widget.NewLabelWithStyle("Cloud Storage", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Cloud storage description
	descriptionLabel := widget.NewLabel("Connect your cloud storage to organize files across your devices")
	descriptionLabel.Alignment = fyne.TextAlignCenter

	// Cloud provider selection
	providerLabel := widget.NewLabel("Cloud Provider:")
	providerSelect := widget.NewSelect([]string{"Dropbox", "Google Drive", "OneDrive", "iCloud", "S3"}, func(value string) {
		// Cloud provider selection logic
	})

	// Authentication status
	statusLabel := widget.NewLabel("Status: Not connected")

	// Connect button
	connectButton := widget.NewButton("Connect", func() {
		a.showInfo("This would connect to cloud storage")
	})

	// Sync directory path
	syncDirLabel := widget.NewLabel("Sync Folder:")
	syncDirEntry := widget.NewEntry()
	syncDirEntry.SetPlaceHolder("/path/to/sync/folder")

	// Sync options
	autoSyncCheck := widget.NewCheck("Auto-sync", func(value bool) {
		// Auto-sync logic
	})

	// Sync now button
	syncButton := widget.NewButton("Sync Now", func() {
		a.showInfo("This would sync cloud files now")
	})

	// Layout cloud tab
	cloudForm := container.NewVBox(
		container.NewHBox(providerLabel, providerSelect),
		statusLabel,
		connectButton,
		container.NewHBox(syncDirLabel, syncDirEntry),
		autoSyncCheck,
		syncButton,
	)

	cloudCard := widget.NewCard("Cloud Storage Configuration", "", cloudForm)

	return container.NewVBox(
		titleLabel,
		descriptionLabel,
		container.NewPadded(cloudCard),
		layout.NewSpacer(),
	)
}

// applyPresetRule applies a preset organization rule
func (a *App) applyPresetRule(ruleType string) {
	// Sample rule definitions
	var rules []struct {
		Pattern string
		Target  string
	}

	switch ruleType {
	case "photos_by_date":
		rules = []struct {
			Pattern string
			Target  string
		}{
			{Pattern: "*.jpg", Target: "Photos/{{.Year}}/{{.Month}}"},
			{Pattern: "*.png", Target: "Photos/{{.Year}}/{{.Month}}"},
			{Pattern: "*.gif", Target: "Photos/{{.Year}}/{{.Month}}"},
		}
		a.showInfo("Applied photo organization by date")

	case "documents_by_type":
		rules = []struct {
			Pattern string
			Target  string
		}{
			{Pattern: "*.pdf", Target: "Documents/PDFs"},
			{Pattern: "*.doc*", Target: "Documents/Word"},
			{Pattern: "*.xls*", Target: "Documents/Excel"},
			{Pattern: "*.ppt*", Target: "Documents/PowerPoint"},
			{Pattern: "*.txt", Target: "Documents/Text"},
		}
		a.showInfo("Applied document organization by type")

	case "music_by_artist":
		rules = []struct {
			Pattern string
			Target  string
		}{
			{Pattern: "*.mp3", Target: "Music/{{.Artist}}"},
			{Pattern: "*.flac", Target: "Music/{{.Artist}}"},
			{Pattern: "*.wav", Target: "Music/{{.Artist}}"},
		}
		a.showInfo("Applied music organization by artist")

	case "clean_downloads":
		rules = []struct {
			Pattern string
			Target  string
		}{
			{Pattern: "*.zip", Target: "Downloads/Archives"},
			{Pattern: "*.rar", Target: "Downloads/Archives"},
			{Pattern: "*.exe", Target: "Downloads/Programs"},
			{Pattern: "*.dmg", Target: "Downloads/Programs"},
			{Pattern: "*.jpg", Target: "Downloads/Images"},
			{Pattern: "*.png", Target: "Downloads/Images"},
			{Pattern: "*.pdf", Target: "Downloads/Documents"},
			{Pattern: "*.doc*", Target: "Downloads/Documents"},
		}
		a.showInfo("Applied downloads folder cleanup")
	}

	// Apply the rules
	if len(rules) > 0 {
		// Convert to the proper struct format that config expects
		for _, rule := range rules {
			a.cfg.Rules = append(a.cfg.Rules, struct {
				Pattern string `yaml:"pattern"`
				Target  string `yaml:"target"`
			}{
				Pattern: rule.Pattern,
				Target:  rule.Target,
			})
		}

		// Save the config with new rules
		a.saveConfig()
	}
}

// handleNaturalLanguageCommand processes natural language commands
func (a *App) handleNaturalLanguageCommand(command string) {
	// This is a placeholder for natural language processing
	// In a real implementation, this would use NLP or a more sophisticated
	// matching system to interpret the user's intent

	command = strings.ToLower(command)

	if strings.Contains(command, "photo") || strings.Contains(command, "image") {
		a.applyPresetRule("photos_by_date")
	} else if strings.Contains(command, "document") || strings.Contains(command, "doc") {
		a.applyPresetRule("documents_by_type")
	} else if strings.Contains(command, "music") || strings.Contains(command, "song") {
		a.applyPresetRule("music_by_artist")
	} else if strings.Contains(command, "download") || strings.Contains(command, "clean") {
		a.applyPresetRule("clean_downloads")
	} else {
		a.showInfo("I'm not sure how to handle that request yet. Please try one of the preset tasks.")
	}
}

// createSettingsTab creates the settings tab
func (a *App) createSettingsTab() fyne.CanvasObject {
	// General settings
	dryRunCheck := widget.NewCheck("Dry Run (Simulate Operations)", func(value bool) {
		a.cfg.Settings.DryRun = value
	})
	dryRunCheck.SetChecked(a.cfg.Settings.DryRun)

	createDirsCheck := widget.NewCheck("Create Destination Directories", func(value bool) {
		a.cfg.Settings.CreateDirs = value
	})
	createDirsCheck.SetChecked(a.cfg.Settings.CreateDirs)

	backupCheck := widget.NewCheck("Create Backups Before Moving", func(value bool) {
		a.cfg.Settings.Backup = value
	})
	backupCheck.SetChecked(a.cfg.Settings.Backup)

	improvedCatCheck := widget.NewCheck("Use Improved Categorization", func(value bool) {
		a.cfg.Settings.ImprovedCategorization = value
	})
	improvedCatCheck.SetChecked(a.cfg.Settings.ImprovedCategorization)

	// Collision strategy
	collisionLabel := widget.NewLabel("Collision Strategy:")
	collisionSelect := widget.NewSelect([]string{"rename", "skip", "ask"}, func(value string) {
		a.cfg.Settings.Collision = value
	})
	collisionSelect.SetSelected(a.cfg.Settings.Collision)

	// Default directory
	defaultDirLabel := widget.NewLabel("Default Directory:")
	defaultDirEntry := widget.NewEntry()
	defaultDirEntry.SetText(a.cfg.Directories.Default)
	defaultDirEntry.OnChanged = func(text string) {
		a.cfg.Directories.Default = text
	}

	// Save button
	saveButton := widget.NewButton("Save Settings", func() {
		a.saveConfig()
	})

	// Layout the settings
	return container.NewVBox(
		widget.NewCard("General Settings", "", container.NewVBox(
			dryRunCheck,
			createDirsCheck,
			backupCheck,
			improvedCatCheck,
			container.NewHBox(collisionLabel, collisionSelect),
			container.NewHBox(defaultDirLabel, defaultDirEntry),
		)),
		layout.NewSpacer(),
		container.NewHBox(
			layout.NewSpacer(),
			saveButton,
		),
	)
}
