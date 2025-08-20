package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wizzomafizzo/bumpers/internal/logger"
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

func TestNewAppWithWorkDir(t *testing.T) {
	t.Parallel()

	configPath := "/path/to/config.yml"
	workDir := "/test/working/directory"

	app := NewAppWithWorkDir(configPath, workDir)

	if app.configPath != configPath {
		t.Errorf("Expected configPath %s, got %s", configPath, app.configPath)
	}

	if app.workDir != workDir {
		t.Errorf("Expected workDir %s, got %s", workDir, app.workDir)
	}
}

func TestInstallClaudeHooksWithWorkDir(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Create bin directory and dummy bumpers binary in the temp directory
	binDir := filepath.Join(tempDir, "bin")
	err := os.MkdirAll(binDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	bumpersPath := filepath.Join(binDir, "bumpers")
	err = os.WriteFile(bumpersPath, []byte("#!/bin/bash\necho test"), 0o750) //nolint:gosec // exec perms
	if err != nil {
		t.Fatal(err)
	}

	// Create the app with the temp directory as the working directory
	app := NewAppWithWorkDir(configPath, tempDir)

	// This should use the tempDir as the working directory instead of calling os.Getwd()
	err = app.installClaudeHooks()
	if err != nil {
		t.Fatalf("installClaudeHooks failed: %v", err)
	}

	// Check that the Claude settings file was created in the correct location
	claudeSettingsPath := filepath.Join(tempDir, ".claude", "settings.local.json")
	if _, err := os.Stat(claudeSettingsPath); os.IsNotExist(err) {
		t.Errorf("Claude settings file should exist at %s", claudeSettingsPath)
	}
}

func TestProcessHook(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - pattern: "go test"
    response: "Use make test instead for better TDD integration"
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
  - pattern: "go test"
    response: "Use make test instead for better TDD integration"`

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
  - pattern: "rm -rf /*"
    response: "⚠️  Dangerous rm command detected"
    alternatives:
      - "Be more specific with your rm command"
      - "Use a safer alternative like moving to trash"
    use_claude: true
    prompt: "Explain why this rm command is dangerous and suggest safer alternatives"`

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
  - pattern: "go test"
    response: "Use make test instead for better TDD integration"
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
  - pattern: "go test"
    response: "Use make test instead for better TDD integration"
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
  - pattern: "go test"
    response: "Use make test instead for better TDD integration"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	result, err := app.TestCommand("go test ./...")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should contain the response message
	if !strings.Contains(result, "Use make test instead") {
		t.Errorf("Result should contain the response message, got: %s", result)
	}
}

func TestInitialize(t *testing.T) { //nolint:paralleltest // Test resets shared logger state
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Reset logger state for clean test
	logger.Reset()

	// Create bin directory and dummy bumpers binary (required for Initialize)
	binDir := filepath.Join(tempDir, "bin")
	err := os.MkdirAll(binDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	bumpersPath := filepath.Join(binDir, "bumpers")
	err = os.WriteFile(bumpersPath, []byte("#!/bin/bash\necho test"), 0o750) //nolint:gosec // exec perms
	if err != nil {
		t.Fatal(err)
	}

	// Use the new constructor with working directory instead of os.Chdir()
	app := NewAppWithWorkDir(configPath, tempDir)

	err = app.Initialize()
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
  - pattern: "go test"
    response: "Use make test instead"
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

func TestInstallUsesProjectClaudeDirectory(t *testing.T) { //nolint:paralleltest // Test resets shared logger state
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Reset logger state for clean test
	logger.Reset()

	// Create bin directory and dummy bumpers binary
	binDir := filepath.Join(tempDir, "bin")
	err := os.MkdirAll(binDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	bumpersPath := filepath.Join(binDir, "bumpers")
	err = os.WriteFile(bumpersPath, []byte("#!/bin/bash\necho test"), 0o750) //nolint:gosec // exec perms
	if err != nil {
		t.Fatal(err)
	}

	// Use the new constructor with working directory instead of os.Chdir()
	app := NewAppWithWorkDir(configPath, tempDir)

	err = app.Initialize()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should create project .claude directory
	claudeDir := filepath.Join(tempDir, ".claude")
	if _, statErr := os.Stat(claudeDir); os.IsNotExist(statErr) {
		t.Error("Expected .claude directory to be created in project directory")
	}
}

func TestInitializeInstallsClaudeHooksInProjectDirectory(t *testing.T) { //nolint:paralleltest // Resets logger
	// Test resets shared logger state
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Reset logger state for clean test
	logger.Reset()

	// Create bin directory and dummy bumpers binary
	binDir := filepath.Join(tempDir, "bin")
	err := os.MkdirAll(binDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	bumpersPath := filepath.Join(binDir, "bumpers")
	err = os.WriteFile(bumpersPath, []byte("#!/bin/bash\necho test"), 0o750) //nolint:gosec // exec perms
	if err != nil {
		t.Fatal(err)
	}

	// Use the new constructor with working directory instead of os.Chdir()
	app := NewAppWithWorkDir(configPath, tempDir)

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

func TestProcessHookLogsErrors(t *testing.T) { //nolint:paralleltest // Test resets shared logger state
	tempDir := t.TempDir()

	// Reset logger state for clean test
	logger.Reset()

	// Initialize logger with temp directory to avoid creating .claude in wrong place
	err := logger.Initialize(tempDir)
	if err != nil {
		t.Fatalf("Logger initialization failed: %v", err)
	}

	// Use non-existent config path to trigger error
	app := NewAppWithWorkDir("non-existent-config.yml", tempDir)

	hookInput := `{
		"tool_input": {
			"command": "test command",
			"description": "Test"
		}
	}`

	// This should trigger an error (logging is a side effect we can't easily test with global logger)
	result, err := app.ProcessHook(strings.NewReader(hookInput))
	if err == nil {
		t.Fatalf("Expected ProcessHook to return error for non-existent config, got result: %s", result)
	}

	// Verify the error is related to config loading
	if !strings.Contains(err.Error(), "config") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Expected config-related error, got: %v", err)
	}
}

func TestInstallActuallyAddsHook(t *testing.T) { //nolint:paralleltest // Test resets shared logger state
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Reset logger state for clean test
	logger.Reset()

	// Create bin directory and dummy bumpers binary
	binDir := filepath.Join(tempDir, "bin")
	err := os.MkdirAll(binDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	bumpersPath := filepath.Join(binDir, "bumpers")
	err = os.WriteFile(bumpersPath, []byte("#!/bin/bash\necho test"), 0o750) //nolint:gosec // exec perms
	if err != nil {
		t.Fatal(err)
	}

	// Use the new constructor with working directory instead of os.Chdir()
	app := NewAppWithWorkDir(configPath, tempDir)

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

func TestProcessHookSimplifiedSchemaAlwaysDenies(t *testing.T) {
	t.Parallel()

	// Setup test config with simplified schema (no name or action fields)
	// Any pattern match should result in denial
	configContent := `rules:
  - pattern: "go test"
    response: "Use make test instead"
  - pattern: "rm -rf"
    response: "Dangerous command detected"
`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Test first rule - should be blocked because it matches (no action field needed)
	hookInput1 := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		}
	}`
	result1, err := app.ProcessHook(strings.NewReader(hookInput1))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}
	if result1 != "Use make test instead" {
		t.Errorf("Expected 'Use make test instead', got %q", result1)
	}

	// Test second rule - should be blocked because it matches (no action field needed)
	hookInput2 := `{
		"tool_input": {
			"command": "rm -rf temp",
			"description": "Remove directory"
		}
	}`
	result2, err := app.ProcessHook(strings.NewReader(hookInput2))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}
	if result2 != "Dangerous command detected" {
		t.Errorf("Expected 'Dangerous command detected', got %q", result2)
	}

	// Test non-matching command - should be allowed
	hookInput3 := `{
		"tool_input": {
			"command": "make build",
			"description": "Build project"
		}
	}`
	result3, err := app.ProcessHook(strings.NewReader(hookInput3))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}
	if result3 != "" {
		t.Errorf("Expected empty result for allowed command, got %q", result3)
	}
}

func TestCommandWithoutBlockedPrefix(t *testing.T) {
	t.Parallel()

	// Test that TestCommand doesn't add "Command blocked:" prefix
	configContent := `rules:
  - pattern: "go test"
    response: "Use make test instead"
`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	result, err := app.TestCommand("go test ./...")
	if err != nil {
		t.Fatalf("TestCommand failed: %v", err)
	}

	// Should not have "Command blocked:" prefix
	if strings.HasPrefix(result, "Command blocked:") {
		t.Errorf("Expected no 'Command blocked:' prefix, but got: %q", result)
	}

	// Should contain the response directly
	if !strings.Contains(result, "Use make test instead") {
		t.Errorf("Expected response to contain 'Use make test instead', got: %q", result)
	}
}

func TestInstallHandlesMissingBumpersBinary(t *testing.T) { //nolint:paralleltest // Test resets shared logger state
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Reset logger state for clean test
	logger.Reset()

	// Use the new constructor with working directory instead of os.Chdir()
	app := NewAppWithWorkDir(configPath, tempDir)

	// Create bin directory but without bumpers binary
	binDir := filepath.Join(tempDir, "bin")
	err := os.MkdirAll(binDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}
	// Intentionally DON'T create the bumpers binary to test error handling

	err = app.Initialize()

	// Now it should fail with a proper error message about missing binary
	if err == nil {
		t.Error("Expected Initialize to fail when bumpers binary is missing")
		return
	}

	// Should contain helpful error message
	expectedPath := filepath.Join(tempDir, "bin", "bumpers")
	if !strings.Contains(err.Error(), expectedPath) {
		t.Errorf("Error should mention the expected path %s, got: %v", expectedPath, err)
	}

	if !strings.Contains(err.Error(), "make build") {
		t.Errorf("Error should suggest running 'make build', got: %v", err)
	}
}

func TestHookInstallationDoesNotIncludeTimeout(t *testing.T) { //nolint:paralleltest // Test resets shared logger state
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Reset logger state for clean test
	logger.Reset()

	// Create bin directory and dummy bumpers binary
	binDir := filepath.Join(tempDir, "bin")
	err := os.MkdirAll(binDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	bumpersPath := filepath.Join(binDir, "bumpers")
	err = os.WriteFile(bumpersPath, []byte("#!/bin/bash\necho test"), 0o750) //nolint:gosec // exec perms
	if err != nil {
		t.Fatal(err)
	}

	// Use the new constructor with working directory instead of os.Chdir()
	app := NewAppWithWorkDir(configPath, tempDir)

	err = app.Initialize()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that hooks were installed without timeout field
	claudeDir := filepath.Join(tempDir, ".claude")
	localSettingsPath := filepath.Join(claudeDir, "settings.local.json")
	content, err := os.ReadFile(localSettingsPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	// Should NOT contain timeout field in the JSON
	if strings.Contains(contentStr, `"timeout"`) {
		t.Error("Expected hook installation to NOT include timeout field in JSON")
	}

	// Should still contain the essential hook fields
	if !strings.Contains(contentStr, `"matcher": "Bash"`) {
		t.Error("Expected Bash hook to be added to settings.local.json")
	}

	if !strings.Contains(contentStr, "bumpers") {
		t.Error("Expected hook command to contain bumpers")
	}
}

func TestInitializeInitializesLogger(t *testing.T) { //nolint:paralleltest // Test resets shared logger state
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Reset logger state for clean test
	logger.Reset()

	// Create bin directory and dummy bumpers binary
	binDir := filepath.Join(tempDir, "bin")
	err := os.MkdirAll(binDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	bumpersPath := filepath.Join(binDir, "bumpers")
	err = os.WriteFile(bumpersPath, []byte("#!/bin/bash\necho test"), 0o750) //nolint:gosec // exec perms
	if err != nil {
		t.Fatal(err)
	}

	// Use the new constructor with working directory
	app := NewAppWithWorkDir(configPath, tempDir)

	err = app.Initialize()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify logger was initialized by attempting to log something
	logger.Info("test log message", "initialized", "true")

	// Give it a moment for the write to complete
	time.Sleep(10 * time.Millisecond)

	// Check if log file was created in the correct directory
	logFile := filepath.Join(tempDir, ".claude", "bumpers", "bumpers.log")
	content, err := os.ReadFile(logFile) //nolint:gosec // controlled log file path in test
	if err != nil {
		t.Fatalf("Expected log file to be created at %s: %v", logFile, err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "test log message") {
		t.Error("Expected log file to contain 'test log message'")
	}

	if !strings.Contains(contentStr, "\"initialized\":\"true\"") {
		t.Error("Expected log file to contain structured data")
	}
}
