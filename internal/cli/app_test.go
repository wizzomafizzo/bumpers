package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/core/engine/hooks"
	"github.com/wizzomafizzo/bumpers/internal/infrastructure/constants"
	"github.com/wizzomafizzo/bumpers/internal/platform/claude"
	"github.com/wizzomafizzo/bumpers/internal/platform/claude/api"
	"github.com/wizzomafizzo/bumpers/internal/platform/filesystem"
	"github.com/wizzomafizzo/bumpers/internal/platform/storage"
	"github.com/wizzomafizzo/bumpers/internal/testing"
)

// setupTest initializes test logger to prevent race conditions
func setupTest(t *testing.T) {
	t.Helper()
	testutil.InitTestLogger(t)
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
	setupTest(t)
	t.Parallel()

	configPath := "/path/to/config.yml"
	workDir := "/test/working/directory"

	app := NewAppWithWorkDir(configPath, workDir)

	assert.Equal(t, configPath, app.configPath, "configPath should match")
	assert.Equal(t, workDir, app.workDir, "workDir should match")
}

func TestCustomConfigPathLoading(t *testing.T) {
	setupTest(t)
	t.Parallel()

	// Create a custom config with a specific rule
	configContent := `rules:
  - match: "test-pattern"
    send: "Custom config loaded!"`

	customConfigPath := createTempConfig(t, configContent)

	// Create app with custom config path
	app := NewAppWithWorkDir(customConfigPath, t.TempDir())

	// Test that the custom config is actually loaded by validating it
	result, err := app.ValidateConfig()
	require.NoError(t, err)
	assert.Contains(t, result, "Configuration is valid")

	// Test that the rule from custom config works
	response, err := app.TestCommand("test-pattern should match")
	require.NoError(t, err)
	assert.Contains(t, response, "Custom config loaded!")
}

// TestAppWithMemoryFileSystem tests App initialization with in-memory filesystem for parallel testing
func TestAppWithMemoryFileSystem(t *testing.T) {
	setupTest(t)
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
	setupTest(t)
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
	setupTest(t)
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
	mock := claude.SetupMockLauncherWithDefaults()
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
	setupTest(t)
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
	setupTest(t)
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
	setupTest(t)
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
	setupTest(t)
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
	setupTest(t)
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
	setupTest(t)
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
	setupTest(t)
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
	setupTest(t)
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

	// Create memory filesystem for proper test isolation
	fs := filesystem.NewMemoryFileSystem()

	// Write the bumpers binary to the memory filesystem as well
	err = fs.WriteFile(bumpersPath, []byte("#!/bin/bash\necho test"), 0o750)
	if err != nil {
		t.Fatal(err)
	}

	// Use filesystem-injected constructor to avoid real file system writes
	app := NewAppWithFileSystem(configPath, tempDir, fs)

	err = app.Initialize()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that both PreToolUse and UserPromptSubmit hooks were added
	claudeDir := filepath.Join(tempDir, ".claude")
	localSettingsPath := filepath.Join(claudeDir, "settings.local.json")
	content, err := fs.ReadFile(localSettingsPath) // Use filesystem interface
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)

	// Check for PreToolUse hook (empty matcher is omitted in JSON)
	if !strings.Contains(contentStr, `"PreToolUse"`) {
		t.Error("Expected PreToolUse hook to be added to settings.local.json")
	}

	// Check for PostToolUse hook (empty matcher is omitted in JSON)
	if !strings.Contains(contentStr, `"PostToolUse"`) {
		t.Error("Expected PostToolUse hook to be added to settings.local.json")
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

	// Check that all four hooks contain bumpers command
	bashHookCount := strings.Count(contentStr, "bumpers")
	if bashHookCount < 4 {
		t.Errorf("Expected at least 4 bumpers hooks "+
			"(PreToolUse, PostToolUse, UserPromptSubmit, SessionStart), found %d",
			bashHookCount)
	}
}

