package gui

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"sortd/internal/config"
	"sortd/internal/log"
	"sortd/internal/organize"
	"sortd/internal/watch"
	"sortd/pkg/types"
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
	fyneApp        fyne.App
	mainWindow     fyne.Window
	cfg            *config.Config
	organizeEngine *organize.Engine
	watchDaemon    *watch.Daemon
	pathLabel      *widget.Label // Reference to the path display label
	statusUpdater  func()        // Function to update system tray status
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

	watchDaemon := watch.NewDaemon(cfg)

	a := &App{
		fyneApp:        fyneApp,
		cfg:            cfg,
		organizeEngine: organizeEngine,
		watchDaemon:    watchDaemon,
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
	borderColor := color.NRGBA{R: 255, G: 165, B: 0, A: 200}

	background := canvas.NewRectangle(bgColor)
	background.Resize(fyne.NewSize(900, 700))

	logoText := `
  █████████     ███████    ███████████   ███████████ ██████████  
 ███░░░░░███  ███░░░░░███ ░░███░░░░░███ ░█░░░███░░░█░░███░░░░███ 
░███    ░░░  ███     ░░███ ░███    ░███ ░   ░███  ░  ░███   ░░███
░░█████████ ░███      ░███ ░██████████      ░███     ░███    ░███
 ░░░░░░░░███░███      ░███ ░███░░░░░███     ░███     ░███    ░███
 ███    ░███░░███     ███  ░███    ░███     ░███     ░███    ███ 
░░█████████  ░░░███████░   █████   █████    █████    ██████████  
 ░░░░░░░░░     ░░░░░░░    ░░░░░   ░░░░░    ░░░░░    ░░░░░░░░░░   
                                                                 
                                                                 
                                                                 
`
	logoDisplay := canvas.NewText(logoText, accentColor)
	logoDisplay.TextStyle.Monospace = true
	logoDisplay.TextSize = 18
	logoDisplay.Alignment = fyne.TextAlignCenter

	fileListLabel := widget.NewLabelWithStyle("File Browser", fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: true})

	a.pathLabel = widget.NewLabelWithStyle("Location: "+a.cfg.Directories.Default,
		fyne.TextAlignLeading,
		fyne.TextStyle{Monospace: true})

	pathHeader := container.NewVBox(
		fileListLabel,
		container.NewHBox(a.pathLabel),
	)

	fileList := widget.NewList(
		func() int { files, _ := a.getDirectoryFiles(a.cfg.Directories.Default); return len(files) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.DocumentIcon()),
				widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			hbox := obj.(*fyne.Container)
			iconObj := hbox.Objects[0].(*widget.Icon)
			label := hbox.Objects[1].(*widget.Label)

			files, err := a.getDirectoryFiles(a.cfg.Directories.Default)
			if err != nil || id >= len(files) {
				label.SetText("Error reading directory")
				return
			}

			fileInfo := files[id]

			if fileInfo.IsDir() {
				iconObj.SetResource(theme.FolderIcon())
			} else {
				iconObj.SetResource(theme.DocumentIcon())
			}
			label.SetText(fileInfo.Name())
		},
	)

	parentDirButton := widget.NewButton("↑ Parent Directory", func() {
		parent := filepath.Dir(a.cfg.Directories.Default)
		if parent != a.cfg.Directories.Default {
			a.updateDirectoryPath(parent)
			fileList.Refresh()
		}
	})
	pathHeader.Add(parentDirButton)

	fileListContainer := container.NewBorder(
		pathHeader,
		nil,
		nil,
		nil,
		container.NewPadded(fileList),
	)

	selectedFilesLabel := widget.NewLabelWithStyle("Selected Files:", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true, Bold: true})

	selectionEntry := widget.NewEntry()
	selectionEntry.SetPlaceHolder("No files selected")
	selectionEntry.Disable()

	organizeButton := widget.NewButton("Organize Files", func() {
		a.organizeEngine.SetDryRun(a.cfg.Settings.DryRun)
		results, err := a.organizeEngine.OrganizeDirectory(a.cfg.Directories.Default)
		if err != nil {
			a.ShowError("Failed to organize directory", err)
		} else {
			var movedCount, errorCount int
			var errors []string
			for _, res := range results {
				if res.Error != nil {
					errorCount++
					errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(res.SourcePath), res.Error))
				} else if res.Moved {
					movedCount++
				}
			}
			msg := fmt.Sprintf("Organization complete. %d files processed/moved.", movedCount)
			if errorCount > 0 {
				msg += fmt.Sprintf("\nEncountered %d errors:\n%s", errorCount, strings.Join(errors, "\n"))
				a.ShowError("Organization encountered errors", fmt.Errorf(strings.Join(errors, "\n"))) // Show first error
			} else {
				a.ShowInfo(msg)
			}
		}
	})
	organizeButton.Importance = widget.HighImportance

	selectionContainer := container.NewBorder(
		selectedFilesLabel,
		container.NewPadded(organizeButton),
		nil,
		nil,
		container.NewPadded(selectionEntry),
	)

	leftBorder := canvas.NewRectangle(color.Transparent)
	leftBorder.StrokeColor = borderColor
	leftBorder.StrokeWidth = 1

	rightBorder := canvas.NewRectangle(color.Transparent)
	rightBorder.StrokeColor = borderColor
	rightBorder.StrokeWidth = 1

	leftPanel := container.NewMax(leftBorder, fileListContainer)
	rightPanel := container.NewMax(rightBorder, selectionContainer)

	mainContent := container.NewHSplit(leftPanel, rightPanel)
	mainContent.Offset = 0.4

	content := container.NewBorder(
		logoDisplay,
		nil,
		nil,
		nil,
		container.NewMax(background, mainContent),
	)

	a.mainWindow.SetContent(content)
	a.mainWindow.Resize(fyne.NewSize(900, 700))

	a.mainWindow.SetCloseIntercept(func() {
		a.mainWindow.Hide()

		if a.watchDaemon.Status().Running {
			a.showNotification("Sortd is still running in the background", "Watch mode is active")
		}
	})

	a.mainWindow.Canvas().SetOnTypedKey(func(ke *fyne.KeyEvent) {
		switch ke.Name {
		case fyne.KeyQ:
			a.fyneApp.Quit()
		case fyne.KeyD:
			a.cfg.Settings.DryRun = !a.cfg.Settings.DryRun
			a.saveConfig()
			if a.cfg.Settings.DryRun {
				a.ShowInfo("Dry run mode enabled")
			} else {
				a.ShowInfo("Dry run mode disabled")
			}
		}
	})

	fileList.OnSelected = func(id widget.ListItemID) {
		files, err := a.getDirectoryFiles(a.cfg.Directories.Default)
		if err != nil || id >= len(files) {
			return
		}

		fileInfo := files[id]

		if fileInfo.IsDir() {
			newPath := filepath.Join(a.cfg.Directories.Default, fileInfo.Name())
			a.updateDirectoryPath(newPath)
			fileList.Refresh()
			fileList.UnselectAll()
		} else {
			selectionEntry.Enable()
			selectionEntry.SetText(fileInfo.Name())
			selectionEntry.Disable()
		}
	}
}

