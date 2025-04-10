package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sortd/pkg/types"
	"sortd/pkg/workflow"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// WizardStep represents a single step in the wizard
type WizardStep struct {
	title       string
	description string
	content     fyne.CanvasObject
	onNext      func() bool // Function to execute when moving to the next step, returns true if valid
}

// WorkflowWizard provides a step-by-step interface for creating workflows
type WorkflowWizard struct {
	app           *App
	window        fyne.Window
	currentStep   int
	workflowData  types.Workflow
	triggerType   string
	conditionType string
	actionType    string
	stepContent   fyne.CanvasObject
	steps         []WizardStep // All wizard steps

	// Navigation buttons
	nextButton   *widget.Button
	backButton   *widget.Button
	doneButton   *widget.Button
	cancelButton *widget.Button

	// Container for steps
	contentContainer *fyne.Container
	stepIndicator    *widget.Label

	// Workflow visualization
	visualPreview *fyne.Container

	// Edit mode flag
	isEditMode bool

	// Update progress function
	updateStepProgress func()
}

// NewWorkflowWizard creates a new workflow creation wizard
func NewWorkflowWizard(app *App) *WorkflowWizard {
	w := &WorkflowWizard{
		app:         app,
		window:      app.fyneApp.NewWindow("Create Workflow"),
		currentStep: 0,
		workflowData: types.Workflow{
			ID:          fmt.Sprintf("workflow-%d", time.Now().Unix()),
			Name:        "New Workflow",
			Description: "",
			Enabled:     true,
			Priority:    5,
			Trigger: types.Trigger{
				Type: types.FileCreated,
			},
			Conditions: []types.Condition{},
			Actions:    []types.Action{},
		},
	}

	// Set up the wizard window
	w.window.Resize(fyne.NewSize(950, 650))

	// Set up step indicator
	w.stepIndicator = widget.NewLabelWithStyle("Step 1 of 5", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Create navigation buttons
	w.backButton = widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), func() {
		if w.currentStep > 0 {
			w.currentStep--
			w.updateStepContent()
		}
	})

	w.nextButton = widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), func() {
		// Run validation check for current step if available
		if w.currentStep < len(w.steps) && w.steps[w.currentStep].onNext != nil {
			if !w.steps[w.currentStep].onNext() {
				// Validation failed, stay on current step
				return
			}
		}
		w.currentStep++
		w.updateStepContent()
	})

	w.doneButton = widget.NewButtonWithIcon("Finish", theme.ConfirmIcon(), func() {
		w.saveWorkflow()
	})

	w.cancelButton = widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		// Confirm cancellation with unsaved changes
		dialog.ShowConfirm("Cancel Workflow Creation",
			"Are you sure you want to cancel? Any unsaved changes will be lost.",
			func(confirmed bool) {
				if confirmed {
					w.window.Close()
				}
			},
			w.window)
	})

	// Create content container
	w.contentContainer = container.NewStack()

	// Create visualization preview with padding
	w.visualPreview = container.NewVBox()

	// Define all wizard steps
	w.steps = []WizardStep{
		{
			title:       "Basic Information",
			description: "Enter the general information about this workflow",
			onNext: func() bool {
				// Validate name not empty
				if strings.TrimSpace(w.workflowData.Name) == "" {
					dialog.ShowError(fmt.Errorf("workflow name cannot be empty"), w.window)
					return false
				}
				return true
			},
		},
		{
			title:       "Triggers",
			description: "Configure what will trigger this workflow",
			onNext:      func() bool { return true }, // Always valid
		},
		{
			title:       "Conditions",
			description: "Add conditions to further refine when this workflow runs",
			onNext:      func() bool { return true }, // Always valid
		},
		{
			title:       "Actions",
			description: "Define what actions will be taken when triggered",
			onNext: func() bool {
				// Must have at least one action
				if len(w.workflowData.Actions) == 0 {
					dialog.ShowError(fmt.Errorf("workflow must have at least one action"), w.window)
					return false
				}
				return true
			},
		},
		{
			title:       "Review",
			description: "Review and finalize your workflow",
			onNext:      func() bool { return true }, // Always valid
		},
	}

	// Initial layout setup
	w.updateStepContent()

	// Create preview header
	previewHeader := container.NewVBox(
		widget.NewLabelWithStyle("Workflow Preview", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
	)

	// Create a scroll container with the preview content
	scrollContainer := container.NewScroll(w.visualPreview)

	// Put everything in a border layout to ensure the scroll gets the remaining space
	previewContainer := container.NewBorder(
		previewHeader,   // Top is header
		nil,             // No bottom content
		nil,             // No left content
		nil,             // No right content
		scrollContainer, // The scroll gets all remaining space
	)

	// Progress indicator with step number
	progressBar := widget.NewProgressBar()
	progressBar.Min = 0
	progressBar.Max = float64(len(w.steps) - 1)
	progressBar.SetValue(0)

	// Create the split container with a 0.65 offset (65% left, 35% right)
	splitContainer := container.NewHSplit(
		container.NewBorder(
			container.NewVBox(
				w.stepIndicator,
				progressBar,
			),
			nil,
			nil,
			nil,
			w.contentContainer,
		),
		previewContainer,
	)
	splitContainer.Offset = 0.65

	// Update the progress indicator when step changes
	w.updateStepProgress = func() {
		w.stepIndicator.SetText(fmt.Sprintf("Step %d of %d: %s",
			w.currentStep+1,
			len(w.steps),
			w.steps[w.currentStep].title))
		progressBar.SetValue(float64(w.currentStep))
	}
	w.updateStepProgress()

	// Create a toolbar for actions
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			// Add action button - functionality depends on current step
			if w.currentStep == 2 {
				// Add condition
				w.addNewCondition()
			} else if w.currentStep == 3 {
				// Add action
				w.addNewAction()
			}
		}),
		widget.NewToolbarAction(theme.ContentClearIcon(), func() {
			// Clear button - functionality depends on current step
			if w.currentStep == 2 && len(w.workflowData.Conditions) > 0 {
				// Clear all conditions
				dialog.ShowConfirm("Clear All Conditions",
					"Are you sure you want to remove all conditions?",
					func(confirmed bool) {
						if confirmed {
							w.workflowData.Conditions = []types.Condition{}
							w.updateStepContent()
						}
					},
					w.window)
			} else if w.currentStep == 3 && len(w.workflowData.Actions) > 0 {
				// Clear all actions
				dialog.ShowConfirm("Clear All Actions",
					"Are you sure you want to remove all actions?",
					func(confirmed bool) {
						if confirmed {
							w.workflowData.Actions = []types.Action{}
							w.updateStepContent()
						}
					},
					w.window)
			}
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			// Show help for current step
			helpText := "No help available for this step."
			switch w.currentStep {
			case 0:
				helpText = "Enter a unique name and description for your workflow. " +
					"The priority determines the order in which workflows are processed."
			case 1:
				helpText = "Select what event will trigger this workflow. " +
					"For example, a new file being created or modified."
			case 2:
				helpText = "Add conditions to narrow down when the workflow will run. " +
					"For example, only run for certain file types or sizes."
			case 3:
				helpText = "Add actions that will be performed when the workflow runs. " +
					"You must add at least one action."
			case 4:
				helpText = "Review all settings before finalizing. " +
					"You can go back to any step to make changes."
			}
			dialog.ShowInformation("Help: "+w.steps[w.currentStep].title, helpText, w.window)
		}),
	)

	// Set the window content
	w.window.SetContent(
		container.NewBorder(
			nil, // Top
			container.NewBorder(
				nil, nil, nil, nil,
				container.NewHBox(
					layout.NewSpacer(),
					w.cancelButton,
					w.backButton,
					w.nextButton,
					w.doneButton,
				),
			), // Bottom
			toolbar, // Left
			nil,     // Right
			splitContainer,
		),
	)

	return w
}

