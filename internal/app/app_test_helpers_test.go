package app

import (
	"os"
	"path/filepath"
	"testing"
)

// Test constants
const (
	testConfigContent = `rules:
  - pattern: "test *"
    action: allow`
	expectedConfigFileName = "test-config.yaml"
	emptyLogOutput         = ""
)

// Test assertion helpers
func assertContextNotNil(t *testing.T, ctx any) {
	t.Helper()
	if ctx == nil {
		t.Fatal("Expected context to be non-nil")
	}
}

func assertFunctionNotNil(t *testing.T, fn any) {
	t.Helper()
	if fn == nil {
		t.Fatal("Expected function to be non-nil")
	}
}

func assertEmptyLogOutput(t *testing.T, output string) {
	t.Helper()
	if output != emptyLogOutput {
		t.Errorf("Expected empty log output initially, got: %s", output)
	}
}

func assertContextNotCancelled(t *testing.T, ctx <-chan struct{}) {
	t.Helper()
	select {
	case <-ctx:
		t.Error("Context should not be cancelled initially")
	default:
		// Expected - context should not be done
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("Expected file to exist at %s", path)
	}
}

func assertFileContent(t *testing.T, path, expectedContent string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, string(content))
	}
}

func assertAbsolutePath(t *testing.T, path string) {
	t.Helper()
	if !filepath.IsAbs(path) {
		t.Errorf("Expected absolute path, got %s", path)
	}
}

func assertFileName(t *testing.T, path, expectedName string) {
	t.Helper()
	if filepath.Base(path) != expectedName {
		t.Errorf("Expected filename %s, got %s", expectedName, filepath.Base(path))
	}
}

// Test cases
func TestSetupTestWithContext(t *testing.T) {
	t.Parallel()
	ctx, getLogOutput := setupTestWithContext(t)

	assertContextNotNil(t, ctx)
	assertFunctionNotNil(t, getLogOutput)

	output := getLogOutput()
	assertEmptyLogOutput(t, output)

	assertContextNotCancelled(t, ctx.Done())
}

func TestCreateTempConfig(t *testing.T) {
	t.Parallel()
	configPath := createTempConfig(t, testConfigContent)

	assertFileExists(t, configPath)
	assertFileContent(t, configPath, testConfigContent)
	assertFileName(t, configPath, expectedConfigFileName)
	assertAbsolutePath(t, configPath)
}
