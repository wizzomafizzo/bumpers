package config

import "testing"

// This file contains placeholder for any remaining tests that don't fit into the focused categories
// All major tests have been moved to:
// - config_loading_test.go: Config loading, parsing, validation, defaults, benchmarks, examples
// - config_rules_test.go: Rule validation, matching, event/source processing
// - config_commands_test.go: Command and session configuration tests

func TestConvertSourcesSliceWithValidInputs(t *testing.T) {
	t.Parallel()

	sources := []any{"command", "intent", "tool_output"}
	result, err := convertSourcesSlice(sources)
	if err != nil {
		t.Fatalf("Expected no error for valid inputs, got: %v", err)
	}

	expected := []string{"command", "intent", "tool_output"}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}

	for i, expected := range expected {
		if result[i] != expected {
			t.Errorf("Expected result[%d] = %q, got %q", i, expected, result[i])
		}
	}
}

func TestConvertSourcesSliceWithInvalidTypesReturnsError(t *testing.T) {
	t.Parallel()

	// This test verifies the new behavior: return error for invalid types
	sources := []any{"command", 123, "intent", nil}
	result, err := convertSourcesSlice(sources)

	if err == nil {
		t.Fatal("Expected error for invalid types, but got none")
	}
	if result != nil {
		t.Errorf("Expected nil result when error occurs, got %v", result)
	}
}
