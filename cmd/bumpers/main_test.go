package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestInstallCommandExistence(t *testing.T) {
	t.Parallel()

	// Test that buildMainRootCommand creates a command with install subcommand
	rootCmd := buildMainRootCommand()

	// Look for install command
	installCmd, _, err := rootCmd.Find([]string{"install"})
	if err != nil {
		t.Fatalf("Expected install command to exist, got error: %v", err)
	}

	if installCmd == nil {
		t.Error("Expected install command to exist")
	}
}

func TestInstallCommandHasRunFunction(t *testing.T) {
	t.Parallel()

	// Test that the install command has a Run function wired up
	rootCmd := buildMainRootCommand()
	installCmd, _, err := rootCmd.Find([]string{"install"})
	if err != nil {
		t.Fatalf("Expected install command to exist, got error: %v", err)
	}

	if installCmd.Run == nil {
		t.Error("Expected install command to have Run function")
	}
}

func TestInstallCommandActuallyWorks(t *testing.T) {
	t.Parallel()

	// Test that install command actually initializes
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

	rootCmd := buildMainRootCommand()
	rootCmd.SetArgs([]string{"install"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected install command to execute successfully, got: %v", err)
	}

	// Should create bumpers.yml
	if _, err := os.Stat("bumpers.yml"); os.IsNotExist(err) {
		t.Error("Expected bumpers.yml to be created")
	}
}

func TestTestCommandActuallyWorks(t *testing.T) {
	t.Parallel()

	// Test that test command tests rules
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

	// First install to create config
	rootCmd := buildMainRootCommand()
	rootCmd.SetArgs([]string{"install"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	// Reset command and test a blocked command
	rootCmd = buildMainRootCommand()
	rootCmd.SetArgs([]string{"test", "go test"})

	var output bytes.Buffer
	rootCmd.SetOut(&output)

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected test command to execute successfully, got: %v", err)
	}

	result := output.String()
	if !strings.Contains(result, "make test") {
		t.Errorf("Expected output to contain 'make test', got: %s", result)
	}
}

func TestConfigFlagWorks(t *testing.T) { //nolint:paralleltest // changes working directory
	// Test that config flag is properly passed to subcommands
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

	// Create a custom config file
	customConfigPath := "custom-config.yaml"
	err = os.WriteFile(customConfigPath, []byte(`rules:
  - pattern: "echo test"
    response: "Custom config loaded"
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// Test with custom config flag
	rootCmd := buildMainRootCommand()
	rootCmd.SetArgs([]string{"--config", customConfigPath, "test", "echo test"})

	var output bytes.Buffer
	var errOutput bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&errOutput)

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected test command to execute successfully, got: %v", err)
	}

	result := output.String()
	if !strings.Contains(result, "Custom config loaded") {
		t.Errorf("Expected output to contain custom config response, got: %s", result)
	}
}

func TestHookDeniedCommandOutputsToStderrAndExitsCode2(t *testing.T) {
	t.Parallel()

	// Test that denied commands output to stderr and exit with code 2 for Claude Code hooks
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
	configPath := "bumpers.yml"
	err = os.WriteFile(configPath, []byte(`rules:
  - pattern: "go test"
    response: "Test command blocked"
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// Create hook input
	hookInput := `{"tool": "Bash", "command": "go test ./..."}`

	// Test main hook processing
	rootCmd := buildMainRootCommand()
	rootCmd.SetArgs([]string{"--config", configPath})

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetIn(strings.NewReader(hookInput))

	err = rootCmd.Execute()

	// Should exit with code 2 (this will be an ExitError)
	if err == nil {
		t.Fatal("Expected command to exit with non-zero code")
	}

	// Check stderr contains the denial message
	stderrOutput := stderr.String()
	if !strings.Contains(stderrOutput, "Test command blocked") {
		t.Errorf("Expected stderr to contain denial message, got: %s", stderrOutput)
	}

	// Check stdout is empty (message should go to stderr)
	stdoutOutput := stdout.String()
	if stdoutOutput != "" {
		t.Errorf("Expected stdout to be empty, got: %s", stdoutOutput)
	}
}
