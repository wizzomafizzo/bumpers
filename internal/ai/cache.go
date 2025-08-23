package ai

import (
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

var bucketName = []byte("ai_cache")

// Cache represents the AI message cache
type Cache struct {
	db *bbolt.DB
}

// NewCache creates a new cache instance
func NewCache(dbPath string) (*Cache, error) {
	db, err := bbolt.Open(dbPath, 0o600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create bucket if it doesn't exist
	err = db.Update(func(tx *bbolt.Tx) error {
		_, bucketErr := tx.CreateBucketIfNotExists(bucketName)
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

	return &Cache{db: db}, nil
}

// Close closes the cache
func (c *Cache) Close() error {
	return c.db.Close() //nolint:wrapcheck // BBolt close error is self-explanatory
}

// Put stores an entry in the cache
func (c *Cache) Put(key string, entry *CacheEntry) error {
	err := c.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		data, marshErr := json.Marshal(entry)
		if marshErr != nil {
			return fmt.Errorf("failed to marshal entry: %w", marshErr)
		}
		return b.Put([]byte(key), data)
	})
	if err != nil {
		return fmt.Errorf("failed to update cache: %w", err)
	}
	return nil
}

// Get retrieves an entry from the cache
func (c *Cache) Get(key string) (*CacheEntry, error) {
	var entry *CacheEntry
	err := c.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		data := b.Get([]byte(key))
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
