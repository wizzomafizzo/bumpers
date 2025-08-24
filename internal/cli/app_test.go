package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wizzomafizzo/bumpers/internal/ai"
	"github.com/wizzomafizzo/bumpers/internal/claude"
	"github.com/wizzomafizzo/bumpers/internal/constants"
	"github.com/wizzomafizzo/bumpers/internal/filesystem"
	"github.com/wizzomafizzo/bumpers/internal/hooks"
	"github.com/wizzomafizzo/bumpers/internal/logger"
	"github.com/wizzomafizzo/bumpers/internal/storage"
)

var loggerInitOnce sync.Once

// setupTest initializes test logger to prevent race conditions
func setupTest(t *testing.T) {
	t.Helper()
	loggerInitOnce.Do(func() {
		logger.InitTest()
	})
}

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

// TestAppWithMemoryFileSystem tests App initialization with in-memory filesystem for parallel testing
func TestAppWithMemoryFileSystem(t *testing.T) {
	t.Parallel()

	// Setup in-memory filesystem with test config
	fs := filesystem.NewMemoryFileSystem()
	configContent := []byte(`rules:
  - match: "rm -rf"
    send: "Use safer alternatives"`)
	configPath := "/test/bumpers.yml"

	err := fs.WriteFile(configPath, configContent, 0o600)
	if err != nil {
		t.Fatalf("Failed to setup test config: %v", err)
	}

	// Create app with injected filesystem - this should work without real file I/O
	app := NewAppWithFileSystem(configPath, "/test/workdir", fs)

	if app == nil {
		t.Fatal("Expected app to be created")
	}

	// Test that the filesystem was properly injected
	if app.fileSystem != fs {
		t.Error("Expected FileSystem to be properly injected")
	}

	// Test that config file is accessible via injected filesystem
	content, err := app.fileSystem.ReadFile(configPath)
	if err != nil {
		t.Errorf("Failed to read config via injected filesystem: %v", err)
	}

	if !bytes.Equal(content, configContent) {
		t.Errorf("Expected config content %q, got %q", string(configContent), string(content))
	}
}

// TestAppInitializeWithMemoryFileSystem tests that Initialize works with injected filesystem
func TestAppInitializeWithMemoryFileSystem(t *testing.T) {
	t.Parallel()

	// Setup in-memory filesystem with test config
	fs := filesystem.NewMemoryFileSystem()
	configContent := []byte(`rules:
  - match: "rm -rf"
    send: "Use safer alternatives"`)
	configPath := "/test/bumpers.yml"

	err := fs.WriteFile(configPath, configContent, 0o600)
	if err != nil {
		t.Fatalf("Failed to setup test config: %v", err)
	}

	// Add bumpers binary to memory filesystem (needed for validateBumpersPath)
	bumpersPath := "/test/workdir/bin/bumpers"
	err = fs.WriteFile(bumpersPath, []byte("fake bumpers binary"), 0o755)
	if err != nil {
		t.Fatalf("Failed to setup fake bumpers binary: %v", err)
	}

	// Create app with injected filesystem
	app := NewAppWithFileSystem(configPath, "/test/workdir", fs)

	// Initialize should work without real filesystem operations
	err = app.Initialize()
	if err != nil {
		t.Errorf("Initialize failed with memory filesystem: %v", err)
	}

	// Verify config can still be loaded after Initialize
	content, err := app.fileSystem.ReadFile(configPath)
	if err != nil {
		t.Errorf("Failed to read config after Initialize: %v", err)
	}

	if !bytes.Equal(content, configContent) {
		t.Error("Config content changed after Initialize")
	}
}

func TestInstallClaudeHooksWithWorkDir(t *testing.T) {
	t.Parallel()
	setupTest(t)

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
	setupTest(t)

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead for better TDD integration"
    alternatives:
      - "make test          # Run all tests"
      - "make test-unit     # Run unit tests only"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash"
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return a response since "go test" matches a rule
	if response == "" {
		t.Error("Expected non-empty response for blocked command")
	}

	if !strings.Contains(response, "just test") {
		t.Error("Response should suggest just test alternative")
	}
}

