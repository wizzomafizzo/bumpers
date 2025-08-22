package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateHookCommand(t *testing.T) {
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

func TestHookCommandProcessesInput(t *testing.T) { //nolint:paralleltest // changes working directory
	// Test that hook command can process hook input
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create config with deny rule
	configContent := `rules:
  - pattern: "go test"
    response: "Test command blocked"`

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
	// Command should not return error, but should exit with status code
	if err == nil {
		// If it doesn't error, check stderr for blocking message
		stderrOutput := stderr.String()
		if !strings.Contains(stderrOutput, "Test command blocked") {
			t.Errorf("Expected stderr to contain denial message, got: %s", stderrOutput)
		}
	}
}
