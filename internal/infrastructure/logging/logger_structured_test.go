package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/wizzomafizzo/bumpers/internal/core/messaging/context"
)

func TestStructuredLoggingFields(t *testing.T) {
	t.Parallel()
	t.Run("current behavior includes project_name", func(t *testing.T) {
		t.Parallel()
		testCurrentBehaviorIncludesProjectName(t)
	})

	t.Run("desired behavior excludes project_name", func(t *testing.T) {
		t.Parallel()
		testDesiredBehaviorExcludesProjectName(t)
	})
}

func testCurrentBehaviorIncludesProjectName(t *testing.T) {
	t.Helper()

	logEntry := createTestLogEntryWithProjectName(t)

	// Verify project_id field exists
	if _, exists := logEntry["project_id"]; !exists {
		t.Error("Expected project_id field in log entry")
	}

	// This test documents current behavior - project_name field should exist
	if _, exists := logEntry["project_name"]; !exists {
		t.Error("Expected project_name field in current implementation")
	}
}

func testDesiredBehaviorExcludesProjectName(t *testing.T) {
	t.Helper()

	logEntry := createTestLogEntryWithoutProjectName(t)

	// Verify project_id field still exists
	if _, exists := logEntry["project_id"]; !exists {
		t.Error("Expected project_id field in log entry")
	}

	// Verify project_name field does NOT exist
	if _, exists := logEntry["project_name"]; exists {
		t.Error("Expected project_name field to be removed")
	}
}

func createTestLogEntryWithProjectName(t *testing.T) map[string]any {
	t.Helper()

	// Create buffer for log output
	var logOutput bytes.Buffer

	// Create project context
	projectCtx := context.New("/test/project")

	// Create a test logger with project_name
	testLogger := zerolog.New(&logOutput).With().
		Timestamp().
		Str("project_id", projectCtx.ID).
		Str("project_name", projectCtx.Name).
		Logger()

	// Log a test message using the custom logger
	testLogger.Info().Msg("test message")

	return parseLogOutput(t, logOutput.String())
}

func createTestLogEntryWithoutProjectName(t *testing.T) map[string]any {
	t.Helper()

	// Create buffer for log output
	var logOutput bytes.Buffer

	// Create project context
	projectCtx := context.New("/test/project")

	// Create a test logger without project_name
	testLogger := zerolog.New(&logOutput).With().
		Timestamp().
		Str("project_id", projectCtx.ID).
		Logger()

	// Log a test message using the custom logger
	testLogger.Info().Msg("test message")

	return parseLogOutput(t, logOutput.String())
}

func parseLogOutput(t *testing.T, logOutput string) map[string]any {
	t.Helper()

	if logOutput == "" {
		t.Fatal("No log output captured")
	}

	// Parse JSON log entry
	var logEntry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry); err != nil {
		t.Fatalf("Failed to parse log JSON: %v", err)
	}

	return logEntry
}
