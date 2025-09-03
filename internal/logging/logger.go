package logging

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/rs/zerolog"
	"github.com/spf13/afero"
	"github.com/wizzomafizzo/bumpers/internal/storage"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	maxLogSizeMB  = 10
	maxLogBackups = 3
	maxLogAgeDays = 30
)

// Log levels - aliases for zerolog levels
const (
	PanicLevel = zerolog.PanicLevel
	FatalLevel = zerolog.FatalLevel
	ErrorLevel = zerolog.ErrorLevel
	WarnLevel  = zerolog.WarnLevel
	InfoLevel  = zerolog.InfoLevel
	DebugLevel = zerolog.DebugLevel
	TraceLevel = zerolog.TraceLevel
)

// Config defines the configuration for logger creation
type Config struct {
	Writer    io.Writer
	ProjectID string
	Level     zerolog.Level
}

// New creates a new context with a logger attached
// For production: provide fs and projectID, leave Writer nil for file logging
// For tests: provide a custom Writer (like strings.Builder) for in-memory logging
func New(ctx context.Context, fs afero.Fs, config Config) (context.Context, error) {
	var writer io.Writer

	if config.Writer != nil {
		// Use provided writer (typically for tests)
		writer = config.Writer
	} else {
		// Create file writer for production
		if fs == nil {
			return nil, errors.New("filesystem required when no writer provided")
		}

		storageManager := storage.New(fs)
		logFile, err := storageManager.GetLogPath()
		if err != nil {
			return nil, fmt.Errorf("failed to get log path: %w", err)
		}

		writer = &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    maxLogSizeMB,
			MaxBackups: maxLogBackups,
			MaxAge:     maxLogAgeDays,
		}
	}

	// Create logger with consistent configuration
	logger := zerolog.New(writer).With().
		Timestamp().
		Str("project_id", config.ProjectID).
		Logger().
		Level(config.Level)

	return logger.WithContext(ctx), nil
}

// Get retrieves the logger from the provided context
// Returns the logger associated with the context, or a disabled logger if none exists
func Get(ctx context.Context) *zerolog.Logger {
	return zerolog.Ctx(ctx)
}