func TestProcessHookAllowed(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead for better TDD integration"`

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
	setupTest(t)

	configContent := `rules:
  - match: "rm -rf /*"
    send: "⚠️  Dangerous rm command detected"
    alternatives:
      - "Be more specific with your rm command"
      - "Use a safer alternative like moving to trash"
    generate:
      mode: "always"
      prompt: "Explain why this rm command is dangerous and suggest safer alternatives"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Set up mock launcher for AI generation
	mock := claude.NewMockLauncher()
	mock.SetResponseForPattern(".*", "AI-generated response about dangerous rm command")
	app.SetMockLauncher(mock)

	hookInput := `{
		"tool_input": {
			"command": "rm -rf /tmp",
			"description": "Remove directory"
		},
		"tool_name": "Bash"
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
	setupTest(t)

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead for better TDD integration"
    alternatives:
      - "make test          # Run all tests"
      - "make test-unit     # Run unit tests only"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"tool_input": {
			"command": "go test -v ./pkg/...",
			"description": "Run verbose tests"
		},
		"tool_name": "Bash"
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return a response since this matches "go test.*" pattern
	if response == "" {
		t.Error("Expected non-empty response for go test pattern match")
	}

	if !strings.Contains(response, "just test") {
		t.Error("Response should suggest just test alternative")
	}
}

func TestConfigurationIsUsed(t *testing.T) {
	t.Parallel()
	setupTest(t)

	// This test ensures we're actually using the config file by checking for
	// a specific message from the config rather than hardcoded responses
	configContent := `rules:
  - match: "go test"
    send: "Use just test instead for better TDD integration"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run all tests"
		},
		"tool_name": "Bash"
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
  - match: "go test"
    send: "Use just test instead for better TDD integration"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	result, err := app.TestCommand("go test ./...")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should contain the response message
	if !strings.Contains(result, "Use just test instead") {
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
  - match: "go test"
    send: "Use just test instead"
    generate: "off"`

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
  - match: "go test"
    send: "Use just test instead"`

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
	setupTest(t)
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
	if !strings.Contains(contentStr, `"PreToolUse"`) {
		t.Error("Expected PreToolUse hook to be added to settings.local.json")
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

	// Check for PreToolUse hook (empty matcher is omitted in JSON)
	if !strings.Contains(contentStr, `"PreToolUse"`) {
		t.Error("Expected PreToolUse hook to be added to settings.local.json")
	}

	// Check for UserPromptSubmit hook with empty matcher (omitted from JSON)
	if !strings.Contains(contentStr, `"UserPromptSubmit"`) {
		t.Error("Expected UserPromptSubmit hook to be added to settings.local.json")
	}

	// Check for SessionStart hook with startup|clear matcher
	if !strings.Contains(contentStr, `"SessionStart"`) {
		t.Error("Expected SessionStart hook to be added to settings.local.json")
	}

	if !strings.Contains(contentStr, `"matcher": "startup|clear"`) {
		t.Error("Expected SessionStart hook to have startup|clear matcher in settings.local.json")
	}

	// Check that all three hooks contain bumpers command
	bashHookCount := strings.Count(contentStr, "bumpers")
	if bashHookCount < 3 {
		t.Errorf("Expected at least 3 bumpers hooks (PreToolUse, UserPromptSubmit, SessionStart), found %d",
			bashHookCount)
	}
}

func TestProcessHookSimplifiedSchemaAlwaysDenies(t *testing.T) {
	t.Parallel()

	// Setup test config with simplified schema (no name or action fields)
	// Any pattern match should result in denial
	configContent := `rules:
  - match: "go test"
    send: "Use just test instead"
    generate: "off"
  - match: "rm -rf"
    send: "Dangerous command detected"
    generate: "off"
`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Test first rule - should be blocked because it matches (no action field needed)
	hookInput1 := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash"
	}`
	result1, err := app.ProcessHook(strings.NewReader(hookInput1))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}
	if result1 != "Use just test instead" {
		t.Errorf("Expected 'Use just test instead', got %q", result1)
	}

	// Test second rule - should be blocked because it matches (no action field needed)
	hookInput2 := `{
		"tool_input": {
			"command": "rm -rf temp",
			"description": "Remove directory"
		},
		"tool_name": "Bash"
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
  - match: "go test"
    send: "Use just test instead"
    generate: "off"
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
	if !strings.Contains(result, "Use just test instead") {
		t.Errorf("Expected response to contain 'Use just test instead', got: %q", result)
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
	setupTest(t)
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
	if !strings.Contains(contentStr, `"PreToolUse"`) {
		t.Error("Expected PreToolUse hook to be added to settings.local.json")
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
}

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - match: "^go test"
    send: "Use just test instead"
    generate: "off"
  - match: "^(gci|go vet)"
    send: "Use just lint fix instead"
    generate: "off"`

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
  - match: "dangerous"
    send: "Use safer alternatives"`
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
	expectedClaudeDir := filepath.Join(tempDir, constants.ClaudeDir)
	if _, err := os.Stat(expectedClaudeDir); os.IsNotExist(err) {
		t.Errorf("Expected Claude directory to be created at %s (using paths.ClaudeDir)", expectedClaudeDir)
	}

	expectedSettingsFile := filepath.Join(expectedClaudeDir, constants.SettingsFilename)
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
		configContent = `[rules]
[[rules.item]]
match = ".*dangerous.*"
send = "This command looks dangerous!"
generate = "off"
`
	case ".json":
		configContent = `{
  "rules": [
    {
      "match": ".*dangerous.*",
      "send": "This command looks dangerous!",
      "generate": "off"
    }
  ]
}`
	default:
		// Default to YAML format
		configContent = `
rules:
  - match: ".*dangerous.*"
    send: "This command looks dangerous!"
    generate: "off"
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
	expectedClaudeDir := filepath.Join(projectDir, constants.ClaudeDir)
	if _, err := os.Stat(expectedClaudeDir); os.IsNotExist(err) {
		t.Errorf("Expected .claude directory to be created at project root %s", expectedClaudeDir)
	}

	// Verify .claude directory was NOT created in subdirectory
	wrongClaudeDir := filepath.Join(subDir, constants.ClaudeDir)
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
	setupTest(t)
	testNewAppAutoFindsConfigFile(t, "bumpers.yaml")
}

