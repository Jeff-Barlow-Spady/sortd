package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sortd/pkg/types"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// createOrganizeTab creates the "Organize" tab
func (a *App) createOrganizeTab() fyne.CanvasObject {
	// Create path selection component
	pathEntry := widget.NewEntry()
	pathEntry.SetText(a.cfg.Directories.Default)
	pathEntry.OnChanged = func(path string) {
		a.cfg.Directories.Default = path
	}

	browseButton := widget.NewButton("Browse...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			path := uri.Path()
			pathEntry.SetText(path)
			a.cfg.Directories.Default = path
		}, a.mainWindow)
	})

	pathContainer := container.NewBorder(
		nil, nil, nil, browseButton,
		pathEntry,
	)

	// Directory preview
	dirPreview := widget.NewLabel("Select a directory above to see contents")
	a.pathLabel = widget.NewLabel("")
	if a.cfg.Directories.Default != "" {
		a.pathLabel.SetText(fmt.Sprintf("Location: %s", a.cfg.Directories.Default))
		// Update preview
		files, err := filepath.Glob(filepath.Join(a.cfg.Directories.Default, "*"))
		if err == nil && len(files) > 0 {
			sb := strings.Builder{}
			sb.WriteString("Directory Contents:\n")
			for i, file := range files {
				if i >= 10 {
					sb.WriteString(fmt.Sprintf("... and %d more files\n", len(files)-10))
					break
				}
				info, err := os.Stat(file)
				if err == nil {
					fileType := "File"
					if info.IsDir() {
						fileType = "Dir"
					}
					sb.WriteString(fmt.Sprintf("%s: %s (%d bytes)\n", fileType, filepath.Base(file), info.Size()))
				}
			}
			dirPreview.SetText(sb.String())
		} else if err == nil {
			dirPreview.SetText("Directory is empty")
		} else {
			dirPreview.SetText(fmt.Sprintf("Error reading directory: %v", err))
		}
	}

	// Refresh button
	refreshButton := widget.NewButton("Refresh Preview", func() {
		if a.cfg.Directories.Default == "" {
			return
		}

		a.pathLabel.SetText(fmt.Sprintf("Location: %s", a.cfg.Directories.Default))
		// Update preview
		files, err := filepath.Glob(filepath.Join(a.cfg.Directories.Default, "*"))
		if err == nil && len(files) > 0 {
			sb := strings.Builder{}
			sb.WriteString("Directory Contents:\n")
			for i, file := range files {
				if i >= 10 {
					sb.WriteString(fmt.Sprintf("... and %d more files\n", len(files)-10))
					break
				}
				info, err := os.Stat(file)
				if err == nil {
					fileType := "File"
					if info.IsDir() {
						fileType = "Dir"
					}
					sb.WriteString(fmt.Sprintf("%s: %s (%d bytes)\n", fileType, filepath.Base(file), info.Size()))
				}
			}
			dirPreview.SetText(sb.String())
		} else if err == nil {
			dirPreview.SetText("Directory is empty")
		} else {
			dirPreview.SetText(fmt.Sprintf("Error reading directory: %v", err))
		}
	})

	// Create organization patterns list
	patternData := make([]string, 0, len(a.cfg.Organize.Patterns))
	for _, pattern := range a.cfg.Organize.Patterns {
		patternData = append(patternData, fmt.Sprintf("%s -> %s", pattern.Match, pattern.Target))
	}

	patternsList := widget.NewList(
		func() int {
			return len(patternData)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(patternData[i])
		},
	)

	patternsList.OnSelected = func(id widget.ListItemID) {
		a.selectedPatternIndex = id
	}
	patternsList.OnUnselected = func(id widget.ListItemID) {
		a.selectedPatternIndex = -1 // Reset when unselected
	}

	// Add pattern button
	patternMatchEntry := widget.NewEntry()
	patternMatchEntry.SetPlaceHolder("e.g., *.jpg")

	patternTargetEntry := widget.NewEntry()
	patternTargetEntry.SetPlaceHolder("e.g., Images/")

	addPatternButton := widget.NewButton("Add Pattern", func() {
		match := patternMatchEntry.Text
		target := patternTargetEntry.Text

		if match == "" || target == "" {
			a.ShowError("Missing pattern info", fmt.Errorf("both pattern match and target must be specified"))
			return
		}

		newPattern := types.Pattern{
			Match:  match,
			Target: target,
		}

		// Add pattern and update UI
		a.cfg.Organize.Patterns = append(a.cfg.Organize.Patterns, newPattern)
		patternData = append(patternData, fmt.Sprintf("%s -> %s", match, target))
		patternsList.Refresh()
		patternsList.Unselect(a.selectedPatternIndex) // Unselect after adding

		// Save the new pattern
		a.saveConfig()

		// Clear entries
		patternMatchEntry.SetText("")
		patternTargetEntry.SetText("")
	})

	// Remove pattern button
	removePatternButton := widget.NewButton("Remove Selected", func() {
		selected := a.selectedPatternIndex // Use the tracked index
		if selected < 0 || selected >= len(a.cfg.Organize.Patterns) {
			a.ShowInfo("Please select a pattern to remove.")
			return
		}

		// Remove the pattern
		a.cfg.Organize.Patterns = append(a.cfg.Organize.Patterns[:selected], a.cfg.Organize.Patterns[selected+1:]...)
		patternData = append(patternData[:selected], patternData[selected+1:]...)
		patternsList.Refresh()
		a.selectedPatternIndex = -1 // Reset selection after removal
		patternsList.UnselectAll()  // Clear visual selection

		// Save the updated patterns
		a.saveConfig()
	})

	// Pattern form
	patternForm := widget.NewForm(
		widget.NewFormItem("Pattern", patternMatchEntry),
		widget.NewFormItem("Target", patternTargetEntry),
	)

	// Pattern buttons container
	patternButtonsContainer := container.NewHBox(
		addPatternButton,
		removePatternButton,
	)

	// Quick preset patterns
	presetLabel := widget.NewLabel("Quick Preset Rules:")

	imagesPresetButton := widget.NewButton("Images", func() {
		a.applyPresetRule("Images")
		// Refresh the pattern list
		patternData = make([]string, 0, len(a.cfg.Organize.Patterns))
		for _, pattern := range a.cfg.Organize.Patterns {
			patternData = append(patternData, fmt.Sprintf("%s -> %s", pattern.Match, pattern.Target))
		}
		patternsList.Refresh()
	})

	docsPresetButton := widget.NewButton("Documents", func() {
		a.applyPresetRule("Documents")
		// Refresh the pattern list
		patternData = make([]string, 0, len(a.cfg.Organize.Patterns))
		for _, pattern := range a.cfg.Organize.Patterns {
			patternData = append(patternData, fmt.Sprintf("%s -> %s", pattern.Match, pattern.Target))
		}
		patternsList.Refresh()
	})

	videosPresetButton := widget.NewButton("Videos", func() {
		a.applyPresetRule("Videos")
		// Refresh the pattern list
		patternData = make([]string, 0, len(a.cfg.Organize.Patterns))
		for _, pattern := range a.cfg.Organize.Patterns {
			patternData = append(patternData, fmt.Sprintf("%s -> %s", pattern.Match, pattern.Target))
		}
		patternsList.Refresh()
	})

	audioPresetButton := widget.NewButton("Audio", func() {
		a.applyPresetRule("Audio")
		// Refresh the pattern list
		patternData = make([]string, 0, len(a.cfg.Organize.Patterns))
		for _, pattern := range a.cfg.Organize.Patterns {
			patternData = append(patternData, fmt.Sprintf("%s -> %s", pattern.Match, pattern.Target))
		}
		patternsList.Refresh()
	})

	// AI-powered organization
	aiOrganizerInput := widget.NewMultiLineEntry()
	aiOrganizerInput.SetPlaceHolder("e.g., Organize all my photos into a Photos folder")

	executeAIButton := widget.NewButton("Execute", func() {
		command := aiOrganizerInput.Text
		if command == "" {
			a.ShowInfo("Please enter a description of what you want to organize.")
			return
		}

		a.handleNaturalLanguageCommand(command)
		aiOrganizerInput.SetText("")
	})

	// Main organize button
	organizeButton := widget.NewButton("Organize Now", func() {
		if a.cfg.Directories.Default == "" {
			a.ShowError("No Directory Selected", fmt.Errorf("please select a directory to organize"))
			return
		}

		// Check if patterns exist
		if len(a.cfg.Organize.Patterns) == 0 {
			a.ShowError("No Patterns Defined", fmt.Errorf("please define at least one organization pattern"))
			return
		}

		// Set the engine's dry run mode from the config
		a.organizeEngine.SetDryRun(a.cfg.Settings.DryRun)

		// Run organization
		results, err := a.organizeEngine.OrganizeDirectory(a.cfg.Directories.Default)
		if err != nil {
			a.ShowError("Organization Failed", err)
			return
		}

		// Count successful and failed operations
		var movedCount, errorCount int
		for _, result := range results {
			if result.Error != nil {
				errorCount++
			} else if result.Moved {
				movedCount++
			}
		}

		// Show results
		if errorCount > 0 {
			a.ShowError("Organization Partially Completed", fmt.Errorf("moved %d files, encountered %d errors", movedCount, errorCount))
		} else if a.cfg.Settings.DryRun {
			a.ShowInfo(fmt.Sprintf("Dry run complete. Would organize %d files.", movedCount))
		} else {
			a.ShowInfo(fmt.Sprintf("Organization complete. %d files organized.", movedCount))
		}

		// Refresh the directory preview
		refreshButton.OnTapped()
	})

	// Watch mode toggle button
	var watchButton *widget.Button
	watchButton = widget.NewButton("Start Watch Mode", func() {
		if a.IsDaemonRunning() {
			a.stopWatchMode()
			watchButton.SetText("Start Watch Mode")
		} else {
			a.startWatchMode()
			watchButton.SetText("Stop Watch Mode")
		}
	})

	// Update watch button text based on initial state
	if a.IsDaemonRunning() {
		watchButton.SetText("Stop Watch Mode")
	}

	// Create main tool buttons container
	mainButtonsContainer := container.NewHBox(
		layout.NewSpacer(),
		organizeButton,
		watchButton,
	)

	// Create preset buttons container
	presetButtonsContainer := container.NewHBox(
		presetLabel,
		imagesPresetButton,
		docsPresetButton,
		videosPresetButton,
		audioPresetButton,
	)

	// Create the layout for the organize tab
	return container.NewVBox(
		widget.NewLabel("Directory to Organize:"),
		pathContainer,
		a.pathLabel,
		container.NewBorder(nil, nil, nil, refreshButton, dirPreview),
		widget.NewCard("Organization Patterns", "Define how files should be organized",
			container.NewBorder(
				nil,
				container.NewVBox(
					patternForm,
					patternButtonsContainer,
					presetButtonsContainer,
				),
				nil, nil,
				container.NewScroll(patternsList),
			),
		),
		widget.NewCard("Smart Organization", "",
			container.NewBorder(
				nil, nil, nil, executeAIButton,
				aiOrganizerInput,
			),
		),
		mainButtonsContainer,
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
				errorMsg := fmt.Sprintf("Encountered %d errors:\\n%s", errorCount, strings.Join(errors, "\\n"))
				msg += "\\n" + errorMsg
				a.ShowError("Organization encountered errors", fmt.Errorf(strings.Join(errors, "\\n"))) // Show first error
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
