package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/wizzomafizzo/bumpers/internal/config"
)

// testLoggerSetup creates a new logger instance and returns the temp directory and log file path
func testLoggerSetup(t *testing.T) (tempDir, logFile string, logger *Logger) {
	t.Helper()
	tempDir = t.TempDir()

	// Create logger with specific working directory
	var err error
	logger, err = New(tempDir)
	if err != nil {
		t.Fatalf("New logger failed: %v", err)
	}

	logFile = filepath.Join(tempDir, ".claude", "bumpers", "bumpers.log")
	return
}

// verifyLogContent checks that the log file contains the expected message and structured data
func verifyLogContent(t *testing.T, logFile, expectedMessage, expectedStructuredData string) {
	t.Helper()

	content, err := os.ReadFile(logFile) //nolint:gosec // controlled log file path in test
	if err != nil {
		t.Fatalf("Expected log file to be created at %s: %v", logFile, err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, expectedMessage) {
		t.Errorf("Expected log file to contain '%s'", expectedMessage)
	}

	if !strings.Contains(contentStr, expectedStructuredData) {
		t.Error("Expected log file to contain structured data")
	}
}

func TestLoggerWritesToFile(t *testing.T) {
	t.Parallel()

	_, logFile, logger := testLoggerSetup(t)
	defer func() { _ = logger.Close() }()

	// Log a test message
	logger.Info().Str("key", "value").Msg("test message")

	verifyLogContent(t, logFile, "test message", "\"key\":\"value\"")
}

func TestNewLoggerWithWorkingDirectory(t *testing.T) {
	t.Parallel()

	_, logFile, logger := testLoggerSetup(t)
	defer func() { _ = logger.Close() }()

	// Log a test message
	logger.Info().Bool("initialized", true).Msg("test message from initialized logger")

	verifyLogContent(t, logFile, "test message from initialized logger", "\"initialized\":true")
}

func TestMultipleLoggerInstances(t *testing.T) {
	t.Parallel()

	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// First logger instance
	logger1, err := New(tempDir1)
	if err != nil {
		t.Fatalf("First New failed: %v", err)
	}
	defer func() { _ = logger1.Close() }()

	logger1.Info().Int("test", 1).Msg("first message")

	// Second logger instance with different directory
	logger2, err := New(tempDir2)
	if err != nil {
		t.Fatalf("Second New failed: %v", err)
	}
	defer func() { _ = logger2.Close() }()

	logger2.Info().Int("test", 2).Msg("second message")

	// Check that second log file was created in the correct directory
	logFile := filepath.Join(tempDir2, ".claude", "bumpers", "bumpers.log")

	content, err := os.ReadFile(logFile) //nolint:gosec // controlled log file path in test
	if err != nil {
		t.Fatalf("Expected log file to be created at %s: %v", logFile, err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "second message") {
		t.Error("Expected log file to contain 'second message'")
	}
}

func TestLoggerWithConfig(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := &config.Config{
		Logging: config.LoggingConfig{
			Level:      "debug",
			Path:       "", // Use default path
			MaxSize:    5,  // 5MB
			MaxBackups: 3,
			MaxAge:     7,
		},
	}

	logger, err := NewWithConfig(cfg, tempDir)
	if err != nil {
		t.Fatalf("NewWithConfig failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Test that logger works with config
	logger.Debug().Str("config_test", "true").Msg("debug message")

	// Verify log file was created at default path
	logFile := filepath.Join(tempDir, ".claude", "bumpers", "bumpers.log")
	verifyLogContent(t, logFile, "debug message", "\"config_test\":\"true\"")
}

func TestLoggerWithConfigLevel(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := &config.Config{
		Logging: config.LoggingConfig{
			Level: "info", // Set to info level - debug messages should be filtered out
		},
	}

	logger, err := NewWithConfig(cfg, tempDir)
	if err != nil {
		t.Fatalf("NewWithConfig failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Log debug message (should be filtered out)
	logger.Debug().Str("level_test", "debug").Msg("debug message should not appear")

	// Log info message (should appear)
	logger.Info().Str("level_test", "info").Msg("info message should appear")

	// Verify only info message appears in log
	logFile := filepath.Join(tempDir, ".claude", "bumpers", "bumpers.log")
	content, err := os.ReadFile(logFile) //nolint:gosec // controlled log file path in test
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if strings.Contains(contentStr, "debug message should not appear") {
		t.Error("Debug message should have been filtered out at info level")
	}

	if !strings.Contains(contentStr, "info message should appear") {
		t.Error("Info message should appear at info level")
	}
}

func TestLoggerWithRotationConfig(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := &config.Config{
		Logging: config.LoggingConfig{
			MaxSize:    1, // 1MB for testing
			MaxBackups: 2,
			MaxAge:     5,
		},
	}

	logger, err := NewWithConfig(cfg, tempDir)
	if err != nil {
		t.Fatalf("NewWithConfig failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Test basic functionality
	logger.Info().Str("rotation_test", "true").Msg("test message with rotation config")

	// Verify log file was created
	logFile := filepath.Join(tempDir, ".claude", "bumpers", "bumpers.log")
	verifyLogContent(t, logFile, "test message with rotation config", "\"rotation_test\":\"true\"")
}

func TestLoggerSupportsRotation(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := &config.Config{
		Logging: config.LoggingConfig{
			MaxSize:    1, // When MaxSize > 0, should use lumberjack
			MaxBackups: 2,
			MaxAge:     5,
		},
	}

	logger, err := NewWithConfig(cfg, tempDir)
	if err != nil {
		t.Fatalf("NewWithConfig failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Test that logger has a Rotate method (exposed through the logger interface)
	// This test will fail until we implement lumberjack
	if !logger.SupportsRotation() {
		t.Error("Logger should support rotation when MaxSize > 0")
	}
}

func TestLoggerFallbackToStderr(t *testing.T) {
	t.Parallel()

	// Use a read-only directory to force file creation to fail
	readOnlyDir := t.TempDir()
	err := os.Chmod(readOnlyDir, 0o444) //nolint:gosec // test setup needs readable directory
	if err != nil {
		t.Fatalf("Failed to make directory read-only: %v", err)
	}
	// Restore permissions for cleanup
	defer func() { _ = os.Chmod(readOnlyDir, 0o755) }() //nolint:gosec // test cleanup

	cfg := &config.Config{
		Logging: config.LoggingConfig{
			Path: filepath.Join(readOnlyDir, "cannot-create.log"),
		},
	}

	// This should not fail even if file creation fails - should fall back to stderr
	logger, err := NewWithConfig(cfg, "")
	if err != nil {
		t.Fatalf("NewWithConfig should not fail even if file creation fails, got: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Test that logger still works (logs to stderr)
	logger.Info().Str("fallback_test", "true").Msg("fallback message")

	// We can't easily verify stderr output in tests, but the fact that
	// logger creation didn't fail shows that fallback is working
}

func TestLoggerRotateMethod(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := &config.Config{
		Logging: config.LoggingConfig{
			MaxSize:    1, // Enable rotation
			MaxBackups: 2,
			MaxAge:     5,
		},
	}

	logger, err := NewWithConfig(cfg, tempDir)
	if err != nil {
		t.Fatalf("NewWithConfig failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Test that we can call Rotate method when rotation is supported
	if logger.SupportsRotation() {
		err := logger.Rotate()
		if err != nil {
			t.Errorf("Rotate() should not fail when rotation is supported, got: %v", err)
		}
	} else {
		t.Error("Logger should support rotation when MaxSize > 0")
	}
}

func TestLoggerCloseIdempotent(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := &config.Config{
		Logging: config.LoggingConfig{},
	}

	logger, err := NewWithConfig(cfg, tempDir)
	if err != nil {
		t.Fatalf("NewWithConfig failed: %v", err)
	}

	// Close should work the first time
	err = logger.Close()
	if err != nil {
		t.Errorf("First Close() failed: %v", err)
	}

	// Close should be safe to call multiple times
	err = logger.Close()
	if err != nil {
		t.Errorf("Second Close() should not fail: %v", err)
	}

	// Third time should also be safe
	err = logger.Close()
	if err != nil {
		t.Errorf("Third Close() should not fail: %v", err)
	}
}

func TestInitLogger(t *testing.T) { //nolint:paralleltest // modifies global logger state
	tempDir := t.TempDir()

	// Initialize global logger
	err := InitLogger(tempDir)
	if err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}

	// Verify log file was created
	logFile := filepath.Join(tempDir, ".claude", "bumpers", "bumpers.log")
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Expected log file to be created at %s", logFile)
	}
}

func TestInitLoggerSetsGlobalLogger(t *testing.T) { //nolint:paralleltest // modifies global logger state
	tempDir := t.TempDir()

	// Initialize global logger
	err := InitLogger(tempDir)
	if err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}

	// Use global logger to write a message
	log.Info().Str("test", "global").Msg("test global logger message")

	// Verify message was written to log file
	logFile := filepath.Join(tempDir, ".claude", "bumpers", "bumpers.log")
	verifyLogContent(t, logFile, "test global logger message", "\"test\":\"global\"")
}
