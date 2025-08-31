package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/platform/database"
)

func createSQLTestManager(t *testing.T) (manager *Manager, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	dbManager, err := database.NewManager(ctx, ":memory:")
	require.NoError(t, err)

	manager, err = NewSQLManager(dbManager.DB(), "test-project")
	require.NoError(t, err)

	cleanup = func() {
		_ = manager.Close()
		_ = dbManager.Close()
	}
	return
}

func TestNewSQLManager(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Create in-memory database for testing
	dbManager, err := database.NewManager(ctx, ":memory:")
	require.NoError(t, err)
	defer func() { _ = dbManager.Close() }()

	manager, err := NewSQLManager(dbManager.DB(), "test-project")
	require.NoError(t, err)
	require.NotNil(t, manager)

	err = manager.Close()
	require.NoError(t, err)
}

func TestSQLManager_GetRulesEnabled_DefaultTrue(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	manager, cleanup := createSQLTestManager(t)
	defer cleanup()

	enabled, err := manager.GetRulesEnabled(ctx)
	require.NoError(t, err)
	require.True(t, enabled)
}

func TestSQLManager_SetRulesEnabled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	manager, cleanup := createSQLTestManager(t)
	defer cleanup()

	// Set to false
	err := manager.SetRulesEnabled(ctx, false)
	require.NoError(t, err)

	// Verify it was set
	enabled, err := manager.GetRulesEnabled(ctx)
	require.NoError(t, err)
	require.False(t, enabled)
}

func TestSQLManager_GetSkipNext_DefaultFalse(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	manager, cleanup := createSQLTestManager(t)
	defer cleanup()

	skip, err := manager.GetSkipNext(ctx)
	require.NoError(t, err)
	require.False(t, skip)
}

func TestSQLManager_SetSkipNext(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	manager, cleanup := createSQLTestManager(t)
	defer cleanup()

	// Set to true
	err := manager.SetSkipNext(ctx, true)
	require.NoError(t, err)

	// Verify it was set
	skip, err := manager.GetSkipNext(ctx)
	require.NoError(t, err)
	require.True(t, skip)
}

func TestSQLManager_ConsumeSkipNext(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	manager, cleanup := createSQLTestManager(t)
	defer cleanup()

	// Initially false, consuming should return false and leave it false
	consumed, err := manager.ConsumeSkipNext(ctx)
	require.NoError(t, err)
	require.False(t, consumed)

	// Set to true
	err = manager.SetSkipNext(ctx, true)
	require.NoError(t, err)

	// Consuming should return true and reset to false
	consumed, err = manager.ConsumeSkipNext(ctx)
	require.NoError(t, err)
	require.True(t, consumed)

	// Verify it was reset to false
	skip, err := manager.GetSkipNext(ctx)
	require.NoError(t, err)
	require.False(t, skip)
}
