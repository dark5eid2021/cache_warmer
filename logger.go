package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel represents the logging level
type LogLevel int

const (
	// LogLevelDebug represents debug level logging
	LogLevelDebug LogLevel = iota
	// LogLevelInfo represents info level logging
	LogLevelInfo
	// LogLevelWarn represents warning level logging
	LogLevelWarn
	// LogLevelError represents error level logging
	LogLevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging capabilities for the cache warmer
type Logger struct {
	logger  *log.Logger
	level   LogLevel
	verbose bool
}

// NewLogger creates a new logger instance
func NewLogger(verbose bool) *Logger {
	// Create a logger that writes to stdout with timestamp
	logger := log.New(os.Stdout, "", 0)

	level := LogLevelInfo
	if verbose {
		level = LogLevelDebug
	}

	return &Logger{
		logger:  logger,
		level:   level,
		verbose: verbose,
	}
}

// log writes a log message with the specified level
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	// Skip if log level is below configured level
	if level < l.level {
		return
	}

	// Format timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Format message
	message := fmt.Sprintf(format, args...)

	// Create full log line
	logLine := fmt.Sprintf("[%s] %s: %s", timestamp, level.String(), message)

	// Write to logger
	l.logger.Println(logLine)
}

// Debug logs a debug message (only shown in verbose mode)
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LogLevelDebug, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LogLevelInfo, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LogLevelWarn, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LogLevelError, format, args...)
}

// Fatal logs a fatal error message and exits the program
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LogLevelError, format, args...)
	os.Exit(1)
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// IsDebugEnabled returns true if debug logging is enabled
func (l *Logger) IsDebugEnabled() bool {
	return l.level <= LogLevelDebug
}

// IsVerbose returns true if verbose mode is enabled
func (l *Logger) IsVerbose() bool {
	return l.verbose
}
