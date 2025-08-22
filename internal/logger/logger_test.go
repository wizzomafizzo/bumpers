package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/wizzomafizzo/bumpers/internal/constants"
)

func TestInit(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	// Test basic initialization
	err := Init(tempDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify log directory was created
	logDir := filepath.Join(tempDir, constants.ClaudeDir, constants.AppSubDir)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("Expected log directory to be created")
	}

	// Verify log file exists after writing a message
	log.Info().Msg("test message")
	logFile := filepath.Join(logDir, constants.LogFilename)
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Expected log file to be created")
	}
}

func TestInitTest(_ *testing.T) { //nolint:paralleltest // modifies global logger state
	// Note: No t.Parallel() as this modifies global logger state
	// Just verify it doesn't panic
	InitTest()
	log.Info().Msg("test message") // Should go to discard
}
