package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wizzomafizzo/bumpers/internal/constants"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps zerolog.Logger with lumberjack for automatic log rotation
type Logger struct {
	zerolog.Logger
	lumberjack       *lumberjack.Logger
	supportsRotation bool
	closeOnce        sync.Once
}

// New creates a new logger instance with automatic log rotation
func New(workDir string) (*Logger, error) {
	logDir := filepath.Join(workDir, constants.ClaudeDir, constants.AppSubDir)
	logFile := filepath.Join(logDir, constants.LogFilename)

	// Create log directory if it doesn't exist
	err := os.MkdirAll(logDir, 0o750)
	if err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// Create lumberjack logger for automatic rotation
	lj := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10, // MB
		MaxBackups: 3,
		MaxAge:     30, // days
	}

	// Create zerolog logger
	zl := zerolog.New(lj).With().Timestamp().Logger()

	return &Logger{
		Logger:           zl,
		lumberjack:       lj,
		supportsRotation: true,
	}, nil
}

// SupportsRotation returns true if the logger uses log rotation
func (l *Logger) SupportsRotation() bool {
	return l.supportsRotation
}

// Rotate manually triggers log rotation if supported
func (l *Logger) Rotate() error {
	if l.lumberjack != nil {
		err := l.lumberjack.Rotate()
		if err != nil {
			return fmt.Errorf("failed to rotate log file: %w", err)
		}
	}
	return nil
}

// Close closes the lumberjack logger
func (l *Logger) Close() error {
	var err error
	l.closeOnce.Do(func() {
		if l.lumberjack != nil {
			err = l.lumberjack.Close()
		}
	})
	return err //nolint:wrapcheck // simple logger cleanup
}

// InitLogger initializes the global logger instance
func InitLogger(workDir string) error {
	logDir := filepath.Join(workDir, constants.ClaudeDir, constants.AppSubDir)
	logFilePath := filepath.Join(logDir, constants.LogFilename)

	// Create log directory if it doesn't exist
	err := os.MkdirAll(logDir, 0o750)
	if err != nil {
		return fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// Open log file for appending
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // safe path
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", logFilePath, err)
	}

	// Set global logger
	log.Logger = zerolog.New(file).With().Timestamp().Logger()

	// Note: Global logger file reference not stored for cleanup
	// Consider using the Logger instance approach instead of global logger

	return nil
}

// Init initializes the global logger with lumberjack rotation
func Init(workDir string) error {
	logDir := filepath.Join(workDir, constants.ClaudeDir, constants.AppSubDir)
	logFile := filepath.Join(logDir, constants.LogFilename)

	// Create log directory if it doesn't exist
	err := os.MkdirAll(logDir, 0o750)
	if err != nil {
		return fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// Create lumberjack logger for automatic rotation
	lj := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10, // MB
		MaxBackups: 3,  // number of old files to keep
		MaxAge:     30, // days
	}

	// Create and set global zerolog logger
	log.Logger = zerolog.New(lj).With().Timestamp().Logger()

	return nil
}

// InitTest initializes logger for testing (outputs to discard)
func InitTest() {
	log.Logger = zerolog.New(io.Discard)
}
