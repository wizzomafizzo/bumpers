package testutil

import (
	"testing"

	"go.uber.org/goleak"
)

// VerifyNoLeaks verifies that no goroutines are leaked during test execution.
// Call this at the beginning of tests that create resources like database connections,
// file handles, or goroutines.
//
// Example usage:
//
//	func TestResourceIntensiveFunction(t *testing.T) {
//	    defer VerifyNoLeaks(t)
//	    // Test code that may create resources
//	}
func VerifyNoLeaks(t *testing.T) {
	t.Helper()
	goleak.VerifyNone(t)
}

// VerifyNoLeaksWithOptions provides more control over leak detection.
// Use this when you need to ignore certain goroutines or customize behavior.
//
// Example usage:
//
//	func TestWithIgnoredGoroutines(t *testing.T) {
//	    defer VerifyNoLeaksWithOptions(t,
//	        goleak.IgnoreTopFunction("some.package.backgroundWorker"))
//	    // Test code
//	}
func VerifyNoLeaksWithOptions(t *testing.T, options ...goleak.Option) {
	t.Helper()
	goleak.VerifyNone(t, options...)
}
