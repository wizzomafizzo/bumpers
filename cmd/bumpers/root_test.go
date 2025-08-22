package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCreateRootCommand(t *testing.T) {
	t.Parallel()

	cmd := createNewRootCommand()

	if cmd.Use != "bumpers" {
		t.Errorf("Expected command use 'bumpers', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected non-empty short description")
	}

	// Should have a hook subcommand
	hookCmd, _, err := cmd.Find([]string{"hook"})
	if err != nil {
		t.Fatalf("Expected hook command to exist, got error: %v", err)
	}
	if hookCmd.Use != "hook" {
		t.Errorf("Expected hook command use 'hook', got '%s'", hookCmd.Use)
	}
}

func TestNewRootCommandShowsHelp(t *testing.T) {
	t.Parallel()

	cmd := createNewRootCommand()

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Expected root command to execute successfully, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("Expected help output to contain 'Available Commands', got: %s", output)
	}
}

func TestNewRootCommandHasAllSubcommands(t *testing.T) {
	t.Parallel()

	cmd := createNewRootCommand()

	// Should have hook command
	hookCmd, _, err := cmd.Find([]string{"hook"})
	if err != nil {
		t.Fatalf("Expected hook command to exist, got error: %v", err)
	}
	if hookCmd.Use != "hook" {
		t.Errorf("Expected hook command use 'hook', got '%s'", hookCmd.Use)
	}

	// Should have install command
	installCmd, _, err := cmd.Find([]string{"install"})
	if err != nil {
		t.Fatalf("Expected install command to exist, got error: %v", err)
	}
	if installCmd.Use != "install" {
		t.Errorf("Expected install command use 'install', got '%s'", installCmd.Use)
	}

	// Should have status command
	statusCmd, _, err := cmd.Find([]string{"status"})
	if err != nil {
		t.Fatalf("Expected status command to exist, got error: %v", err)
	}
	if statusCmd.Use != "status" {
		t.Errorf("Expected status command use 'status', got '%s'", statusCmd.Use)
	}

	// Should have validate command
	validateCmd, _, err := cmd.Find([]string{"validate"})
	if err != nil {
		t.Fatalf("Expected validate command to exist, got error: %v", err)
	}
	if validateCmd.Use != "validate" {
		t.Errorf("Expected validate command use 'validate', got '%s'", validateCmd.Use)
	}
}

func TestNewRootCommandHasConfigFlag(t *testing.T) {
	t.Parallel()

	cmd := createNewRootCommand()

	configFlag := cmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("Expected config flag to exist")
		return
	}

	if configFlag.DefValue != "bumpers.yml" {
		t.Errorf("Expected config flag default value 'bumpers.yml', got '%s'", configFlag.DefValue)
	}
}

func TestCreateAppFromCommand(t *testing.T) {
	t.Parallel()

	cmd := createNewRootCommand()
	cmd.SetArgs([]string{"--config", "test.yml"})

	// Parse flags before accessing them
	err := cmd.ParseFlags([]string{"--config", "test.yml"})
	if err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	app, err := createAppFromCommand(cmd)
	if err != nil {
		t.Fatalf("Expected createAppFromCommand to succeed, got error: %v", err)
	}

	if app == nil {
		t.Error("Expected createAppFromCommand to return non-nil app")
	}
}
