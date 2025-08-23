package ai

import (
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Cache represents the AI message cache
type Cache struct {
	db        *bbolt.DB
	projectID string
}

// getBucketName returns the bucket name for the current project
func (c *Cache) getBucketName() []byte {
	if c.projectID == "" {
		return []byte("default")
	}
	return []byte(c.projectID)
}

// newCacheInstance creates a cache instance with common initialization logic
func newCacheInstance(dbPath, projectID string) (*Cache, error) {
	db, err := bbolt.Open(dbPath, 0o600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	cache := &Cache{db: db, projectID: projectID}

	// Create bucket if it doesn't exist
	err = db.Update(func(tx *bbolt.Tx) error {
		_, bucketErr := tx.CreateBucketIfNotExists(cache.getBucketName())
		if bucketErr != nil {
			return fmt.Errorf("failed to create bucket: %w", bucketErr)
		}
		return nil
	})
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to create bucket and close db: %w, %w", closeErr, err)
		}
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return cache, nil
}

// NewCache creates a new cache instance
func NewCache(dbPath string) (*Cache, error) {
	return newCacheInstance(dbPath, "")
}

// NewCacheWithProject creates a new cache instance with project context
func NewCacheWithProject(dbPath, projectID string) (*Cache, error) {
	return newCacheInstance(dbPath, projectID)
}

// Close closes the cache
func (c *Cache) Close() error {
	return c.db.Close() //nolint:wrapcheck // BBolt close error is self-explanatory
}

// makeKey creates a data-type prefixed key for future extensibility
// This allows the same project bucket to store different types of cached data
func (*Cache) makeKey(key string) string {
	return "ai:" + key
}

// Put stores an entry in the cache
func (c *Cache) Put(key string, entry *CacheEntry) error {
	fullKey := c.makeKey(key)

	err := c.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(c.getBucketName())
		data, marshErr := json.Marshal(entry)
		if marshErr != nil {
			return fmt.Errorf("failed to marshal entry: %w", marshErr)
		}
		return b.Put([]byte(fullKey), data)
	})
	if err != nil {
		return fmt.Errorf("failed to update cache: %w", err)
	}
	return nil
}

// Get retrieves an entry from the cache
func (c *Cache) Get(key string) (*CacheEntry, error) {
	fullKey := c.makeKey(key)

	var entry *CacheEntry
	err := c.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(c.getBucketName())
		data := b.Get([]byte(fullKey))
		if data == nil {
			return nil
		}
		entry = &CacheEntry{}
		return json.Unmarshal(data, entry)
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // JSON unmarshal error is self-explanatory
	}
	return entry, nil
}
