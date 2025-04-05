package gui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// CommandRunner handles the execution of sortd commands from the GUI
type CommandRunner struct {
	binaryPath string
	configPath string
}

// NewCommandRunner creates a new command runner
func NewCommandRunner(binaryPath, configPath string) *CommandRunner {
	return &CommandRunner{
		binaryPath: binaryPath,
		configPath: configPath,
	}
}

// StartWatchMode starts the watch mode daemon
func (r *CommandRunner) StartWatchMode(dirs []string, interval int, requireConfirmation bool, confirmationPeriod int) error {
	args := []string{"watch"}

	// Add directories
	for _, dir := range dirs {
		args = append(args, "--dir", dir)
	}

	// Add interval
	args = append(args, "--interval", fmt.Sprintf("%d", interval))

	// Add confirmation settings
	if requireConfirmation {
		args = append(args, "--require-confirmation")
		args = append(args, "--confirmation-period", fmt.Sprintf("%d", confirmationPeriod))
	}

	// Add config path
	args = append(args, "--config", r.configPath)

	// Run the command
	cmd := exec.Command(r.binaryPath, args...)
	return cmd.Start()
}

// StopWatchMode stops the watch mode daemon
func (r *CommandRunner) StopWatchMode() error {
	cmd := exec.Command(r.binaryPath, "daemon", "stop")
	return cmd.Run()
}

// OrganizeDirectory organizes the specified directory
func (r *CommandRunner) OrganizeDirectory(dir string, dryRun bool) error {
	args := []string{"organize", dir, "--config", r.configPath}

	if dryRun {
		args = append(args, "--dry-run")
	}

	cmd := exec.Command(r.binaryPath, args...)
	_, err := cmd.CombinedOutput()
	return err
}

// GetBinaryPath attempts to find the sortd binary path
func GetBinaryPath() string {
	// First, check if it's in the PATH
	path, err := exec.LookPath("sortd")
	if err == nil {
		return path
	}

	// Try common locations
	locations := []string{
		"sortd",                      // Current directory
		"./sortd",                    // Current directory (explicit)
		filepath.Join("..", "sortd"), // Parent directory
	}

	for _, loc := range locations {
		if _, err := exec.Command(loc, "--version").Output(); err == nil {
			return loc
		}
	}

	// Default to current directory if all else fails
	return "./sortd"
}

// GetVersion gets the current version of sortd
func (r *CommandRunner) GetVersion() (string, error) {
	cmd := exec.Command(r.binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse the version from output
	versionStr := strings.TrimSpace(string(output))
	return versionStr, nil
}

// CheckStatus checks if the watch daemon is running
func (r *CommandRunner) CheckStatus() (bool, error) {
	cmd := exec.Command(r.binaryPath, "daemon", "status")
	output, err := cmd.CombinedOutput()

	// Check if the output indicates the daemon is running
	isRunning := strings.Contains(string(output), "running")

	// If the command failed but we detected it's not running, don't return an error
	if err != nil && !isRunning {
		return false, nil
	}

	return isRunning, err
}

// ParseInterval parses an interval string to an integer
func ParseInterval(intervalStr string) (int, error) {
	return strconv.Atoi(intervalStr)
}