// GetMainWindow returns the main window for testing purposes
func (a *App) GetMainWindow() fyne.Window {
	return a.mainWindow
}

// updateWatchStatus updates the watch mode status text
func (a *App) updateWatchStatus(statusLabel *canvas.Text) {
	status := a.watchDaemon.Status()
	if status.Running {
		statusLabel.Text = "Watch Mode: Running (Watching " + fmt.Sprint(len(status.WatchDirectories)) + " dirs)"
		statusLabel.Color = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	} else {
		statusLabel.Text = "Watch Mode: Stopped"
		statusLabel.Color = color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	}
	statusLabel.Refresh()
}

// updateWatchButtons updates the watch mode buttons based on current state
func (a *App) updateWatchButtons(startButton, stopButton *widget.Button) {
	status := a.watchDaemon.Status()
	if status.Running {
		startButton.Disable()
		stopButton.Enable()
	} else {
		startButton.Enable()
		stopButton.Disable()
	}
}

// saveConfig saves the current configuration
func (a *App) saveConfig() {
	if err := config.SaveConfig(a.cfg); err != nil {
		a.ShowError("Failed to save configuration", err)
	} else {
		// Notify daemon of config change if it's running?
		// Or assume daemon reads config on start?
		if a.watchDaemon.Status().Running {
			// TODO: Add a way to reload config in the daemon if needed
			log.Info("Configuration saved. Restart watch mode for changes to take effect.")
			a.ShowInfo("Configuration saved. Restart watch mode for changes to take effect.")
		} else {
			a.ShowInfo("Configuration saved.")
		}
	}
}

