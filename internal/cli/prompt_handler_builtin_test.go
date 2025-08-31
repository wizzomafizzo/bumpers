package cli

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessUserPrompt_BuiltinCommand(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Test the builtin command directly with in-memory database
	result, err := ProcessBuiltinCommand(ctx, "bumpers status", ":memory:", "/test/project")
	require.NoError(t, err)
	require.Contains(t, result.(string), "Rules are currently")
}

func TestProcessUserPrompt_BuiltinCommandUsesProjectRoot(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Create a temporary database file that can be shared across calls
	tempDir := t.TempDir()
	dbPath := tempDir + "/test.db"

	// Use the same database but different project IDs to test state isolation
	project1ID := "/test/project1"
	project2ID := "/test/project2"

	// Disable rules in project1
	_, err := ProcessBuiltinCommand(ctx, "bumpers disable", dbPath, project1ID)
	require.NoError(t, err)

	// Check status in project1 - should be disabled
	result1, err := ProcessBuiltinCommand(ctx, "bumpers status", dbPath, project1ID)
	require.NoError(t, err)
	require.Contains(t, result1.(string), "disabled")

	// Check status in project2 - should still be enabled (independent state)
	result2, err := ProcessBuiltinCommand(ctx, "bumpers status", dbPath, project2ID)
	require.NoError(t, err)
	require.Contains(t, result2.(string), "enabled", "project2 should have independent state from project1")
}

func TestProcessUserPrompt_BuiltinCommandIntegration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Create a test config with a command
	tempDir := t.TempDir()
	configPath := tempDir + "/bumpers.yml"
	configContent := `commands:
  - name: test
    send: "Test command response"
rules:
  - match: "dummy"
    send: "dummy rule"`

	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o600))

	// Create prompt handler with test database
	handler := NewPromptHandler(configPath, tempDir)
	// Need a way to inject test database path - this should fail if method doesn't exist
	handler.SetTestDBPath(":memory:")

	// Test regular command first to see expected format
	regularJSON := []byte(`{"prompt": "$test"}`)
	regularResult, err := handler.ProcessUserPrompt(ctx, regularJSON)
	require.NoError(t, err)

	// Regular commands return JSON with hookSpecificOutput
	t.Logf("Regular command result: %s", regularResult)
	require.Contains(t, regularResult, "hookSpecificOutput")

	// Now test builtin command - document what SHOULD happen
	builtinJSON := []byte(`{"prompt": "$bumpers status"}`)
	builtinResult, err := handler.ProcessUserPrompt(ctx, builtinJSON)

	// This test demonstrates that we need database path injection for testing
	// Currently fails because it tries to access live database
	require.NoError(t, err, "Need database path injection to make builtin commands testable")

	// Builtin commands should BLOCK the prompt and show result to user
	// They should NOT continue to Claude like regular commands do
	t.Logf("Builtin command result: %s", builtinResult)
	require.Contains(t, builtinResult, `"decision":"block"`)
	require.Contains(t, builtinResult, `"reason"`)
	require.Contains(t, builtinResult, "Rules are currently")
}