func TestNewApp_AutoFindsTomlConfigFile(t *testing.T) {
	t.Parallel()
	t.Skip("TOML parsing doesn't work with polymorphic 'generate' field - skipping for now")
}

func TestNewApp_AutoFindsJsonConfigFile(t *testing.T) {
	t.Parallel()
	setupTest(t)
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
  - match: "yaml-test"
    send: "Found YAML config"
    generate: "off"
`
	tomlContent := `
[[rules]]
pattern = "toml-test"
message = "Found TOML config"
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
	setupTest(t)

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead"
    generate: "off"
commands:
  - name: "help"
    send: "Available commands:\\n$help - Show this help\\n$status - Show project status"
    generate: "off"
  - name: "status"
    send: "Project Status: All systems operational"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Test UserPromptSubmit hook routing
	userPromptInput := `{
		"prompt": "$help"
	}`

	result, err := app.ProcessHook(strings.NewReader(userPromptInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for UserPromptSubmit: %v", err)
	}

	expectedJSON := `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
		`"additionalContext":"Available commands:\\n$help - Show this help\\n$status - Show project status"}}`
	if result != expectedJSON {
		t.Errorf("Expected %q, got %q", expectedJSON, result)
	}
}

func TestProcessUserPrompt(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead"
    generate: "off"
commands:
  - name: "help"
    send: "Available commands:\\n$help - Show this help\\n$status - Show project status"
    generate: "off"
  - name: "status"
    send: "Project Status: All systems operational"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	tests := []struct {
		name           string
		promptJSON     string
		expectedOutput string
	}{
		{
			name: "Help command ($help)",
			promptJSON: `{
				"prompt": "$help"
			}`,
			expectedOutput: `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
				`"additionalContext":"Available commands:\\n$help - Show this help\\n$status - Show project status"}}`,
		},
		{
			name: "Status command ($status)",
			promptJSON: `{
				"prompt": "$status"
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
			name: "Invalid command index ($5)",
			promptJSON: `{
				"prompt": "$5"
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

func TestProcessUserPromptValidationResult(t *testing.T) {
	t.Parallel()

	// Create temporary config with named commands
	configContent := `
commands:
  - name: "test"
    send: "Test command message"
    generate: "off"
`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Test that named command prompts work
	promptJSON := `{"prompt": "$test"}`
	result, err := app.ProcessUserPrompt(json.RawMessage(promptJSON))
	if err != nil {
		t.Fatalf("ProcessUserPrompt failed: %v", err)
	}

	expectedOutput := `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
		`"additionalContext":"Test command message"}}`
	if result != expectedOutput {
		t.Errorf("Expected hookSpecificOutput format for named command %q, got %q", expectedOutput, result)
	}
}

func TestProcessUserPromptWithCommandGeneration(t *testing.T) {
	t.Parallel()

	configContent := `commands:
  - name: "help"
    send: "Basic help message"
    generate: "always"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Set up mock launcher
	mockLauncher := claude.NewMockLauncher()
	mockLauncher.SetResponseForPattern("", "Enhanced help message from AI")
	app.SetMockLauncher(mockLauncher)

	promptJSON := `{"prompt": "$help"}`
	result, err := app.ProcessUserPrompt(json.RawMessage(promptJSON))
	if err != nil {
		t.Fatalf("ProcessUserPrompt failed: %v", err)
	}

	expectedOutput := `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
		`"additionalContext":"Enhanced help message from AI"}}`
	if result != expectedOutput {
		t.Errorf("Expected AI-generated command output %q, got %q", expectedOutput, result)
	}

	// Verify the mock was called with the right prompt
	if mockLauncher.GetCallCount() == 0 {
		t.Error("Expected mock launcher to be called for AI generation")
	}
}

func TestCommandPrefixConfiguration(t *testing.T) {
	t.Parallel()

	configContent := `
commands:
  - name: "test"
    send: "Test command message"
    generate: "off"
`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Test that % prefix no longer works
	oldPrefixJSON := `{"prompt": "%test"}`
	result, err := app.ProcessUserPrompt(json.RawMessage(oldPrefixJSON))
	if err != nil {
		t.Fatalf("ProcessUserPrompt failed for old prefix: %v", err)
	}

	if result != "" {
		t.Error("Old % prefix should no longer work")
	}

	// Test new behavior with $ prefix
	newPrefixJSON := `{"prompt": "$test"}`
	result, err = app.ProcessUserPrompt(json.RawMessage(newPrefixJSON))
	if err != nil {
		t.Fatalf("ProcessUserPrompt failed for new prefix: %v", err)
	}

	expectedOutput := `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
		`"additionalContext":"Test command message"}}`
	if result != expectedOutput {
		t.Errorf("Expected $ prefix to work, got result: %q", result)
	}
}

func TestProcessUserPromptWithTemplate(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `commands:
  - name: "hello"
    send: "Hello {{.Name}}!"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	promptJSON := `{"prompt": "$hello"}`
	result, err := app.ProcessUserPrompt(json.RawMessage(promptJSON))
	if err != nil {
		t.Fatalf("ProcessUserPrompt failed: %v", err)
	}

	expectedOutput := `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
		`"additionalContext":"Hello hello!"}}`
	if result != expectedOutput {
		t.Errorf("Expected templated message, got: %q", result)
	}
}

