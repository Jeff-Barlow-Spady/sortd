package gui_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"
)

// TestMinimalBorderAssertion isolates the issue of asserting window content to *fyne.Container.
func TestMinimalBorderAssertion(t *testing.T) {
	// 1. Create a simple *container.Border
	// We need to provide at least one piece of content for it to be valid.
	borderContent := container.NewBorder(
		widget.NewLabel("Top"),
		widget.NewLabel("Bottom"),
		widget.NewLabel("Left"),
		widget.NewLabel("Right"),
		widget.NewLabel("Center"), // The content that *should* be a *container.Border
	)

	// 2. Create a test window and set the border as its content
	w := test.NewTempWindow(t, borderContent)
	defer w.Close() // Ensure window resources are cleaned up

	// 3. Get the window's content
	retrievedContent := w.Content()
	require.NotNil(t, retrievedContent, "Window content should not be nil")

	// 4. Attempt the problematic type assertion - corrected to *fyne.Container
	rootContainer, ok := retrievedContent.(*fyne.Container) // Use generic *fyne.Container

	// 5. Assert the result (the compiler error should happen before this if the issue persists)
	require.True(t, ok, "Content retrieved from window should be assertable to *fyne.Container")
	require.NotNil(t, rootContainer, "*fyne.Container should not be nil after assertion")

	// Optional: Further check if needed
	// _, centerOk := rootContainer.Center.(*widget.Label)
	// require.True(t, centerOk, "Center object should be the label we put in")
	// Note: Accessing .Center might require checking rootContainer.Layout is *layout.BorderLayout first

	t.Log("Minimal test completed: Type assertion to *fyne.Container was successful.")
}
