package log

import (
	"fmt"
	"os"
	"time"
)

var (
	isDebug = false
	logger  = NewLogger()
)

type Logger struct {
	out *os.File
}

func NewLogger() *Logger {
	return &Logger{
		out: os.Stdout,
	}
}

func SetDebug(debug bool) {
	isDebug = debug
}

func Info(format string, args ...interface{}) {
	logger.log("INFO", format, args...)
}

// Debug logs a message with arguments
func Debug(msg string, args ...interface{}) {
	if isDebug {
		logger.log("DEBUG", msg+": %v", args...)
	}
}

// Debugf logs a formatted message
func Debugf(format string, args ...interface{}) {
	if isDebug {
		logger.log("DEBUG", format, args...)
	}
}

// Error logs an error message with arguments
func Error(msg string, args ...interface{}) {
	logger.log("ERROR", msg+": %v", args...)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	logger.log("ERROR", format, args...)
}

func (l *Logger) log(level, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.out, "[%s] %s: %s\n", timestamp, level, msg)
}

// Warn logs a warning message with arguments
func Warn(msg string, args ...interface{}) {
	logger.log("WARN", msg+": %v", args...)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	logger.log("WARN", format, args...)
}
