package logger

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

var (
	loggerInstance *slog.Logger
	once           sync.Once
	errInit        error
	workingDir     string
)

// Initialize sets up the logger with a working directory
func Initialize(workDir string) error {
	workingDir = workDir
	return nil
}

// Reset allows tests to reinitialize the logger (not safe for concurrent use)
func Reset() {
	once = sync.Once{}
	loggerInstance = nil
	errInit = nil
	workingDir = ""
}

// getLogger returns a singleton logger instance
func getLogger() *slog.Logger {
	once.Do(func() {
		// Use working directory if set, otherwise fall back to current directory
		baseDir := workingDir
		if baseDir == "" {
			var err error
			baseDir, err = os.Getwd()
			if err != nil {
				errInit = err
				return
			}
		}

		logDir := filepath.Join(baseDir, ".claude", "bumpers")
		logFile := filepath.Join(logDir, "bumpers.log")

		// Create log directory if it doesn't exist
		err := os.MkdirAll(logDir, 0o750)
		if err != nil {
			errInit = err
			return
		}

		// Open log file for appending
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // safe path
		if err != nil {
			errInit = err
			return
		}

		// Create JSON logger with the file handler
		loggerInstance = slog.New(slog.NewJSONHandler(file, nil))
	})

	if errInit != nil {
		return nil
	}
	return loggerInstance
}

// Info logs an info message
func Info(msg string, args ...any) {
	logger := getLogger()
	if logger != nil {
		logger.Info(msg, args...)
	}
}

// Error logs an error message
func Error(msg string, args ...any) {
	logger := getLogger()
	if logger != nil {
		logger.Error(msg, args...)
	}
}
