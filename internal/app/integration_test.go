package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/rules"
	"github.com/wizzomafizzo/bumpers/internal/storage"
)

func TestPromptHandler_CreatedWithStateManager(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	stateManager, err := storage.NewStateManager(dbPath, "test-project")
	require.NoError(t, err)

	t.Cleanup(func() { _ = stateManager.Close() })

	// Create prompt handler with state manager (this should work with current constructor)
	handler := NewPromptHandler(filepath.Join(tempDir, "bumpers.yml"), tempDir, stateManager)
	require.NotNil(t, handler.stateManager)

	// Test that operation mode actually works through this handler
	ctx := context.Background()
	discussionState := &rules.OperationState{
		Mode:         rules.PlanMode,
		TriggerCount: 0,
		UpdatedAt:    123456789,
	}
	err = stateManager.SetOperationMode(ctx, discussionState)
	require.NoError(t, err)

	// Create trigger phrase event
	event := UserPromptEvent{
		Prompt: "go ahead",
	}
	eventJSON, err := json.Marshal(event)
	require.NoError(t, err)

	// Process the prompt - should switch to implementation mode
	response, err := handler.ProcessUserPrompt(ctx, json.RawMessage(eventJSON))
	require.NoError(t, err)
	require.Empty(t, response)

	// Verify mode switched
	newState, err := stateManager.GetOperationMode(ctx)
	require.NoError(t, err)
	require.Equal(t, rules.ExecuteMode, newState.Mode)
}
