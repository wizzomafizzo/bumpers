package ai

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCacheBasicOperations(t *testing.T) {
	t.Parallel()
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	cache, err := NewCacheWithProject(dbPath, "test-project")
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Failed to close cache: %v", closeErr)
		}
	}()

	key := "test-key"
	entry := &CacheEntry{
		GeneratedMessage: "Generated message",
		OriginalMessage:  "Original message",
		Timestamp:        time.Now(),
		ExpiresAt:        nil,
	}

	// Test Put
	err = cache.Put(key, entry)
	if err != nil {
		t.Fatalf("Failed to put entry: %v", err)
	}

	// Test Get
	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Retrieved entry is nil")
	}
	if retrieved.GeneratedMessage != entry.GeneratedMessage {
		t.Errorf("GeneratedMessage mismatch: got %q, want %q", retrieved.GeneratedMessage, entry.GeneratedMessage)
	}

	// Test Get non-existent key
	nonExistent, err := cache.Get("non-existent")
	if err != nil {
		t.Fatalf("Get non-existent key should not error: %v", err)
	}
	if nonExistent != nil {
		t.Error("Get non-existent key should return nil")
	}
}

func TestCachePersistenceBetweenSessions(t *testing.T) {
	t.Parallel()
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	key := "persistent-key"
	entry := &CacheEntry{
		GeneratedMessage: "Persistent message",
		OriginalMessage:  "Original message",
		Timestamp:        time.Now(),
		ExpiresAt:        nil,
	}

	// First session: create cache and store entry
	cache1, err := NewCacheWithProject(dbPath, "test-project")
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	err = cache1.Put(key, entry)
	if err != nil {
		t.Fatalf("Failed to put entry: %v", err)
	}

	if closeErr := cache1.Close(); closeErr != nil {
		t.Fatalf("Failed to close cache1: %v", closeErr)
	}

	// Second session: reopen cache and retrieve entry
	cache2, err := NewCacheWithProject(dbPath, "test-project")
	if err != nil {
		t.Fatalf("Failed to reopen cache: %v", err)
	}
	defer func() {
		if closeErr := cache2.Close(); closeErr != nil {
			t.Logf("Failed to close cache2: %v", closeErr)
		}
	}()

	retrieved, err := cache2.Get(key)
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Entry should persist between sessions")
	}
	if retrieved.GeneratedMessage != entry.GeneratedMessage {
		t.Errorf("GeneratedMessage mismatch: got %q, want %q", retrieved.GeneratedMessage, entry.GeneratedMessage)
	}
}

func TestCacheWithProjectContext(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create cache with project context
	projectID := "test-a1b2"
	cache, err := NewCacheWithProject(dbPath, projectID)
	if err != nil {
		t.Fatalf("Failed to create cache with project: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Failed to close cache: %v", closeErr)
		}
	}()

	// Store entry - should be prefixed internally
	key := "test-key"
	entry := &CacheEntry{
		GeneratedMessage: "Project-specific message",
		OriginalMessage:  "Original message",
		Timestamp:        time.Now(),
	}

	err = cache.Put(key, entry)
	if err != nil {
		t.Fatalf("Failed to put entry: %v", err)
	}

	// Retrieve entry
	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved entry is nil")
	}

	if retrieved.GeneratedMessage != entry.GeneratedMessage {
		t.Errorf("GeneratedMessage mismatch: got %q, want %q", retrieved.GeneratedMessage, entry.GeneratedMessage)
	}
}

func TestCacheClearSessionCache(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	cache, err := NewCacheWithProject(dbPath, "test-project")
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Failed to close cache: %v", closeErr)
		}
	}()

	// Setup test data
	setupClearSessionCacheTest(t, cache)

	// Clear session cache
	err = cache.ClearSessionCache()
	if err != nil {
		t.Fatalf("Failed to clear session cache: %v", err)
	}

	// Verify results
	verifyClearSessionCacheResults(t, cache)
}

