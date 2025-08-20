package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// createTempConfig creates a temporary config file for testing
func createTempConfig(t *testing.T, content string) string {
	t.Helper()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")
	err := os.WriteFile(configPath, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}
	return configPath
}

func TestProcessHook(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - name: "block-go-test"
    pattern: "go test"
    action: "deny"
    message: "Use make test instead for better TDD integration"
    alternatives:
      - "make test          # Run all tests"
      - "make test-unit     # Run unit tests only"
    use_claude: false`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		}
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return a response since "go test" matches a rule
	if response == "" {
		t.Error("Expected non-empty response for blocked command")
	}

	if !strings.Contains(response, "make test") {
		t.Error("Response should suggest make test alternative")
	}
}

func TestProcessHookAllowed(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - name: "block-go-test"
    pattern: "go test"
    action: "deny"
    message: "Use make test instead for better TDD integration"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"tool_input": {
			"command": "make test",
			"description": "Run tests with make"
		}
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return empty response since "make test" doesn't match any deny rule
	if response != "" {
		t.Errorf("Expected empty response for allowed command, got %s", response)
	}
}

func TestProcessHookDangerousCommand(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - name: "dangerous-rm"
    pattern: "rm -rf /*"
    action: "deny"
    message: "⚠️  Dangerous rm command detected"
    alternatives:
      - "Be more specific with your rm command"
      - "Use a safer alternative like moving to trash"
    use_claude: true
    claude_prompt: "Explain why this rm command is dangerous and suggest safer alternatives"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"tool_input": {
			"command": "rm -rf /tmp",
			"description": "Remove directory"
		}
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return a response since dangerous rm command matches a rule
	if response == "" {
		t.Error("Expected non-empty response for dangerous command")
	}
}

func TestProcessHookPatternMatching(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - name: "block-go-test"
    pattern: "go test"
    action: "deny"
    message: "Use make test instead for better TDD integration"
    alternatives:
      - "make test          # Run all tests"
      - "make test-unit     # Run unit tests only"
    use_claude: false`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"tool_input": {
			"command": "go test -v ./pkg/...",
			"description": "Run verbose tests"
		}
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return a response since this matches "go test.*" pattern
	if response == "" {
		t.Error("Expected non-empty response for go test pattern match")
	}

	if !strings.Contains(response, "make test") {
		t.Error("Response should suggest make test alternative")
	}
}

func TestConfigurationIsUsed(t *testing.T) {
	t.Parallel()

	// This test ensures we're actually using the config file by checking for
	// a specific message from the config rather than hardcoded responses
	configContent := `rules:
  - name: "block-go-test"
    pattern: "go test"
    action: "deny"
    message: "Use make test instead for better TDD integration"
    use_claude: false`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run all tests"
		}
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should contain the exact message from config file
	if !strings.Contains(response, "better TDD integration") {
		t.Error("Response should contain message from config file")
	}
}

func TestTestCommand(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - name: "block-go-test"
    pattern: "go test"
    action: "deny"
    message: "Use make test instead for better TDD integration"
    use_claude: false`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	result, err := app.TestCommand("go test ./...")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !strings.Contains(result, "blocked") {
		t.Error("Result should indicate command is blocked")
	}
}

func TestInitialize(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")
	app := NewApp(configPath)

	err := app.Initialize()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should create config file
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Expected config file to be created")
	}
}

func TestStatus(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - name: "block-go-test"
    pattern: "go test"
    action: "deny"
    message: "Use make test instead"
    use_claude: false`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	status, err := app.Status()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if status == "" {
		t.Error("Expected non-empty status message")
	}
}

func TestInstallUsesProjectClaudeDirectory(t *testing.T) { //nolint:paralleltest // changes working directory
	// Don't run in parallel - changes working directory

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Change to temp directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	app := NewApp(configPath)
	err = app.Initialize()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should create project .claude directory, not home directory
	claudeDir := filepath.Join(tempDir, ".claude")
	if _, statErr := os.Stat(claudeDir); os.IsNotExist(statErr) {
		t.Error("Expected .claude directory to be created in project directory")
	}

	// Should NOT create in home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	homeClaudeDir := filepath.Join(homeDir, ".claude", "settings.local.json")
	if _, err := os.Stat(homeClaudeDir); err == nil {
		// Check if it was modified recently (within last 10 seconds)
		stat, _ := os.Stat(homeClaudeDir)
		if time.Since(stat.ModTime()) < 10*time.Second {
			t.Error("Install should not modify home directory .claude settings")
		}
	}
}

func TestInitializeInstallsClaudeHooksInProjectDirectory(t *testing.T) { //nolint:paralleltest // changes cwd
	// Don't run in parallel - changes working directory

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Change to temp directory to simulate working in project directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	app := NewApp(configPath)
	err = app.Initialize()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should create config file
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		t.Error("Expected config file to be created")
	}

	// Should create .claude/settings.local.json in current directory
	claudeDir := filepath.Join(tempDir, ".claude")
	localSettingsPath := filepath.Join(claudeDir, "settings.local.json")
	if _, statErr := os.Stat(localSettingsPath); os.IsNotExist(statErr) {
		t.Error("Expected .claude/settings.local.json to be created in project directory")
	}

	// Check that hook was installed with absolute paths
	content, err := os.ReadFile(localSettingsPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "bumpers") {
		t.Error("Expected settings.local.json to contain bumpers hook")
	}

	// Hook should contain absolute path to binary and config
	if !strings.Contains(contentStr, tempDir) {
		t.Error("Expected hook command to contain absolute path to project directory")
	}
}

func TestProcessHookLogsErrors(t *testing.T) {
	t.Parallel()

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

	// Use non-existent config path to trigger error
	app := NewApp("non-existent-config.yml")

	hookInput := `{
		"tool_input": {
			"command": "test command",
			"description": "Test"
		}
	}`

	_, _ = app.ProcessHook(strings.NewReader(hookInput))

	// Check if error was logged
	logFile := filepath.Join(tempDir, ".claude", "bumpers", "bumpers.log")
	content, err := os.ReadFile(logFile) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("Expected log file to be created: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "config") && !strings.Contains(contentStr, "error") {
		t.Error("Expected error to be logged when config file doesn't exist")
	}
}

func TestInstallActuallyAddsHook(t *testing.T) { //nolint:paralleltest // changes working directory
	// Don't run in parallel to avoid directory race conditions

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Change to temp directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		// Always restore directory, ignore errors if temp dir was cleaned up
		_ = os.Chdir(originalWd)
	}()

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create bin directory and dummy bumpers binary
	binDir := filepath.Join(tempDir, "bin")
	err = os.MkdirAll(binDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	bumpersPath := filepath.Join(binDir, "bumpers")
	err = os.WriteFile(bumpersPath, []byte("#!/bin/bash\necho test"), 0o750) //nolint:gosec // exec perms
	if err != nil {
		t.Fatal(err)
	}

	app := NewApp(configPath)
	err = app.Initialize()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that Bash hook was actually added
	claudeDir := filepath.Join(tempDir, ".claude")
	localSettingsPath := filepath.Join(claudeDir, "settings.local.json")
	content, err := os.ReadFile(localSettingsPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `"matcher": "Bash"`) {
		t.Error("Expected Bash hook to be added to settings.local.json")
	}

	if !strings.Contains(contentStr, "bumpers") {
		t.Error("Expected hook command to contain bumpers")
	}
}
