package tests

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"sortd/internal/config"
	"sortd/internal/organize"
	"sortd/pkg/types"
	"sortd/tests/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCliVersionCommand verifies that the CLI version command works.
func TestCliVersionCommand(t *testing.T) {
	output, err := testutils.RunCliCommand(t, "", "version")
	require.NoError(t, err, "Version command should not fail")
	assert.Contains(t, output, "sortd version", "Version output should contain version information")
}

// TestCliHelpCommand verifies that help output is displayed.
func TestCliHelpCommand(t *testing.T) {
	output, err := testutils.RunCliCommand(t, "", "--help")
	require.NoError(t, err, "Help command should not fail")

	// Verify that the output shows available commands
	assert.Contains(t, output, "Available Commands:", "Help output should list available commands")
	assert.Contains(t, output, "organize", "Help output should mention organize command")
	assert.Contains(t, output, "setup", "Help output should mention setup command")
	assert.Contains(t, output, "rules", "Help output should mention rules command")
	assert.Contains(t, output, "watch", "Help output should mention watch command")
	assert.Contains(t, output, "daemon", "Help output should mention daemon command")
}

// TestCliSetupCommand tests the setup command with various flags.
func TestCliSetupCommand(t *testing.T) {
	if os.Getenv("CI") == "true" || os.Getenv("SORTD_BIN") != "" {
		t.Skip("Skipping interactive test in CI environment or when using test binary")
	}

	// Set test mode to avoid interactive prompts
	originalTestMode := os.Getenv("TESTMODE")
	os.Setenv("TESTMODE", "true")
	defer os.Setenv("TESTMODE", originalTestMode)

	// Test the --dry-run flag which should show what would be done without interactive prompts
	output, err := testutils.RunCliCommand(t, "", "setup", "--dry-run")
	require.NoError(t, err, "Setup command with dry-run should not fail")
	assert.Contains(t, output, "configuration", "Dry run should mention configuration setup")
}

// TestCliOrganizeCommand tests the organize command with various flags.
func TestCliOrganizeCommand(t *testing.T) {
	if os.Getenv("SORTD_BIN") != "" {
		t.Skip("Skipping organize command test when using test binary")
	}

	// Create temp dir for test
	tempDir := t.TempDir()

	// Create a subdirectory for test files
	testDir := filepath.Join(tempDir, "001")
	require.NoError(t, os.MkdirAll(testDir, 0755))

	// Copy test files to the temp directory
	srcPath := testutils.CopyFixture(t, "sample.txt", testDir)
	t.Logf("Copied fixture file: %s", srcPath)

	// Copy an image fixture
	imgPath := testutils.CopyFixture(t, "photo.jpg", testDir)
	t.Logf("Copied fixture file: %s", imgPath)

	// Set environment variable for test mode
	originalTestMode := os.Getenv("TESTMODE")
	os.Setenv("TESTMODE", "true")
	defer os.Setenv("TESTMODE", originalTestMode)

	// Test organize command with dry-run
	output, err := testutils.RunCliCommand(t, "", "organize", testDir, "--dry-run")
	require.NoError(t, err, "Organize command with dry-run should not fail")
	assert.Contains(t, output, "sample.txt", "Output should include text file")
	assert.Contains(t, output, "photo.jpg", "Output should include image file")
	assert.Contains(t, output, "would be", "Output should indicate files would be moved")
}

// TestCliOrganizeWithMock tests the organize command with a mock organizer
func TestCliOrganizeWithMock(t *testing.T) {
	// Create a mock organizer
	mockOrganizer := &MockOrganizer{}

	// Save the original factory
	originalFactory := organize.CurrentOrganizerFactory
	defer func() {
		// Restore the original factory when the test completes
		organize.CurrentOrganizerFactory = originalFactory
	}()

	// Set our mock factory
	organize.SetOrganizerFactory(func() organize.Organizer {
		return mockOrganizer
	})

	// Create temp dir for test
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err, "Failed to create test file")

	// Set environment variable for test mode
	originalTestMode := os.Getenv("TESTMODE")
	os.Setenv("TESTMODE", "true")
	defer os.Setenv("TESTMODE", originalTestMode)

	// Test organize command with our mock
	_, err = testutils.RunCliCommand(t, "", "organize", tempDir)
	require.NoError(t, err, "Organize command with mock should not fail")

	// Verify the mock was called correctly
	assert.True(t, mockOrganizer.configSet, "SetConfig should have been called")
	assert.Equal(t, false, mockOrganizer.dryRun, "Dry run should be false")
	assert.True(t, mockOrganizer.patternsCalled, "OrganizeByPatterns should have been called")
}

// MockOrganizer is a mock implementation of the Organizer interface for testing
type MockOrganizer struct {
	configSet      bool
	dryRun         bool
	patternsCalled bool
}

func (m *MockOrganizer) SetConfig(cfg *config.Config) {
	m.configSet = true
}

func (m *MockOrganizer) SetDryRun(dryRun bool) {
	m.dryRun = dryRun
}

func (m *MockOrganizer) AddPattern(pattern types.Pattern) {
	// Nothing to do
}

func (m *MockOrganizer) OrganizeFile(path string) error {
	return nil
}

func (m *MockOrganizer) MoveFile(src, dest string) error {
	return nil
}

func (m *MockOrganizer) OrganizeFiles(files []string, destDir string) error {
	return nil
}

func (m *MockOrganizer) OrganizeByPatterns(files []string) error {
	m.patternsCalled = true
	return nil
}

func (m *MockOrganizer) OrganizeDir(dir string) ([]string, error) {
	return []string{}, nil
}

// TestCliRulesCommand tests the rules command.
func TestCliRulesCommand(t *testing.T) {
	// Test list command
	output, err := testutils.RunCliCommand(t, "", "rules", "list")
	require.NoError(t, err, "Rules list command should not fail")
	assert.Contains(t, output, "Rules:", "Output should display rules list heading")
}

// TestCliWatchCommand tests the watch command with a short timeout.
// This is just a basic test to ensure it starts without error.
func TestCliWatchCommand(t *testing.T) {
	// Create temp dir for test
	tempDir := t.TempDir()

	// Get the binary path through the standard mechanism
	binPath := testutils.GetBinaryPath(t)

	// We want to run this asynchronously with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Run the watch command with a timeout context but no --timeout flag since it's not supported
	cmd := exec.CommandContext(ctx, binPath, "watch", tempDir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start but don't wait for completion
	err := cmd.Start()
	require.NoError(t, err, "Watch command should start without errors")

	// Wait a short time for output
	time.Sleep(500 * time.Millisecond)

	// Kill the process
	_ = cmd.Process.Kill()

	// Check any output that was generated
	output := stdout.String()
	t.Logf("Command output: %s", output)
	t.Logf("Command stderr: %s", stderr.String())

	// Simple check that it started properly
	assert.NotContains(t, stderr.String(), "Error:", "Should not contain error messages")
}

// TestCliDaemonCommand tests the daemon command.
func TestCliDaemonCommand(t *testing.T) {
	// Test status command
	output, err := testutils.RunCliCommand(t, "", "daemon", "status")
	require.NoError(t, err, "Daemon status command should not fail")
	assert.Contains(t, output, "Daemon", "Output should mention daemon status")
}
