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

func Debug(format string, args ...interface{}) {
	if isDebug {
		logger.log("DEBUG", format, args...)
	}
}

func Error(format string, args ...interface{}) {
	logger.log("ERROR", format, args...)
}

func (l *Logger) log(level, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.out, "[%s] %s: %s\n", timestamp, level, msg)
}
