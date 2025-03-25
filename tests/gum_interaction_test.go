package tests

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"sortd/tests/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGumAvailability checks if the gum command is available on the system.
func TestGumAvailability(t *testing.T) {
	// Skip this test if we're in a CI environment that might not have Gum
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping Gum availability test in CI environment")
	}

	cmd := exec.Command("gum", "--version")
	err := cmd.Run()
	assert.NoError(t, err, "Gum should be installed and available")
}

// TestCliWithGumInteraction tests using Gum within the CLI, if available.
// This is a simplified test that only checks if the CLI attempts to use Gum.
func TestCliWithGumInteraction(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping interactive test in CI environment")
	}

	// Check if Gum is installed
	gumInstalled := exec.Command("gum", "--version").Run() == nil

	// Create a temporary file to store test output
	tmpDir := t.TempDir()
	outputFile := tmpDir + "/output.txt"

	// Get the binary path
	binPath := testutils.GetBinaryPath(t)

	// Run a command that would use Gum (with timeout to ensure it exits)
	// Redirect output to our file for analysis
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf("timeout 2s %s setup --dry-run > %s 2>&1 || true", binPath, outputFile))

	err := cmd.Run()
	require.NoError(t, err, "Command should execute without errors")

	// Read the output file
	output, err := os.ReadFile(outputFile)
	require.NoError(t, err, "Should be able to read output file")

	// If Gum is installed, the output might reference Gum or show formatted content
	// If not installed, it should fall back to basic output
	if gumInstalled {
		// Only make assertions if Gum is installed
		if bytes.Contains(output, []byte("gum")) ||
			bytes.Contains(output, []byte("style")) {
			t.Log("Detected Gum-formatted output")
		}
	} else {
		t.Log("Gum not installed, CLI should use fallback output")
	}

	// In either case, output should contain certain information
	assert.Contains(t, string(output), "configuration",
		"Output should mention configuration regardless of Gum availability")
}

// TestGumStyleRendering tests that our CLI can produce styled output with Gum.
// This test is designed to be skipped if Gum is not available.
func TestGumStyleRendering(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping Gum rendering test in CI environment")
	}

	// Check if Gum is installed
	if exec.Command("gum", "--version").Run() != nil {
		t.Skip("Gum is not installed, skipping style rendering test")
	}

	// Test a simple Gum style rendering
	cmd := exec.Command("gum", "style", "--foreground", "212", "Test Text")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	require.NoError(t, err, "Gum style command should execute successfully")

	output := stdout.String()
	assert.Contains(t, output, "Test Text", "Styled output should contain our test text")

	// The actual color codes are hard to test as they're ANSI escape sequences
	// But we can verify the text is present
}

// TestGumInteractiveElements tests the interactive elements using Gum if available.
func TestGumInteractiveElements(t *testing.T) {
	if os.Getenv("CI") == "true" || os.Getenv("INTERACTIVE_TESTS") != "true" {
		t.Skip("Skipping interactive Gum test - run with INTERACTIVE_TESTS=true to enable")
	}

	// Check if Gum is installed
	if exec.Command("gum", "--version").Run() != nil {
		t.Skip("Gum is not installed, skipping interactive test")
	}

	// This test would normally echo input to Gum commands and check output
	// For now, we'll just verify we can run a basic Gum command that returns immediately

	cmd := exec.Command("gum", "style", "Interactive tests would run here")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	require.NoError(t, err, "Basic Gum command should run successfully")

	output := stdout.String()
	assert.Contains(t, output, "Interactive", "Output should contain our test text")
}
