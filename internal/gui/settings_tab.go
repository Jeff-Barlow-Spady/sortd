package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// createSettingsTab creates the settings tab
func (a *App) createSettingsTab() fyne.CanvasObject {
	// --- General Settings ---
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

	generalSettingsCard := widget.NewCard("General Settings", "", container.NewVBox(
		dryRunCheck,
		createDirsCheck,
		backupCheck,
		improvedCatCheck,
		container.NewHBox(collisionLabel, collisionSelect),
		container.NewHBox(defaultDirLabel, defaultDirEntry),
	))

	// --- Watch Mode Settings ---
	watchDirsData := []string{}
	watchDirsList := widget.NewList(
		func() int {
			return len(a.cfg.WatchDirectories)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(a.cfg.WatchDirectories[i])
			watchDirsData = append(watchDirsData, a.cfg.WatchDirectories[i])
		},
	)

	watchDirsList.OnSelected = func(id widget.ListItemID) {
		a.selectedWatchDirIndex = id
	}
	watchDirsList.OnUnselected = func(id widget.ListItemID) {
		a.selectedWatchDirIndex = -1 // Reset when unselected
	}

	// --- Watch Directory Management ---
	watchDirsLabel := widget.NewLabel("Directories to Watch:")
	addWatchDirButton := widget.NewButton("Add Directory...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			path := uri.Path()
			// Check if directory already exists
			for _, dir := range a.cfg.WatchDirectories {
				if dir == path {
					a.ShowInfo(fmt.Sprintf("Directory '%s' is already being watched.", path))
					return
				}
			}
			a.cfg.WatchDirectories = append(a.cfg.WatchDirectories, path)
			watchDirsList.Refresh()
			watchDirsList.Unselect(a.selectedWatchDirIndex) // Unselect after adding
			a.saveConfig()
		}, a.mainWindow)
	})

	removeWatchDirButton := widget.NewButton("Remove Selected", func() {
		selected := a.selectedWatchDirIndex // Use the tracked index
		if selected < 0 || selected >= len(a.cfg.WatchDirectories) {
			a.ShowInfo("Please select a directory to remove.")
			return
		}
		a.cfg.WatchDirectories = append(a.cfg.WatchDirectories[:selected], a.cfg.WatchDirectories[selected+1:]...)
		watchDirsList.Refresh()
		a.selectedWatchDirIndex = -1 // Reset selection after removal
		watchDirsList.UnselectAll()  // Clear visual selection
		a.saveConfig()
	})

	watchModeCard := widget.NewCard("Watch Mode Settings", "", container.NewBorder(
		watchDirsLabel,
		container.NewHBox(addWatchDirButton, removeWatchDirButton),
		nil, nil,
		container.NewScroll(watchDirsList),
	))

	// --- Import/Export Settings ---
	importButton := widget.NewButton("Import Configuration...", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()

			// Use the function from utils.go
			newCfg, err := parseImportedConfig(reader)
			if err != nil {
				a.ShowError("Import Failed", err)
				return
			}

			// Set the new config
			*a.cfg = *newCfg

			// Update UI to reflect new settings
			dryRunCheck.SetChecked(a.cfg.Settings.DryRun)
			createDirsCheck.SetChecked(a.cfg.Settings.CreateDirs)
			backupCheck.SetChecked(a.cfg.Settings.Backup)
			collisionSelect.SetSelected(a.cfg.Settings.Collision)
			defaultDirEntry.SetText(a.cfg.Directories.Default)
			watchDirsList.Refresh()

			// Save the imported config
			a.saveConfig()
			a.ShowInfo("Configuration imported successfully")
		}, a.mainWindow)
	})

	exportButton := widget.NewButton("Export Configuration...", func() {
		dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil || writer == nil {
				return
			}
			defer writer.Close()

			// Determine format from extension
			format := "yaml" // Default
			if writer.URI().Extension() == ".json" {
				format = "json"
			}

			// Use the function from utils.go
			if err := exportConfig(a.cfg, writer, format); err != nil {
				a.ShowError("Export Failed", err)
				return
			}
			a.ShowInfo("Configuration exported successfully")
		}, a.mainWindow)
	})

	importExportCard := widget.NewCard("Import/Export", "", container.NewHBox(
		importButton,
		exportButton,
	))

	// --- Save Settings Button ---
	saveSettingsButton := widget.NewButton("Save Settings", func() {
		a.saveConfig()
		a.ShowInfo("Settings saved successfully")
	})

	// Combine all settings sections
	return container.NewVBox(
		generalSettingsCard,
		watchModeCard,
		importExportCard,
		saveSettingsButton,
	)
}

// NOTE: Removed duplicate function definitions for parseImportedConfig and exportConfig.
// These functions are now defined in utils.go and imported.
// Ensure that the import path "sortd/internal/config" is present if needed,
// although it should be if the code compiles correctly after this change.
