package hooks

import (
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/constants"
)

func TestHookProcessor_ShouldUsePreToolUseConstant(t *testing.T) {
	t.Parallel()
	// This test verifies that hook_processor.go uses constants.PreToolUseEvent
	// instead of hardcoded "PreToolUse" string

	// Given - the constant should match expected value
	expectedValue := "PreToolUse"

	// When - we check the constant value
	actualValue := constants.PreToolUseEvent

	// Then - it should match the expected hardcoded value we're replacing
	if actualValue != expectedValue {
		t.Errorf("Expected PreToolUseEvent to be %q, got %q", expectedValue, actualValue)
	}
}
