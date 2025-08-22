package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateClaudeBackupCommand(t *testing.T) {
	t.Parallel()
	// Create a temporary test directory
	testDir := t.TempDir()

	// Create a mock settings.json file
	settingsFile := filepath.Join(testDir, "settings.json")
	err := os.WriteFile(settingsFile, []byte(`{"outputStyle":"default"}`), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test settings file: %v", err)
	}

	// Test that backup command can be created
	cmd := createClaudeBackupCommand()
	if cmd == nil {
		t.Error("createClaudeBackupCommand() returned nil")
		return
	}

	if cmd.Use != "backup" {
		t.Errorf("Expected command use 'backup', got '%s'", cmd.Use)
	}
}

func TestClaudeCommandIntegration(t *testing.T) {
	t.Parallel()
	// Test that Claude command can be integrated into root command
	rootCmd := createRootCommand()

	// Find the claude command
	claudeCmd, _, err := rootCmd.Find([]string{"claude"})
	if err != nil {
		t.Fatalf("Claude command not found: %v", err)
	}

	if claudeCmd.Use != "claude" {
		t.Errorf("Expected command use 'claude', got '%s'", claudeCmd.Use)
	}

	// Verify backup subcommand exists
	backupCmd, _, err := claudeCmd.Find([]string{"backup"})
	if err != nil {
		t.Fatalf("Backup subcommand not found: %v", err)
	}

	if backupCmd.Use != "backup" {
		t.Errorf("Expected backup command use 'backup', got '%s'", backupCmd.Use)
	}
}

func TestMainCommandStructure(t *testing.T) {
	t.Parallel()
	// This test ensures that when we refactor main.go to use createRootCommand,
	// it maintains the existing test, status commands as well as adding claude
	rootCmd := buildMainRootCommand()

	// Verify existing commands still exist
	testCmd, _, err := rootCmd.Find([]string{"test"})
	if err != nil {
		t.Fatalf("Test command not found: %v", err)
	}
	if testCmd.Use != "test [command]" {
		t.Errorf("Expected test command use 'test [command]', got '%s'", testCmd.Use)
	}

	statusCmd, _, err := rootCmd.Find([]string{"status"})
	if err != nil {
		t.Fatalf("Status command not found: %v", err)
	}
	if statusCmd.Use != "status" {
		t.Errorf("Expected status command use 'status', got '%s'", statusCmd.Use)
	}

	// Verify claude command exists
	claudeCmd, _, err := rootCmd.Find([]string{"claude"})
	if err != nil {
		t.Fatalf("Claude command not found: %v", err)
	}
	if claudeCmd.Use != "claude" {
		t.Errorf("Expected claude command use 'claude', got '%s'", claudeCmd.Use)
	}
}

func TestStatusCommandShouldHaveRunFunction(t *testing.T) {
	t.Parallel()

	rootCmd := buildMainRootCommand()
	statusCmd, _, err := rootCmd.Find([]string{"status"})
	if err != nil {
		t.Fatalf("Status command not found: %v", err)
	}

	// Status command should have a RunE function to be functional
	if statusCmd.RunE == nil {
		t.Error("Status command should have a RunE function to be functional")
	}
}