// ShowError displays an error message
func (a *App) ShowError(message string, err error) {
	log.Errorf("%s: %v", message, err)
	dialog.ShowError(fmt.Errorf("%s: %w", message, err), a.mainWindow) // Keep formatting here as we combine msg+err
}

// ShowInfo displays an information message
func (a *App) ShowInfo(message string) {
	log.Info(message)
	dialog.ShowInformation("Info", message, a.mainWindow) // Pass message directly
}

// showNotification shows a notification
func (a *App) showNotification(title, message string) {
	if a.fyneApp != nil {
		a.fyneApp.SendNotification(fyne.NewNotification(title, message))
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

// createCloudTab creates the cloud storage tab
func (a *App) createCloudTab() fyne.CanvasObject {
	cloudLabel := widget.NewLabel("Cloud Storage Integration (Coming Soon)")

	providerSelect := widget.NewSelect([]string{"AWS S3", "Google Cloud Storage", "Azure Blob"}, func(value string) {
		// Handle provider selection
	})
	providerSelect.PlaceHolder = "Select Provider"

	bucketEntry := widget.NewEntry()
	bucketEntry.SetPlaceHolder("Bucket Name")
	accessKeyEntry := widget.NewPasswordEntry()
	accessKeyEntry.SetPlaceHolder("Access Key")
	secretKeyEntry := widget.NewPasswordEntry()
	secretKeyEntry.SetPlaceHolder("Secret Key")

	saveButton := widget.NewButton("Save Cloud Config", func() {
		// Save cloud config logic
	})

	return container.NewVBox(
		cloudLabel,
		widget.NewForm(
			widget.NewFormItem("Provider", providerSelect),
			widget.NewFormItem("Bucket", bucketEntry),
			widget.NewFormItem("Access Key", accessKeyEntry),
			widget.NewFormItem("Secret Key", secretKeyEntry),
		),
		saveButton,
	)
}

// applyPresetRule applies a preset organization rule
func (a *App) applyPresetRule(ruleType string) {
	var newPattern types.Pattern
	var destDir string

	switch ruleType {
	case "Images":
		destDir = "Images"
		newPattern = types.Pattern{Match: "*.{jpg,jpeg,png,gif,bmp,tiff}", Target: destDir}
	case "Documents":
		destDir = "Documents"
		newPattern = types.Pattern{Match: "*.{pdf,doc,docx,xls,xlsx,ppt,pptx,txt,rtf}", Target: destDir}
	case "Videos":
		destDir = "Videos"
		newPattern = types.Pattern{Match: "*.{mp4,avi,mov,wmv,mkv}", Target: destDir}
	case "Audio":
		destDir = "Audio"
		newPattern = types.Pattern{Match: "*.{mp3,wav,aac,flac}", Target: destDir}
	case "Archives":
		destDir = "Archives"
		newPattern = types.Pattern{Match: "*.{zip,rar,tar,gz,7z}", Target: destDir}
	default:
		a.ShowError("Invalid Preset", fmt.Errorf("unknown preset rule type: %s", ruleType))
		return
	}

	found := false
	for _, p := range a.cfg.Organize.Patterns {
		if p.Match == newPattern.Match && p.Target == newPattern.Target {
			found = true
			break
		}
	}

	if found {
		a.ShowInfo("The '" + ruleType + "' preset rule already exists.")
		return
	}

	a.cfg.Organize.Patterns = append(a.cfg.Organize.Patterns, newPattern)

	if a.cfg.Settings.CreateDirs {
		fullDestPath := filepath.Join(a.cfg.Directories.Default, destDir)
		if err := os.MkdirAll(fullDestPath, 0755); err != nil {
			a.ShowError("Directory Creation Failed", fmt.Errorf("could not create target directory '%s': %w", fullDestPath, err))
		}
	}

	a.saveConfig()

	a.ShowInfo("Added '" + ruleType + "' preset rule.")
}

// handleNaturalLanguageCommand processes natural language commands
func (a *App) handleNaturalLanguageCommand(command string) {
	lowerCmd := strings.ToLower(command)

	if strings.Contains(lowerCmd, "organize") {
		a.organizeEngine.SetDryRun(a.cfg.Settings.DryRun)
		results, err := a.organizeEngine.OrganizeDirectory(a.cfg.Directories.Default)
		if err != nil {
			a.ShowError("Natural Language Organize Failed", err)
		} else {
			var movedCount, errorCount int
			var errors []string
			for _, res := range results {
				if res.Error != nil {
					errorCount++
					errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(res.SourcePath), res.Error))
				} else if res.Moved {
					movedCount++
				}
			}
			msg := fmt.Sprintf("Organization complete. %d files processed/moved.", movedCount)
			if errorCount > 0 {
				msg += fmt.Sprintf("\nEncountered %d errors:\n%s", errorCount, strings.Join(errors, "\n"))
				a.ShowError("Organization encountered errors", fmt.Errorf(strings.Join(errors, "\n"))) // Show first error
			} else {
				a.ShowInfo(msg)
			}
		}
	} else if strings.Contains(lowerCmd, "watch") {
		if strings.Contains(lowerCmd, "start") {
			a.startWatchMode()
		} else if strings.Contains(lowerCmd, "stop") {
			a.stopWatchMode()
		} else {
			a.ShowInfo("Specify 'start watch' or 'stop watch'.")
		}
	} else {
		a.ShowInfo("I'm not sure how to handle that request yet. Please try one of the preset tasks.")
	}
}

