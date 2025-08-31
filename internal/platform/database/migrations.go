package database

import (
	"context"
	"fmt"
)

type migration struct {
	sql     string
	version int
}

var migrations = []migration{
	{
		version: 1,
		sql: `
			CREATE TABLE cache (
				key TEXT PRIMARY KEY,
				project_id TEXT NOT NULL,
				value BLOB NOT NULL,
				expires_at INTEGER,
				created_at INTEGER NOT NULL DEFAULT (unixepoch())
			);

			CREATE TABLE state (
				key TEXT PRIMARY KEY,
				project_id TEXT NOT NULL,
				value BLOB NOT NULL,
				updated_at INTEGER NOT NULL DEFAULT (unixepoch())
			);

			CREATE INDEX idx_cache_project ON cache(project_id);
			CREATE INDEX idx_cache_expires ON cache(expires_at);
			CREATE INDEX idx_state_project ON state(project_id);
		`,
	},
}

func (m *Manager) runMigrations(ctx context.Context) error {
	// Get current version
	var currentVersion int
	err := m.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get current database version: %w", err)
	}

	// Run migrations
	for _, migration := range migrations {
		if migration.version <= currentVersion {
			continue
		}
		if err := m.executeMigration(ctx, migration); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) executeMigration(ctx context.Context, migration migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if _, err := tx.ExecContext(ctx, migration.sql); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to execute migration %d: %w", migration.version, err)
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", migration.version)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to update database version to %d: %w", migration.version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration %d: %w", migration.version, err)
	}
	return nil
}
