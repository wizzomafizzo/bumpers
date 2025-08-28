//go:build e2e

package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/testing"
)

func TestCreateHookCommand(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging for e2e tests
	t.Parallel()

	cmd := createHookCommand()

	if cmd.Use != "hook" {
		t.Errorf("Expected command use 'hook', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected non-empty short description")
	}

	if cmd.RunE == nil {
		t.Error("Expected hook command to have RunE function")
	}
}

func TestHookCommandBlocksWithProperToolName(t *testing.T) { //nolint:paralleltest // changes working directory
	_, _ = testutil.NewTestContext(t) // Context-aware logging for e2e tests
	// Test that hook command blocks when tool_name is provided correctly
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create config with deny rule and AI generation disabled
	configContent := `rules:
  - match: "^go test"
    send: "Test command blocked"
    generate: "off"`

	configPath := filepath.Join(tempDir, "bumpers.yml")
	err = os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// Create hook input WITH tool_name
	hookInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash"
	}`

	// Test hook command execution
	rootCmd := createNewRootCommand()
	rootCmd.SetArgs([]string{"hook", "--config", configPath})

	var stderr bytes.Buffer
	rootCmd.SetErr(&stderr)
	rootCmd.SetIn(strings.NewReader(hookInput))

	err = rootCmd.Execute()
	// Command should return HookExitError for blocked commands (exit code 2)
	if err == nil {
		t.Error("Expected command to be blocked with error, but got no error")
		return
	}

	// Check if it's a HookExitError with the expected message and exit code
	hookErr := &HookExitError{}
	if !errors.As(err, &hookErr) {
		t.Errorf("Expected HookExitError, got: %T", err)
		return
	}

	if hookErr.Code != 2 {
		t.Errorf("Expected exit code 2 for blocked command, got %d", hookErr.Code)
	}
	if !strings.Contains(hookErr.Message, "Test command blocked") {
		t.Errorf("Expected error message to contain 'Test command blocked', got: %s", hookErr.Message)
	}
}

func TestHookCommandNoDuplicateOutput(t *testing.T) { //nolint:paralleltest // changes working directory
	_, _ = testutil.NewTestContext(t) // Context-aware logging for e2e tests
	// Test that blocked commands don't output the message twice
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create config with deny rule and AI generation disabled
	configContent := `rules:
  - match: "^go test"
    send: "Test command blocked"
    generate: "off"`

	configPath := filepath.Join(tempDir, "bumpers.yml")
	err = os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// Create hook input WITH tool_name
	hookInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash"
	}`

	// Test hook command execution
	rootCmd := createNewRootCommand()
	rootCmd.SetArgs([]string{"hook", "--config", configPath})

	var stderr bytes.Buffer
	var stdout bytes.Buffer
	rootCmd.SetErr(&stderr)
	rootCmd.SetOut(&stdout)
	rootCmd.SetIn(strings.NewReader(hookInput))

	_ = rootCmd.Execute()

	// Verify the message only appears once in stderr (not duplicated)
	stderrContent := stderr.String()
	messageCount := strings.Count(stderrContent, "Test command blocked")
	if messageCount != 1 {
		t.Errorf("Expected message to appear exactly once in stderr, found %d occurrences in: %s",
			messageCount, stderrContent)
	}

	// Verify stdout is empty for blocked commands
	if stdout.String() != "" {
		t.Errorf("Expected empty stdout for blocked command, got: %s", stdout.String())
	}
}

func TestHookCommandAllowsInputWithMissingToolName(t *testing.T) { //nolint:paralleltest // changes working directory
	_, _ = testutil.NewTestContext(t) // Context-aware logging for e2e tests
	// Test that hook command allows input when tool_name is missing (safe default)
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create config with deny rule and AI generation disabled
	configContent := `rules:
  - match: "^go test"
    send: "Test command blocked"
    generate: "off"`

	configPath := filepath.Join(tempDir, "bumpers.yml")
	err = os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// Create hook input
	hookInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		}
	}`

	// Test hook command execution
	rootCmd := createNewRootCommand()
	rootCmd.SetArgs([]string{"hook", "--config", configPath})

	var stderr bytes.Buffer
	rootCmd.SetErr(&stderr)
	rootCmd.SetIn(strings.NewReader(hookInput))

	err = rootCmd.Execute()
	// Command should be allowed when tool_name is missing (safe default behavior)
	if err != nil {
		t.Errorf("Expected command to be allowed when tool_name is missing, but got error: %v", err)
	}
}