// Show displays the workflow wizard
func (w *WorkflowWizard) Show() {
	w.window.Show()
}

// updateStepContent changes the content based on the current step
func (w *WorkflowWizard) updateStepContent() {
	// Update button states
	w.backButton.Disable()
	w.nextButton.Show()
	w.doneButton.Hide()

	if w.currentStep > 0 {
		w.backButton.Enable()
	}

	// Last step shows done instead of next
	if w.currentStep == len(w.steps)-1 {
		w.nextButton.Hide()
		w.doneButton.Show()
	}

	// Update step indicator
	w.updateStepProgress()

	// Update content based on step
	switch w.currentStep {
	case 0:
		w.contentContainer.Objects = []fyne.CanvasObject{w.createBasicInfoStep()}
	case 1:
		w.contentContainer.Objects = []fyne.CanvasObject{w.createTriggerStep()}
	case 2:
		w.contentContainer.Objects = []fyne.CanvasObject{w.createConditionsStep()}
	case 3:
		w.contentContainer.Objects = []fyne.CanvasObject{w.createActionsStep()}
	case 4:
		w.contentContainer.Objects = []fyne.CanvasObject{w.createReviewStep()}
	}

	w.contentContainer.Refresh()
	w.updateVisualization()
}

// createBasicInfoStep creates the basic workflow information step
func (w *WorkflowWizard) createBasicInfoStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		w.steps[w.currentStep].description,
		fyne.TextAlignLeading,
		fyne.TextStyle{Italic: true})

	nameEntry := widget.NewEntry()
	nameEntry.SetText(w.workflowData.Name)
	nameEntry.OnChanged = func(value string) {
		w.workflowData.Name = value
		w.updateVisualization()
	}

	idEntry := widget.NewEntry()
	idEntry.SetText(w.workflowData.ID)
	idEntry.OnChanged = func(value string) {
		w.workflowData.ID = value
	}

	descEntry := widget.NewMultiLineEntry()
	descEntry.SetPlaceHolder("Enter a description of this workflow")
	descEntry.SetText(w.workflowData.Description)
	descEntry.OnChanged = func(value string) {
		w.workflowData.Description = value
		w.updateVisualization()
	}

	enabledCheck := widget.NewCheck("Enabled", func(value bool) {
		w.workflowData.Enabled = value
		w.updateVisualization()
	})
	enabledCheck.SetChecked(w.workflowData.Enabled)

	prioritySlider := widget.NewSlider(1, 10)
	prioritySlider.Value = float64(w.workflowData.Priority)
	priorityValue := widget.NewLabel(fmt.Sprintf("%d", w.workflowData.Priority))

	prioritySlider.OnChanged = func(value float64) {
		w.workflowData.Priority = int(value)
		priorityValue.SetText(fmt.Sprintf("%d", w.workflowData.Priority))
		w.updateVisualization()
	}

	// Container for priority with both slider and value
	priorityContainer := container.NewBorder(
		nil, nil,
		widget.NewLabel("Priority: "), priorityValue,
		prioritySlider)

	form := widget.NewForm(
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("ID", idEntry),
		widget.NewFormItem("Description", descEntry),
		widget.NewFormItem("Status", enabledCheck),
		widget.NewFormItem("Priority", priorityContainer),
	)

	// Help text for priority
	helpText := widget.NewRichTextFromMarkdown("**Workflow Priority**\n\nHigher priority (10) workflows are processed before lower priority (1) workflows.")

	return container.NewBorder(
		container.NewVBox(
			title,
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewCard("Help", "", helpText),
		),
		nil, nil,
		form,
	)
}

