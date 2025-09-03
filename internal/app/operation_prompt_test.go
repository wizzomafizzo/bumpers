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

func TestProcessUserPrompt_TriggerPhraseDetection(t *testing.T) {
	t.Parallel()

	// Create test prompt handler with state manager
	handler, stateManager := createTestPromptHandler(t)

	// Set operation to plan mode
	ctx := context.Background()
	discussionState := &rules.OperationState{
		Mode:         rules.PlanMode,
		TriggerCount: 0,
		UpdatedAt:    123456789,
	}
	err := stateManager.SetOperationMode(ctx, discussionState)
	require.NoError(t, err)

	// Create a user prompt with trigger phrase
	event := UserPromptEvent{
		Prompt: "make it so",
	}

	eventJSON, err := json.Marshal(event)
	require.NoError(t, err)

	// Process the prompt
	response, err := handler.ProcessUserPrompt(ctx, json.RawMessage(eventJSON))
	require.NoError(t, err)

	// Should be empty response (allowing through) since we switched to execute mode
	require.Empty(t, response)

	// Verify operation mode changed to execute
	newState, err := stateManager.GetOperationMode(ctx)
	require.NoError(t, err)
	require.Equal(t, rules.ExecuteMode, newState.Mode)
	require.Equal(t, 1, newState.TriggerCount)
}

// createTestPromptHandler creates a test prompt handler with state manager
func createTestPromptHandler(t *testing.T) (*DefaultPromptHandler, *storage.StateManager) {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	stateManager, err := storage.NewStateManager(dbPath, "test-project")
	require.NoError(t, err)

	t.Cleanup(func() { _ = stateManager.Close() })

	handler := &DefaultPromptHandler{
		configPath:   filepath.Join(tempDir, "bumpers.yml"),
		projectRoot:  tempDir,
		aiHelper:     NewAIHelper(AIHelperOptions{ProjectRoot: tempDir}),
		stateManager: stateManager,
	}

	return handler, stateManager
}
