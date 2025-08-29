package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/wizzomafizzo/bumpers/internal/infrastructure/constants"
)

func TestAppInitializeWithMemoryFileSystem(t *testing.T) {
	t.Parallel()

	// Setup in-memory filesystem with test config
	fs := afero.NewMemMapFs()
	configContent := []byte(`rules:
  - match: "rm -rf"
    send: "Use safer alternatives"`)
	configPath := "/test/bumpers.yml"

	err := afero.WriteFile(fs, configPath, configContent, 0o600)
	if err != nil {
		t.Fatalf("Failed to setup test config: %v", err)
	}

	// Add bumpers binary to memory filesystem (needed for validateBumpersPath)
	bumpersPath := "/test/workdir/bin/bumpers"
	err = afero.WriteFile(fs, bumpersPath, []byte("fake bumpers binary"), 0o755)
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
	content, err := afero.ReadFile(app.fileSystem, configPath)
	if err != nil {
		t.Errorf("Failed to read config after Initialize: %v", err)
	}

	if !bytes.Equal(content, configContent) {
		t.Error("Config content changed after Initialize")
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

	// Create memory filesystem for proper test isolation
	fs := afero.NewMemMapFs()

	// Write the bumpers binary to the memory filesystem as well
	err = afero.WriteFile(fs, bumpersPath, []byte("#!/bin/bash\necho test"), 0o750)
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
	content, err := afero.ReadFile(fs, localSettingsPath) // Use filesystem interface
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

// createTempConfig creates a temporary config file for testing

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

	// Recreate InstallManager with correct project root
	app.installManager = NewInstallManager(app.configPath, app.workDir, app.projectRoot, nil)

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
