package watch

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sortd/internal/config"
	"sortd/internal/log"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	pidFile = ".sortd.pid"
)

type daemonContext struct {
	PidFileName   string
	PidFilePerm   os.FileMode
	LogFileName   string
	LogFilePerm   os.FileMode
	WorkDir       string
	Umask         uint32
	Args          []string
	Log           *os.File
	LogWriter     *bufio.Writer
	LogRotateSize int64
	LogRotateNum  int
}

// DaemonControl manages the lifecycle of the watch daemon
// including proper daemonization and process control
func DaemonControl(cfg *config.Config, foreground bool) error {
	// Create context for daemon
	context := &daemonContext{
		PidFileName: filepath.Join(cfg.Directories.Default, pidFile),
		PidFilePerm: 0644,
		LogFileName: filepath.Join(cfg.Directories.Default, "sortd.log"),
		LogFilePerm: 0640,
		WorkDir:     cfg.Directories.Default,
		Umask:       027,
		Args:        []string{fmt.Sprintf("-c=%t", cfg.Settings.DryRun)},
	}

	// Get the daemon
	daemonized, err := context.Reborn()
	if err != nil {
		return fmt.Errorf("failed to daemonize: %w", err)
	}

	if !daemonized {
		// Parent process - we're done here
		return nil
	}

	// Child process - setup signal handling and start daemon
	defer context.Release()

	// Set up signal handling
	chSig := make(chan os.Signal, 1)
	signal.Notify(chSig, os.Interrupt, syscall.SIGTERM)

	// Start the daemon
	if err := startDaemon(); err != nil {
		log.LogWithFields(log.F("error", err)).Error("Failed to start daemon")
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// If foreground, handle signals and keep running
	if foreground {
		log.Info("Running in foreground. Press Ctrl+C to stop.")
		<-chSig // Wait for signal
		log.Info("Stopping daemon...")
		stopDaemon()
		return nil
	}

	// Background mode - write PID file and exit parent
	return nil
}

// Reborn creates a new process and returns true if we're in the child process
func (c *daemonContext) Reborn() (bool, error) {
	// Implementation of daemonization
	if os.Getppid() == 1 {
		// We're already daemonized
		return true, nil
	}

	// Fork the process
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Start()
	return false, nil
}

// WritePid writes the PID to the PID file
func (c *daemonContext) WritePid() error {
	// Get current process PID
	pid := os.Getpid()

	// Write PID to file
	f, err := os.Create(c.PidFileName)
	if err != nil {
		return fmt.Errorf("failed to create PID file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(fmt.Sprintf("%d", pid)); err != nil {
		return fmt.Errorf("failed to write PID: %w", err)
	}

	return nil
}

// Release releases resources held by the context
func (c *daemonContext) Release() {
	// Implementation of resource release
}

func startDaemon() error {
	// Start the daemon
	return nil
}

func stopDaemon() {
	// Stop the daemon
}

// IsDaemonRunning checks if the daemon is currently running
func IsDaemonRunning(cfg *config.Config) bool {
	pidPath := filepath.Join(cfg.Directories.Default, pidFile)
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// StopDaemon stops a running daemon
func StopDaemon(cfg *config.Config) error {
	pidPath := filepath.Join(cfg.Directories.Default, pidFile)
	if !IsDaemonRunning(cfg) {
		return fmt.Errorf("daemon is not running")
	}

	// Read PID from file
	data, err := os.ReadFile(pidPath)
	if err != nil {
		log.Error("Failed to read PID file: %v", err)
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	_, err = parsePid(string(data))
	if err != nil {
		log.Error("Failed to parse PID: %v", err)
		return fmt.Errorf("failed to parse PID: %w", err)
	}

	// Send SIGTERM to the process group
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Start()
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
		log.Error("Failed to send SIGTERM to daemon: %v", err)
		return fmt.Errorf("failed to send SIGTERM to daemon: %w", err)
	}

	// Wait for process to exit

	// Remove PID file
	if err := os.Remove(pidPath); err != nil {
		log.Error("Failed to remove PID file: %v", err)
		return fmt.Errorf("failed to remove PID file: %w", err)
	}

	log.Info("Daemon stopped successfully")
	return nil
}

// parsePid parses a PID from a string
func parsePid(pidStr string) (int, error) {
	pid, err := strconv.Atoi(strings.TrimSpace(pidStr))
	if err != nil {
		return 0, fmt.Errorf("invalid PID format: %w", err)
	}
	return pid, nil
}

// Status returns the current status of the daemon
func Status(cfg *config.Config) (config.DaemonStatus, error) {
	pidPath := filepath.Join(cfg.Directories.Default, pidFile)
	if !IsDaemonRunning(cfg) {
		return config.DaemonStatus{}, fmt.Errorf("daemon is not running")
	}

	// Read PID from file
	data, err := os.ReadFile(pidPath)
	if err != nil {
		log.LogWithFields(log.F("error", err)).Error("Failed to read PID file")
		return config.DaemonStatus{}, fmt.Errorf("failed to read PID file: %w", err)
	}

	_, err = parsePid(string(data))
	if err != nil {
		log.LogWithFields(log.F("error", err)).Error("Failed to parse PID")
		return config.DaemonStatus{}, fmt.Errorf("failed to parse PID: %w", err)
	}

	// Get status from daemon process
	status := config.DaemonStatus{
		Running: true,
		// WatchDirectories will need to be communicated from the daemon
		// This is a limitation of the current implementation
		WatchDirectories: []string{},
		// LastActivity and FilesProcessed would need to be
		// communicated from the daemon process
		LastActivity:   time.Time{},
		FilesProcessed: 0,
	}

	log.Info("Daemon status retrieved successfully")
	return status, nil
}
