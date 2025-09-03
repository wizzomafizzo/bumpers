package database

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"
)

// Test constants
const (
	expectedVersion = 1
)

var (
	expectedTables  = []string{"cache", "state"}
	expectedIndexes = []string{
		"idx_cache_project",
		"idx_cache_expires",
		"idx_state_project",
	}
)

// Test helper factory functions
func createTestManager(t *testing.T) (*Manager, *sql.DB, func()) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	manager := &Manager{db: db}
	cleanup := func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}

	return manager, db, cleanup
}

func setDatabaseVersion(t *testing.T, db *sql.DB, version int) {
	t.Helper()

	query := fmt.Sprintf("PRAGMA user_version = %d", version)
	_, err := db.ExecContext(context.Background(), query)
	if err != nil {
		t.Fatalf("Failed to set database version: %v", err)
	}
}

// Test assertion helpers
func assertTablesExist(t *testing.T, db *sql.DB, tables []string) {
	t.Helper()

	for _, tableName := range tables {
		exists := checkTableExists(t, db, tableName)
		if !exists {
			t.Errorf("Expected table '%s' to exist, but it doesn't", tableName)
		}
	}
}

func assertIndexesExist(t *testing.T, db *sql.DB, indexes []string) {
	t.Helper()

	for _, indexName := range indexes {
		exists := checkIndexExists(t, db, indexName)
		if !exists {
			t.Errorf("Expected index '%s' to exist, but it doesn't", indexName)
		}
	}
}

func assertDatabaseVersion(t *testing.T, db *sql.DB, expectedVersion int) {
	t.Helper()

	var version int
	err := db.QueryRowContext(context.Background(), "PRAGMA user_version").Scan(&version)
	if err != nil {
		t.Fatalf("Failed to get database version: %v", err)
	}
	if version != expectedVersion {
		t.Errorf("Expected database version %d, got %d", expectedVersion, version)
	}
}

// Test cases
func TestRunMigrations_FreshDatabase(t *testing.T) {
	t.Parallel()
	manager, db, cleanup := createTestManager(t)
	defer cleanup()

	ctx := context.Background()
	err := manager.runMigrations(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	assertTablesExist(t, db, expectedTables)
	assertIndexesExist(t, db, expectedIndexes)
	assertDatabaseVersion(t, db, expectedVersion)
}

func TestRunMigrations_SkipWhenAtCurrentVersion(t *testing.T) {
	t.Parallel()
	manager, db, cleanup := createTestManager(t)
	defer cleanup()

	// Set database to current version (should skip migrations)
	setDatabaseVersion(t, db, expectedVersion)

	ctx := context.Background()
	err := manager.runMigrations(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Verify tables were NOT created since migrations were skipped
	for _, tableName := range expectedTables {
		exists := checkTableExists(t, db, tableName)
		if exists {
			t.Errorf("Expected table '%s' to NOT exist since migrations should be skipped, but it does", tableName)
		}
	}

	// Verify version remains unchanged
	assertDatabaseVersion(t, db, expectedVersion)
}

func TestExecuteMigration_ValidMigration(t *testing.T) {
	t.Parallel()
	manager, db, cleanup := createTestManager(t)
	defer cleanup()

	testMigration := migration{
		version: 2,
		sql:     "CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)",
	}

	ctx := context.Background()
	err := manager.executeMigration(ctx, testMigration)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Verify table was created
	exists := checkTableExists(t, db, "test_table")
	if !exists {
		t.Error("Expected test_table to exist after migration")
	}

	// Verify version was updated
	assertDatabaseVersion(t, db, 2)
}

// Database schema check helpers
func checkTableExists(t *testing.T, db *sql.DB, tableName string) bool {
	t.Helper()

	var count int
	query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
	err := db.QueryRowContext(context.Background(), query, tableName).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}
	return count > 0
}

func checkIndexExists(t *testing.T, db *sql.DB, indexName string) bool {
	t.Helper()

	var count int
	query := "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?"
	err := db.QueryRowContext(context.Background(), query, indexName).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check index existence: %v", err)
	}
	return count > 0
}