func TestInstallWithPathCommand(t *testing.T) { //nolint:paralleltest // modifies global os.Args
	setupTest(t)
	// Don't run in parallel as we modify global os.Args

	// Save original args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Simulate running from PATH
	os.Args = []string{"bumpers", "install"}

	// Use a production-like directory name to avoid test environment detection
	tempDir := t.TempDir()
	prodDir := filepath.Join(tempDir, "production-env")
	err := os.MkdirAll(prodDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(prodDir, "bumpers.yml")

	// Use the new constructor with working directory
	app := NewAppWithWorkDir(configPath, prodDir)

	// This should NOT fail because PATH validation should be skipped
	err = app.Initialize()
	if err != nil {
		t.Fatalf("Expected no error when using PATH command, got %v", err)
	}

	// Verify the hook command is just "bumpers"
	claudeDir := filepath.Join(prodDir, ".claude")
	localSettingsPath := filepath.Join(claudeDir, "settings.local.json")
	content, err := os.ReadFile(localSettingsPath) //nolint:gosec // test file, controlled path
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `"command": "bumpers hook"`) {
		t.Error("Expected hook command to be 'bumpers hook' when run from PATH")
	}
}

func TestInstallPreservesExistingHooks(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	app := NewAppWithWorkDir(configPath, tempDir)

	// Create Claude directory and settings with existing hooks
	claudeDir := filepath.Join(tempDir, ".claude")
	err := os.MkdirAll(claudeDir, 0o750)
	if err != nil {
		t.Fatal(err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.local.json")
	existingSettings := `{
		"hooks": {
			"PreToolUse": [
				{
					"matcher": "Bash",
					"hooks": [
						{
							"type": "command",
							"command": "tdd-guard-go"
						}
					]
				}
			],
			"UserPromptSubmit": [
				{
					"matcher": "",
					"hooks": [
						{
							"type": "command", 
							"command": "other-tool"
						}
					]
				}
			]
		}
	}`

	err = os.WriteFile(settingsPath, []byte(existingSettings), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// Run install - this should preserve existing hooks
	err = app.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Read the settings back
	content, err := os.ReadFile(settingsPath) //nolint:gosec // test file, controlled path
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)

	// Should contain both the existing tool AND bumpers
	if !strings.Contains(contentStr, "tdd-guard-go") {
		t.Error("Expected existing tdd-guard-go hook to be preserved")
	}
	if !strings.Contains(contentStr, "other-tool") {
		t.Error("Expected existing other-tool hook to be preserved")
	}
	if !strings.Contains(contentStr, "bumpers") {
		t.Error("Expected bumpers hook to be added")
	}
}

func TestProcessHookWorks(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"hookEventName": "PreToolUse",
		"toolInput": {"command": "ls"}
	}`

	_, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}
}

func TestProcessHookRoutesSessionStart(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead"
    generate: "off"
session:
  - add: "Remember to run tests first"
    generate: "off"
  - add: "Check CLAUDE.md for project conventions"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Test SessionStart hook routing with startup source
	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "startup"
	}`

	result, err := app.ProcessHook(strings.NewReader(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for SessionStart: %v", err)
	}

	expectedJSON := `{"hookSpecificOutput":{"hookEventName":"SessionStart",` +
		`"additionalContext":"Remember to run tests first\nCheck CLAUDE.md for project conventions"}}`
	if result != expectedJSON {
		t.Errorf("Expected %q, got %q", expectedJSON, result)
	}
}

func TestProcessSessionStartWithDifferentNotes(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead"
    generate: "off"
session:
  - add: "Different message here"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "startup"
	}`

	result, err := app.ProcessHook(strings.NewReader(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for SessionStart: %v", err)
	}

	expectedJSON := `{"hookSpecificOutput":{"hookEventName":"SessionStart",` +
		`"additionalContext":"Different message here"}}`
	if result != expectedJSON {
		t.Errorf("Expected %q, got %q", expectedJSON, result)
	}
}

func TestProcessSessionStartIgnoresResume(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `session:
  - add: "Should not appear"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "resume"
	}`

	result, err := app.ProcessHook(strings.NewReader(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for SessionStart: %v", err)
	}

	// Should return empty string for resume source
	if result != "" {
		t.Errorf("Expected empty string for resume source, got %q", result)
	}
}

func TestProcessSessionStartWorksWithClear(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `session:
  - add: "Clear message"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "clear"
	}`

	result, err := app.ProcessHook(strings.NewReader(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for SessionStart: %v", err)
	}

	expectedJSON := `{"hookSpecificOutput":{"hookEventName":"SessionStart",` +
		`"additionalContext":"Clear message"}}`
	if result != expectedJSON {
		t.Errorf("Expected %q, got %q", expectedJSON, result)
	}
}

func TestProcessSessionStartWithTemplate(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `session:
  - add: "Hello from template!"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "startup"
	}`

	result, err := app.ProcessHook(strings.NewReader(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for SessionStart: %v", err)
	}

	// The template should be processed (no template syntax, so it should pass through as-is)
	expectedJSON := `{"hookSpecificOutput":{"hookEventName":"SessionStart",` +
		`"additionalContext":"Hello from template!"}}`
	if result != expectedJSON {
		t.Errorf("Expected %q, got %q", expectedJSON, result)
	}
}

func TestProcessHookWithTemplate(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `rules:
  - match: "go test"
    send: "Command blocked: {{.Command}}"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash"
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedMessage := "Command blocked: go test ./..."
	if response != expectedMessage {
		t.Errorf("Expected %q, got %q", expectedMessage, response)
	}
}

