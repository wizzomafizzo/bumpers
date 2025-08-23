package ai

import (
	"path/filepath"
	"testing"
)

func TestGeneratorGenerateMessage(t *testing.T) {
	t.Parallel()
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	generator, err := NewGenerator(dbPath)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	defer func() {
		if closeErr := generator.Close(); closeErr != nil {
			t.Logf("Failed to close generator: %v", closeErr)
		}
	}()

	req := &GenerateRequest{
		OriginalMessage: "Use 'just test' instead of 'go test'",
		GenerateMode:    "once",
		Pattern:         "^go test",
	}

	result, err := generator.GenerateMessage(req)
	// We expect either a result or an error about Claude not being available
	if err != nil {
		t.Logf("GenerateMessage failed (expected in test environment): %v", err)
		// Should still return original message as fallback
		if result != req.OriginalMessage {
			t.Errorf("Expected fallback to original message, got %q", result)
		}
		return
	}

	// If successful, result should be non-empty
	if result == "" {
		t.Error("GenerateMessage should return a non-empty result")
	}
}

func TestGeneratorCaching(t *testing.T) {
	t.Parallel()
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	generator, err := NewGenerator(dbPath)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	defer func() {
		if closeErr := generator.Close(); closeErr != nil {
			t.Logf("Failed to close generator: %v", closeErr)
		}
	}()

	req := &GenerateRequest{
		OriginalMessage: "Use 'just test' instead of 'go test'",
		GenerateMode:    "once",
		Pattern:         "^go test",
	}

	// First call should potentially generate (or fallback)
	result1, err1 := generator.GenerateMessage(req)
	if err1 != nil {
		t.Logf("First GenerateMessage failed (expected in test environment): %v", err1)
		// Should still return original message as fallback
		if result1 != req.OriginalMessage {
			t.Errorf("Expected fallback to original message, got %q", result1)
		}
	}

	// Second call with same request should use cache for "once" mode
	result2, err2 := generator.GenerateMessage(req)
	if err2 != nil {
		t.Logf("Second GenerateMessage failed (expected in test environment): %v", err2)
		// Should still return original message as fallback
		if result2 != req.OriginalMessage {
			t.Errorf("Expected fallback to original message, got %q", result2)
		}
	}

	// Results should be the same (cached)
	if result1 != result2 {
		t.Errorf("Expected cached result to match: %q vs %q", result1, result2)
	}
}

func TestGeneratorShouldUseCache(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	generator, err := NewGenerator(dbPath)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	defer func() {
		if closeErr := generator.Close(); closeErr != nil {
			t.Logf("Failed to close generator: %v", closeErr)
		}
	}()

	req := &GenerateRequest{
		OriginalMessage: "Use 'just test' instead of 'go test'",
		GenerateMode:    "once",
		Pattern:         "^go test",
	}

	// The result should NOT be the stub implementation
	result1, err := generator.GenerateMessage(req)
	if err != nil {
		t.Logf("GenerateMessage failed (expected in test environment): %v", err)
		// Should return original message as fallback
		if result1 != req.OriginalMessage {
			t.Errorf("Expected fallback to original message, got %q", result1)
		}
		return // Test passes - we handled Claude failure correctly
	}

	expectedStub := "[AI] " + req.OriginalMessage
	if result1 == expectedStub {
		t.Error("Generator is still using stub implementation, expected actual cache/claude integration")
	}
}

// mockLauncher implements a Claude launcher that returns different results each call
type mockLauncher struct {
	callCount int
}

func (m *mockLauncher) GenerateMessage(_ string) (string, error) {
	m.callCount++
	return "Mock AI response " + string(rune(m.callCount+'A'-1)), nil
}

func TestGeneratorCachingWithMock(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	generator, err := NewGenerator(dbPath)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	defer func() {
		if closeErr := generator.Close(); closeErr != nil {
			t.Logf("Failed to close generator: %v", closeErr)
		}
	}()

	// Replace launcher with mock that returns different results each call
	mock := &mockLauncher{}
	generator.launcher = mock

	req := &GenerateRequest{
		OriginalMessage: "Use 'just test' instead of 'go test'",
		GenerateMode:    "once",
		Pattern:         "^go test",
	}

	// First call should get "Mock AI response A"
	result1, err := generator.GenerateMessage(req)
	if err != nil {
		t.Fatalf("First GenerateMessage failed: %v", err)
	}

	// Second call should return cached result, NOT "Mock AI response B"
	result2, err := generator.GenerateMessage(req)
	if err != nil {
		t.Fatalf("Second GenerateMessage failed: %v", err)
	}

	// If caching works, both results should be identical
	if result1 != result2 {
		t.Errorf("Cache not working: first call %q != second call %q", result1, result2)
	}

	// Mock should only have been called once if caching works
	if mock.callCount != 1 {
		t.Errorf("Expected mock to be called once due to caching, but was called %d times", mock.callCount)
	}
}
