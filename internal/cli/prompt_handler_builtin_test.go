package cli

import (
	"context"
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
