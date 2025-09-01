package cli

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/core/engine/hooks"
	"github.com/wizzomafizzo/bumpers/internal/core/engine/operation"
	"github.com/wizzomafizzo/bumpers/internal/platform/state"
)

func TestProcessPreToolUse_OperationBlocking(t *testing.T) {
	t.Parallel()

	// Create test processor with mock state manager
	processor, stateManager := createTestProcessor(t)

	// Set operation to plan mode (blocking mode)
	ctx := context.Background()
	discussionState := &operation.OperationState{
		Mode:         operation.PlanMode,
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
func createTestProcessor(t *testing.T) (*DefaultHookProcessor, *state.Manager) {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	stateManager, err := state.NewManager(dbPath, "test-project")
	require.NoError(t, err)

	t.Cleanup(func() { _ = stateManager.Close() })

	processor := &DefaultHookProcessor{
		stateManager: stateManager,
	}

	return processor, stateManager
}
