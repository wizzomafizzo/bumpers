package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoggerWritesToFile(t *testing.T) {
	t.Parallel()

	// Change to temp directory for this test
	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Log a test message
	Info("test message", "key", "value")

	// Check if log file was created and contains the message
	logFile := filepath.Join(tempDir, ".claude", "bumpers", "bumpers.log")

	// Give it a moment for the write to complete
	time.Sleep(10 * time.Millisecond)

	content, err := os.ReadFile(logFile) //nolint:gosec // controlled log file path in test
	if err != nil {
		t.Fatalf("Expected log file to be created at %s: %v", logFile, err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "test message") {
		t.Error("Expected log file to contain 'test message'")
	}

	if !strings.Contains(contentStr, "\"key\":\"value\"") {
		t.Error("Expected log file to contain structured data")
	}
}
