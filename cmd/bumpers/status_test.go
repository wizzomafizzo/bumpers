package main

import (
	"testing"

	testutil "github.com/wizzomafizzo/bumpers/internal/testing"
)

func TestCreateStatusCommand(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging
	t.Parallel()

	cmd := createStatusCommand()

	if cmd.Use != "status" {
		t.Errorf("Expected command use 'status', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected non-empty short description")
	}

	if cmd.RunE == nil {
		t.Error("Expected status command to have RunE function")
	}
}
