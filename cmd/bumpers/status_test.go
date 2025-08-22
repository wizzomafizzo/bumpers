package main

import (
	"testing"
)

func TestCreateStatusCommand(t *testing.T) {
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
