package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/platform/claude"
)

func TestProcessHookWithContext(t *testing.T) {
	t.Parallel()
	ctx, getLogs := setupTestWithContext(t)

	// Create a temporary config
	configContent := `
rules: []
`
	configPath := createTempConfig(t, configContent)

	// Create app with same context
	app := NewApp(ctx, configPath)

	// Create a simple hook input - UserPromptSubmit with no matching rules
	input := `{"hook_event":"user_prompt_submit","intent":"echo hello"}`
	reader := strings.NewReader(input)

	// This should not fail and should accept context
	result, err := app.ProcessHook(ctx, reader)

	// Should not return an error
	require.NoError(t, err)
	assert.Empty(t, result) // No rules to match

	// Should be able to capture logs without race conditions
	logOutput := getLogs()
	assert.Contains(t, logOutput, "processing hook input")
}

func TestProcessHook(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead for better TDD integration"
    alternatives:
      - "make test          # Run all tests"
      - "make test-unit     # Run unit tests only"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	hookInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash"
	}`

	response, err := app.ProcessHook(ctx, strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return a response since "go test" matches a rule
	if response == "" {
		t.Error("Expected non-empty response for blocked command")
	}

	if !strings.Contains(response, "just test") {
		t.Error("Response should suggest just test alternative")
	}
}

func TestProcessHookAllowed(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead for better TDD integration"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	hookInput := `{
		"tool_input": {
			"command": "make test",
			"description": "Run tests with make"
		}
	}`

	response, err := app.ProcessHook(ctx, strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return empty response since "make test" doesn't match any deny rule
	if response != "" {
		t.Errorf("Expected empty response for allowed command, got %s", response)
	}
}

func TestProcessHookDangerousCommand(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `rules:
  - match: "rm -rf /*"
    send: "⚠️  Dangerous rm command detected"
    alternatives:
      - "Be more specific with your rm command"
      - "Use a safer alternative like moving to trash"
    generate:
      mode: "always"
      prompt: "Explain why this rm command is dangerous and suggest safer alternatives"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Set up mock launcher for AI generation
	mock := claude.SetupMockLauncherWithDefaults()
	mock.SetResponseForPattern(".*", "AI-generated response about dangerous rm command")
	app.SetMockLauncher(mock)

	hookInput := `{
		"tool_input": {
			"command": "rm -rf /tmp",
			"description": "Remove directory"
		},
		"tool_name": "Bash"
	}`

	response, err := app.ProcessHook(ctx, strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return a response since dangerous rm command matches a rule
	if response == "" {
		t.Error("Expected non-empty response for dangerous command")
	}
}

func TestProcessHookPatternMatching(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead for better TDD integration"
    alternatives:
      - "make test          # Run all tests"
      - "make test-unit     # Run unit tests only"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	hookInput := `{
		"tool_input": {
			"command": "go test -v ./pkg/...",
			"description": "Run verbose tests"
		},
		"tool_name": "Bash"
	}`

	response, err := app.ProcessHook(ctx, strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return a response since this matches "go test.*" pattern
	if response == "" {
		t.Error("Expected non-empty response for go test pattern match")
	}

	if !strings.Contains(response, "just test") {
		t.Error("Response should suggest just test alternative")
	}
}

func TestProcessHookWorks(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	hookInput := `{
		"hookEventName": "PreToolUse",
		"tool_input": {"command": "ls"},
		"tool_name": "Bash"
	}`

	_, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}
}

func TestProcessHookPreToolUseMatchesCommand(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `rules:
  - match: "ls"
    send: "Use file explorer instead"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	hookInput := `{
		"hookEventName": "PreToolUse",
		"tool_input": {"command": "ls"},
		"tool_name": "Bash"
	}`

	response, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput))
	require.NoError(t, err)
	assert.NotEmpty(t, response, "Should deny command with message")
	assert.Contains(t, response, "Use file explorer instead")
}

func TestProcessHookPreToolUseRespectsEventField(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	// Rule with event: "post" should NOT match PreToolUse hooks
	configContent := `rules:
  - match:
      pattern: "ls"
      event: "post"
    send: "Use file explorer instead"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	hookInput := `{
		"hookEventName": "PreToolUse",
		"tool_input": {"command": "ls"},
		"tool_name": "Bash"
	}`

	response, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput))
	require.NoError(t, err)
	assert.Empty(t, response, "Rule with event=post should not match PreToolUse hooks")
}

func TestProcessHookPreToolUseSourcesFiltering(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	// Rule with sources=[command] should not match description field
	configContent := `rules:
  - match:
      pattern: "delete"
      sources: ["command"]
    send: "Be careful with delete operations"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	hookInput := `{
		"hookEventName": "PreToolUse",
		"tool_input": {"command": "ls", "description": "delete files"},
		"tool_name": "Bash"
	}`

	response, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput))
	require.NoError(t, err)
	assert.Empty(t, response, "Rule with sources=[command] should not match description field")
}

func TestProcessHookLogsErrors(t *testing.T) {
	ctx, _ := setupTestWithContext(t)
	t.Parallel()
	tempDir := t.TempDir()

	// Use non-existent config path to trigger error
	app := NewAppWithWorkDir("non-existent-config.yml", tempDir)

	// Create logger for the app
	var err error

	hookInput := `{
		"tool_input": {
			"command": "test command",
			"description": "Test"
		}
	}`

	// This should trigger an error (logging is a side effect we can't easily test with global logger)
	result, err := app.ProcessHook(ctx, strings.NewReader(hookInput))
	if err == nil {
		t.Fatalf("Expected ProcessHook to return error for non-existent config, got result: %s", result)
	}

	// Verify the error is related to config loading
	if !strings.Contains(err.Error(), "config") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Expected config-related error, got: %v", err)
	}
}

func TestProcessHookSimplifiedSchemaAlwaysDenies(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	// Setup test config with simplified schema (no name or action fields)
	// Any pattern match should result in denial
	configContent := `rules:
  - match: "go test"
    send: "Use just test instead"
    generate: "off"
  - match: "rm -rf"
    send: "Dangerous command detected"
    generate: "off"
`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Test first rule - should be blocked because it matches (no action field needed)
	hookInput1 := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash"
	}`
	result1, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput1))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}
	if result1 != "Use just test instead" {
		t.Errorf("Expected 'Use just test instead', got %q", result1)
	}

	// Test second rule - should be blocked because it matches (no action field needed)
	hookInput2 := `{
		"tool_input": {
			"command": "rm -rf temp",
			"description": "Remove directory"
		},
		"tool_name": "Bash"
	}`
	result2, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput2))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}
	if result2 != "Dangerous command detected" {
		t.Errorf("Expected 'Dangerous command detected', got %q", result2)
	}

	// Test non-matching command - should be allowed
	hookInput3 := `{
		"tool_input": {
			"command": "make build",
			"description": "Build project"
		}
	}`
	result3, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput3))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}
	if result3 != "" {
		t.Errorf("Expected empty result for allowed command, got %q", result3)
	}
}