// createTriggerStep creates the trigger configuration step
func (w *WorkflowWizard) createTriggerStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Step 2: Set Trigger", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	triggerTypes := []string{
		"File Created",
		"File Modified",
		"File Pattern Match",
		"Manual",
		"Scheduled",
	}

	triggerSelect := widget.NewSelect(triggerTypes, func(value string) {
		w.triggerType = value
		switch value {
		case "File Created":
			w.workflowData.Trigger.Type = types.FileCreated
		case "File Modified":
			w.workflowData.Trigger.Type = types.FileModified
		case "File Pattern Match":
			w.workflowData.Trigger.Type = types.FilePatternMatch
		case "Manual":
			w.workflowData.Trigger.Type = types.ManualTrigger
		case "Scheduled":
			w.workflowData.Trigger.Type = types.ScheduledTrigger
		}
	})

	// Set initial selection based on current data
	switch w.workflowData.Trigger.Type {
	case types.FileCreated:
		triggerSelect.SetSelected("File Created")
	case types.FileModified:
		triggerSelect.SetSelected("File Modified")
	case types.FilePatternMatch:
		triggerSelect.SetSelected("File Pattern Match")
	case types.ManualTrigger:
		triggerSelect.SetSelected("Manual")
	case types.ScheduledTrigger:
		triggerSelect.SetSelected("Scheduled")
	default:
		triggerSelect.SetSelected("File Created")
	}

	patternEntry := widget.NewEntry()
	patternEntry.SetText(w.workflowData.Trigger.Pattern)
	patternEntry.SetPlaceHolder("e.g., *.{jpg,png,pdf}")
	patternEntry.OnChanged = func(value string) {
		w.workflowData.Trigger.Pattern = value
	}

	scheduleEntry := widget.NewEntry()
	scheduleEntry.SetText(w.workflowData.Trigger.Schedule)
	scheduleEntry.SetPlaceHolder("e.g., 0 * * * * (cron format)")
	scheduleEntry.OnChanged = func(value string) {
		w.workflowData.Trigger.Schedule = value
	}

	helpText := widget.NewLabel("Pattern triggers use file glob patterns to match files.")

	// Simple pattern presets
	imagePreset := widget.NewButton("Image Files", func() {
		patternEntry.SetText("*.{jpg,jpeg,png,gif,webp,svg}")
	})

	docPreset := widget.NewButton("Document Files", func() {
		patternEntry.SetText("*.{pdf,doc,docx,txt,rtf,odt}")
	})

	presetBox := container.NewHBox(
		widget.NewLabel("Presets:"),
		imagePreset,
		docPreset,
	)

	return container.NewVBox(
		title,
		widget.NewForm(
			widget.NewFormItem("Trigger Type", triggerSelect),
			widget.NewFormItem("File Pattern", patternEntry),
			widget.NewFormItem("Schedule", scheduleEntry),
		),
		helpText,
		presetBox,
	)
}

