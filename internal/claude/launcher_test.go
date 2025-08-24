package claude

import (
	"errors"
	"testing"
)

func TestIsClaudeNotFoundError(t *testing.T) {
	t.Parallel()

	// Test with NotFoundError
	notFoundErr := &NotFoundError{AttemptedPaths: []string{"test"}}
	if !IsClaudeNotFoundError(notFoundErr) {
		t.Error("IsClaudeNotFoundError should return true for NotFoundError")
	}

	// Test with other error types
	otherErr := errors.New("some other error")
	if IsClaudeNotFoundError(otherErr) {
		t.Error("IsClaudeNotFoundError should return false for non-NotFoundError")
	}

	// Test with nil
	if IsClaudeNotFoundError(nil) {
		t.Error("IsClaudeNotFoundError should return false for nil")
	}
}
