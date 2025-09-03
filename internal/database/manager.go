package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Manager struct {
	db *sql.DB
}

func NewManager(ctx context.Context, dsn string) (*Manager, error) {
	db, err := sql.Open("sqlite", dsn)
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

	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to execute pragma %s: %w", pragma, err)
		}
	}

	// Configure connection pool for multi-process scenarios
	db.SetMaxOpenConns(10) // Consider total across all processes
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	manager := &Manager{db: db}

	// Run migrations
	if err := manager.runMigrations(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return manager, nil
}

func (m *Manager) DB() *sql.DB {
	return m.db
}

func (m *Manager) Close() error {
	if m.db != nil {
		if err := m.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}
	return nil
}
