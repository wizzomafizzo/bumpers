package ai

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestNewSQLCache(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create tables manually for this test
	_, err = db.ExecContext(ctx, `
		CREATE TABLE cache (
			key TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			value BLOB NOT NULL,
			expires_at INTEGER,
			created_at INTEGER NOT NULL DEFAULT (unixepoch())
		)
	`)
	require.NoError(t, err)

	cache, err := NewSQLCache(db, "test-project")
	require.NoError(t, err)
	require.NotNil(t, cache)
}
