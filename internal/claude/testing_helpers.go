package claude

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Common test response patterns for easy setup
var CommonTestResponses = map[string]string{
	".*just test.*": "Great choice! Using 'just test' provides better integration with TDD guard and coverage.",
	".*go test.*":   "Consider using 'just test' instead of 'go test' for enhanced testing capabilities.",
	".*rm -rf.*":    "For safety, consider using specific file paths instead of recursive removal.",
	".*password.*":  "Avoid hardcoding sensitive information. Consider using environment variables.",
	".*secret.*":    "Avoid hardcoding secrets. Use secure configuration management instead.",
	".*npm.*":       "Consider using the project's preferred package manager as specified in the documentation.",
}

// Common test errors
var (
	ErrMockNetworkTimeout = errors.New("mock network timeout")
	ErrMockAPIError       = errors.New("mock API error: rate limit exceeded")
	ErrMockInvalidJSON    = errors.New("mock JSON parsing error")
)

// SetupMockLauncherWithDefaults creates a mock launcher with common patterns
func SetupMockLauncherWithDefaults() *MockLauncher {
	return NewMockLauncherWithResponses(CommonTestResponses)
}

// SetupMockClaudeBinary prepares the mock shell script for integration tests
// Returns the path to the mock binary that can be used in tests
func SetupMockClaudeBinary(t *testing.T) string {
	t.Helper()

	// Get the project root (where testdata should be)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Find project root by looking for go.mod
	projectRoot := wd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatal("Could not find project root (go.mod)")
		}
		projectRoot = parent
	}

	mockPath := filepath.Join(projectRoot, "testdata", "mock-claude.sh")

	// Verify the mock script exists
	if _, err := os.Stat(mockPath); err != nil {
		t.Fatalf("Mock Claude script not found at %s: %v", mockPath, err)
	}

	return mockPath
}

// UseMockClaudePath temporarily overrides PATH to use the mock binary
func UseMockClaudePath(t *testing.T, mockPath string) {
	t.Helper()

	// Add the directory containing the mock to PATH
	mockDir := filepath.Dir(mockPath)
	originalPath := os.Getenv("PATH")
	newPath := mockDir + ":" + originalPath

	// Set the new PATH using t.Setenv for automatic cleanup
	t.Setenv("PATH", newPath)
}

// AssertMockCalled verifies that the mock was called
func AssertMockCalled(t *testing.T, mock *MockLauncher, expectedCalls int) {
	t.Helper()
	if mock.GetCallCount() != expectedCalls {
		t.Errorf("Expected %d calls to mock launcher, got %d", expectedCalls, mock.GetCallCount())
	}
}

// AssertMockCalledWithPattern verifies that the mock was called with a prompt matching the pattern
func AssertMockCalledWithPattern(t *testing.T, mock *MockLauncher, pattern string) {
	t.Helper()
	if !mock.WasCalledWithPattern(pattern) {
		t.Errorf("Expected mock to be called with pattern %q, but no matching calls found", pattern)
		t.Log("Actual calls:")
		for i, call := range mock.Calls {
			t.Logf("  %d: %q", i+1, call.Prompt)
		}
	}
}

// AssertMockNotCalled verifies that the mock was never called
func AssertMockNotCalled(t *testing.T, mock *MockLauncher) {
	t.Helper()
	if mock.GetCallCount() > 0 {
		t.Errorf("Expected no calls to mock launcher, but got %d calls", mock.GetCallCount())
	}
}