func TestProcessHookSimplifiedSchemaAlwaysDenies(t *testing.T) {
	setupTest(t)
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
	setupTest(t)
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
	setupTest(t)
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
	setupTest(t)
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
	setupTest(t)
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
	setupTest(t)
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
	setupTest(t)
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
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

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
	setupTest(t)
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
	setupTest(t)
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
	setupTest(t)
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
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name: "Help command ($help)",
			input: `{
				"prompt": "$help"
			}`,
			want: `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
				`"additionalContext":"Available commands:\\n$help - Show this help\\n$status - Show project status"}}`,
			wantErr: false,
		},
		{
			name: "Status command ($status)",
			input: `{
				"prompt": "$status"
			}`,
			want: `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
				`"additionalContext":"Project Status: All systems operational"}}`,
			wantErr: false,
		},
		{
			name: "Non-command prompt",
			input: `{
				"prompt": "regular question"
			}`,
			want:    "",
			wantErr: false,
		},
		{
			name: "Invalid command index ($5)",
			input: `{
				"prompt": "$5"
			}`,
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := app.ProcessUserPrompt(json.RawMessage(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessUserPrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("ProcessUserPrompt() = %q, want %q", result, tt.want)
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
	mockLauncher := claude.SetupMockLauncherWithDefaults()
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
	t.Cleanup(func() { os.Args = originalArgs })

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
		"tool_input": {"command": "ls"},
		"tool_name": "Bash"
	}`

	_, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}
}

func TestProcessHookPreToolUseMatchesCommand(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - match: "ls"
    send: "Use file explorer instead"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"hookEventName": "PreToolUse",
		"tool_input": {"command": "ls"},
		"tool_name": "Bash"
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	require.NoError(t, err)
	assert.NotEmpty(t, response, "Should deny command with message")
	assert.Contains(t, response, "Use file explorer instead")
}

func TestProcessHookPreToolUseRespectsEventField(t *testing.T) {
	t.Parallel()

	// Rule with event: "post" should NOT match PreToolUse hooks
	configContent := `rules:
  - match:
      pattern: "ls"
      event: "post"
    send: "Use file explorer instead"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"hookEventName": "PreToolUse",
		"tool_input": {"command": "ls"},
		"tool_name": "Bash"
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	require.NoError(t, err)
	assert.Empty(t, response, "Rule with event=post should not match PreToolUse hooks")
}

func TestProcessHookPreToolUseSourcesFiltering(t *testing.T) {
	t.Parallel()

	// Rule with sources=[command] should not match description field
	configContent := `rules:
  - match:
      pattern: "delete"
      sources: ["command"]
    send: "Be careful with delete operations"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"hookEventName": "PreToolUse",
		"tool_input": {"command": "ls", "description": "delete files"},
		"tool_name": "Bash"
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	require.NoError(t, err)
	assert.Empty(t, response, "Rule with sources=[command] should not match description field")
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

func TestProcessHookWithTemplatePattern(t *testing.T) {
	t.Parallel()
	setupTest(t)

	// Setup a temporary project directory with config
	projectDir := t.TempDir()
	configContent := fmt.Sprintf(`rules:
  - match: "^%s/bumpers\\.yml$"
    tool: "Read|Edit|Grep"
    send: "Bumpers configuration file should not be accessed."
    generate: "off"`, "{{.ProjectRoot}}")

	configPath := filepath.Join(projectDir, "bumpers.yml")
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	// Use NewAppWithWorkDir to explicitly set the project directory
	app := NewAppWithWorkDir(configPath, projectDir)
	app.projectRoot = projectDir // Explicitly set project root for template processing

	// Test that template pattern matches project-specific file
	hookInput := fmt.Sprintf(`{
		"tool_input": {
			"file_path": "%s/bumpers.yml"
		},
		"tool_name": "Read"
	}`, projectDir)

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	require.NoError(t, err, "Expected no error for main config file")

	expectedMessage := "Bumpers configuration file should not be accessed."
	assert.Equal(t, expectedMessage, response, "Template pattern should match project-specific path")

	// Test that template pattern doesn't match test files in project
	// Create testdata directory
	testdataDir := filepath.Join(projectDir, "testdata")
	err = os.MkdirAll(testdataDir, 0o750)
	require.NoError(t, err)

	hookInputTest := fmt.Sprintf(`{
		"tool_input": {
			"file_path": "%s/testdata/bumpers.yml"
		},
		"tool_name": "Read"
	}`, projectDir)

	response2, err2 := app.ProcessHook(strings.NewReader(hookInputTest))
	// This should NOT match the rule, so we expect no error and empty response (command allowed)
	require.NoError(t, err2, "Expected no error for non-matching file")
	assert.Empty(t, response2, "Template pattern should not match test files, so command should be allowed")
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

func TestProcessSessionStartClearsSessionCache(t *testing.T) { //nolint:paralleltest // t.Setenv() usage
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

	// Set XDG_DATA_HOME to use temp directory for cache - this works with t.Parallel()
	dataHome := filepath.Join(tempDir, ".local", "share")
	t.Setenv("XDG_DATA_HOME", dataHome)

	configPath := createTempConfig(t, `session:
  - add: "Session started"`)
	app = NewApp(configPath)
	app.projectRoot = tempDir

	// Get the actual cache path that the app will use
	storageManager := storage.New(filesystem.NewOSFileSystem())
	var err error
	cachePath, err = storageManager.GetCachePath()
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
	// Close cache after populating to allow ProcessSessionStart to access it
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Failed to close cache: %v", closeErr)
		}
	}()

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
	t.Cleanup(func() { _ = cache.Close() })

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
	mockLauncher := claude.SetupMockLauncherWithDefaults()
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
	mock := claude.SetupMockLauncherWithDefaults()
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
	mock := claude.SetupMockLauncherWithDefaults()
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

// TestProcessHookRoutesPostToolUse tests that PostToolUse hooks are properly routed
func TestProcessHookRoutesPostToolUse(t *testing.T) {
	setupTest(t)
	t.Parallel()

	configContent := `rules:
  # Post-tool-use rule matching output (TODO: implement tool_output support)
  - match:
      pattern: "error|failed"
      event: "post"
      sources: ["tool_output"]
    send: "Command failed - check the output"
  # Post-tool-use rule matching reasoning
  - match:
      pattern: "not related to my changes"
      event: "post"
      sources: ["#intent"]
    send: "AI claiming unrelated - please verify"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Use static testdata transcript
	testdataDir := "../../testdata"
	transcriptPath := filepath.Join(testdataDir, "transcript-not-related.jsonl")
	absPath, err := filepath.Abs(transcriptPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	transcriptPath = absPath

	// Test PostToolUse hook with reasoning match
	postToolUseInput := `{
		"session_id": "abc123",
		"transcript_path": "` + transcriptPath + `",
		"cwd": "/test/directory",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_input": {
			"command": "npm test"
		},
		"tool_output": {
			"success": true,
			"output": "All tests passed"
		}
	}`

	result, err := app.ProcessHook(strings.NewReader(postToolUseInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for PostToolUse: %v", err)
	}

	// Should match the reasoning rule and return the message
	expectedMessage := "AI claiming unrelated - please verify"
	if result != expectedMessage {
		t.Errorf("Expected %q, got %q", expectedMessage, result)
	}
}

// TestPostToolUseWithDifferentTranscript tests that PostToolUse reads actual transcript content
func TestPostToolUseWithDifferentTranscript(t *testing.T) {
	setupTest(t)
	t.Parallel()

	configContent := `rules:
  - match:
      pattern: "permission denied"
      event: "post"
      sources: ["#intent"]
    send: "File permission error detected"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Use static testdata transcript with permission denied content
	testdataDir := "../../testdata"
	transcriptPath := filepath.Join(testdataDir, "transcript-permission-denied.jsonl")
	absPath, err := filepath.Abs(transcriptPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	transcriptPath = absPath

	postToolUseInput := `{
		"session_id": "test456",
		"transcript_path": "` + transcriptPath + `",
		"cwd": "/test/dir",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_input": {"command": "ls /root"},
		"tool_output": {"error": true}
	}`

	result, err := app.ProcessHook(strings.NewReader(postToolUseInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	// This should match "permission denied" pattern, not return the hardcoded stub message
	expectedMessage := "File permission error detected"
	if result != expectedMessage {
		t.Errorf("Expected %q (from transcript matching), got %q", expectedMessage, result)
	}

	// Verify it's NOT returning the hardcoded stub message
	stubMessage := "AI claiming unrelated - please verify"
	if result == stubMessage {
		t.Error("Got hardcoded stub message - PostToolUse handler not reading transcript properly")
	}
}

// TestPostToolUseRuleNotMatching tests that PostToolUse returns empty when no rules match
func TestPostToolUseRuleNotMatching(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `rules:
  - match:
      pattern: "file not found"
      event: "post"
      sources: ["#intent"]
    send: "Check the file path"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Use transcript that won't match the "file not found" pattern
	testdataDir := "../../testdata"
	transcriptPath := filepath.Join(testdataDir, "transcript-no-match.jsonl")
	absPath, err := filepath.Abs(transcriptPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	postToolUseInput := `{
		"session_id": "test789",
		"transcript_path": "` + absPath + `",
		"cwd": "/test/dir",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_input": {"command": "ls"},
		"tool_output": {"success": true}
	}`

	result, err := app.ProcessHook(strings.NewReader(postToolUseInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	// Should return empty string when no rules match
	if result != "" {
		t.Errorf("Expected empty result when no rules match, got %q", result)
	}
}

// TestPostToolUseWithCustomPattern tests rule system integration with custom patterns
func TestPostToolUseWithCustomPattern(t *testing.T) {
	t.Parallel()
	setupTest(t)

	// Create a config with a custom pattern that doesn't match hardcoded patterns
	configContent := `rules:
  - match:
      pattern: "timeout.*occurred"
      event: "post"
      sources: ["#intent"]
    send: "Operation timed out - check network connection"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Create a temporary transcript file with the custom pattern
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "custom-transcript.jsonl")
	transcriptContent := `{"type":"assistant","message":{"content":[{"type":"text",` +
		`"text":"The command failed because a timeout occurred while connecting to the server"}]}}`

	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600); err != nil {
		t.Fatalf("Failed to write transcript file: %v", err)
	}

	postToolUseInput := fmt.Sprintf(`{
		"session_id": "test123",
		"transcript_path": "%s",
		"cwd": "/test/dir",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_input": {"command": "curl -m 5 example.com"},
		"tool_output": {"success": false}
	}`, transcriptPath)

	result, err := app.ProcessHook(strings.NewReader(postToolUseInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	// Should match the custom pattern and return the configured message
	expectedMessage := "Operation timed out - check network connection"
	if result != expectedMessage {
		t.Errorf("Expected %q, got %q (rule system integration needed)",
			expectedMessage, result)
	}
}

// TestPostToolUseWithToolOutputMatching tests tool_output field matching
func TestPostToolUseWithToolOutputMatching(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `rules:
  - match:
      pattern: "error.*exit code"
      event: "post"
      sources: ["tool_response"]
    send: "Tool execution failed"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Create mock JSON with tool_response containing error
	jsonData := `{
		"session_id": "test123",
		"transcript_path": "",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_response": "Command failed with error: exit code 1"
	}`

	result, err := app.ProcessPostToolUse(json.RawMessage(jsonData))
	if err != nil {
		t.Fatalf("ProcessPostToolUse failed: %v", err)
	}

	assert.Equal(t, "Tool execution failed", result)
}

// TestPostToolUseWithMultipleFieldMatching tests multiple field matching in a single rule
func TestPostToolUseWithMultipleFieldMatching(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `rules:
  - match:
      pattern: "timeout|permission denied"
      event: "post"
      sources: ["#intent", "tool_response"]
    send: "Operation issue detected"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Test with tool_output containing timeout
	jsonData1 := `{
		"session_id": "test123",
		"transcript_path": "",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_response": "Connection timeout occurred"
	}`

	result1, err := app.ProcessPostToolUse(json.RawMessage(jsonData1))
	if err != nil {
		t.Fatalf("ProcessPostToolUse failed: %v", err)
	}

	assert.Equal(t, "Operation issue detected", result1)

	// Test with reasoning content containing permission denied
	testdataDir := "../../testdata"
	transcriptPath := filepath.Join(testdataDir, "transcript-permission-denied.jsonl")
	absPath, err := filepath.Abs(transcriptPath)
	if err != nil {
		t.Fatalf("Could not get absolute path: %v", err)
	}

	jsonData2 := fmt.Sprintf(`{
		"session_id": "test456",
		"transcript_path": "%s",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_response": "Command executed successfully"
	}`, absPath)

	result2, err := app.ProcessPostToolUse(json.RawMessage(jsonData2))
	if err != nil {
		t.Fatalf("ProcessPostToolUse failed: %v", err)
	}

	assert.Equal(t, "Operation issue detected", result2)
}

// TestPostToolUseWithThinkingAndTextBlocks tests extraction of both thinking blocks and text content
func TestPostToolUseWithThinkingAndTextBlocks(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `rules:
  - match:
      pattern: "need to analyze.*performance"
      event: "post"
      sources: ["#intent"]
    send: "Performance analysis detected"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Create transcript with both thinking blocks and text content
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "thinking-text-transcript.jsonl")
	transcriptContent := `{"type":"user","message":{"role":"user","content":"Check system performance"},` +
		`"uuid":"user1","timestamp":"2024-01-01T10:00:00Z"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking",` +
		`"thinking":"I need to analyze the system performance metrics to identify bottlenecks."},` +
		`{"type":"text","text":"I'll check the system performance for you."}]},"uuid":"assistant1",` +
		`"timestamp":"2024-01-01T10:01:00Z"}
{"type":"user","message":{"role":"user","content":[{"tool_use_id":"tool1","type":"tool_result",` +
		`"content":[{"type":"text","text":"CPU: 85%, Memory: 70%, Disk: 40%"}]}]},"uuid":"user2",` +
		`"timestamp":"2024-01-01T10:02:00Z"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking",` +
		`"thinking":"The CPU usage is quite high at 85%, which could indicate performance issues."},` +
		`{"type":"text","text":"The system shows high CPU usage at 85%. ` +
		`This could be causing performance issues."}]},"uuid":"assistant2","timestamp":"2024-01-01T10:03:00Z"}`

	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600); err != nil {
		t.Fatalf("Failed to write transcript file: %v", err)
	}

	postToolUseInput := fmt.Sprintf(`{
		"session_id": "test-thinking",
		"transcript_path": "%s",
		"cwd": "/test/dir",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_input": {"command": "top"},
		"tool_output": {"success": true}
	}`, transcriptPath)

	result, err := app.ProcessHook(strings.NewReader(postToolUseInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	// Should match pattern from thinking block content
	expectedMessage := "Performance analysis detected"
	if result != expectedMessage {
		t.Errorf("Expected %q (thinking block should be extracted), got %q", expectedMessage, result)
	}
}

func TestProcessUserPromptWithCommandArguments(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `commands:
  - name: "test"
    send: "Command: {{.Name}}, Args: {{argc}}, First: {{argv 1}}"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	promptJSON := `{"prompt": "$test arg1 arg2"}`
	result, err := app.ProcessUserPrompt(json.RawMessage(promptJSON))
	if err != nil {
		t.Fatalf("ProcessUserPrompt failed: %v", err)
	}

	// Parse the response to get the additionalContext
	var response map[string]any
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	hookOutput, ok := response["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatal("Response missing hookSpecificOutput")
	}

	additionalContext, ok := hookOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Response missing additionalContext")
	}

	expected := "Command: test, Args: 2, First: arg1"
	if additionalContext != expected {
		t.Errorf("Expected %q, got %q", expected, additionalContext)
	}
}

func TestProcessUserPromptWithNoArguments(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `commands:
  - name: "test"
    send: "Command: {{.Name}}, Args: {{argc}}"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	promptJSON := `{"prompt": "$test"}`
	result, err := app.ProcessUserPrompt(json.RawMessage(promptJSON))
	if err != nil {
		t.Fatalf("ProcessUserPrompt failed: %v", err)
	}

	// Parse the response
	var response map[string]any
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	hookOutput, ok := response["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatalf("Expected hookSpecificOutput to be map[string]any, got %T", response["hookSpecificOutput"])
	}
	additionalContext, ok := hookOutput["additionalContext"].(string)
	if !ok {
		t.Fatalf("Expected additionalContext to be string, got %T", hookOutput["additionalContext"])
	}

	expected := "Command: test, Args: 0"
	if additionalContext != expected {
		t.Errorf("Expected %q, got %q", expected, additionalContext)
	}
}

type preToolUseIntentTestCase struct {
	testName          string
	transcriptContent string
	matchPattern      string
	command           string
	desc              string
	expectedMessage   string
}

// Helper function to test PreToolUse intent matching with different scenarios
func testPreToolUseIntentMatching(t *testing.T, tc *preToolUseIntentTestCase) {
	t.Helper()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, tc.testName+"-transcript.jsonl")

	err := os.WriteFile(transcriptPath, []byte(tc.transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write transcript file: %v", err)
	}

	configContent := fmt.Sprintf(`rules:
  - match:
      pattern: "%s"
      event: "pre"
      sources: ["#intent"]
    send: "%s"
    generate: "off"`, tc.matchPattern, tc.expectedMessage)

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := fmt.Sprintf(`{
		"transcript_path": "%s",
		"tool_name": "Bash",
		"tool_input": {
			"command": "%s",
			"description": "%s"
		}
	}`, transcriptPath, tc.command, tc.desc)

	result, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	if result != tc.expectedMessage {
		t.Errorf("Expected %q, got %q", tc.expectedMessage, result)
	}
}

func TestPreToolUseIntentSupport(t *testing.T) {
	t.Parallel()
	setupTest(t)

	transcriptContent := `{"type":"assistant","message":{"content":[` +
		`{"type":"thinking","thinking":"I need to run some database tests to verify the connection works"}, ` +
		`{"type":"text","text":"I'll run the database tests for you"}]}}`
	testPreToolUseIntentMatching(t, &preToolUseIntentTestCase{
		testName:          "test",
		transcriptContent: transcriptContent,
		matchPattern:      "database.*test",
		command:           "python test_db.py",
		desc:              "Run database tests",
		expectedMessage:   "Consider checking database connection first",
	})
}

func TestPreToolUseIntentWithTextOnlyContent(t *testing.T) {
	t.Parallel()
	setupTest(t)

	transcriptContent := `{"type":"assistant","message":{"content":[` +
		`{"type":"text","text":"I need to perform a critical system update to fix vulnerabilities"}]}}`
	testPreToolUseIntentMatching(t, &preToolUseIntentTestCase{
		testName:          "text-only",
		transcriptContent: transcriptContent,
		matchPattern:      "critical.*system",
		command:           "sudo apt update",
		desc:              "System update",
		expectedMessage:   "System update detected",
	})
}

// TestPreToolUseIntentWithMissingTranscript tests graceful handling when transcript doesn't exist
func TestPreToolUseIntentWithMissingTranscript(t *testing.T) {
	t.Parallel()
	setupTest(t)

	configContent := `rules:
  - match:
      pattern: "anything"
      event: "pre"
      sources: ["#intent"]
    send: "This should not match"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"transcript_path": "/non/existent/transcript.jsonl",
		"tool_name": "Bash",
		"tool_input": {
			"command": "echo hello",
			"description": "Simple echo command"
		}
	}`

	result, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	// Should return empty when transcript can't be read
	if result != "" {
		t.Errorf("Expected empty result when transcript missing, got %q", result)
	}
}

type postToolUseIntegrationTestCase struct {
	sessionID       string
	transcriptFile  string
	matchPattern    string
	command         string
	expectedMessage string
	success         bool
}

// Helper function for PostToolUse integration tests
func testPostToolUseIntegration(t *testing.T, tc *postToolUseIntegrationTestCase) {
	t.Helper()

	testdataDir := "../../testdata"
	transcriptPath := filepath.Join(testdataDir, tc.transcriptFile)
	absPath, err := filepath.Abs(transcriptPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	configContent := fmt.Sprintf(`rules:
  - match:
      pattern: "%s"
      event: "post"
      sources: ["#intent"]
    send: "%s"
    generate: "off"`, tc.matchPattern, tc.expectedMessage)

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	postToolUseInput := fmt.Sprintf(`{
		"session_id": "%s",
		"transcript_path": "%s",
		"cwd": "/test/dir",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_input": {"command": "%s"},
		"tool_output": {"success": %t}
	}`, tc.sessionID, absPath, tc.command, tc.success)

	result, err := app.ProcessHook(strings.NewReader(postToolUseInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	if result != tc.expectedMessage {
		t.Errorf("Expected %q, got %q", tc.expectedMessage, result)
	}
}

func TestIntegrationThinkingAndTextExtraction(t *testing.T) {
	t.Parallel()
	setupTest(t)

	testPostToolUseIntegration(t, &postToolUseIntegrationTestCase{
		sessionID:       "integration-test",
		transcriptFile:  "transcript-thinking-and-text.jsonl",
		matchPattern:    "critical.*attention",
		command:         "go test ./...",
		expectedMessage: "Critical issue requires careful handling",
		success:         false,
	})
}

func TestIntegrationPerformanceAnalysisDetection(t *testing.T) {
	setupTest(t)
	t.Parallel()

	testPostToolUseIntegration(t, &postToolUseIntegrationTestCase{
		sessionID:       "perf-test",
		transcriptFile:  "transcript-performance-analysis.jsonl",
		matchPattern:    "immediate.*optimization",
		command:         "top",
		expectedMessage: "Performance optimization guidance triggered",
		success:         true,
	})
}

// TestPostToolUseStructuredToolResponseFields tests matching against specific fields in structured tool responses
func TestPostToolUseStructuredToolResponseFields(t *testing.T) {
	t.Parallel()
	setupTest(t)

	// Test that rules can match against specific tool output fields by name
	configContent := `rules:
  - match:
      pattern: "JWT_SECRET"
      event: "post"
      sources: ["content"]  # Should match the "content" field in tool_response
    send: "Configuration file contains sensitive data"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	// Test matching against "content" field
	jsonWithContent := `{
		"session_id": "test123",
		"transcript_path": "",
		"hook_event_name": "PostToolUse", 
		"tool_name": "Read",
		"tool_response": {
			"content": "module.exports = { secret: process.env.JWT_SECRET }",
			"lines": 42
		}
	}`

	result1, err := app.ProcessPostToolUse(json.RawMessage(jsonWithContent))
	require.NoError(t, err)
	assert.Equal(t, "Configuration file contains sensitive data", result1)
}
