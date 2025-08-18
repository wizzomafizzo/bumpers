package cli

import (
	"strings"
	"testing"
)

func TestProcessHook(t *testing.T) {
	app := NewApp("../../configs/bumpers.yaml")
	
	hookInput := `{
		"command": "go test ./...",
		"args": ["go", "test", "./..."],
		"cwd": "/path/to/project"
	}`
	
	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Should return a response since "go test" matches a rule
	if response == "" {
		t.Error("Expected non-empty response for blocked command")
	}
	
	if !strings.Contains(response, "make test") {
		t.Error("Response should suggest make test alternative")
	}
}

func TestProcessHookAllowed(t *testing.T) {
	app := NewApp("../../configs/bumpers.yaml")
	
	hookInput := `{
		"command": "make test",
		"args": ["make", "test"],
		"cwd": "/path/to/project"
	}`
	
	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Should return empty response since "make test" doesn't match any deny rule
	if response != "" {
		t.Errorf("Expected empty response for allowed command, got %s", response)
	}
}

func TestProcessHookDangerousCommand(t *testing.T) {
	app := NewApp("../../configs/bumpers.yaml")
	
	hookInput := `{
		"command": "rm -rf /tmp",
		"args": ["rm", "-rf", "/tmp"],
		"cwd": "/path/to/project"
	}`
	
	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Should return a response since dangerous rm command matches a rule
	if response == "" {
		t.Error("Expected non-empty response for dangerous command")
	}
}

func TestProcessHookPatternMatching(t *testing.T) {
	app := NewApp("../../configs/bumpers.yaml")
	
	hookInput := `{
		"command": "go test -v ./pkg/...",
		"args": ["go", "test", "-v", "./pkg/..."],
		"cwd": "/path/to/project"
	}`
	
	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Should return a response since this matches "go test.*" pattern
	if response == "" {
		t.Error("Expected non-empty response for go test pattern match")
	}
	
	if !strings.Contains(response, "make test") {
		t.Error("Response should suggest make test alternative")
	}
}

func TestConfigurationIsUsed(t *testing.T) {
	// This test ensures we're actually using the config file by checking for 
	// a specific message from the config rather than hardcoded responses
	app := NewApp("../../configs/bumpers.yaml")
	
	hookInput := `{
		"command": "go test ./...",
		"args": ["go", "test", "./..."],
		"cwd": "/path/to/project"
	}`
	
	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Should contain the exact message from config file
	if !strings.Contains(response, "better TDD integration") {
		t.Error("Response should contain message from config file")
	}
}

func TestTestCommand(t *testing.T) {
	app := NewApp("../../configs/bumpers.yaml")
	
	result, err := app.TestCommand("go test ./...")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if !strings.Contains(result, "blocked") {
		t.Error("Result should indicate command is blocked")
	}
}