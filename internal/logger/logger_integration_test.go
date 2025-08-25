//go:build integration

package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/wizzomafizzo/bumpers/internal/constants"
	"github.com/wizzomafizzo/bumpers/internal/context"
)

func TestInit(t *testing.T) { //nolint:paralleltest // modifies global logger state
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

func TestInitTest(t *testing.T) { //nolint:paralleltest // modifies global logger state
	// Note: No t.Parallel() as this modifies global logger state
	// Just verify it doesn't panic
	InitTest()
	log.Info().Msg("test message") // Should go to discard
}

func TestInitWithProjectContext(t *testing.T) { //nolint:paralleltest // modifies global logger state
	// This will fail until we implement the new XDG + project context approach
	projectCtx := &context.ProjectContext{
		ID:   "test-a1b2",
		Name: "test",
		Path: "/tmp/test",
	}

	err := InitWithProjectContext(projectCtx)
	if err != nil {
		t.Fatalf("InitWithProjectContext failed: %v", err)
	}

	// Write a log message and verify it includes project ID
	log.Info().Msg("test message with project context")
	// TODO: Add verification that log entries include project ID
}
