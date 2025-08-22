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
