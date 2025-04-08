package gui_test

import (
	"os"
	"path/filepath"
	"sortd/internal/config"
	"sortd/internal/gui"
	"sortd/internal/organize"
	"sortd/internal/watch"
	"sortd/pkg/types"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/layout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockOrganizeEngine is a mock implementation of organize.Engine
type MockOrganizeEngine struct {
	OrganizeDirectoryFunc func(directory string) ([]types.OrganizeResult, error)
}

func (m *MockOrganizeEngine) OrganizeDirectory(directory string) ([]types.OrganizeResult, error) {
	if m.OrganizeDirectoryFunc != nil {
		return m.OrganizeDirectoryFunc(directory)
	}
	return nil, nil // Default behavior
}

func (m *MockOrganizeEngine) OrganizeFile(filePath string) (types.OrganizeResult, error) {
	// Mock implementation - not the focus for GUI tests yet
	return types.OrganizeResult{}, nil
}

// MockWatchDaemon is a mock implementation of watch.Daemon
type MockWatchDaemon struct {
	startCalled     bool
	stopCalled      bool
	status          watch.DaemonStatus
	SetDryRunFunc   func(bool)
	StartFunc       func() error
	StopFunc        func() error
	StatusFunc      func() watch.DaemonStatus
	AddWatchFunc    func(string) error
	RemoveWatchFunc func(string) error
}

func (m *MockWatchDaemon) SetDryRun(dryRun bool) {
	if m.SetDryRunFunc != nil {
		m.SetDryRunFunc(dryRun)
	}
}

func (m *MockWatchDaemon) Start() error {
	m.startCalled = true
	m.status.Running = true
	if m.StartFunc != nil {
		return m.StartFunc()
	}
	return nil
}

func (m *MockWatchDaemon) Stop() error {
	m.stopCalled = true
	m.status.Running = false
	if m.StopFunc != nil {
		return m.StopFunc()
	}
	return nil
}

func (m *MockWatchDaemon) Status() watch.DaemonStatus {
	if m.StatusFunc != nil {
		return m.StatusFunc()
	}
	return m.status // Return internal mock status
}

func (m *MockWatchDaemon) AddWatch(dir string) error {
	if m.AddWatchFunc != nil {
		return m.AddWatchFunc(dir)
	}
	return nil
}

func (m *MockWatchDaemon) RemoveWatch(dir string) error {
	if m.RemoveWatchFunc != nil {
		return m.RemoveWatchFunc(dir)
	}
	return nil
}

// TestNewApp checks if the GUI application initializes without errors.
func TestNewApp(t *testing.T) {
	cfg := config.New() // Use config.New() for default config

	// Create a minimal engine for this test
	engine := organize.NewWithConfig(cfg) // Use NewWithConfig

	// Create the GUI App instance (Correct signature: func NewApp(cfg *config.Config, engine *organize.Engine) *App)
	guiApp := gui.NewApp(cfg, engine)

	if guiApp == nil {
		t.Fatal("gui.NewApp returned nil")
	}

	// We test component creation and interactions separately.
	t.Log("GUI App initialized successfully")
}

// TestGetMainWindow tests the retrieval of the main window.
func TestGetMainWindow(t *testing.T) {
	cfg := config.New()                   // Use config.New() for default config
	engine := organize.NewWithConfig(cfg) // Use NewWithConfig
	guiApp := gui.NewApp(cfg, engine)

	if guiApp == nil {
		t.Fatal("gui.NewApp returned nil")
	}

	mainWindow := guiApp.GetMainWindow()
	if mainWindow == nil {
		t.Fatal("GetMainWindow returned nil")
	}

	t.Log("Main window retrieved successfully")
}

// TestGUI_OrganizeFilesButton tests the integration of the "Organize Files" button
// with the real organize.Engine.
func TestGUI_OrganizeFilesButton(t *testing.T) {
	// --- Setup ---
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "documents")
	require.NoError(t, os.Mkdir(sourceDir, 0755))
	// No need to Mkdir targetDir if CreateDirs is true in config

	testFileName := "testfile.txt"
	sourceFilePath := filepath.Join(sourceDir, testFileName)
	targetFilePath := filepath.Join(targetDir, testFileName)

	// Create a dummy file
	require.NoError(t, os.WriteFile(sourceFilePath, []byte("test content"), 0644))

	// Configure to move the file
	cfg := config.NewTestConfig()
	cfg.Directories.Default = sourceDir // Set GUI default dir
	cfg.Organize.Patterns = []types.Pattern{
		{Match: "*.txt", Target: targetDir},
	}
	cfg.Settings.DryRun = false
	cfg.Settings.CreateDirs = true // Ensure target dir is created

	// Use the real organize engine
	realEngine := organize.NewWithConfig(cfg) // Use NewWithConfig

	// Create the GUI App instance (Correct signature: func NewApp(cfg *config.Config, engine *organize.Engine) *App)
	guiApp := gui.NewApp(cfg, realEngine) // Use correct NewApp signature
	require.NotNil(t, guiApp)

	// Setup the main window content implicitly via NewApp, get window
	w := guiApp.GetMainWindow()
	require.NotNil(t, w, "Main window should not be nil")

	// --- Find the Button ---
	// Traverse the widget tree to find the 'Organize Files' button
	var organizeButton *widget.Button

	// 1. Window Content -> Root Container (*fyne.Container, *layout.NewBorderLayout)
	winContent := w.Content()
	require.NotNil(t, winContent, "Window content is nil")
	rootContainer, ok := winContent.(*fyne.Container)
	require.True(t, ok, "Window content assertion failed: expected *fyne.Container")
	require.NotNil(t, rootContainer, "Root container is nil")
	rootLayout, ok := rootContainer.Layout.(*layout.BorderLayout)
	require.True(t, ok, "Root container layout assertion failed: expected *layout.BorderLayout")
	require.NotNil(t, rootLayout, "Root layout is nil")

	// 2. Root Border Layout -> Center Object (expect *container.Split)
	centerSplitGeneric := rootLayout.Center
	require.NotNil(t, centerSplitGeneric, "Center object in root layout is nil")
	centerSplit, ok := centerSplitGeneric.(*container.Split)
	require.True(t, ok, "Center object assertion failed: expected *container.Split")
	require.NotNil(t, centerSplit, "Center split container is nil")

	// 3. Center Split -> Trailing Object (Right Pane - expect *container.Max)
	rightSideGeneric := centerSplit.Trailing
	require.NotNil(t, rightSideGeneric, "Trailing object (right side) in split is nil")
	rightMax, ok := rightSideGeneric.(*container.Max)
	require.True(t, ok, "Right side assertion failed: expected *container.Max")
	require.NotNil(t, rightMax, "Right Max container is nil")
	require.NotEmpty(t, rightMax.Objects, "Right Max container has no objects")

	// 4. Max Container -> First Object (Details View - expect *fyne.Container with *layout.NewBorderLayout)
	detailsContainerGeneric := rightMax.Objects[0]
	require.NotNil(t, detailsContainerGeneric, "Details view generic object is nil")
	detailsContainer, ok := detailsContainerGeneric.(*fyne.Container)
	require.True(t, ok, "Details view assertion failed: expected *fyne.Container")
	require.NotNil(t, detailsContainer, "Details container is nil")
	detailsLayout, ok := detailsContainer.Layout.(*layout.BorderLayout)
	require.True(t, ok, "Details container layout assertion failed: expected *layout.BorderLayout")
	require.NotNil(t, detailsLayout, "Details layout is nil")

	// 5. Details Border Layout -> Bottom Object (Button Area - expect *container.Padded)
	bottomAreaGeneric := detailsLayout.Bottom
	require.NotNil(t, bottomAreaGeneric, "Bottom object in details layout is nil")
	paddedButtonContainer, ok := bottomAreaGeneric.(*container.Padded)
	require.True(t, ok, "Bottom area assertion failed: expected *container.Padded")
	require.NotNil(t, paddedButtonContainer, "Padded button container is nil")

	// 6. Padded Container -> Content (The Button - expect *widget.Button)
	buttonGeneric := paddedButtonContainer.Content
	require.NotNil(t, buttonGeneric, "Content of padded container is nil")
	organizeButton, ok = buttonGeneric.(*widget.Button)
	require.True(t, ok, "Button assertion failed: expected *widget.Button")
	require.NotNil(t, organizeButton, "Organize button is nil after assertion")

	// 7. Verify Button Text
	require.Equal(t, "Organize Files", organizeButton.Text, "Button text mismatch")

	// --- Action ---
	test.Tap(organizeButton)

	// --- Verification ---
	// File should be moved
	_, errSource := os.Stat(sourceFilePath)
	assert.True(t, os.IsNotExist(errSource), "Source file should not exist after organize")

	_, errTarget := os.Stat(targetFilePath)
	assert.NoError(t, errTarget, "Target file should exist after organize")
}