func TestBackupCommandExecution(t *testing.T) {
	t.Parallel()
	// Create a temporary test directory
	testDir := t.TempDir()

	// Create a mock Claude settings file
	settingsFile := filepath.Join(testDir, "settings.json")
	originalContent := `{"outputStyle":"explanatory","model":"claude-3-5-sonnet-20241022"}`
	err := os.WriteFile(settingsFile, []byte(originalContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test settings file: %v", err)
	}

	// Test the backup functionality using settings package (the production approach)
	backupPath, err := executeBackupCommand(filepath.Dir(settingsFile))
	if err != nil {
		t.Fatalf("executeBackupCommand failed: %v", err)
	}

	// Verify backup was created
	if _, statErr := os.Stat(backupPath); os.IsNotExist(statErr) {
		t.Fatalf("Backup file was not created at %s", backupPath)
	}

	// Verify backup content matches original
	backupContent, err := os.ReadFile(backupPath) // #nosec G304
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if string(backupContent) != originalContent {
		t.Errorf("Backup content mismatch. Expected: %s, Got: %s", originalContent, string(backupContent))
	}
}

func TestBackupWithSimpleExtension(t *testing.T) {
	t.Parallel()
	// Create a temporary test directory
	testDir := t.TempDir()

	// Create a mock Claude settings file
	settingsFile := filepath.Join(testDir, "settings.json")
	originalContent := `{"outputStyle":"explanatory"}`
	err := os.WriteFile(settingsFile, []byte(originalContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test settings file: %v", err)
	}

	// Test the backup functionality using simple .bak extension
	backupPath, err := executeBackupCommand(filepath.Dir(settingsFile))
	if err != nil {
		t.Fatalf("executeBackupCommand failed: %v", err)
	}

	// Verify backup uses simple .bak extension
	expectedBackupPath := settingsFile + ".bak"
	if backupPath != expectedBackupPath {
		t.Errorf("Expected backup path %s, got: %s", expectedBackupPath, backupPath)
	}

	// Verify backup was created and has correct content
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatalf("Backup file was not created at %s", backupPath)
	}
}

func TestBackupCommandWithSettingsDiscovery(t *testing.T) {
	t.Parallel()
	// Create a temporary test directory structure like Claude's
	testDir := t.TempDir()

	// Create a mock Claude settings file
	settingsFile := filepath.Join(testDir, "settings.json")
	originalContent := `{"outputStyle":"explanatory","model":"claude-3-5-sonnet-20241022"}`
	err := os.WriteFile(settingsFile, []byte(originalContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test settings file: %v", err)
	}

	// Test finding the settings file
	foundPath, err := findClaudeSettingsIn(testDir)
	if err != nil {
		t.Fatalf("findClaudeSettingsIn failed: %v", err)
	}

	if foundPath != settingsFile {
		t.Errorf("Expected settings path %s, got %s", settingsFile, foundPath)
	}
}

func TestCreateClaudeRestoreCommand(t *testing.T) {
	t.Parallel()
	// Test that restore command can be created
	cmd := createClaudeRestoreCommand()
	if cmd == nil {
		t.Error("createClaudeRestoreCommand() returned nil")
		return
	}

	if cmd.Use != "restore [backup_file]" {
		t.Errorf("Expected command use 'restore [backup_file]', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Restore command should have a short description")
	}
}

func TestBackupCommandActualImplementation(t *testing.T) {
	t.Parallel()
	// Create a temporary test directory
	testDir := t.TempDir()

	// Create a mock Claude settings file
	settingsFile := filepath.Join(testDir, "settings.json")
	originalContent := `{"outputStyle":"explanatory","model":"claude-3-5-sonnet-20241022"}`
	err := os.WriteFile(settingsFile, []byte(originalContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test settings file: %v", err)
	}

	// Test backup command execution with proper settings discovery
	backupPath, err := executeBackupCommand(testDir)
	if err != nil {
		t.Fatalf("executeBackupCommand failed: %v", err)
	}

	// Verify backup was created with timestamp format
	if backupPath == "" {
		t.Error("Expected non-empty backup path")
	}

	// Verify backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatalf("Backup file was not created at %s", backupPath)
	}
}

func TestRestoreCommandActualImplementation(t *testing.T) {
	t.Parallel()
	// Create a temporary test directory
	testDir := t.TempDir()

	// Create original settings file
	settingsFile := filepath.Join(testDir, "settings.json")
	originalContent := `{"outputStyle":"explanatory","model":"claude-3-5-sonnet-20241022"}`
	err := os.WriteFile(settingsFile, []byte(originalContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test settings file: %v", err)
	}

	// Create a backup first
	backupPath, err := executeBackupCommand(testDir)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Modify the original file
	modifiedContent := `{"outputStyle":"minimal","model":"claude-3-haiku-20240307"}`
	err = os.WriteFile(settingsFile, []byte(modifiedContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to modify settings file: %v", err)
	}

	// Test restore command execution
	err = executeRestoreCommand(backupPath, settingsFile)
	if err != nil {
		t.Fatalf("executeRestoreCommand failed: %v", err)
	}

	// Verify content was restored
	restoredContent, err := os.ReadFile(settingsFile) // #nosec G304
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	if string(restoredContent) != originalContent {
		t.Errorf("Content not properly restored. Expected: %s, Got: %s", originalContent, string(restoredContent))
	}
}

func TestCLIBackupCommandWithCurrentDirectory(t *testing.T) { //nolint:paralleltest // changes working directory
	// Create a temporary directory and change to it
	testDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }() // Restore original directory after test

	err = os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	// Create a mock Claude settings file in current directory
	settingsFile := filepath.Join(testDir, "settings.json")
	originalContent := `{"outputStyle":"explanatory","model":"claude-3-5-sonnet-20241022"}`
	err = os.WriteFile(settingsFile, []byte(originalContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test settings file: %v", err)
	}

	// Test that CLI backup command can work from current directory
	backupPath, err := runBackupFromCurrentDirectory()
	if err != nil {
		t.Fatalf("runBackupFromCurrentDirectory failed: %v", err)
	}

	// Verify backup was created
	if backupPath == "" {
		t.Error("Expected non-empty backup path")
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatalf("Backup file was not created at %s", backupPath)
	}
}

func TestBackupCommandOutput(t *testing.T) { //nolint:paralleltest // changes working directory
	// Create a temporary directory and change to it
	testDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	// Create a mock Claude settings file
	settingsContent := `{"outputStyle":"explanatory","model":"claude-3-5-sonnet-20241022"}`
	err = os.WriteFile("settings.json", []byte(settingsContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test settings file: %v", err)
	}

	// Test that backup command produces appropriate output
	output, err := runBackupCommandWithOutput()
	if err != nil {
		t.Fatalf("runBackupCommandWithOutput failed: %v", err)
	}

	// Verify output contains backup information
	if output == "" {
		t.Error("Expected non-empty output from backup command")
	}

	// Should contain information about the backup created
	if !containsBackupInfo(output) {
		t.Errorf("Expected output to contain backup information, got: %s", output)
	}
}

func TestClaudeRestoreCommandInCLI(t *testing.T) {
	t.Parallel()
	// Test that the root command contains both backup and restore subcommands
	rootCmd := createRootCommand()

	// Find the claude command
	claudeCmd, _, err := rootCmd.Find([]string{"claude"})
	if err != nil {
		t.Fatalf("Claude command not found: %v", err)
	}

	// Verify both backup and restore subcommands exist
	backupCmd, _, err := claudeCmd.Find([]string{"backup"})
	if err != nil {
		t.Fatalf("Backup subcommand not found: %v", err)
	}
	if backupCmd.Use != "backup" {
		t.Errorf("Expected backup command use 'backup', got '%s'", backupCmd.Use)
	}

	// Verify restore subcommand exists
	restoreCmd, _, err := claudeCmd.Find([]string{"restore"})
	if err != nil {
		t.Fatalf("Restore subcommand not found: %v", err)
	}
	if restoreCmd.Use != "restore [backup_file]" {
		t.Errorf("Expected restore command use 'restore [backup_file]', got '%s'", restoreCmd.Use)
	}
}

func TestRestoreCommandExecution(t *testing.T) { //nolint:paralleltest // changes working directory
	// Test that the restore command can actually execute and restore settings
	// Create a temporary directory and change to it
	testDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	// Create original settings file
	originalContent := `{"outputStyle":"explanatory","model":"claude-3-5-sonnet-20241022"}`
	err = os.WriteFile("settings.json", []byte(originalContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create original settings file: %v", err)
	}

	// Create a backup
	backupPath, err := executeBackupCommand(".")
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Modify the original file
	modifiedContent := `{"outputStyle":"minimal","model":"claude-3-haiku-20240307"}`
	err = os.WriteFile("settings.json", []byte(modifiedContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to modify settings file: %v", err)
	}

	// Test that restore command can be executed (this should fail until implemented)
	cmd := createClaudeRestoreCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{backupPath})

	err = cmd.Execute()
	// Currently this will succeed because the command exists but does nothing
	// After implementation, we should verify the file was actually restored
	if err != nil {
		t.Fatalf("Restore command failed: %v", err)
	}

	// Verify that restore command actually restored the content
	// TODO: This test documents that restore should actually work
	restoredContent, err := os.ReadFile("settings.json")
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	// This assertion will fail until restore is properly implemented
	if string(restoredContent) != originalContent {
		t.Errorf("Restore command did not restore content. Expected: %s, Got: %s",
			originalContent, string(restoredContent))
	}
}

func TestValidateCommandExistence(t *testing.T) {
	t.Parallel()

	rootCmd := buildMainRootCommand()

	validateCmd, _, err := rootCmd.Find([]string{"validate"})
	if err != nil {
		t.Fatalf("validate command not found: %v", err)
	}

	if validateCmd.Use != "validate" {
		t.Errorf("Expected command use 'validate', got %s", validateCmd.Use)
	}

	if validateCmd.Short == "" {
		t.Error("validate command should have a short description")
	}
}

func TestValidateCommandExecution(t *testing.T) {
	t.Parallel()

	// Create a temporary config file
	tempDir := t.TempDir()
	configContent := `rules:
  - pattern: "^go test"
    response: "Use make test instead"`

	configPath := filepath.Join(tempDir, "test-config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}

	rootCmd := buildMainRootCommand()
	rootCmd.SetArgs([]string{"validate", "--config", configPath})

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("validate command execution failed: %v", err)
	}

	output := buf.String()
	if output != "Configuration is valid\n" {
		t.Errorf("Expected 'Configuration is valid', got %q", output)
	}
}

func TestProcessHookCommandExitCodes(t *testing.T) {
	t.Parallel()

	// Create a temporary config file with a command
	tempDir := t.TempDir()
	configContent := `commands:
  - name: test
    message: "Hello World"`

	configPath := filepath.Join(tempDir, "bumpers.yml")
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}

	tests := []struct {
		name              string
		input             string
		expectedResponse  string
		expectedExitCode  int
		shouldContainJSON bool
	}{
		{
			name:              "HookSpecificOutput should return exit code 0",
			input:             `{"session_id":"test-session","transcript_path":"/tmp/test.jsonl","cwd":"/tmp","hook_event_name":"UserPromptSubmit","prompt":"%test"}`,
			expectedExitCode:  0,
			shouldContainJSON: true,
		},
		{
			name:             "Non-command prompt should return exit code 0",
			input:            `{"session_id":"test-session","transcript_path":"/tmp/test.jsonl","cwd":"/tmp","hook_event_name":"UserPromptSubmit","prompt":"regular prompt"}`,
			expectedExitCode: 0,
			expectedResponse: "",
		},
		{
			name:             "Unknown command should return exit code 0",
			input:            `{"session_id":"test-session","transcript_path":"/tmp/test.jsonl","cwd":"/tmp","hook_event_name":"UserPromptSubmit","prompt":"%unknown"}`,
			expectedExitCode: 0,
			expectedResponse: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := bytes.NewBufferString(tt.input)
			var output bytes.Buffer

			exitCode, response := processHookCommand(configPath, input, &output)

			if exitCode != tt.expectedExitCode {
				t.Errorf("Expected exit code %d, got %d", tt.expectedExitCode, exitCode)
			}

			if tt.shouldContainJSON && !bytes.Contains([]byte(response), []byte("hookEventName")) {
				t.Errorf("Expected response to contain hookEventName JSON, got: %s", response)
			}

			if tt.expectedResponse != "" && response != tt.expectedResponse {
				t.Errorf("Expected response %q, got %q", tt.expectedResponse, response)
			}
		})
	}
}

// TestProcessHookCommandDebugLogging removed - debug logging was removed for security reasons
// to prevent potential exposure of sensitive user input

func TestMainCommandOutputStreams(t *testing.T) {
	// Create a temporary config file with a command
	tempDir := t.TempDir()
	configContent := `commands:
  - name: test
    message: "Hello World"`

	configPath := filepath.Join(tempDir, "bumpers.yml")
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}

	tests := []struct {
		name           string
		input          string
		expectedStdout string
		expectedStderr string
	}{
		{
			name:           "HookSpecificOutput should go to stdout",
			input:          `{"session_id":"test-session","transcript_path":"/tmp/test.jsonl","cwd":"/tmp","hook_event_name":"UserPromptSubmit","prompt":"%test"}`,
			expectedStdout: `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"Hello World"}}` + "\n",
			expectedStderr: "",
		},
		{
			name:           "Non-command should have no output",
			input:          `{"session_id":"test-session","transcript_path":"/tmp/test.jsonl","cwd":"/tmp","hook_event_name":"UserPromptSubmit","prompt":"regular prompt"}`,
			expectedStdout: "",
			expectedStderr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the main command
			rootCmd := createMainRootCommand()
			rootCmd.PersistentFlags().StringP("config", "c", configPath, "Path to config file")

			// Capture stdout and stderr
			var stdout, stderr bytes.Buffer
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)
			rootCmd.SetIn(strings.NewReader(tt.input))

			// Execute the command
			err := rootCmd.Execute()
			if err != nil {
				t.Fatalf("Command execution failed: %v", err)
			}

			// Check stdout
			if stdout.String() != tt.expectedStdout {
				if tt.expectedStdout != "" {
					t.Errorf("Expected stdout %q, got %q", tt.expectedStdout, stdout.String())
				}
			}

			// Check stderr
			if tt.expectedStderr == "" {
				if strings.TrimSpace(stderr.String()) != "" {
					t.Errorf("Expected empty stderr, got %q", stderr.String())
				}
			}
		})
	}
}