// createConditionsStep creates the conditions configuration step
func (w *WorkflowWizard) createConditionsStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Step 3: Set Conditions", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Display existing conditions
	var conditionList *widget.List
	var selectedConditionIndex int = -1 // Track selected index

	conditionList = widget.NewList(
		func() int {
			return len(w.workflowData.Conditions)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Type: "),
				widget.NewLabel("Condition Type"),
				widget.NewLabel(" | Field: "),
				widget.NewLabel("Field Name"),
				widget.NewLabel(" | Value: "),
				widget.NewLabel("Value"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(w.workflowData.Conditions) {
				return
			}
			cond := w.workflowData.Conditions[id]
			container := obj.(*fyne.Container)

			typeLabel := container.Objects[1].(*widget.Label)
			fieldLabel := container.Objects[3].(*widget.Label)
			valueLabel := container.Objects[5].(*widget.Label)

			typeLabel.SetText(string(cond.Type))
			fieldLabel.SetText(cond.Field)
			valueLabel.SetText(fmt.Sprintf("%s %s %s", cond.Operator, cond.Value, cond.ValueUnit))
		},
	)

	conditionList.OnSelected = func(id widget.ListItemID) {
		selectedConditionIndex = int(id)
	}
	conditionList.OnUnselected = func(id widget.ListItemID) {
		if selectedConditionIndex == int(id) {
			selectedConditionIndex = -1
		}
	}

	// Condition type selection
	conditionTypes := []string{
		"File Size",
		"File Type",
		"File Name",
		"File Age",
	}

	conditionTypeSelect := widget.NewSelect(conditionTypes, func(value string) {
		w.conditionType = value
	})
	conditionTypeSelect.PlaceHolder = "Select condition type..."

	// Fields depending on type
	fieldEntry := widget.NewEntry()
	fieldEntry.SetPlaceHolder("Field name")

	// Operators
	operatorSelect := widget.NewSelect([]string{
		"Equals",
		"Not Equals",
		"Contains",
		"Starts With",
		"Ends With",
		"Greater Than",
		"Less Than",
	}, nil)
	operatorSelect.PlaceHolder = "Select operator..."

	valueEntry := widget.NewEntry()
	valueEntry.SetPlaceHolder("Value")

	unitEntry := widget.NewEntry()
	unitEntry.SetPlaceHolder("Unit (e.g., MB, KB, days)")

	// Add button
	addButton := widget.NewButton("Add Condition", func() {
		if conditionTypeSelect.Selected == "" || operatorSelect.Selected == "" || valueEntry.Text == "" {
			w.app.ShowError("Missing Fields", fmt.Errorf("please fill in all required fields"))
			return
		}

		var condType types.ConditionType
		switch conditionTypeSelect.Selected {
		case "File Size":
			condType = types.FileSizeCondition
			fieldEntry.Text = "size"
		case "File Type":
			condType = types.FileTypeCondition
			fieldEntry.Text = "type"
		case "File Name":
			condType = types.FileNameCondition
			fieldEntry.Text = "name"
		case "File Age":
			condType = types.FileAgeCondition
			fieldEntry.Text = "age"
		}

		var opType types.OperatorType
		switch operatorSelect.Selected {
		case "Equals":
			opType = types.Equals
		case "Not Equals":
			opType = types.NotEquals
		case "Contains":
			opType = types.Contains
		case "Starts With":
			opType = types.StartsWith
		case "Ends With":
			opType = types.EndsWith
		case "Greater Than":
			opType = types.GreaterThan
		case "Less Than":
			opType = types.LessThan
		}

		newCondition := types.Condition{
			Type:      condType,
			Field:     fieldEntry.Text,
			Operator:  opType,
			Value:     valueEntry.Text,
			ValueUnit: unitEntry.Text,
		}

		w.workflowData.Conditions = append(w.workflowData.Conditions, newCondition)
		conditionList.Refresh()
		w.updateVisualization()

		// Reset inputs
		conditionTypeSelect.ClearSelected()
		operatorSelect.ClearSelected()
		valueEntry.SetText("")
		unitEntry.SetText("")
	})

	// Remove button (removes selected condition)
	removeButton := widget.NewButton("Remove Selected", func() {
		if selectedConditionIndex >= 0 && selectedConditionIndex < len(w.workflowData.Conditions) {
			w.workflowData.Conditions = append(
				w.workflowData.Conditions[:selectedConditionIndex],
				w.workflowData.Conditions[selectedConditionIndex+1:]...,
			)
			conditionList.Refresh()
			w.updateVisualization()
		}
	})

	// Create a fixed height container for the list with scroll
	listContainer := container.NewBorder(
		widget.NewLabel("Existing Conditions:"),
		nil,
		nil,
		nil,
		container.NewVScroll(conditionList),
	)

	// Use a fixed height with VBox to ensure the list gets enough space
	listWithHeight := container.NewVBox(
		listContainer,
		layout.NewSpacer(),
	)

	return container.NewBorder(
		container.NewVBox(
			title,
			listWithHeight,
		),
		container.NewVBox(
			widget.NewSeparator(),
			widget.NewLabel("Add New Condition:"),
			container.NewGridWithColumns(2,
				widget.NewForm(
					widget.NewFormItem("Condition Type", conditionTypeSelect),
					widget.NewFormItem("Operator", operatorSelect),
				),
				widget.NewForm(
					widget.NewFormItem("Value", valueEntry),
					widget.NewFormItem("Unit", unitEntry),
				),
			),
			container.NewHBox(
				layout.NewSpacer(),
				addButton,
				removeButton,
			),
		),
		nil,
		nil,
		nil,
	)
}