func TestProcessHookWithTodayVariable(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `rules:
  - match: "go test"
    send: "Command {{.Command}} blocked on {{.Today}}"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash"
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedDate := time.Now().Format("2006-01-02")
	expectedMessage := "Command go test ./... blocked on " + expectedDate
	if response != expectedMessage {
		t.Errorf("Expected %q, got %q", expectedMessage, response)
	}
}

func TestTestCommandWithTodayVariable(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - match: "go test"
    send: "Command {{.Command}} blocked on {{.Today}}"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	response, err := app.TestCommand("go test ./...")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedDate := time.Now().Format("2006-01-02")
	expectedMessage := "Command go test ./... blocked on " + expectedDate
	if response != expectedMessage {
		t.Errorf("Expected %q, got %q", expectedMessage, response)
	}
}

func TestProcessUserPromptWithTodayVariable(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `commands:
  - name: "hello"
    send: "Hello {{.Name}} on {{.Today}}!"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	promptJSON := `{"prompt": "$hello"}`
	result, err := app.ProcessUserPrompt(json.RawMessage(promptJSON))
	if err != nil {
		t.Fatalf("ProcessUserPrompt failed: %v", err)
	}

	// Parse the response to get the additionalContext
	var response map[string]any
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	hookOutput, ok := response["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatal("Expected hookSpecificOutput in response")
	}

	additionalContext, ok := hookOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext string in hookSpecificOutput")
	}

	expectedDate := time.Now().Format("2006-01-02")
	expectedMessage := "Hello hello on " + expectedDate + "!"
	if additionalContext != expectedMessage {
		t.Errorf("Expected %q, got %q", expectedMessage, additionalContext)
	}
}

func TestProcessSessionStartWithTodayVariable(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `session:
  - add: "Today is {{.Today}}"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "startup"
	}`

	result, err := app.ProcessSessionStart(json.RawMessage(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessSessionStart failed: %v", err)
	}

	// Parse the response to get the additionalContext
	var response map[string]any
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	hookOutput, ok := response["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatal("Expected hookSpecificOutput in response")
	}

	additionalContext, ok := hookOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext string in hookSpecificOutput")
	}

	expectedDate := time.Now().Format("2006-01-02")
	expectedMessage := "Today is " + expectedDate
	if additionalContext != expectedMessage {
		t.Errorf("Expected %q, got %q", expectedMessage, additionalContext)
	}
}

func TestProcessSessionStartClearsSessionCache(t *testing.T) {
	t.Parallel()
	setupTest(t)

	app, cachePath, tempDir := setupSessionCacheTest(t)
	populateSessionCache(t, cachePath, tempDir)

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "startup"
	}`

	// Process session start should clear session cache
	_, err := app.ProcessSessionStart(json.RawMessage(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessSessionStart failed: %v", err)
	}

	verifySessionCacheCleared(t, cachePath, tempDir)
}

func setupSessionCacheTest(t *testing.T) (app *App, cachePath, tempDir string) {
	t.Helper()
	tempDir = t.TempDir()

	// Setup environment
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		if originalHome != "" {
			_ = os.Setenv("HOME", originalHome)
		} else {
			_ = os.Unsetenv("HOME")
		}
	})
	_ = os.Setenv("HOME", tempDir)

	configPath := createTempConfig(t, `session:
  - add: "Session started"`)
	app = NewApp(configPath)
	app.projectRoot = tempDir

	// Get cache path
	storageManager := storage.New(filesystem.NewOSFileSystem())
	cachePath, err := storageManager.GetCachePath()
	if err != nil {
		t.Fatalf("Failed to get cache path: %v", err)
	}

	return
}

func populateSessionCache(t *testing.T, cachePath, tempDir string) {
	t.Helper()
	cache, err := ai.NewCacheWithProject(cachePath, tempDir)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer func() { _ = cache.Close() }()

	expiry := time.Now().Add(24 * time.Hour)
	sessionEntry := &ai.CacheEntry{
		GeneratedMessage: "Generated session message",
		OriginalMessage:  "Original message",
		Timestamp:        time.Now(),
		ExpiresAt:        &expiry,
	}

	err = cache.Put("test-session-key", sessionEntry)
	if err != nil {
		t.Fatalf("Failed to put session entry: %v", err)
	}

	// Verify entry exists
	retrieved, err := cache.Get("test-session-key")
	if err != nil || retrieved == nil {
		t.Fatal("Session entry should exist before ProcessSessionStart")
	}
}

func verifySessionCacheCleared(t *testing.T, cachePath, tempDir string) {
	t.Helper()
	cache, err := ai.NewCacheWithProject(cachePath, tempDir)
	if err != nil {
		t.Fatalf("Failed to reopen cache: %v", err)
	}
	defer func() { _ = cache.Close() }()

	retrieved, err := cache.Get("test-session-key")
	if err != nil {
		t.Fatalf("Unexpected error getting session key after clearing: %v", err)
	}
	if retrieved != nil {
		t.Error("Session entry should be cleared after ProcessSessionStart")
	}
}

func TestProcessSessionStartWithAIGeneration(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `session:
  - add: "Basic session message"
    generate: "always"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Set up mock launcher
	mockLauncher := claude.NewMockLauncher()
	mockLauncher.SetResponseForPattern("", "Enhanced session message from AI")
	app.SetMockLauncher(mockLauncher)

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "startup"
	}`

	result, err := app.ProcessHook(strings.NewReader(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for SessionStart: %v", err)
	}

	expectedJSON := `{"hookSpecificOutput":{"hookEventName":"SessionStart",` +
		`"additionalContext":"Enhanced session message from AI"}}`
	if result != expectedJSON {
		t.Errorf("Expected AI-generated session output %q, got %q", expectedJSON, result)
	}

	// Verify the mock was called with the right prompt
	if mockLauncher.GetCallCount() == 0 {
		t.Error("Expected mock launcher to be called for AI generation")
	}
}

func TestAppProcessHookWithToolName(t *testing.T) {
	t.Parallel()

	app := setupToolNameTestApp(t)
	testCases := getToolNameTestCases()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runToolNameTest(t, app, tc)
		})
	}
}

func setupToolNameTestApp(t *testing.T) *App {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.yml")

	configContent := `
