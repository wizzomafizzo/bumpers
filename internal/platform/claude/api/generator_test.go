package ai

import (
	"path/filepath"
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/platform/claude"
	"github.com/wizzomafizzo/bumpers/internal/testing"
)

// setupTest initializes test logger to prevent race conditions
func setupTest(t *testing.T) {
	t.Helper()
	testutil.InitTestLogger(t)
}

func TestGeneratorGenerateMessage(t *testing.T) {
	t.Parallel()
	setupTest(t)
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	generator, err := NewGenerator(dbPath, "test-project")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := generator.Close(); closeErr != nil {
			t.Logf("Failed to close generator: %v", closeErr)
		}
	})

	// Replace launcher with mock to avoid slow Claude CLI discovery
	mock := claude.SetupMockLauncherWithDefaults()
	mock.SetResponseForPattern(".*", "AI enhanced test response")
	generator.launcher = mock

	req := &GenerateRequest{
		OriginalMessage: "Use 'just test' instead of 'go test'",
		GenerateMode:    "once",
		Pattern:         "^go test",
	}

	result, err := generator.GenerateMessage(req)
	if err != nil {
		t.Fatalf("GenerateMessage failed: %v", err)
	}

	// Should return the mocked response
	expectedResponse := "AI enhanced test response"
	if result != expectedResponse {
		t.Errorf("Expected mocked response %q, got %q", expectedResponse, result)
	}
}

func TestGeneratorCaching(t *testing.T) {
	t.Parallel()
	setupTest(t)
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	generator, err := NewGenerator(dbPath, "test-project")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := generator.Close(); closeErr != nil {
			t.Logf("Failed to close generator: %v", closeErr)
		}
	})

	// Replace launcher with mock to avoid slow Claude CLI discovery
	mock := claude.SetupMockLauncherWithDefaults()
	mock.SetResponseForPattern(".*", "AI cached test response")
	generator.launcher = mock

	req := &GenerateRequest{
		OriginalMessage: "Use 'just test' instead of 'go test'",
		GenerateMode:    "once",
		Pattern:         "^go test",
	}

	// First call should generate the response
	result1, err1 := generator.GenerateMessage(req)
	if err1 != nil {
		t.Fatalf("First GenerateMessage failed: %v", err1)
	}

	expectedResponse := "AI cached test response"
	if result1 != expectedResponse {
		t.Errorf("Expected mocked response %q, got %q", expectedResponse, result1)
	}

	// Second call with same request should use cache for "once" mode
	result2, err2 := generator.GenerateMessage(req)
	if err2 != nil {
		t.Fatalf("Second GenerateMessage failed: %v", err2)
	}

	// Both calls should return the same result due to caching
	if result1 != result2 {
		t.Errorf("Caching failed: first result %q != second result %q", result1, result2)
	}
}

func TestGeneratorShouldUseCache(t *testing.T) {
	t.Parallel()
	setupTest(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	generator, err := NewGenerator(dbPath, "test-project")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := generator.Close(); closeErr != nil {
			t.Logf("Failed to close generator: %v", closeErr)
		}
	})

	// Replace launcher with mock to avoid slow Claude CLI discovery
	mock := claude.SetupMockLauncherWithDefaults()
	mock.SetResponseForPattern(".*", "AI should use cache test response")
	generator.launcher = mock

	req := &GenerateRequest{
		OriginalMessage: "Use 'just test' instead of 'go test'",
		GenerateMode:    "once",
		Pattern:         "^go test",
	}

	// Should return the mocked response, demonstrating actual integration
	result1, err := generator.GenerateMessage(req)
	if err != nil {
		t.Fatalf("GenerateMessage failed: %v", err)
	}

	expectedResponse := "AI should use cache test response"
	if result1 != expectedResponse {
		t.Errorf("Expected mocked response %q, got %q", expectedResponse, result1)
	}

	// Verify it's not using a stub implementation
	expectedStub := "[AI] " + req.OriginalMessage
	if result1 == expectedStub {
		t.Error("Generator is using stub implementation, expected actual cache/claude integration")
	}
}

func TestGeneratorCachingWithMock(t *testing.T) {
	t.Parallel()
	setupTest(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	generator, err := NewGenerator(dbPath, "test-project")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := generator.Close(); closeErr != nil {
			t.Logf("Failed to close generator: %v", closeErr)
		}
	})

	// Replace launcher with enhanced mock that returns different results each call
	mock := claude.SetupMockLauncherWithDefaults()
	mock.SetResponseForPattern(".*", "Mock AI response A")
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
	claude.AssertMockCalled(t, mock, 1)
}

func TestGeneratorLogsCache(t *testing.T) {
	t.Parallel()
	setupTest(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	generator, err := NewGenerator(dbPath, "test-project")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := generator.Close(); closeErr != nil {
			t.Logf("Failed to close generator: %v", closeErr)
		}
	})

	// Replace launcher with enhanced mock
	mock := claude.SetupMockLauncherWithDefaults()
	mock.SetResponseForPattern(".*", "Mock cache test response")
	generator.launcher = mock

	req := &GenerateRequest{
		OriginalMessage: "Use 'just test' instead of 'go test'",
		GenerateMode:    "once",
		Pattern:         "^go test",
	}

	// First call should log cache miss and AI generation
	_, err = generator.GenerateMessage(req)
	if err != nil {
		t.Fatalf("First GenerateMessage failed: %v", err)
	}

	// Second call should log cache hit
	_, err = generator.GenerateMessage(req)
	if err != nil {
		t.Fatalf("Second GenerateMessage failed: %v", err)
	}

	// Verify caching behavior through mock call count
	// Should only call the launcher once due to caching
	claude.AssertMockCalled(t, mock, 1)
}
