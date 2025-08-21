package logger

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wizzomafizzo/bumpers/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps zerolog.Logger with our specific configuration
type Logger struct {
	zerolog.Logger
	file             *os.File
	lumberjack       *lumberjack.Logger
	supportsRotation bool
	closeOnce        sync.Once
}

// New creates a new logger instance
func New(workDir string) (*Logger, error) {
	logDir := filepath.Join(workDir, ".claude", "bumpers")
	logFile := filepath.Join(logDir, "bumpers.log")

	// Create log directory if it doesn't exist
	err := os.MkdirAll(logDir, 0o750)
	if err != nil {
		return nil, err //nolint:wrapcheck // simple logger setup
	}

	// Open log file for appending
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // safe path
	if err != nil {
		return nil, err //nolint:wrapcheck // simple logger setup
	}

	// Create zerolog logger
	zl := zerolog.New(file).With().Timestamp().Logger()

	return &Logger{
		Logger:           zl,
		file:             file,
		supportsRotation: false,
	}, nil
}

// SupportsRotation returns true if the logger uses log rotation
func (l *Logger) SupportsRotation() bool {
	return l.supportsRotation
}

// Rotate manually triggers log rotation if supported
func (l *Logger) Rotate() error {
	if l.lumberjack != nil {
		return l.lumberjack.Rotate() //nolint:wrapcheck // simple logger operation
	}
	return nil
}

// NewWithConfig creates a new logger instance using configuration
func NewWithConfig(cfg *config.Config, workDir string) (*Logger, error) {
	// Determine log path
	var logDir, logFile string
	if cfg.Logging.Path != "" {
		logFile = cfg.Logging.Path
		logDir = filepath.Dir(logFile)
	} else {
		logDir = filepath.Join(workDir, ".claude", "bumpers")
		logFile = filepath.Join(logDir, "bumpers.log")
	}

	// Try to create log file, fall back to stderr if it fails
	var writer io.Writer
	var file *os.File

	// Create log directory if it doesn't exist
	err := os.MkdirAll(logDir, 0o750)
	if err != nil {
		writer = os.Stderr
	} else {
		// Open log file for appending
		file, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // safe log path
		if err != nil {
			writer = os.Stderr
		} else {
			writer = file
		}
	}

	// Create zerolog logger
	zl := zerolog.New(writer).With().Timestamp().Logger()

	// Set log level if specified in config
	if cfg.Logging.Level != "" {
		level, err := zerolog.ParseLevel(cfg.Logging.Level)
		if err == nil {
			zl = zl.Level(level)
		}
	}

	return &Logger{
		Logger:           zl,
		file:             file,
		supportsRotation: cfg.Logging.MaxSize > 0,
	}, nil
}

// Close closes the log file
func (l *Logger) Close() error {
	var err error
	l.closeOnce.Do(func() {
		if l.file != nil {
			err = l.file.Close()
		}
	})
	return err //nolint:wrapcheck // simple logger cleanup
}

// InitLogger initializes the global logger instance
func InitLogger(workDir string) error {
	logDir := filepath.Join(workDir, ".claude", "bumpers")
	logFilePath := filepath.Join(logDir, "bumpers.log")

	// Create log directory if it doesn't exist
	err := os.MkdirAll(logDir, 0o750)
	if err != nil {
		return err //nolint:wrapcheck // simple logger setup
	}

	// Open log file for appending
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // safe path
	if err != nil {
		return err //nolint:wrapcheck // simple logger setup
	}

	// Set global logger
	log.Logger = zerolog.New(file).With().Timestamp().Logger()

	// Note: Global logger file reference not stored for cleanup
	// Consider using the Logger instance approach instead of global logger

	return nil
}
