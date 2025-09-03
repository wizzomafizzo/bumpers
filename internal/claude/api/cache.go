package ai

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wizzomafizzo/bumpers/internal/database"
	_ "modernc.org/sqlite"
)

// Cache represents the AI message cache
type Cache struct {
	db        *sql.DB
	manager   *database.Manager
	storage   map[string]*CacheEntry
	projectID string
}

// newCacheInstance creates a cache instance with common initialization logic
func newCacheInstance(ctx context.Context, dbPath, projectID string) (*Cache, error) {
	// Create database manager
	manager, err := database.NewManager(ctx, dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database manager: %w", err)
	}

	cache := &Cache{
		db:        manager.DB(),
		projectID: projectID,
		manager:   manager,
		storage:   make(map[string]*CacheEntry),
	}

	return cache, nil
}

// NewCache creates a new cache instance
func NewCache(ctx context.Context, dbPath string) (*Cache, error) {
	return newCacheInstance(ctx, dbPath, "")
}

// NewCacheWithProject creates a new cache instance with project context
func NewCacheWithProject(ctx context.Context, dbPath, projectID string) (*Cache, error) {
	return newCacheInstance(ctx, dbPath, projectID)
}

// NewCacheWithDB creates a new cache instance with a database connection
func NewCacheWithDB(_ any, _ string) (*Cache, error) {
	return nil, nil //nolint:nilnil // Stub function not implemented
}

// Close closes the cache
func (c *Cache) Close() error {
	if err := c.db.Close(); err != nil {
		return fmt.Errorf("failed to close cache database: %w", err)
	}
	return nil
}

// Put stores an entry in the cache
func (c *Cache) Put(ctx context.Context, key string, entry *CacheEntry) error {
	var expiresAt *int64
	if entry.ExpiresAt != nil {
		timestamp := entry.ExpiresAt.Unix()
		expiresAt = &timestamp
	}

	_, err := c.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO cache (key, project_id, value, expires_at) VALUES (?, ?, ?, ?)",
		key, c.projectID, entry.GeneratedMessage, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to store cache entry for key %q: %w", key, err)
	}
	return nil
}

// Get retrieves an entry from the cache
func (c *Cache) Get(ctx context.Context, key string) (*CacheEntry, error) {
	// Try memory first
	entry, exists := c.storage[key]
	if exists {
		return entry, nil
	}

	// Try database
	var message string
	var expiresAt *int64
	err := c.db.QueryRowContext(ctx, "SELECT value, expires_at FROM cache WHERE key = ? AND project_id = ?",
		key, c.projectID).Scan(&message, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil //nolint:nilnil // Cache miss returns nil value and nil error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cache entry for key %q: %w", key, err)
	}

	entry = &CacheEntry{GeneratedMessage: message}
	if expiresAt != nil {
		timestamp := time.Unix(*expiresAt, 0)
		entry.ExpiresAt = &timestamp
	}

	return entry, nil
}

// ClearSessionCache clears all cached entries with "session" generate mode
func (c *Cache) ClearSessionCache(ctx context.Context) error {
	// Clear in-memory storage - but only session entries
	// Session entries have ExpiresAt set, "once" entries have ExpiresAt as nil
	for key, entry := range c.storage {
		if entry.ExpiresAt != nil {
			delete(c.storage, key)
		}
	}

	// Delete session cache entries from database - entries with expires_at NOT NULL
	// Session entries have expires_at set, "once" entries have expires_at as NULL
	_, err := c.db.ExecContext(ctx,
		"DELETE FROM cache WHERE project_id = ? AND expires_at IS NOT NULL",
		c.projectID)
	if err != nil {
		return fmt.Errorf("failed to clear session cache from database: %w", err)
	}

	return nil
}

// NewSQLCache creates a new cache instance with a SQL database connection
func NewSQLCache(db *sql.DB, projectID string) (*Cache, error) {
	return &Cache{
		db:        db,
		projectID: projectID,
		storage:   make(map[string]*CacheEntry),
	}, nil
}
