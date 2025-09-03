package testutil

import (
	"testing"

	"go.uber.org/goleak"
)

// Test constants and helpers
const (
	expectedResult = 2
	testString     = "test"
)

func performSimpleOperation() int {
	return 1 + 1
}

func performStringOperation() string {
	return testString
}

// Test cases
func TestVerifyNoLeaks_NoGoroutineLeaks(t *testing.T) {
	t.Parallel()

	// Test function should pass when no goroutines are leaked
	t.Run("clean test", func(t *testing.T) {
		t.Parallel()
		defer VerifyNoLeaks(t)

		// Simple operation that doesn't create goroutines
		result := performSimpleOperation()
		if result != expectedResult {
			t.Errorf("Expected %d, got %d", expectedResult, result)
		}
	})
}

func TestVerifyNoLeaksWithOptions_CustomIgnore(t *testing.T) {
	t.Parallel()
	// This test verifies that the options parameter is properly passed to goleak
	// We use a custom option that should work (IgnoreTopFunction with non-existent function)
	t.Run("with ignore option", func(t *testing.T) {
		t.Parallel()
		defer VerifyNoLeaksWithOptions(t,
			goleak.IgnoreTopFunction("non.existent.function"))

		// Simple operation
		result := performStringOperation()
		if result != testString {
			t.Errorf("Expected '%s', got %s", testString, result)
		}
	})
}
