package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/rules"
)

func TestNewManager(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	manager, err := NewStateManager(dbPath, "test-project")
	require.NoError(t, err)
	require.NotNil(t, manager)

	err = manager.Close()
	require.NoError(t, err)
}

func TestGetRulesEnabled_DefaultTrue(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	manager := createTestManager(t)

	enabled, err := manager.GetRulesEnabled(ctx)
	require.NoError(t, err)
	require.True(t, enabled)
}

// createTestManager creates a state manager for testing
func createTestManager(t *testing.T) *StateManager {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	manager, err := NewStateManager(dbPath, "test-project")
	require.NoError(t, err)

	t.Cleanup(func() { _ = manager.Close() })

	return manager
}

func TestCloseActuallyClosesDatabase(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	manager, err := NewStateManager(dbPath, "test-project")
	require.NoError(t, err)

	// Close the manager
	err = manager.Close()
	require.NoError(t, err)

	// Try to create another manager with the same database path
	// This should work if the first one properly closed the database
	manager2, err := NewStateManager(dbPath, "test-project")
	require.NoError(t, err)
	require.NotNil(t, manager2)

	err = manager2.Close()
	require.NoError(t, err)
}

func TestSetRulesEnabled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	manager := createTestManager(t)

	testBooleanToggle(ctx, t, [2]bool{false, true},
		manager.SetRulesEnabled, manager.GetRulesEnabled)
}

func TestGetSkipNext_DefaultFalse(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	manager := createTestManager(t)

	skip, err := manager.GetSkipNext(ctx)
	require.NoError(t, err)
	require.False(t, skip)
}

func TestSetSkipNext(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	manager := createTestManager(t)

	testBooleanToggle(ctx, t, [2]bool{true, false},
		manager.SetSkipNext, manager.GetSkipNext)
}

// testBooleanToggle tests setting a boolean value to firstValue, then to secondValue
func testBooleanToggle(ctx context.Context, t *testing.T,
	values [2]bool,
	setter func(context.Context, bool) error,
	getter func(context.Context) (bool, error),
) {
	t.Helper()

	// Set to first value
	err := setter(ctx, values[0])
	require.NoError(t, err)

	// Verify it was set
	actual, err := getter(ctx)
	require.NoError(t, err)
	require.Equal(t, values[0], actual)

	// Set to second value
	err = setter(ctx, values[1])
	require.NoError(t, err)

	// Verify it was set
	actual, err = getter(ctx)
	require.NoError(t, err)
	require.Equal(t, values[1], actual)
}

func TestConsumeSkipNext(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	manager := createTestManager(t)

	// Initially false, consuming should return false and leave it false
	consumed, err := manager.ConsumeSkipNext(ctx)
	require.NoError(t, err)
	require.False(t, consumed)

	// Verify it's still false
	skip, err := manager.GetSkipNext(ctx)
	require.NoError(t, err)
	require.False(t, skip)

	// Set to true
	err = manager.SetSkipNext(ctx, true)
	require.NoError(t, err)

	// Consuming should return true and reset to false
	consumed, err = manager.ConsumeSkipNext(ctx)
	require.NoError(t, err)
	require.True(t, consumed)

	// Verify it was reset to false
	skip, err = manager.GetSkipNext(ctx)
	require.NoError(t, err)
	require.False(t, skip)

	// Consuming again should return false
	consumed, err = manager.ConsumeSkipNext(ctx)
	require.NoError(t, err)
	require.False(t, consumed)
}

func TestGetOperationMode_DefaultExecute(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	manager := createTestManager(t)

	state, err := manager.GetOperationMode(ctx)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Equal(t, rules.ExecuteMode, state.Mode)
	require.Equal(t, 0, state.TriggerCount)
}

func TestSetOperationMode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	manager := createTestManager(t)

	// Set to execute mode
	newState := &rules.OperationState{
		Mode:         rules.ExecuteMode,
		TriggerCount: 1,
		UpdatedAt:    123456789,
	}

	err := manager.SetOperationMode(ctx, newState)
	require.NoError(t, err)

	// Verify it was stored
	stored, err := manager.GetOperationMode(ctx)
	require.NoError(t, err)
	require.Equal(t, rules.ExecuteMode, stored.Mode)
	require.Equal(t, 1, stored.TriggerCount)
	require.Equal(t, int64(123456789), stored.UpdatedAt)
}
