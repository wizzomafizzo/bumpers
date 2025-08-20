package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testLoggerSetup initializes the logger and returns the temp directory and log file path
func testLoggerSetup(t *testing.T) (tempDir, logFile string) {
	t.Helper()
	tempDir = t.TempDir()
	Reset() // Ensure clean state

	// Initialize logger with specific working directory
	err := Initialize(tempDir)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	logFile = filepath.Join(tempDir, ".claude", "bumpers", "bumpers.log")
	return
}

// verifyLogContent checks that the log file contains the expected message and structured data
func verifyLogContent(t *testing.T, logFile, expectedMessage, expectedStructuredData string) {
	t.Helper()

	// Give it a moment for the write to complete
	time.Sleep(10 * time.Millisecond)

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

func TestLoggerWritesToFile(t *testing.T) { //nolint:paralleltest // Logger tests share global state
	_, logFile := testLoggerSetup(t)

	// Log a test message
	Info("test message", "key", "value")

	verifyLogContent(t, logFile, "test message", "\"key\":\"value\"")
}

func TestInitializeWithWorkingDirectory(t *testing.T) { //nolint:paralleltest // Logger tests share global state
	_, logFile := testLoggerSetup(t)

	// Log a test message
	Info("test message from initialized logger", "initialized", "true")

	verifyLogContent(t, logFile, "test message from initialized logger", "\"initialized\":\"true\"")
}

func TestResetAllowsReinitialization(t *testing.T) { //nolint:paralleltest // Logger tests share global state
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// First initialization
	err := Initialize(tempDir1)
	if err != nil {
		t.Fatalf("First Initialize failed: %v", err)
	}

	Info("first message", "test", "1")

	// Reset logger to allow reinitialization
	Reset()

	// Second initialization with different directory
	err = Initialize(tempDir2)
	if err != nil {
		t.Fatalf("Second Initialize failed: %v", err)
	}

	Info("second message", "test", "2")

	// Check that second log file was created in the correct directory
	logFile := filepath.Join(tempDir2, ".claude", "bumpers", "bumpers.log")

	time.Sleep(10 * time.Millisecond)

	content, err := os.ReadFile(logFile) //nolint:gosec // controlled log file path in test
	if err != nil {
		t.Fatalf("Expected log file to be created at %s: %v", logFile, err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "second message") {
		t.Error("Expected log file to contain 'second message'")
	}
}
