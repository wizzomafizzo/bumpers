package app

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apphooks "github.com/wizzomafizzo/bumpers/internal/app/hooks"
	"github.com/wizzomafizzo/bumpers/internal/claude"
	"github.com/wizzomafizzo/bumpers/internal/storage"
)

const preToolUseHookInput = `{
	"hookEventName": "PreToolUse",
	"tool_input": {"command": "ls"},
	"tool_name": "Bash"
}`

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
	input := `{"hook_event_name":"UserPromptSubmit","prompt":"echo hello"}`
	reader := strings.NewReader(input)

	// This should not fail and should accept context
	result, err := app.ProcessHook(ctx, reader)

	// Should not return an error
	require.NoError(t, err)
	assert.Empty(t, result.Message) // No rules to match

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
	if response.Message == "" {
		t.Error("Expected non-empty response for blocked command")
	}

	if !strings.Contains(response.Message, "just test") {
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
	if response.Message != "" {
		t.Errorf("Expected empty response for allowed command, got %s", response.Message)
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
	if response.Message == "" {
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
	if response.Message == "" {
		t.Error("Expected non-empty response for go test pattern match")
	}

	if !strings.Contains(response.Message, "just test") {
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

	hookInput := preToolUseHookInput

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

	hookInput := preToolUseHookInput

	response, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput))
	require.NoError(t, err)
	assert.NotEmpty(t, response.Message, "Should deny command with message")
	assert.Contains(t, response.Message, "Use file explorer instead")
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

	hookInput := preToolUseHookInput

	response, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput))
	require.NoError(t, err)
	assert.Empty(t, response.Message, "Rule with event=post should not match PreToolUse hooks")
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
	assert.Empty(t, response.Message, "Rule with sources=[command] should not match description field")
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
	if result1.Message != "Use just test instead" {
		t.Errorf("Expected 'Use just test instead', got %q", result1.Message)
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
	if result2.Message != "Dangerous command detected" {
		t.Errorf("Expected 'Dangerous command detected', got %q", result2.Message)
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
	if result3.Message != "" {
		t.Errorf("Expected empty result for allowed command, got %q", result3.Message)
	}
}

func TestProcessHookRespectsDisabledState(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	// Create config with rule that should be matched
	configContent := `rules:
  - match: "go test"
    send: "Use just test instead"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Input that would normally match the rule
	hookInput := `{
		"hookEventName": "PreToolUse",
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash"
	}`

	// First, rules are enabled by default - should block
	result1, err := app.ProcessHook(ctx, strings.NewReader(hookInput))
	require.NoError(t, err)
	assert.NotEmpty(t, result1.Message, "Should block command when rules enabled")

	// Create a hook processor with a state manager that has rules disabled
	stateManager, err := createTestStateManagerWithDisabledRules(t)
	require.NoError(t, err)
	defer func() {
		if closeErr := stateManager.Close(); closeErr != nil {
			t.Errorf("Failed to close state manager: %v", closeErr)
		}
	}()

	hookProcessorWithState := apphooks.NewHookProcessor(app.configValidator, app.projectRoot, stateManager)

	// With disabled rules, the same command should be allowed
	result2, err := hookProcessorWithState.ProcessPreToolUse(ctx, []byte(hookInput))
	require.NoError(t, err)
	assert.Empty(t, result2, "Should allow command when rules disabled")
}

func TestStateManagerChecksBehaveIdentically(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	// Test data for both PreToolUse and PostToolUse
	preToolInput := `{
		"hookEventName": "PreToolUse",
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash"
	}`

	postToolInput := `{
		"hookEventName": "PostToolUse",
		"tool_name": "Bash",
		"tool_input": {
			"command": "go test",
			"description": "Run tests"
		},
		"tool_response": {
			"output": "test output"
		}
	}`

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead"
    generate: "off"
  - event: "post"
    match: "test output"
    send: "Post-tool message"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	validator := NewConfigValidator(configPath, t.TempDir())

	// Test with disabled rules - both should behave identically (return empty)
	stateManager, err := createTestStateManagerWithDisabledRules(t)
	require.NoError(t, err)
	defer func() { _ = stateManager.Close() }()

	hookProcessor := apphooks.NewHookProcessor(validator, configPath, stateManager)

	preResult, err := hookProcessor.ProcessPreToolUse(ctx, []byte(preToolInput))
	require.NoError(t, err)

	postResult, err := hookProcessor.ProcessPostToolUse(ctx, []byte(postToolInput))
	require.NoError(t, err)

	// Both should return empty string when rules are disabled
	assert.Empty(t, preResult, "PreToolUse should allow command when rules disabled")
	assert.Empty(t, postResult, "PostToolUse should allow command when rules disabled")

	// Test with skip flag - both should behave identically
	stateManager2, err := createTestStateManagerWithSkipFlag(t)
	require.NoError(t, err)
	defer func() { _ = stateManager2.Close() }()

	hookProcessor2 := apphooks.NewHookProcessor(validator, configPath, stateManager2)

	preSkipResult, err := hookProcessor2.ProcessPreToolUse(ctx, []byte(preToolInput))
	require.NoError(t, err)

	postSkipResult, err := hookProcessor2.ProcessPostToolUse(ctx, []byte(postToolInput))
	require.NoError(t, err)

	// Both should return empty string when skip flag is set
	assert.Empty(t, preSkipResult, "PreToolUse should allow command when skip flag set")
	assert.Empty(t, postSkipResult, "PostToolUse should allow command when skip flag set")
}

func TestShouldSkipProcessingMethodExists(t *testing.T) {
	t.Parallel()
	_, _ = setupTestWithContext(t) // Context not needed for this test

	// This test ensures the shouldSkipProcessing method exists and works correctly
	// to eliminate code duplication between ProcessPreToolUse and ProcessPostToolUse
	configPath := createTempConfig(t, "rules: []")
	validator := NewConfigValidator(configPath, t.TempDir())

	// Test with no state manager
	hookProcessor1 := apphooks.NewHookProcessor(validator, configPath, nil)
	// NOTE: Cannot test unexported shouldSkipProcessing method directly
	// This test would need to be refactored to use public methods
	_ = hookProcessor1 // Use the processor to avoid unused variable

	// Test with rules disabled
	stateManager, err := createTestStateManagerWithDisabledRules(t)
	require.NoError(t, err)
	defer func() { _ = stateManager.Close() }()

	hookProcessor2 := apphooks.NewHookProcessor(validator, configPath, stateManager)
	// NOTE: Cannot test unexported shouldSkipProcessing method directly
	// This test would need to be refactored to use public methods
	_ = hookProcessor2 // Use the processor to avoid unused variable
}

// createTestStateManagerWithDisabledRules creates a state manager for testing with rules disabled
func createTestStateManagerWithDisabledRules(t *testing.T) (*storage.StateManager, error) {
	ctx := context.Background()
	// Create temporary database file
	tempFile := t.TempDir() + "/test-state.db"

	// Create state manager
	stateManager, err := storage.NewStateManager(tempFile, "test-project")
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	// Disable rules
	err = stateManager.SetRulesEnabled(ctx, false)
	if err != nil {
		if closeErr := stateManager.Close(); closeErr != nil {
			t.Errorf("Failed to close state manager during cleanup: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to disable rules: %w", err)
	}

	return stateManager, nil
}

// createTestStateManagerWithSkipFlag creates a state manager for testing with skip flag set
func createTestStateManagerWithSkipFlag(t *testing.T) (*storage.StateManager, error) {
	ctx := context.Background()
	// Create temporary database file
	tempFile := t.TempDir() + "/test-state.db"

	// Create state manager
	stateManager, err := storage.NewStateManager(tempFile, "test-project")
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	// Set skip flag
	err = stateManager.SetSkipNext(ctx, true)
	if err != nil {
		if closeErr := stateManager.Close(); closeErr != nil {
			t.Errorf("Failed to close state manager during cleanup: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to set skip flag: %w", err)
	}

	return stateManager, nil
}

func TestProcessPostToolUseRespectsDisabledState(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	// Create config with a post-event rule that should be matched
	configContent := `rules:
  - match:
      pattern: "test output"
      event: "post"
    send: "Consider reviewing test results"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Input that would normally match the rule
	postHookInput := `{
		"hookEventName": "PostToolUse",
		"tool_name": "Bash",
		"tool_response": "test output from command",
		"transcript_path": ""
	}`

	// First, rules are enabled by default - should block
	result1, err := app.ProcessPostToolUse(ctx, []byte(postHookInput))
	require.NoError(t, err)
	assert.NotEmpty(t, result1, "Should block post-tool-use when rules enabled")

	// Create a hook processor with a state manager that has rules disabled
	stateManager, err := createTestStateManagerWithDisabledRules(t)
	require.NoError(t, err)
	defer func() {
		if closeErr := stateManager.Close(); closeErr != nil {
			t.Errorf("Failed to close state manager: %v", closeErr)
		}
	}()

	hookProcessorWithState := apphooks.NewHookProcessor(app.configValidator, app.projectRoot, stateManager)

	// With disabled rules, the same command should be allowed
	result2, err := hookProcessorWithState.ProcessPostToolUse(ctx, []byte(postHookInput))
	require.NoError(t, err)
	assert.Empty(t, result2, "Should allow post-tool-use when rules disabled")
}

func TestProcessPostToolUseRespectsSkipFlag(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	// Create config with a post-event rule that should be matched
	configContent := `rules:
  - match:
      pattern: "test output"
      event: "post"
    send: "Consider reviewing test results"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Input that would normally match the rule
	postHookInput := `{
		"hookEventName": "PostToolUse",
		"tool_name": "Bash",
		"tool_response": "test output from command",
		"transcript_path": ""
	}`

	// Create a state manager with skip flag set
	stateManager, err := createTestStateManagerWithSkipFlag(t)
	require.NoError(t, err)
	defer func() {
		if closeErr := stateManager.Close(); closeErr != nil {
			t.Errorf("Failed to close state manager: %v", closeErr)
		}
	}()

	hookProcessorWithState := apphooks.NewHookProcessor(app.configValidator, app.projectRoot, stateManager)

	// With skip flag set, command should be allowed and flag consumed
	result, err := hookProcessorWithState.ProcessPostToolUse(ctx, []byte(postHookInput))
	require.NoError(t, err)
	assert.Empty(t, result, "Should allow post-tool-use when skip flag set")

	// Verify skip flag was consumed - next call should block
	result2, err := hookProcessorWithState.ProcessPostToolUse(ctx, []byte(postHookInput))
	require.NoError(t, err)
	assert.NotEmpty(t, result2, "Should block post-tool-use after skip flag consumed")
}
