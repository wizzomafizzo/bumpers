package testutil

import (
	"strings"
	"testing"
)

// AssertNoError fails the test with a formatted message if err is not nil
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected error but got nil", msg)
	}
}

// AssertEqual fails the test if expected != actual
func AssertEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// AssertEqualMsg fails the test if expected != actual with custom message
func AssertEqualMsg[T comparable](t *testing.T, expected, actual T, msg string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// AssertContains fails the test if haystack does not contain needle
func AssertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("Expected %q to contain %q", haystack, needle)
	}
}

// AssertNotContains fails the test if haystack contains needle
func AssertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("Expected %q to not contain %q", haystack, needle)
	}
}

// AssertTrue fails the test if condition is false
func AssertTrue(t *testing.T, condition bool, msg string) { //nolint:revive // condition parameter is part of assertion API
	t.Helper()
	if !condition {
		t.Errorf("%s: expected true, got false", msg)
	}
}

// AssertFalse fails the test if condition is true
func AssertFalse(t *testing.T, condition bool, msg string) { //nolint:revive // condition parameter is part of assertion API
	t.Helper()
	if condition {
		t.Errorf("%s: expected false, got true", msg)
	}
}

// AssertNil fails the test if value is not nil
func AssertNil(t *testing.T, value any, msg string) {
	t.Helper()
	if value != nil {
		t.Errorf("%s: expected nil, got %v", msg, value)
	}
}

// AssertNotNil fails the test if value is nil
func AssertNotNil(t *testing.T, value any, msg string) {
	t.Helper()
	if value == nil {
		t.Errorf("%s: expected non-nil value", msg)
	}
}

// AssertLen fails the test if the slice/map/string length doesn't match expected
func AssertLen[T any](t *testing.T, items []T, expectedLen int, msg string) {
	t.Helper()
	if len(items) != expectedLen {
		t.Errorf("%s: expected length %d, got %d", msg, expectedLen, len(items))
	}
}

// AssertEmpty fails the test if the slice/map/string is not empty
func AssertEmpty[T any](t *testing.T, items []T, msg string) {
	t.Helper()
	if len(items) != 0 {
		t.Errorf("%s: expected empty, got %d items", msg, len(items))
	}
}

// AssertNotEmpty fails the test if the slice/map/string is empty
func AssertNotEmpty[T any](t *testing.T, items []T, msg string) {
	t.Helper()
	if len(items) == 0 {
		t.Errorf("%s: expected non-empty", msg)
	}
}