// createActionsStep creates the actions configuration step
func (w *WorkflowWizard) createActionsStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Step 4: Set Actions", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Display existing actions
	var actionList *widget.List
	var selectedActionIndex int = -1 // Track selected index

	actionList = widget.NewList(
		func() int {
			return len(w.workflowData.Actions)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Type: "),
				widget.NewLabel("Action Type"),
				widget.NewLabel(" | Target: "),
				widget.NewLabel("Target"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(w.workflowData.Actions) {
				return
			}
			action := w.workflowData.Actions[id]
			container := obj.(*fyne.Container)

			typeLabel := container.Objects[1].(*widget.Label)
			targetLabel := container.Objects[3].(*widget.Label)

			typeLabel.SetText(string(action.Type))
			targetLabel.SetText(action.Target)
		},
	)

	actionList.OnSelected = func(id widget.ListItemID) {
		selectedActionIndex = int(id)
	}
	actionList.OnUnselected = func(id widget.ListItemID) {
		if selectedActionIndex == int(id) {
			selectedActionIndex = -1
		}
	}

	// Action type selection
	actionTypes := []string{
		"Move File",
		"Copy File",
		"Rename File",
		"Tag File",
		"Delete File",
		"Execute Command",
	}

	actionTypeSelect := widget.NewSelect(actionTypes, func(value string) {
		w.actionType = value
	})
	actionTypeSelect.PlaceHolder = "Select action type..."

	targetEntry := widget.NewEntry()
	targetEntry.SetPlaceHolder("Target path, name, or command")

	browseButton := widget.NewButton("Browse...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				targetEntry.SetText(uri.Path())
			}
		}, w.window)
	})

	// Options
	createDirCheck := widget.NewCheck("Create target directory if it doesn't exist", nil)
	overwriteCheck := widget.NewCheck("Overwrite existing files", nil)

	// Add button
	addButton := widget.NewButton("Add Action", func() {
		if actionTypeSelect.Selected == "" || targetEntry.Text == "" {
			w.app.ShowError("Missing Fields", fmt.Errorf("please fill in all required fields"))
			return
		}

		var actionType types.ActionType
		switch actionTypeSelect.Selected {
		case "Move File":
			actionType = types.MoveAction
		case "Copy File":
			actionType = types.CopyAction
		case "Rename File":
			actionType = types.RenameAction
		case "Tag File":
			actionType = types.TagAction
		case "Delete File":
			actionType = types.DeleteAction
		case "Execute Command":
			actionType = types.ExecuteAction
		}

		options := make(map[string]string)

		if createDirCheck.Checked {
			options["createTargetDir"] = "true"
		}

		if overwriteCheck.Checked {
			options["overwrite"] = "true"
		}

		newAction := types.Action{
			Type:    actionType,
			Target:  targetEntry.Text,
			Options: options,
		}

		w.workflowData.Actions = append(w.workflowData.Actions, newAction)
		actionList.Refresh()
		w.updateVisualization()

		// Reset inputs
		actionTypeSelect.ClearSelected()
		targetEntry.SetText("")
		createDirCheck.SetChecked(false)
		overwriteCheck.SetChecked(false)
	})

	// Remove button (removes selected action)
	removeButton := widget.NewButton("Remove Selected", func() {
		if selectedActionIndex >= 0 && selectedActionIndex < len(w.workflowData.Actions) {
			w.workflowData.Actions = append(
				w.workflowData.Actions[:selectedActionIndex],
				w.workflowData.Actions[selectedActionIndex+1:]...,
			)
			actionList.Refresh()
			w.updateVisualization()
		}
	})

	// Create a fixed height container for the list with scroll
	listContainer := container.NewBorder(
		widget.NewLabel("Existing Actions:"),
		nil,
		nil,
		nil,
		container.NewVScroll(actionList),
	)

	// Use a fixed height with VBox to ensure the list gets enough space
	listWithHeight := container.NewVBox(
		listContainer,
		layout.NewSpacer(),
	)

	return container.NewBorder(
		container.NewVBox(
			title,
			listWithHeight,
		),
		container.NewVBox(
			widget.NewSeparator(),
			widget.NewLabel("Add New Action:"),
			container.NewGridWithColumns(2,
				widget.NewForm(
					widget.NewFormItem("Action Type", actionTypeSelect),
					widget.NewFormItem("Target", container.NewBorder(nil, nil, nil, browseButton, targetEntry)),
				),
			),
			container.NewVBox(
				createDirCheck,
				overwriteCheck,
			),
			container.NewHBox(
				layout.NewSpacer(),
				addButton,
				removeButton,
			),
		),
		nil,
		nil,
		nil,
	)
}

// createReviewStep creates the final review step
func (w *WorkflowWizard) createReviewStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Step 5: Review & Save", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Summary of the workflow
	nameSummary := widget.NewLabel(fmt.Sprintf("Name: %s", w.workflowData.Name))
	idSummary := widget.NewLabel(fmt.Sprintf("ID: %s", w.workflowData.ID))
	statusSummary := widget.NewLabel(fmt.Sprintf("Status: %s", map[bool]string{true: "Enabled", false: "Disabled"}[w.workflowData.Enabled]))
	triggerSummary := widget.NewLabel(fmt.Sprintf("Trigger: %s", w.workflowData.Trigger.Type))
	conditionsSummary := widget.NewLabel(fmt.Sprintf("Conditions: %d configured", len(w.workflowData.Conditions)))
	actionsSummary := widget.NewLabel(fmt.Sprintf("Actions: %d configured", len(w.workflowData.Actions)))

	testButton := widget.NewButton("Test Workflow (Dry Run)", func() {
		w.testWorkflow()
	})

	return container.NewVBox(
		title,
		widget.NewCard("Workflow Summary", "",
			container.NewVBox(
				nameSummary,
				idSummary,
				statusSummary,
				triggerSummary,
				conditionsSummary,
				actionsSummary,
			),
		),
		testButton,
	)
}

