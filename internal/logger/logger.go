package logger

import (
	"log/slog"
	"os"
	"path/filepath"
)

// Info logs an info message
func Info(msg string, args ...any) {
	// Create log directory if it doesn't exist
	logDir := ".claude/bumpers"
	logFile := filepath.Join(logDir, "bumpers.log")

	err := os.MkdirAll(logDir, 0o750)
	if err != nil {
		return
	}

	// Open log file for appending
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // safe path
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	// Create JSON logger and log the message
	logger := slog.New(slog.NewJSONHandler(file, nil))
	logger.Info(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	// Create log directory if it doesn't exist
	logDir := ".claude/bumpers"
	logFile := filepath.Join(logDir, "bumpers.log")

	err := os.MkdirAll(logDir, 0o750)
	if err != nil {
		return
	}

	// Open log file for appending
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // safe path
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	// Create JSON logger and log the message
	logger := slog.New(slog.NewJSONHandler(file, nil))
	logger.Error(msg, args...)
}