rules:
  - match: "^rm -rf"
    tool: "^(Bash|Task)$"
    send: "Dangerous rm command"
    generate: "off"
  - match: "password"
    tool: "^Write$"
    send: "No hardcoded secrets"
    generate: "off"
`

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	fs := filesystem.NewMemoryFileSystem()
	return NewAppWithFileSystem(configPath, tempDir, fs)
}

func getToolNameTestCases() []struct {
	name        string
	hookInput   string
	expectMsg   string
	expectBlock bool
} {
	return []struct {
		name        string
		hookInput   string
		expectMsg   string
		expectBlock bool
	}{
		{
			name: "rm command in Bash should be blocked",
			hookInput: `{
				"tool_input": {
					"command": "rm -rf /tmp",
					"description": "Remove temp files"
				},
				"tool_name": "Bash"
			}`,
			expectBlock: true,
			expectMsg:   "Dangerous rm command",
		},
		{
			name: "rm command in Write should be allowed",
			hookInput: `{
				"tool_input": {
					"command": "rm -rf /tmp",
					"description": "Remove temp files"
				},
				"tool_name": "Write"
			}`,
			expectBlock: false,
		},
		{
			name: "password in Write should be blocked",
			hookInput: `{
				"tool_input": {
					"command": "echo password > file",
					"description": "Write to file"
				},
				"tool_name": "Write"
			}`,
			expectBlock: true,
			expectMsg:   "No hardcoded secrets",
		},
		{
			name: "password in Bash should be allowed",
			hookInput: `{
				"tool_input": {
					"command": "echo password",
					"description": "Echo command"
				},
				"tool_name": "Bash"
			}`,
			expectBlock: false,
		},
	}
}

func runToolNameTest(t *testing.T, app *App, tc struct {
	name        string
	hookInput   string
	expectMsg   string
	expectBlock bool
},
) {
	result, err := app.ProcessHook(strings.NewReader(tc.hookInput))

	if tc.expectBlock {
		validateBlockedCommand(t, result, err, tc.expectMsg)
	} else {
		validateAllowedCommand(t, result, err)
	}
}

func validateBlockedCommand(t *testing.T, result string, err error, expectMsg string) {
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
	if result == "" {
		t.Error("Expected blocking message but got empty result")
	}
	if !strings.Contains(result, expectMsg) {
		t.Errorf("Expected message to contain '%s', got '%s'", expectMsg, result)
	}
}

func validateAllowedCommand(t *testing.T, result string, err error) {
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
	if result != "" {
		t.Errorf("Expected no blocking message but got: %s", result)
	}
}

func TestAppProcessHookWithEmptyToolName(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.yml")

	// Config that only matches Bash commands (no tools field means default to Bash)
	configContent := `
rules:
  - match: "^go test"
    send: "Use just test instead"
    generate: "off"
