package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	manager, err := NewManager(ctx, ":memory:")

	require.NoError(t, err)
	require.NotNil(t, manager)
	require.NotNil(t, manager.DB())

	// Cleanup
	err = manager.Close()
	assert.NoError(t, err)
}

func TestWALModeEnabled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use file database since :memory: doesn't support WAL
	tempFile := t.TempDir() + "/test.db"
	manager, err := NewManager(ctx, tempFile)
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer func() { _ = manager.Close() }()

	// Verify WAL mode is enabled
	var journalMode string
	err = manager.DB().QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode)
	require.NoError(t, err)
	assert.Equal(t, "wal", journalMode)
}

func TestMigrationsExecuted(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	manager, err := NewManager(ctx, ":memory:")
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer func() { _ = manager.Close() }()

	db := manager.DB()

	// Check that cache table exists
	rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='cache'")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	assert.True(t, rows.Next(), "cache table should exist")
	require.NoError(t, rows.Err())
}

func TestMigrationVersion(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	manager, err := NewManager(ctx, ":memory:")
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer func() { _ = manager.Close() }()

	db := manager.DB()

	// Check user_version was set to 1
	var version int
	err = db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	require.NoError(t, err)
	assert.Equal(t, 1, version)
}
