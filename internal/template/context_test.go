package template

import (
	"testing"
	"time"
)

func TestNewSharedContext(t *testing.T) {
	t.Parallel()

	context := NewSharedContext()

	// Verify Today field is set to current date in yyyy-mm-dd format
	expectedDate := time.Now().Format("2006-01-02")
	if context.Today != expectedDate {
		t.Errorf("Expected Today to be %q, got %q", expectedDate, context.Today)
	}
}

func TestMergeContexts_SharedOnly(t *testing.T) {
	t.Parallel()

	shared := SharedContext{Today: "2025-08-22"}
	result := MergeContexts(shared, nil)

	if result["Today"] != "2025-08-22" {
		t.Errorf("Expected Today to be '2025-08-22', got %v", result["Today"])
	}
}

func TestMergeContexts_WithRuleContext(t *testing.T) {
	t.Parallel()

	shared := SharedContext{Today: "2025-08-22"}
	specific := RuleContext{Command: "git commit"}
	result := MergeContexts(shared, specific)

	if result["Today"] != "2025-08-22" {
		t.Errorf("Expected Today to be '2025-08-22', got %v", result["Today"])
	}
	if result["Command"] != "git commit" {
		t.Errorf("Expected Command to be 'git commit', got %v", result["Command"])
	}
}

func TestBuildRuleContext(t *testing.T) {
	t.Parallel()

	result := BuildRuleContext("go test")

	// Should have both shared and rule-specific context
	expectedDate := time.Now().Format("2006-01-02")
	if result["Today"] != expectedDate {
		t.Errorf("Expected Today to be %q, got %v", expectedDate, result["Today"])
	}
	if result["Command"] != "go test" {
		t.Errorf("Expected Command to be 'go test', got %v", result["Command"])
	}
}

func TestBuildCommandContext(t *testing.T) {
	t.Parallel()

	result := BuildCommandContext("lint")

	// Should have both shared and command-specific context
	expectedDate := time.Now().Format("2006-01-02")
	if result["Today"] != expectedDate {
		t.Errorf("Expected Today to be %q, got %v", expectedDate, result["Today"])
	}
	if result["Name"] != "lint" {
		t.Errorf("Expected Name to be 'lint', got %v", result["Name"])
	}
}

func TestBuildNoteContext(t *testing.T) {
	t.Parallel()

	result := BuildNoteContext()

	// Should have shared context
	expectedDate := time.Now().Format("2006-01-02")
	if result["Today"] != expectedDate {
		t.Errorf("Expected Today to be %q, got %v", expectedDate, result["Today"])
	}

	// Should only have shared context fields since NoteContext is empty
	if len(result) != 1 {
		t.Errorf("Expected 1 field in result, got %d", len(result))
	}
}

// Table-driven test for context building functions
func TestBuildContexts(t *testing.T) {
	t.Parallel()

	expectedDate := time.Now().Format("2006-01-02")

	tests := []struct {
		buildFunc    func() map[string]any
		expectedKeys map[string]any
		name         string
		expectedLen  int
	}{
		{
			name:      "rule context",
			buildFunc: func() map[string]any { return BuildRuleContext("go test") },
			expectedKeys: map[string]any{
				"Today":   expectedDate,
				"Command": "go test",
			},
			expectedLen: 2,
		},
		{
			name:      "command context",
			buildFunc: func() map[string]any { return BuildCommandContext("lint") },
			expectedKeys: map[string]any{
				"Today": expectedDate,
				"Name":  "lint",
			},
			expectedLen: 2,
		},
		{
			name:      "note context",
			buildFunc: BuildNoteContext,
			expectedKeys: map[string]any{
				"Today": expectedDate,
			},
			expectedLen: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.buildFunc()

			if len(result) != tc.expectedLen {
				t.Errorf("Expected %d fields in result, got %d", tc.expectedLen, len(result))
			}

			for key, expectedValue := range tc.expectedKeys {
				if result[key] != expectedValue {
					t.Errorf("Expected %s to be %q, got %v", key, expectedValue, result[key])
				}
			}
		})
	}
}