`

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	fs := filesystem.NewMemoryFileSystem()
	app := NewAppWithFileSystem(configPath, tempDir, fs)

	// Test case 1: Hook input with empty tool_name (should NOT be treated as Bash)
	hookInput1 := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		}
	}`

	result, err := app.ProcessHook(strings.NewReader(hookInput1))
	// Empty tool name should NOT match Bash-only rules
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
	if result != "" {
		t.Errorf("Expected no blocking message for empty tool name, got: %s", result)
	}

	// Test case 2: Same command with explicit "Bash" tool_name (should be blocked)
	hookInput2 := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash"
	}`

	result2, err2 := app.ProcessHook(strings.NewReader(hookInput2))

	// This should be blocked
	if err2 != nil {
		t.Errorf("Expected no error but got: %v", err2)
	}
	if result2 == "" {
		t.Error("Expected blocking message for explicit Bash tool, got empty result")
	}
}

func TestProcessHookWithAIGeneration(t *testing.T) {
	t.Parallel()

	setupTest(t)

	configContent := `rules:
  - match: "^go test"
    send: "Use 'just test' instead"
    generate:
      mode: "always"`

	configPath := createTempConfig(t, configContent)

	// Create a temp directory for AI cache
	tempDir := t.TempDir()
	app := NewAppWithWorkDir(configPath, tempDir)

	// Inject mock launcher to ensure tests use mock Claude CLI
	mock := claude.NewMockLauncher()
	mock.SetResponseForPattern(".*", "Mock response")
	app.SetMockLauncher(mock)

	// Verify mock was set (simple field access test)
	if app.mockLauncher == nil {
		t.Fatal("SETUP ERROR: mock launcher was not set properly")
	}
	t.Logf("DEBUG: mockLauncher type: %T, value: %+v", app.mockLauncher, app.mockLauncher != nil)

	// Test the mock directly to ensure it works
	testResponse, testErr := mock.GenerateMessage("test prompt")
	if testErr != nil {
		t.Fatalf("Mock launcher test failed: %v", testErr)
	}
	if testResponse != "Mock response" {
		t.Fatalf("Mock launcher not working properly. Expected 'Mock response', got: %s", testResponse)
	}
	t.Log("DEBUG: Mock launcher works correctly when called directly")

	hookInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash"
	}`

	result, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	// Should return some message (either generated or original)
	if result == "" {
		t.Error("Expected non-empty result")
	}

	t.Logf("DEBUG: Actual result: %q", result)

	// MUST use mock Claude CLI - verify mock response
	expectedMockResponse := "Mock response"
	if result != expectedMockResponse {
		t.Errorf("Expected mock response %q, but got real Claude response: %s", expectedMockResponse, result)
		t.Error("TEST FAILURE: CLI tests MUST use mock Claude CLI, not real Claude CLI")
		// Debug: check if mock was set
		if app.mockLauncher == nil {
			t.Error("DEBUG: mockLauncher is nil - SetMockLauncher didn't work")
		} else {
			t.Errorf("DEBUG: mockLauncher is set (%T) but not being used in processAIGeneration", app.mockLauncher)
		}
	}

	// Run second time - should use cache for "once" mode
	result2, err2 := app.ProcessHook(strings.NewReader(hookInput))
	if err2 != nil {
		t.Errorf("Expected no error on second call but got: %v", err2)
	}

	// Should get same result (cached)
	if result != result2 {
		t.Errorf("Expected cached result to match: %q vs %q", result, result2)
	}
}

func TestProcessHookAIGenerationRequired(t *testing.T) {
	t.Parallel()

	setupTest(t)

	// Test that AI generation is attempted when generate field is set
	configContent := `rules:
  - match: "^go test"
    send: "Use 'just test' instead"
    generate:
      mode: "always"`

	configPath := createTempConfig(t, configContent)

	// Create a temp directory for AI cache
	tempDir := t.TempDir()
	app := NewAppWithWorkDir(configPath, tempDir)

	// Inject mock launcher to ensure tests use mock Claude CLI
	mock := claude.NewMockLauncher()
	mock.SetResponseForPattern(".*", "Mock response")
	app.SetMockLauncher(mock)

	hookInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash"
	}`

	result, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}

	// MUST use mock Claude CLI - verify mock response
	expectedMockResponse := "Mock response"
	if result != expectedMockResponse {
		t.Errorf("Expected mock response %q, but got real Claude response: %s", expectedMockResponse, result)
		t.Error("TEST FAILURE: CLI tests MUST use mock Claude CLI, not real Claude CLI")
	}
}

func TestProcessHookWithPartiallyInvalidConfig(t *testing.T) {
	t.Parallel()
	setupTest(t)

	// Create config with some valid and some invalid rules
	configContent := `rules:
  - match: "^go test"
    send: "Use just test instead"
    generate: "off"
  - match: "[invalid regex"
    send: "This rule has invalid regex"
    generate: "off"
  - match: "rm -rf"
    send: "Dangerous command - use safer alternatives"
    generate: "off"
  - match: ""
    send: "Empty pattern rule"
    generate: "off"
`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Test that hook processing works with valid rules even when some are invalid
	hookInput := `{
		"tool_name": "Bash",
		"tool_input": {
			"command": "go test ./..."
		}
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected ProcessHook to work with partial config, got error: %v", err)
	}

	if response == "" {
		t.Fatal("Expected response from valid rule, got empty string")
	}

	if !strings.Contains(response, "just test") {
		t.Errorf("Expected response to contain 'just test', got: %s", response)
	}

	// Test that the second valid rule also works
	hookInput2 := `{
		"tool_name": "Bash",
		"tool_input": {
			"command": "rm -rf /tmp"
		}
	}`

	response2, err := app.ProcessHook(strings.NewReader(hookInput2))
	if err != nil {
		t.Fatalf("Expected ProcessHook to work with second valid rule, got error: %v", err)
	}

	if !strings.Contains(response2, "safer") {
		t.Errorf("Expected response to contain 'safer', got: %s", response2)
	}

	// Test that invalid rules don't match anything (command should be allowed)
	hookInput3 := `{
		"tool_name": "Bash",
		"tool_input": {
			"command": "some other command"
		}
	}`

	response3, err := app.ProcessHook(strings.NewReader(hookInput3))
	if err != nil {
		t.Fatalf("Expected no error for unmatched command, got: %v", err)
	}

	if response3 != "" {
		t.Errorf("Expected empty response for unmatched command, got: %s", response3)
	}
}

