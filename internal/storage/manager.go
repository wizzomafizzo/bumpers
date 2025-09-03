package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/wizzomafizzo/bumpers/internal/rules"
	_ "modernc.org/sqlite"
)

// StateManager handles persistent state storage for Bumpers configuration using SQLite only
type StateManager struct {
	db        *sql.DB
	operation *rules.OperationState
	projectID string
}

// NewManager creates a new state manager instance using SQLite
func NewStateManager(dbPath, projectID string) (*StateManager, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure WAL mode and other pragmas
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA temp_store = MEMORY",
	}

	ctx := context.Background()
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to set pragma: %w", err)
		}
	}

	// Run schema migration if needed
	if err := runSchemaMigration(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run schema migration: %w", err)
	}

	return &StateManager{db: db, projectID: projectID}, nil
}

// runSchemaMigration ensures the state table exists
func runSchemaMigration(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS state (
			key TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			value BLOB NOT NULL,
			updated_at INTEGER NOT NULL DEFAULT (unixepoch())
		);
		CREATE INDEX IF NOT EXISTS idx_state_project ON state(project_id);
	`)
	if err != nil {
		return fmt.Errorf("failed to create state table: %w", err)
	}
	return nil
}

// Close closes the state manager
func (m *StateManager) Close() error {
	if m.db != nil {
		if err := m.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}
	return nil
}

// getBoolValue retrieves a boolean value from the state table
func (m *StateManager) getBoolValue(
	ctx context.Context, key string, defaultValue bool, errorPrefix string,
) (bool, error) {
	var valueJSON []byte
	err := m.db.QueryRowContext(ctx,
		"SELECT value FROM state WHERE key = ? AND project_id = ?",
		key, m.projectID).Scan(&valueJSON)

	if err == sql.ErrNoRows {
		return defaultValue, nil
	}
	if err != nil {
		return defaultValue, fmt.Errorf("%s: %w", errorPrefix, err)
	}

	var value bool
	if err := json.Unmarshal(valueJSON, &value); err != nil {
		return defaultValue, fmt.Errorf("%s: %w", errorPrefix, err)
	}

	return value, nil
}

// GetRulesEnabled returns whether rules processing is enabled
func (m *StateManager) GetRulesEnabled(ctx context.Context) (bool, error) {
	return m.getBoolValue(ctx, "state:rules_enabled", true, "failed to get rules enabled state")
}

// SetRulesEnabled sets whether rules processing is enabled
func (m *StateManager) SetRulesEnabled(ctx context.Context, enabled bool) error {
	key := "state:rules_enabled"

	data, err := json.Marshal(enabled)
	if err != nil {
		return fmt.Errorf("failed to marshal enabled state: %w", err)
	}

	_, err = m.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO state (key, project_id, value) VALUES (?, ?, ?)",
		key, m.projectID, data)
	if err != nil {
		return fmt.Errorf("failed to set rules enabled state: %w", err)
	}

	return nil
}

// GetSkipNext returns whether the next rule-processing hook should be skipped
func (m *StateManager) GetSkipNext(ctx context.Context) (bool, error) {
	return m.getBoolValue(ctx, "state:skip_next_rule_hook", false, "failed to get skip next state")
}

// SetSkipNext sets whether the next rule-processing hook should be skipped
func (m *StateManager) SetSkipNext(ctx context.Context, skip bool) error {
	key := "state:skip_next_rule_hook"

	data, err := json.Marshal(skip)
	if err != nil {
		return fmt.Errorf("failed to marshal skip state: %w", err)
	}

	_, err = m.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO state (key, project_id, value) VALUES (?, ?, ?)",
		key, m.projectID, data)
	if err != nil {
		return fmt.Errorf("failed to set skip next state: %w", err)
	}

	return nil
}

// ConsumeSkipNext returns the current skip flag value and resets it to false
func (m *StateManager) ConsumeSkipNext(ctx context.Context) (bool, error) {
	value, err := m.GetSkipNext(ctx)
	if err != nil {
		return false, err
	}

	if value {
		err = m.SetSkipNext(ctx, false)
		if err != nil {
			return false, fmt.Errorf("failed to reset skip flag: %w", err)
		}
	}

	return value, nil
}

// GetOperationMode returns the current operation state
func (m *StateManager) GetOperationMode(_ context.Context) (*rules.OperationState, error) {
	if m.operation == nil {
		return rules.DefaultState(), nil
	}
	return m.operation, nil
}

// SetOperationMode sets the operation state
func (m *StateManager) SetOperationMode(_ context.Context, state *rules.OperationState) error {
	m.operation = state
	return nil
}

// NewSQLManager creates a new SQL-based state manager instance
func NewSQLManager(db *sql.DB, projectID string) (*StateManager, error) {
	return &StateManager{
		db:        db,
		projectID: projectID,
	}, nil
}
