package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apphooks "github.com/wizzomafizzo/bumpers/internal/app/hooks"
	"github.com/wizzomafizzo/bumpers/internal/hooks"
	"github.com/wizzomafizzo/bumpers/internal/rules"
	"github.com/wizzomafizzo/bumpers/internal/storage"
)

func TestProcessPreToolUse_OperationBlocking(t *testing.T) {
	t.Parallel()

	// Create test processor with mock state manager
	processor, stateManager := createTestProcessor(t)

	// Set operation to plan mode (blocking mode)
	ctx := context.Background()
	discussionState := &rules.OperationState{
		Mode:         rules.PlanMode,
		TriggerCount: 0,
		UpdatedAt:    123456789,
	}
	err := stateManager.SetOperationMode(ctx, discussionState)
	require.NoError(t, err)

	// Create a hook event for Edit tool (should be blocked)
	event := hooks.HookEvent{
		ToolName: "Edit",
		ToolInput: map[string]any{
			"file_path":  "/test/file.txt",
			"old_string": "old content",
			"new_string": "new content",
		},
	}

	eventJSON, err := json.Marshal(event)
	require.NoError(t, err)

	// Process the hook
	response, err := processor.ProcessPreToolUse(ctx, json.RawMessage(eventJSON))
	require.NoError(t, err)
	require.NotEmpty(t, response)

	// Should contain blocking message about plan mode
	require.Contains(t, strings.ToLower(response), "plan mode")
}

// createTestProcessor creates a minimal test processor
func createTestProcessor(t *testing.T) (*apphooks.DefaultHookProcessor, *storage.StateManager) {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	stateManager, err := storage.NewStateManager(dbPath, "test-project")
	require.NoError(t, err)

	t.Cleanup(func() { _ = stateManager.Close() })

	// Create a minimal config validator for testing
	configValidator := NewConfigValidator("", tempDir)
	processor := apphooks.NewHookProcessor(configValidator, tempDir, stateManager)

	return processor, stateManager
}
