package main

import (
	"testing"
)

const testHookCommand = "hook"

func TestHookExitError_Error(t *testing.T) {
	t.Parallel()
	err := &HookExitError{
		Message: "Test error message",
		Code:    2,
	}

	if err.Error() != "Test error message" {
		t.Errorf("Expected 'Test error message', got '%s'", err.Error())
	}
}

func TestInitLoggingForHook(t *testing.T) {
	t.Parallel()
	ctx, err := initLoggingForHook()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if ctx == nil {
		t.Error("Expected non-nil context")
	}
}

func TestCreateHookCommand(t *testing.T) {
	t.Parallel()

	cmd := createHookCommand()

	if cmd.Use != testHookCommand {
		t.Errorf("Expected Use 'hook', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected non-empty Short description")
	}

	if cmd.Long == "" {
		t.Error("Expected non-empty Long description")
	}

	if !cmd.SilenceUsage {
		t.Error("Expected SilenceUsage to be true")
	}

	if cmd.RunE == nil {
		t.Error("Expected RunE to be set")
	}
}
