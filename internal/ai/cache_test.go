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
		GenerateMode:     "once",
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
		GenerateMode:     "once",
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
		GenerateMode:     "once",
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
