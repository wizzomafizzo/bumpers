package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/paths"
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
	logDir := filepath.Join(workDir, paths.ClaudeDir, paths.AppSubDir)
	logFile := filepath.Join(logDir, paths.LogFilename)

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

// NewWithConfig creates a new logger instance using configuration with lumberjack rotation
// If cfg is nil, uses default configuration
func NewWithConfig(cfg *config.Config, workDir string) (*Logger, error) {
	// Use default config if none provided
	if cfg == nil {
		cfg = &config.Config{
			Logging: config.LoggingConfig{
				Level:      "info",
				Path:       "",
				MaxSize:    10,
				MaxBackups: 3,
				MaxAge:     30,
			},
		}
	}

	// Set defaults for lumberjack if not specified
	if cfg.Logging.MaxSize == 0 {
		cfg.Logging.MaxSize = 10
	}
	if cfg.Logging.MaxBackups == 0 {
		cfg.Logging.MaxBackups = 3
	}
	if cfg.Logging.MaxAge == 0 {
		cfg.Logging.MaxAge = 30
	}

	// Determine log path
	var logFile string
	if cfg.Logging.Path != "" {
		logFile = cfg.Logging.Path
	} else {
		logDir := filepath.Join(workDir, paths.ClaudeDir, paths.AppSubDir)
		logFile = filepath.Join(logDir, paths.LogFilename)

		// Create log directory if it doesn't exist
		err := os.MkdirAll(logDir, 0o750)
		if err != nil {
			return nil, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
		}
	}

	// Create lumberjack logger for automatic rotation
	lj := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    cfg.Logging.MaxSize,    // MB
		MaxBackups: cfg.Logging.MaxBackups, // number of old files to keep
		MaxAge:     cfg.Logging.MaxAge,     // days
	}

	// Create zerolog logger
	zl := zerolog.New(lj).With().Timestamp().Logger()

	// Set log level if specified in config
	if cfg.Logging.Level != "" {
		level, err := zerolog.ParseLevel(cfg.Logging.Level)
		if err == nil {
			zl = zl.Level(level)
		}
	}

	return &Logger{
		Logger:           zl,
		lumberjack:       lj,
		supportsRotation: true,
	}, nil
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
	logDir := filepath.Join(workDir, paths.ClaudeDir, paths.AppSubDir)
	logFilePath := filepath.Join(logDir, paths.LogFilename)

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