func setupClearSessionCacheTest(t *testing.T, cache *Cache) {
	t.Helper()

	now := time.Now()
	sessionExpiry := now.Add(24 * time.Hour)

	entries := map[string]*CacheEntry{
		"session-key-1": {
			GeneratedMessage: "Session message 1",
			OriginalMessage:  "Original 1",
			Timestamp:        now,
			ExpiresAt:        &sessionExpiry,
		},
		"session-key-2": {
			GeneratedMessage: "Session message 2",
			OriginalMessage:  "Original 2",
			Timestamp:        now,
			ExpiresAt:        &sessionExpiry,
		},
		"once-key": {
			GeneratedMessage: "Once message",
			OriginalMessage:  "Original once",
			Timestamp:        now,
			ExpiresAt:        nil,
		},
	}

	for key, entry := range entries {
		err := cache.Put(key, entry)
		if err != nil {
			t.Fatalf("Failed to put %s: %v", key, err)
		}
	}

	// Verify all entries exist
	for key := range entries {
		retrieved, err := cache.Get(key)
		if err != nil || retrieved == nil {
			t.Fatalf("%s should exist before clearing", key)
		}
	}
}

func verifyClearSessionCacheResults(t *testing.T, cache *Cache) {
	t.Helper()

	// Verify session entries are gone
	sessionKeys := []string{"session-key-1", "session-key-2"}
	for _, key := range sessionKeys {
		retrieved, err := cache.Get(key)
		if err != nil {
			t.Fatalf("Unexpected error getting %s: %v", key, err)
		}
		if retrieved != nil {
			t.Errorf("%s should be cleared", key)
		}
	}

	// Verify once entry still exists
	retrieved, err := cache.Get("once-key")
	if err != nil {
		t.Fatalf("Unexpected error getting once key: %v", err)
	}
	if retrieved == nil {
		t.Error("Once entry should still exist after clearing session cache")
	}
}

func TestCacheShouldNotStoreGenerateMode(t *testing.T) {
	t.Parallel()
	// This test demonstrates that cache entries should not store generate mode
	// Cache invalidation should be based on live config, not stored mode
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	cache, err := NewCacheWithProject(dbPath, "test-project")
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Failed to close cache: %v", closeErr)
		}
	}()

	// Create an entry that was cached when mode was "once"
	// GenerateMode should not be stored in cache - removed from struct
	entry := &CacheEntry{
		GeneratedMessage: "Test message",
		OriginalMessage:  "Original message",
		Timestamp:        time.Now(),
	}

	// Store the entry
	key := "test-key"
	err = cache.Put(key, entry)
	if err != nil {
		t.Fatalf("Failed to put entry: %v", err)
	}

	// Retrieve and verify GenerateMode is not stored
	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Retrieved entry is nil")
	}

	// Verify that cache entry was stored and retrieved successfully without GenerateMode
	if retrieved.GeneratedMessage != entry.GeneratedMessage {
		t.Errorf("GeneratedMessage mismatch: got %q, want %q", retrieved.GeneratedMessage, entry.GeneratedMessage)
	}
}

func TestCacheClearByCurrentMode(t *testing.T) {
	t.Parallel()
	// Test that cache clearing uses current config mode, not stored mode
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	cache, err := NewCacheWithProject(dbPath, "test-project")
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Failed to close cache: %v", closeErr)
		}
	}()

	// Add some test entries (without GenerateMode since we removed it)
	now := time.Now()
	sessionExpiry := now.Add(24 * time.Hour)

	entries := map[string]*CacheEntry{
		"entry1": {
			GeneratedMessage: "Add 1",
			OriginalMessage:  "Original 1",
			Timestamp:        now,
			ExpiresAt:        &sessionExpiry, // Session-like expiry
		},
		"entry2": {
			GeneratedMessage: "Add 2",
			OriginalMessage:  "Original 2",
			Timestamp:        now,
			ExpiresAt:        nil, // No expiry (once-like)
		},
	}

	// Store entries
	for key, entry := range entries {
		if putErr := cache.Put(key, entry); putErr != nil {
			t.Fatalf("Failed to put %s: %v", key, putErr)
		}
	}

	// Test clearing with current mode "session" - should clear session entries
	// Currently using old method signature, but we want to pass currentMode parameter
	err = cache.ClearSessionCache()
	if err != nil {
		t.Fatalf("Failed to clear cache with current mode: %v", err)
	}

	// Verify appropriate entries were cleared based on current mode
	entry1, _ := cache.Get("entry1")
	entry2, _ := cache.Get("entry2")

	if entry1 != nil {
		t.Error("Entry with session-like expiry should be cleared when current mode is session")
	}
	if entry2 == nil {
		t.Error("Entry without expiry should remain when current mode is session")
	}
}
