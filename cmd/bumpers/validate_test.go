package main

import (
	"testing"
)

func TestCreateValidateCommand(t *testing.T) {
	t.Parallel()

	cmd := createValidateCommand()

	if cmd.Use != "validate" {
		t.Errorf("Expected command use 'validate', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected non-empty short description")
	}

	if cmd.RunE == nil {
		t.Error("Expected validate command to have RunE function")
	}
}