// saveWorkflow saves the created workflow
func (w *WorkflowWizard) saveWorkflow() {
	// Check if the workflow is valid
	if w.workflowData.ID == "" || w.workflowData.Name == "" {
		w.app.ShowError("Invalid Workflow", fmt.Errorf("workflow must have an ID and name"))
		return
	}

	if len(w.workflowData.Actions) == 0 {
		w.app.ShowError("Invalid Workflow", fmt.Errorf("workflow must have at least one action"))
		return
	}

	// Get workflow directory from app config
	home, err := os.UserHomeDir()
	if err != nil {
		w.app.ShowError("Error Saving Workflow", fmt.Errorf("failed to get home directory: %w", err))
		return
	}

	configDir := filepath.Join(home, ".config", "sortd", "workflows")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		w.app.ShowError("Error Saving Workflow", fmt.Errorf("failed to create workflows directory: %w", err))
		return
	}

	// Create workflow manager
	manager, err := workflow.NewManager(configDir)
	if err != nil {
		w.app.ShowError("Error Saving Workflow", fmt.Errorf("failed to initialize workflow manager: %w", err))
		return
	}

	// Add workflow
	if err := manager.AddWorkflow(w.workflowData); err != nil {
		w.app.ShowError("Error Saving Workflow", fmt.Errorf("failed to save workflow: %w", err))
		return
	}

	// Get file path for informational purposes
	fileName := w.workflowData.ID
	if !strings.HasSuffix(fileName, ".yaml") {
		fileName += ".yaml"
	}
	filePath := filepath.Join(configDir, fileName)

	dialog.ShowInformation("Workflow Created",
		fmt.Sprintf("Workflow '%s' created successfully and saved to '%s'",
			w.workflowData.Name, filePath), w.window)

	w.window.Close()
}

// testWorkflow performs a dry run test of the workflow
func (w *WorkflowWizard) testWorkflow() {
	// Validate workflow first
	if w.workflowData.ID == "" || w.workflowData.Name == "" || len(w.workflowData.Actions) == 0 {
		w.app.ShowError("Invalid Workflow", fmt.Errorf("workflow must have an ID, name, and at least one action"))
		return
	}

	// Show file selection dialog
	dialog.ShowFileOpen(func(file fyne.URIReadCloser, err error) {
		if err != nil || file == nil {
			return // User canceled or error
		}

		filePath := file.URI().Path()
		file.Close()

		// Create temporary workflow manager with dry run mode
		home, err := os.UserHomeDir()
		if err != nil {
			w.app.ShowError("Test Error", fmt.Errorf("failed to get home directory: %w", err))
			return
		}

		configDir := filepath.Join(home, ".config", "sortd", "workflows")
		manager, err := workflow.NewManager(configDir)
		if err != nil {
			w.app.ShowError("Test Error", fmt.Errorf("failed to initialize workflow manager: %w", err))
			return
		}

		// Enable dry run mode
		manager.SetDryRun(true)

		// Creating a temporary ID for the workflow
		origID := w.workflowData.ID
		tempID := fmt.Sprintf("temp-%d", time.Now().Unix())
		w.workflowData.ID = tempID

		// Add workflow temporarily
		if err := manager.AddWorkflow(w.workflowData); err != nil {
			w.workflowData.ID = origID // Restore original ID
			w.app.ShowError("Test Error", fmt.Errorf("failed to setup workflow for testing: %w", err))
			return
		}

		// Run the workflow in dry run mode
		result, err := manager.ExecuteWorkflow(tempID, filePath)

		// Clean up the temporary workflow
		manager.DeleteWorkflow(tempID)

		// Restore original ID
		w.workflowData.ID = origID

		// Handle result
		if err != nil {
			w.app.ShowError("Test Error", fmt.Errorf("failed to execute workflow: %w", err))
			return
		}

		// Show test result
		if result.Success {
			message := fmt.Sprintf("Dry run successful on file: %s\n\n%s\n\nNo actual changes were made.",
				filepath.Base(filePath), result.Message)
			dialog.ShowInformation("Test Successful", message, w.window)
		} else {
			message := fmt.Sprintf("Dry run failed on file: %s\n\n%s\n\nError: %v",
				filepath.Base(filePath), result.Message, result.Error)
			dialog.ShowInformation("Test Failed", message, w.window)
		}
	}, w.window)
}