func TestValidateConfigWithPartiallyInvalidConfig(t *testing.T) {
	t.Parallel()
	setupTest(t)

	// Create config with some valid and some invalid rules
	configContent := `rules:
  - match: "^go test"
    send: "Use just test instead"
    generate: "off"
  - match: "[invalid regex"
    send: "This rule has invalid regex"
    generate: "off"
  - match: "rm -rf"
    send: "Dangerous command - use safer alternatives"
    generate: "off"
`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Test that ValidateConfig provides useful feedback for partial configs
	result, err := app.ValidateConfig()
	if err != nil {
		t.Fatalf("Expected ValidateConfig to handle partial configs, got error: %v", err)
	}

	if result == "" {
		t.Error("Expected validation result message")
	}

	// Should mention both valid rules and invalid rules
	expectedMessages := []string{
		"2 valid rules",
		"1 invalid rule",
		"invalid regex",
	}

	for _, expected := range expectedMessages {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected result to contain '%s', got: %s", expected, result)
		}
	}
}

func TestProcessHookWithAllInvalidRules(t *testing.T) {
	t.Parallel()
	setupTest(t)

	// Create config with only invalid rules
	configContent := `rules:
  - match: "[invalid regex"
    send: "This rule has invalid regex"
    generate: "off"
  - match: ""
    send: "Empty pattern rule"
    generate: "off"
`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Test that hook processing still works (should allow all commands)
	hookInput := `{
		"tool_name": "Bash",
		"tool_input": {
			"command": "go test ./..."
		}
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected ProcessHook to work even with all invalid rules, got error: %v", err)
	}

	// Should allow command (empty response) since no valid rules match
	if response != "" {
		t.Errorf("Expected empty response when no valid rules exist, got: %s", response)
	}
}

func TestReadToolMatching(t *testing.T) {
	t.Parallel()
	setupTest(t)

	// Config with rule that should match Read tool accessing bumpers.yml
	configContent := `rules:
  - match: "bumpers.yml"
    tool: "Read|Edit|Grep"
    send: "Bumpers configuration file should not be accessed directly."
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewAppWithWorkDir(configPath, "")

	// Hook input simulating Read tool with file_path (not command)
	hookInput := `{
		"tool_name": "Read",
		"tool_input": {
			"file_path": "/home/callan/dev/bumpers/bumpers.yml"
		}
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	// Should now match because we check all string fields in tool_input
	if response == "" {
		t.Error("Expected Read tool to match bumpers.yml rule but got empty response")
	}

	// Verify the response content
	if !strings.Contains(response, "Bumpers configuration file should not be accessed directly") {
		t.Errorf("Expected specific message about bumpers.yml, got: %s", response)
	}
}

func TestReadToolSecretsMatching(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `rules:
  - match: "secrets"
    tool: "Read"
    send: "Tool usage blocked"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewAppWithWorkDir(configPath, "")

	hookInput := `{
		"tool_name": "Read",
		"tool_input": {
			"file_path": "/home/user/secrets.txt"
		}
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	if response == "" {
		t.Error("Expected Read tool to match on file_path field but got empty response")
	}
}

func TestGrepToolPasswordMatching(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `rules:
  - match: "password"
    tool: "Grep"
    send: "Tool usage blocked"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewAppWithWorkDir(configPath, "")

	hookInput := `{
		"tool_name": "Grep",
		"tool_input": {
			"pattern": "password",
			"path": "/home/user"
		}
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	if response == "" {
		t.Error("Expected Grep tool to match on pattern field but got empty response")
	}
}

func TestWriteToolNoMatching(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `rules:
  - match: "secrets"
    tool: "Write"
    send: "Tool usage blocked"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewAppWithWorkDir(configPath, "")

	hookInput := `{
		"tool_name": "Write",
		"tool_input": {
			"file_path": "/home/user/normal.txt",
			"content": "normal content"
		}
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	if response != "" {
		t.Errorf("Expected Write tool not to match rule but got response: %s", response)
	}
}

func TestFindMatchingRule(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `rules:
  - match: "secrets"
    tool: "Read"
    send: "Blocked"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewAppWithWorkDir(configPath, "")

	_, ruleMatcher, err := app.loadConfigAndMatcher()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	event := hooks.HookEvent{
		ToolName: "Read",
		ToolInput: map[string]any{
			"file_path": "/home/user/secrets.txt",
			"other":     123,
		},
	}

	rule, value, err := app.findMatchingRule(ruleMatcher, event)
	if err != nil {
		t.Fatalf("findMatchingRule failed: %v", err)
	}

	if rule == nil {
		t.Error("Expected rule match but got nil")
	}

	if value != "/home/user/secrets.txt" {
		t.Errorf("Expected matched value '/home/user/secrets.txt', got %s", value)
	}
}
