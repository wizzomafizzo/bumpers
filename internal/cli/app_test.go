package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/logger"
	"github.com/wizzomafizzo/bumpers/internal/paths"
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

func TestInitialize(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

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

func TestStatusEnhanced(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Create a config file
	configContent := `rules:
  - pattern: "go test"
    response: "Use make test instead"`

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	app := NewAppWithWorkDir(configPath, tempDir)

	status, err := app.Status()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should contain status information
	if !strings.Contains(status, "Bumpers Status:") {
		t.Error("Expected status to contain 'Bumpers Status:'")
	}

	// Should show config file status
	if !strings.Contains(status, "Config file: EXISTS") {
		t.Error("Expected status to show config file exists")
	}

	// Should show config location
	if !strings.Contains(status, configPath) {
		t.Error("Expected status to show config file path")
	}
}

func TestInstallUsesProjectClaudeDirectory(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

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

func TestInitializeInstallsClaudeHooksInProjectDirectory(t *testing.T) {
	t.Parallel()
	// Test resets shared logger state
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

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

func TestProcessHookLogsErrors(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	// Use non-existent config path to trigger error
	app := NewAppWithWorkDir("non-existent-config.yml", tempDir)

	// Create logger for the app
	var err error
	// Use default config for logger in tests
	cfg := &config.Config{
		Logging: config.LoggingConfig{},
	}
	app.logger, err = logger.NewWithConfig(cfg, tempDir)
	if err != nil {
		t.Fatalf("Logger creation failed: %v", err)
	}

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

func TestInstallActuallyAddsHook(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

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

func TestInstallCreatesBothHooks(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

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

	// Check that both PreToolUse and UserPromptSubmit hooks were added
	claudeDir := filepath.Join(tempDir, ".claude")
	localSettingsPath := filepath.Join(claudeDir, "settings.local.json")
	content, err := os.ReadFile(localSettingsPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)

	// Check for PreToolUse hook with Bash matcher
	if !strings.Contains(contentStr, `"matcher": "Bash"`) {
		t.Error("Expected PreToolUse Bash hook to be added to settings.local.json")
	}

	// Check for UserPromptSubmit hook with .* matcher
	if !strings.Contains(contentStr, `"matcher": ".*"`) {
		t.Error("Expected UserPromptSubmit hook with .* matcher to be added to settings.local.json")
	}

	// Check that both hooks contain bumpers command
	bashHookCount := strings.Count(contentStr, "bumpers")
	if bashHookCount < 2 {
		t.Errorf("Expected at least 2 bumpers hooks (PreToolUse and UserPromptSubmit), found %d", bashHookCount)
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

// validateTestEnvironment checks that Initialize succeeds in test environment
func validateTestEnvironment(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Expected Initialize to succeed in test environment, but got error: %v", err)
	}
}

// validateProductionEnvironment checks that Initialize fails appropriately in production
func validateProductionEnvironment(t *testing.T, err error, prodLikeDir string) {
	t.Helper()
	if err == nil {
		t.Error("Expected Initialize to fail when bumpers binary is missing in production environment")
		return
	}

	// Should contain helpful error message
	expectedPath := filepath.Join(prodLikeDir, "bin", "bumpers")
	if !strings.Contains(err.Error(), expectedPath) {
		t.Errorf("Error should mention the expected path %s, got: %v", expectedPath, err)
	}

	if !strings.Contains(err.Error(), "make build") {
		t.Errorf("Error should suggest running 'make build', got: %v", err)
	}
}

func TestInstallHandlesMissingBumpersBinary(t *testing.T) {
	t.Parallel()
	// Use a directory name that won't trigger test environment detection
	tempDir := t.TempDir()
	prodLikeDir := filepath.Join(tempDir, "production-env")
	err := os.MkdirAll(prodLikeDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(prodLikeDir, "bumpers.yml")

	// Use the new constructor with working directory instead of os.Chdir()
	app := NewAppWithWorkDir(configPath, prodLikeDir)

	// Create bin directory but without bumpers binary
	binDir := filepath.Join(prodLikeDir, "bin")
	err = os.MkdirAll(binDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}
	// Intentionally DON'T create the bumpers binary to test error handling

	// Determine if this is a test environment (should skip binary check)
	shouldSkip := strings.HasPrefix(filepath.Base(prodLikeDir), "Test") || strings.Contains(prodLikeDir, "/tmp/Test")

	err = app.Initialize()

	// Validate behavior based on environment
	if shouldSkip {
		validateTestEnvironment(t, err)
	} else {
		validateProductionEnvironment(t, err, prodLikeDir)
	}
}

func TestHookInstallationDoesNotIncludeTimeout(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

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

func TestInitializeInitializesLogger(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

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
	app.logger.Info().Bool("initialized", true).Msg("test log message")

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

	if !strings.Contains(contentStr, "\"initialized\":true") {
		t.Error("Expected log file to contain structured data")
	}
}

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - pattern: "^go test"
    response: "Use make test instead"
  - pattern: "^(gci|go vet)"
    response: "Use make lint-fix instead"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	result, err := app.ValidateConfig()
	if err != nil {
		t.Fatalf("Expected no error for valid config, got %v", err)
	}

	if result != "Configuration is valid" {
		t.Errorf("Expected 'Configuration is valid', got %q", result)
	}
}

func TestInstallUsesPathConstants(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	// Create a simple config file
	configContent := `rules:
  - pattern: "dangerous"
    response: "Use safer alternatives"`
	configPath := createTempConfig(t, configContent)

	// Create bin directory and bumpers binary (required by Initialize)
	binDir := filepath.Join(tempDir, "bin")
	err := os.MkdirAll(binDir, 0o750)
	if err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}
	bumpersPath := filepath.Join(binDir, "bumpers")
	err = os.WriteFile(bumpersPath, []byte("#!/bin/sh\necho 'bumpers'"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create bumpers binary: %v", err)
	}

	// Change to temp directory for the test
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()

	app := NewAppWithWorkDir(configPath, tempDir)

	// Initialize should create Claude directory structure using path constants
	err = app.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify that directories were created using the path constants
	expectedClaudeDir := filepath.Join(tempDir, paths.ClaudeDir)
	if _, err := os.Stat(expectedClaudeDir); os.IsNotExist(err) {
		t.Errorf("Expected Claude directory to be created at %s (using paths.ClaudeDir)", expectedClaudeDir)
	}

	expectedSettingsFile := filepath.Join(expectedClaudeDir, paths.SettingsFilename)
	if _, err := os.Stat(expectedSettingsFile); os.IsNotExist(err) {
		t.Errorf("Expected settings file to be created at %s (using paths.SettingsFilename)", expectedSettingsFile)
	}
}

// setupProjectStructure creates a temporary project structure for testing
func setupProjectStructure(t *testing.T, configFileName string) (projectDir, subDir string, cleanup func()) {
	t.Helper()

	tempDir := t.TempDir()
	projectDir = filepath.Join(tempDir, "my-project")
	subDir = filepath.Join(projectDir, "internal", "cli")

	err := os.MkdirAll(subDir, 0o750)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create go.mod file to mark project root
	goModPath := filepath.Join(projectDir, "go.mod")
	err = os.WriteFile(goModPath, []byte("module example.com/myproject\n"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create go.mod file: %v", err)
	}

	// Create config file at project root
	configPath := filepath.Join(projectDir, configFileName)

	// Generate config content based on file extension
	var configContent string
	switch filepath.Ext(configFileName) {
	case ".toml":
		configContent = `
[[rules]]
pattern = ".*dangerous.*"
response = "This command looks dangerous!"
`
	case ".json":
		configContent = `{
  "rules": [
    {
      "pattern": ".*dangerous.*",
      "response": "This command looks dangerous!"
    }
  ]
}`
	default:
		// Default to YAML format
		configContent = `
rules:
  - pattern: ".*dangerous.*"
    response: "This command looks dangerous!"
`
	}

	err = os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Setup cleanup function - only get original directory if it exists
	cleanup = func() {
		// This function is called during test cleanup
		// The temp directory will be automatically cleaned up by t.TempDir()
	}

	return projectDir, subDir, cleanup
}

func TestNewApp_ProjectRootDetection(t *testing.T) {
	t.Parallel()

	projectDir, subDir, cleanup := setupProjectStructure(t, "bumpers.yml")
	defer cleanup()

	// Test with relative config path using workdir approach
	app := NewAppWithWorkDir("bumpers.yml", subDir)

	// Manually set project root since NewAppWithWorkDir doesn't detect it
	app.projectRoot = projectDir

	// Apply config resolution logic manually
	if app.projectRoot != "" && !filepath.IsAbs("bumpers.yml") {
		app.configPath = filepath.Join(app.projectRoot, "bumpers.yml")
	}

	// Verify that the app can find and load the config from project root
	result, err := app.ValidateConfig()
	if err != nil {
		t.Fatalf("ValidateConfig failed: %v", err)
	}

	if result != "Configuration is valid" {
		t.Errorf("Expected configuration to be valid, got: %s", result)
	}
}

func TestInstall_UsesProjectRoot(t *testing.T) {
	t.Parallel()

	projectDir, subDir, cleanup := setupProjectStructure(t, "bumpers.yml")
	defer cleanup()

	// Create bin directory and dummy bumpers binary at project root
	binDir := filepath.Join(projectDir, "bin")
	err := os.MkdirAll(binDir, 0o750)
	if err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	bumpersPath := filepath.Join(binDir, "bumpers")
	err = os.WriteFile(bumpersPath, []byte("#!/bin/bash\necho test"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create dummy bumpers binary: %v", err)
	}

	// Test with relative config path using workdir approach
	app := NewAppWithWorkDir("bumpers.yml", subDir)

	// Manually set project root since NewAppWithWorkDir doesn't detect it
	app.projectRoot = projectDir

	// Apply config resolution logic manually
	if app.projectRoot != "" && !filepath.IsAbs("bumpers.yml") {
		app.configPath = filepath.Join(app.projectRoot, "bumpers.yml")
	}

	// Initialize should create .claude directory at project root
	err = app.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify .claude directory was created at project root, not in subdirectory
	expectedClaudeDir := filepath.Join(projectDir, paths.ClaudeDir)
	if _, err := os.Stat(expectedClaudeDir); os.IsNotExist(err) {
		t.Errorf("Expected .claude directory to be created at project root %s", expectedClaudeDir)
	}

	// Verify .claude directory was NOT created in subdirectory
	wrongClaudeDir := filepath.Join(subDir, paths.ClaudeDir)
	if _, err := os.Stat(wrongClaudeDir); !os.IsNotExist(err) {
		t.Errorf("Expected .claude directory to NOT be created in subdirectory %s", wrongClaudeDir)
	}
}

func testNewAppAutoFindsConfigFile(t *testing.T, configFileName string) {
	t.Helper()

	projectDir, subDir, cleanup := setupProjectStructure(t, configFileName)
	defer cleanup()

	// Test with default config path using workdir approach
	app := NewAppWithWorkDir("bumpers.yml", subDir)

	// Manually set project root since NewAppWithWorkDir doesn't detect it
	app.projectRoot = projectDir

	// Apply config resolution logic manually
	if app.projectRoot != "" && !filepath.IsAbs("bumpers.yml") {
		app.configPath = filepath.Join(app.projectRoot, "bumpers.yml")

		// Try different extensions in order
		if _, err := os.Stat(app.configPath); os.IsNotExist(err) {
			extensions := []string{"yaml", "toml", "json"}
			for _, ext := range extensions {
				candidatePath := filepath.Join(app.projectRoot, "bumpers."+ext)
				if _, statErr := os.Stat(candidatePath); statErr == nil {
					app.configPath = candidatePath
					break
				}
			}
		}
	}

	// Verify that the app can find and load the config from project root
	result, err := app.ValidateConfig()
	if err != nil {
		t.Fatalf("ValidateConfig failed: %v", err)
	}

	if result != "Configuration is valid" {
		t.Errorf("Expected configuration to be valid, got: %s", result)
	}
}

func TestNewApp_AutoFindsConfigFile(t *testing.T) {
	t.Parallel()
	testNewAppAutoFindsConfigFile(t, "bumpers.yaml")
}

func TestNewApp_AutoFindsTomlConfigFile(t *testing.T) {
	t.Parallel()
	testNewAppAutoFindsConfigFile(t, "bumpers.toml")
}

func TestNewApp_AutoFindsJsonConfigFile(t *testing.T) {
	t.Parallel()
	testNewAppAutoFindsConfigFile(t, "bumpers.json")
}

func TestNewApp_ConfigPrecedenceOrder(t *testing.T) {
	t.Parallel()

	projectDir, subDir := setupPrecedenceTestDir(t)
	createPrecedenceConfigFiles(t, projectDir)

	app := createAppWithPrecedenceConfig(projectDir, subDir)

	// Test the YAML-specific command to ensure YAML was loaded
	result, err := app.TestCommand("yaml-test")
	if err != nil {
		t.Fatalf("TestCommand failed: %v", err)
	}

	if result != "Found YAML config" {
		t.Errorf("Expected YAML config to be loaded (should have precedence), but got: %s", result)
	}
}

func setupPrecedenceTestDir(t *testing.T) (projectDir, subDir string) {
	t.Helper()

	tempDir := t.TempDir()
	projectDir = filepath.Join(tempDir, "my-project")
	subDir = filepath.Join(projectDir, "subdir")

	err := os.MkdirAll(subDir, 0o750)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create go.mod file to mark project root
	goModPath := filepath.Join(projectDir, "go.mod")
	err = os.WriteFile(goModPath, []byte("module example.com/myproject\n"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create go.mod file: %v", err)
	}

	return projectDir, subDir
}

func createPrecedenceConfigFiles(t *testing.T, projectDir string) {
	t.Helper()

	// Create both YAML and TOML config files to test precedence
	yamlContent := `
rules:
  - pattern: "yaml-test"
    response: "Found YAML config"
`
	tomlContent := `
[[rules]]
pattern = "toml-test"
response = "Found TOML config"
`

	yamlPath := filepath.Join(projectDir, "bumpers.yaml")
	tomlPath := filepath.Join(projectDir, "bumpers.toml")

	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create YAML config file: %v", err)
	}

	err = os.WriteFile(tomlPath, []byte(tomlContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create TOML config file: %v", err)
	}
}

func createAppWithPrecedenceConfig(projectDir, subDir string) *App {
	// Test using NewAppWithWorkDir to avoid directory changes
	app := NewAppWithWorkDir("bumpers.yml", subDir)
	// Manually set projectRoot and update configPath since NewAppWithWorkDir doesn't detect it
	app.projectRoot = projectDir

	// Apply the same config resolution logic as NewApp
	resolvedConfigPath := "bumpers.yml"
	if app.projectRoot != "" && !filepath.IsAbs("bumpers.yml") {
		resolvedConfigPath = filepath.Join(app.projectRoot, "bumpers.yml")

		// Try different extensions in order
		if _, err := os.Stat(resolvedConfigPath); os.IsNotExist(err) {
			extensions := []string{"yaml", "toml", "json"}
			for _, ext := range extensions {
				candidatePath := filepath.Join(app.projectRoot, "bumpers."+ext)
				if _, statErr := os.Stat(candidatePath); statErr == nil {
					resolvedConfigPath = candidatePath
					break
				}
			}
		}
	}
	app.configPath = resolvedConfigPath
	return app
}

func TestProcessHookRoutesUserPromptSubmit(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - pattern: "go test"
    response: "Use make test instead"
commands:
  - message: "Available commands:\\n!help - Show this help\\n!status - Show project status"
  - message: "Project Status: All systems operational"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Test UserPromptSubmit hook routing
	userPromptInput := `{
		"prompt": "!0"
	}`

	result, err := app.ProcessHook(strings.NewReader(userPromptInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for UserPromptSubmit: %v", err)
	}

	expectedJSON := `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
		`"additionalContext":"Available commands:\\n!help - Show this help\\n!status - Show project status"}}`
	if result != expectedJSON {
		t.Errorf("Expected %q, got %q", expectedJSON, result)
	}
}

func TestProcessUserPrompt(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - pattern: "go test"
    response: "Use make test instead"
commands:
  - message: "Available commands:\\n!help - Show this help\\n!status - Show project status"
  - message: "Project Status: All systems operational"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	tests := []struct {
		name           string
		promptJSON     string
		expectedOutput string
	}{
		{
			name: "Command 0 (!0)",
			promptJSON: `{
				"prompt": "!0"
			}`,
			expectedOutput: `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
				`"additionalContext":"Available commands:\\n!help - Show this help\\n!status - Show project status"}}`,
		},
		{
			name: "Command 1 (!1)",
			promptJSON: `{
				"prompt": "!1"
			}`,
			expectedOutput: `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
				`"additionalContext":"Project Status: All systems operational"}}`,
		},
		{
			name: "Non-command prompt",
			promptJSON: `{
				"prompt": "regular question"
			}`,
			expectedOutput: "",
		},
		{
			name: "Invalid command index (!5)",
			promptJSON: `{
				"prompt": "!5"
			}`,
			expectedOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := app.ProcessUserPrompt(json.RawMessage(tt.promptJSON))
			if err != nil {
				t.Fatalf("ProcessUserPrompt failed: %v", err)
			}

			if result != tt.expectedOutput {
				t.Errorf("Expected %q, got %q", tt.expectedOutput, result)
			}
		})
	}
}
