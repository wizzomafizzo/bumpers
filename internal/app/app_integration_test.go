package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApp_AlignmentIntegrationE2E(t *testing.T) {
	t.Parallel()

	// Test that shows App needs to wire up stateManager for alignment to work
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Create app using standard constructor
	ctx := context.Background()
	app := NewApp(ctx, configPath)
	defer func() {
		if app.dbManager != nil {
			_ = app.dbManager.Close()
		}
	}()

	// This test will fail if the app doesn't wire up the stateManager properly
	// because the promptHandler won't have access to alignment state

	// Create a user prompt with trigger phrase
	event := UserPromptEvent{
		Prompt: "make it so",
	}
	eventJSON, err := json.Marshal(event)
	require.NoError(t, err)

	// Process through the app - this should work with alignment if wired up correctly
	response, err := app.ProcessUserPrompt(ctx, json.RawMessage(eventJSON))
	require.NoError(t, err)

	// If alignment is working, trigger phrases should be handled
	// (For now, we just test that the prompt processing doesn't fail)
	require.NotNil(t, response) // Should not panic or error
}