// updateVisualization updates the workflow visualization preview
func (w *WorkflowWizard) updateVisualization() {
	w.visualPreview.Objects = nil

	// Add header with workflow name and description
	nameLabel := widget.NewLabelWithStyle(w.workflowData.Name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	nameLabel.Alignment = fyne.TextAlignCenter
	w.visualPreview.Add(nameLabel)

	if w.workflowData.Description != "" {
		descLabel := widget.NewLabel(w.workflowData.Description)
		descLabel.Wrapping = fyne.TextWrapWord
		w.visualPreview.Add(descLabel)
	}

	w.visualPreview.Add(widget.NewSeparator())
	w.visualPreview.Add(widget.NewLabel("")) // Add spacing

	// Add enabled/disabled status and priority
	statusText := "✓ Enabled"
	if !w.workflowData.Enabled {
		statusText = "✗ Disabled"
	}
	w.visualPreview.Add(widget.NewLabel(statusText))
	w.visualPreview.Add(widget.NewLabel(fmt.Sprintf("Priority: %d", w.workflowData.Priority)))
	w.visualPreview.Add(widget.NewLabel("")) // Add spacing

	// Add trigger
	triggerIcon := "⚡" // Lightning bolt
	w.visualPreview.Add(widget.NewLabelWithStyle(
		fmt.Sprintf("%s Trigger: %s", triggerIcon, w.workflowData.Trigger.Type),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	))

	if w.workflowData.Trigger.Pattern != "" {
		patternLabel := widget.NewLabel(fmt.Sprintf("  Pattern: %s", w.workflowData.Trigger.Pattern))
		patternLabel.Wrapping = fyne.TextWrapWord
		w.visualPreview.Add(patternLabel)
	}

	if w.workflowData.Trigger.Schedule != "" {
		w.visualPreview.Add(widget.NewLabel(fmt.Sprintf("  Schedule: %s", w.workflowData.Trigger.Schedule)))
	}

	w.visualPreview.Add(widget.NewLabel("")) // Add spacing
	w.visualPreview.Add(widget.NewSeparator())
	w.visualPreview.Add(widget.NewLabel("")) // Add spacing

	// Add conditions
	if len(w.workflowData.Conditions) > 0 {
		filterIcon := "🔍" // Magnifying glass
		w.visualPreview.Add(widget.NewLabelWithStyle(
			fmt.Sprintf("%s Conditions:", filterIcon),
			fyne.TextAlignLeading,
			fyne.TextStyle{Bold: true},
		))

		for i, cond := range w.workflowData.Conditions {
			unit := ""
			if cond.ValueUnit != "" {
				unit = " " + cond.ValueUnit
			}
			condLabel := widget.NewLabel(
				fmt.Sprintf("  %d. %s %s %s%s",
					i+1, cond.Field, cond.Operator, cond.Value, unit),
			)
			condLabel.Wrapping = fyne.TextWrapWord
			w.visualPreview.Add(condLabel)
		}

		w.visualPreview.Add(widget.NewLabel("")) // Add spacing
		w.visualPreview.Add(widget.NewSeparator())
		w.visualPreview.Add(widget.NewLabel("")) // Add spacing
	}

	// Add actions
	if len(w.workflowData.Actions) > 0 {
		actionIcon := "⚙️" // Gear
		w.visualPreview.Add(widget.NewLabelWithStyle(
			fmt.Sprintf("%s Actions:", actionIcon),
			fyne.TextAlignLeading,
			fyne.TextStyle{Bold: true},
		))

		for i, action := range w.workflowData.Actions {
			actionLabel := widget.NewLabel(
				fmt.Sprintf("  %d. %s to: %s",
					i+1, action.Type, action.Target),
			)
			actionLabel.Wrapping = fyne.TextWrapWord
			w.visualPreview.Add(actionLabel)

			// Add options if present
			if len(action.Options) > 0 {
				optionsText := "     Options: "
				for k, v := range action.Options {
					optionsText += fmt.Sprintf("%s=%s ", k, v)
				}
				optLabel := widget.NewLabel(optionsText)
				optLabel.Wrapping = fyne.TextWrapWord
				w.visualPreview.Add(optLabel)
			}
		}
	}

	w.visualPreview.Refresh()
}

// addNewCondition opens a dialog to add a new condition
func (w *WorkflowWizard) addNewCondition() {
	// Create a dialog for adding a new condition
	conditionTypeSelect := widget.NewSelect([]string{
		"File Size",
		"File Type",
		"File Name",
		"File Age",
		"Custom",
	}, nil)
	conditionTypeSelect.PlaceHolder = "Select condition type..."

	fieldEntry := widget.NewEntry()
	fieldEntry.SetPlaceHolder("Field name (e.g., size, extension)")

	operatorOptions := []string{
		"Equals",
		"Not Equals",
		"Contains",
		"Starts With",
		"Ends With",
		"Greater Than",
		"Less Than",
		"Matches Regex",
	}

	operatorSelect := widget.NewSelect(operatorOptions, nil)
	operatorSelect.PlaceHolder = "Select operator..."

	valueEntry := widget.NewEntry()
	valueEntry.SetPlaceHolder("Value")

	unitEntry := widget.NewEntry()
	unitEntry.SetPlaceHolder("Unit (e.g., MB, KB, days)")

	// Create a form for the dialog
	form := widget.NewForm(
		widget.NewFormItem("Condition Type", conditionTypeSelect),
		widget.NewFormItem("Field", fieldEntry),
		widget.NewFormItem("Operator", operatorSelect),
		widget.NewFormItem("Value", valueEntry),
		widget.NewFormItem("Unit (Optional)", unitEntry),
	)

	// Create a custom confirmation dialog
	dialog.ShowCustomConfirm("Add New Condition", "Add", "Cancel", form, func(add bool) {
		if add {
			// Create new condition based on form values
			if conditionTypeSelect.Selected == "" {
				w.app.ShowError("Missing condition type", fmt.Errorf("please select a condition type"))
				return
			}

			if operatorSelect.Selected == "" {
				w.app.ShowError("Missing operator", fmt.Errorf("please select an operator"))
				return
			}

			if valueEntry.Text == "" {
				w.app.ShowError("Missing value", fmt.Errorf("please enter a value"))
				return
			}

			// Map selected condition type to actual type
			var condType types.ConditionType
			switch conditionTypeSelect.Selected {
			case "File Size":
				condType = types.FileSizeCondition
				if fieldEntry.Text == "" {
					fieldEntry.SetText("size")
				}
			case "File Type":
				condType = types.FileTypeCondition
				if fieldEntry.Text == "" {
					fieldEntry.SetText("type")
				}
			case "File Name":
				condType = types.FileNameCondition
				if fieldEntry.Text == "" {
					fieldEntry.SetText("name")
				}
			case "File Age":
				condType = types.FileAgeCondition
				if fieldEntry.Text == "" {
					fieldEntry.SetText("age")
				}
			case "Custom":
				condType = types.CustomCondition
			default:
				w.app.ShowError("Invalid condition type", fmt.Errorf("please select a valid condition type"))
				return
			}

			// Convert selected operator to OperatorType
			var operator types.OperatorType
			switch operatorSelect.Selected {
			case "Equals":
				operator = types.Equals
			case "Not Equals":
				operator = types.NotEquals
			case "Contains":
				operator = types.Contains
			case "Starts With":
				operator = types.StartsWith
			case "Ends With":
				operator = types.EndsWith
			case "Greater Than":
				operator = types.GreaterThan
			case "Less Than":
				operator = types.LessThan
			case "Matches Regex":
				operator = types.MatchesRegex
			default:
				w.app.ShowError("Invalid operator", fmt.Errorf("please select a valid operator"))
				return
			}

			newCondition := types.Condition{
				Type:      condType,
				Field:     fieldEntry.Text,
				Operator:  operator,
				Value:     valueEntry.Text,
				ValueUnit: unitEntry.Text,
			}

			// Add the condition and update the UI
			w.workflowData.Conditions = append(w.workflowData.Conditions, newCondition)
			w.updateStepContent()
			w.updateVisualization()
			w.app.ShowInfo("The condition has been added successfully.")
		}
	}, w.window)
}

// addNewAction opens a dialog to add a new action
func (w *WorkflowWizard) addNewAction() {
	// Create a dialog for adding a new action
	actionTypeSelect := widget.NewSelect([]string{
		"Move",
		"Copy",
		"Rename",
		"Tag",
		"Delete",
		"Execute",
	}, nil)
	actionTypeSelect.PlaceHolder = "Select action type..."

	targetEntry := widget.NewEntry()
	targetEntry.SetPlaceHolder("Target path (e.g., /path/to/destination)")

	// Options map entries
	formatEntry := widget.NewEntry()
	formatEntry.SetPlaceHolder("Format (e.g., {date}/{extension}/{name})")

	commandEntry := widget.NewEntry()
	commandEntry.SetPlaceHolder("Command to execute")

	// Create a form for the dialog
	form := widget.NewForm(
		widget.NewFormItem("Action Type", actionTypeSelect),
		widget.NewFormItem("Target", targetEntry),
		widget.NewFormItem("Format", formatEntry),
		widget.NewFormItem("Command", commandEntry),
	)

	// Create a custom confirmation dialog
	dialog.ShowCustomConfirm("Add New Action", "Add", "Cancel", form, func(add bool) {
		if add {
			// Validate inputs
			if actionTypeSelect.Selected == "" {
				w.app.ShowError("Missing action type", fmt.Errorf("please select an action type"))
				return
			}

			// Create new action based on form values
			var actionType types.ActionType
			switch actionTypeSelect.Selected {
			case "Move":
				actionType = types.MoveAction
			case "Copy":
				actionType = types.CopyAction
			case "Rename":
				actionType = types.RenameAction
			case "Tag":
				actionType = types.TagAction
			case "Delete":
				actionType = types.DeleteAction
			case "Execute":
				actionType = types.ExecuteAction
			default:
				w.app.ShowError("Invalid action type", fmt.Errorf("please select a valid action type"))
				return
			}

			// Create options map
			options := make(map[string]string)
			if formatEntry.Text != "" {
				options["format"] = formatEntry.Text
			}
			if commandEntry.Text != "" {
				options["command"] = commandEntry.Text
			}

			newAction := types.Action{
				Type:    actionType,
				Target:  targetEntry.Text,
				Options: options,
			}

			// Add the action and update the UI
			w.workflowData.Actions = append(w.workflowData.Actions, newAction)
			w.updateStepContent()
			w.updateVisualization()
			w.app.ShowInfo("The action has been added successfully.")
		}
	}, w.window)
}