// createSettingsTab creates the settings tab
func (a *App) createSettingsTab() fyne.CanvasObject {
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
		// Ignore - not in our config structure
	})
	improvedCatCheck.SetChecked(false)

	collisionLabel := widget.NewLabel("Collision Strategy:")
	collisionSelect := widget.NewSelect([]string{"rename", "skip", "ask"}, func(value string) {
		a.cfg.Settings.Collision = value
	})
	collisionSelect.SetSelected(a.cfg.Settings.Collision)

	defaultDirLabel := widget.NewLabel("Default Directory:")
	defaultDirEntry := widget.NewEntry()
	defaultDirEntry.SetText(a.cfg.Directories.Default)
	defaultDirEntry.OnChanged = func(text string) {
		a.cfg.Directories.Default = text
	}

	saveButton := widget.NewButton("Save Settings", func() {
		a.saveConfig()
	})

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

// getDirectoryFiles returns a list of files in the given directory
func (a *App) getDirectoryFiles(dir string) ([]os.FileInfo, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	fileInfos := make([]os.FileInfo, 0, len(files))
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, info)
	}

	sort.Slice(fileInfos, func(i, j int) bool {
		if fileInfos[i].IsDir() && !fileInfos[j].IsDir() {
			return true
		}
		if !fileInfos[i].IsDir() && fileInfos[j].IsDir() {
			return false
		}
		return fileInfos[i].Name() < fileInfos[j].Name()
	})

	return fileInfos, nil
}

// updateDirectoryPath updates the path label and refreshes the file list
func (a *App) updateDirectoryPath(dir string) {
	a.cfg.Directories.Default = dir

	if a.pathLabel != nil {
		a.pathLabel.SetText("Location: " + dir)
	}
}

// findPathLabels finds all Label widgets in the UI
func (a *App) findPathLabels(obj fyne.CanvasObject) []fyne.CanvasObject {
	var labels []fyne.CanvasObject

	if label, ok := obj.(*widget.Label); ok {
		if strings.HasPrefix(label.Text, "Location:") {
			labels = append(labels, label)
		}
	}

	switch cont := obj.(type) {
	case *fyne.Container:
		for _, child := range cont.Objects {
			childLabels := a.findPathLabels(child)
			labels = append(labels, childLabels...)
		}
	case *container.Split:
		if cont.Leading != nil {
			labels = append(labels, a.findPathLabels(cont.Leading)...)
		}
		if cont.Trailing != nil {
			labels = append(labels, a.findPathLabels(cont.Trailing)...)
		}
	}

	return labels
}
