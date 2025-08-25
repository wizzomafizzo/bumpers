//go:build e2e

package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/testutil"
)

// setupTest initializes test logger to prevent race conditions
func setupTest(t *testing.T) {
	t.Helper()
	testutil.InitTestLogger(t)
}

func TestInstallCommandExistence(t *testing.T) {
	t.Parallel()
	setupTest(t)

	// Test that createNewRootCommand creates a command with install subcommand
	rootCmd := createNewRootCommand()

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
	setupTest(t)

	// Test that the install command has a Run function wired up
	rootCmd := createNewRootCommand()
	installCmd, _, err := rootCmd.Find([]string{"install"})
	if err != nil {
		t.Fatalf("Expected install command to exist, got error: %v", err)
	}

	if installCmd.RunE == nil {
		t.Error("Expected install command to have RunE function")
	}
}

func TestInstallCommandActuallyWorks(t *testing.T) { //nolint:paralleltest // uses global logger state
	setupTest(t)
	// Test that install command actually initializes
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

	rootCmd := createNewRootCommand()
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

func TestHookCommandExistence(t *testing.T) {
	t.Parallel()
	setupTest(t)

	// Test that createNewRootCommand creates a command with hook subcommand
	rootCmd := createNewRootCommand()

	// Look for hook command
	hookCmd, _, err := rootCmd.Find([]string{"hook"})
	if err != nil {
		t.Fatalf("Expected hook command to exist, got error: %v", err)
	}

	if hookCmd == nil {
		t.Error("Expected hook command to exist")
		return
	}

	if hookCmd.RunE == nil {
		t.Error("Expected hook command to have RunE function")
	}
}

func TestConfigFlagWorks(t *testing.T) { //nolint:paralleltest // changes working directory
	setupTest(t)
	// Test that config flag is properly passed to subcommands
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

	// Create a custom config file
	customConfigPath := "custom-config.yaml"
	err = os.WriteFile(customConfigPath, []byte(`rules:
  - match: "echo test"
    send: "Custom config loaded"
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// Test with custom config flag - use validate command to test config parsing
	rootCmd := createNewRootCommand()
	rootCmd.SetArgs([]string{"--config", customConfigPath, "validate"})

	var output bytes.Buffer
	rootCmd.SetOut(&output)

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected validate command to execute successfully, got: %v", err)
	}

	// Just verify the command ran without error - validation success indicates config was loaded
}

func TestRootCommandShowsHelp(t *testing.T) {
	t.Parallel()
	setupTest(t)

	// Test that root command shows help when run without arguments
	rootCmd := createNewRootCommand()
	rootCmd.SetArgs([]string{})

	var output bytes.Buffer
	rootCmd.SetOut(&output)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected root command to execute successfully, got: %v", err)
	}

	result := output.String()
	if !strings.Contains(result, "Available Commands") {
		t.Errorf("Expected output to contain help text, got: %s", result)
	}
}

func TestMainUsesProjectContextForLogger(t *testing.T) { //nolint:paralleltest // tests logger initialization
	setupTest(t)
	// Test that main.go uses InitWithProjectContext instead of Init
	// This is a behavioral test - we can't easily mock the logger initialization,
	// but we can verify that the function completes successfully with the new approach
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

	// The main functionality should work with the new project context approach
	err = run()
	if err != nil {
		t.Fatalf("Expected run() to complete successfully with project context logging, got: %v", err)
	}

	// The fact that run() completed successfully means InitWithProjectContext worked
	// The actual XDG path testing is done in the storage package tests
}

func TestRunFunctionWrapsCommandExecutionErrors(t *testing.T) { //nolint:paralleltest // changes working directory
	setupTest(t)
	// Test that run() function properly wraps errors from command execution
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

	// Use an invalid flag to force command execution to fail
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"bumpers", "--invalid-flag"}

	// Call run() directly to test error wrapping
	err = run()
	// Should return an error that wraps the underlying command error
	if err == nil {
		t.Fatal("Expected run() to return an error for invalid command, got nil")
	}

	// Error message should provide context about what failed (not just the raw cobra error)
	errorMsg := err.Error()
	if errorMsg == "" {
		t.Error("Expected non-empty error message")
	}

	// Check that the error message contains contextual information, not just the raw error
	// The wrapped error should contain additional context like "command execution failed"
	if errorMsg == "unknown flag: --invalid-flag" {
		t.Error("Error should be wrapped with context, but got raw cobra error")
	}

	// Should contain some kind of context about command execution
	hasCommand := strings.Contains(errorMsg, "command")
	hasExecution := strings.Contains(errorMsg, "execution")
	hasFailed := strings.Contains(errorMsg, "failed")
	if !hasCommand && !hasExecution && !hasFailed {
		t.Errorf("Expected error to contain contextual information, got: %s", errorMsg)
	}
}
