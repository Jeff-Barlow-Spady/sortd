package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

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
