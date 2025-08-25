package main

import (
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/testutil"
)

func TestCreateInstallCommand(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	cmd := createInstallCommand()

	if cmd.Use != "install" {
		t.Errorf("Expected command use 'install', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected non-empty short description")
	}

	if cmd.RunE == nil {
		t.Error("Expected install command to have RunE function")
	}
}
