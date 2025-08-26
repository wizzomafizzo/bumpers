package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wizzomafizzo/bumpers/internal/core/messaging/context"
	"github.com/wizzomafizzo/bumpers/internal/infrastructure/constants"
	"github.com/wizzomafizzo/bumpers/internal/platform/filesystem"
	"github.com/wizzomafizzo/bumpers/internal/platform/storage"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	// Lumberjack configuration constants
	maxLogSizeMB  = 10 // Maximum size in MB before rotation
	maxLogBackups = 3  // Number of old files to keep
	maxLogAgeDays = 30 // Maximum age in days before deletion
)

// createLumberjackLogger creates a lumberjack.Logger with standard configuration
func createLumberjackLogger(logFile string) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    maxLogSizeMB,
		MaxBackups: maxLogBackups,
		MaxAge:     maxLogAgeDays,
	}
}

// ensureLogDir creates the log directory if it doesn't exist
func ensureLogDir(logDir string) error {
	err := os.MkdirAll(logDir, 0o750)
	if err != nil {
		return fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}
	return nil
}

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

	if err := ensureLogDir(logDir); err != nil {
		return nil, err
	}

	lj := createLumberjackLogger(logFile)

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

// InitLogger initializes the global logger instance with lumberjack rotation.
// Deprecated: Use Init() for production logging with rotation.
func InitLogger(workDir string) error {
	return Init(workDir)
}

// Init initializes the global logger with lumberjack rotation
func Init(workDir string) error {
	logDir := filepath.Join(workDir, constants.ClaudeDir, constants.AppSubDir)
	logFile := filepath.Join(logDir, constants.LogFilename)

	if err := ensureLogDir(logDir); err != nil {
		return err
	}

	lj := createLumberjackLogger(logFile)

	// Create and set global zerolog logger
	log.Logger = zerolog.New(lj).With().Timestamp().Logger()

	return nil
}

// InitTest initializes logger for testing (outputs to discard)
func InitTest() {
	log.Logger = zerolog.New(io.Discard)
}

// InitWithProjectContextAndFS initializes the global logger with project context and injected filesystem
func InitWithProjectContextAndFS(projectCtx *context.ProjectContext, fs filesystem.FileSystem) error {
	// Use storage manager for XDG-compliant paths with injected filesystem
	storageManager := storage.New(fs)
	logFile, err := storageManager.GetLogPath()
	if err != nil {
		return fmt.Errorf("failed to get log path: %w", err)
	}

	lj := createLumberjackLogger(logFile)

	// Create zerolog logger with project context
	log.Logger = zerolog.New(lj).With().
		Timestamp().
		Str("project_id", projectCtx.ID).
		Logger()

	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	return nil
}

// InitWithProjectContext initializes the global logger with project context and XDG paths
// Uses OSFileSystem by default for backward compatibility
func InitWithProjectContext(projectCtx *context.ProjectContext) error {
	return InitWithProjectContextAndFS(projectCtx, &filesystem.OSFileSystem{})
}
