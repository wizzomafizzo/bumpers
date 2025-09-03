package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessUserPrompt_InvalidJSON(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	handler := NewPromptHandler("/nonexistent", "/test")
	handler.SetTestDBPath(":memory:")

	// Test with completely invalid JSON
	result, err := handler.ProcessUserPrompt(ctx, []byte(`{invalid json`))

	require.Error(t, err)
	require.Empty(t, result)
	require.Contains(t, err.Error(), "invalid character") // JSON parsing error
}

func TestProcessUserPrompt_EmptyJSON(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	handler := NewPromptHandler("/nonexistent", "/test")
	handler.SetTestDBPath(":memory:")

	// Test with valid JSON but no prompt field
	result, err := handler.ProcessUserPrompt(ctx, []byte(`{}`))

	// Empty prompt should not be an error, just pass through
	require.NoError(t, err)
	require.Empty(t, result) // No command to process
}

func TestProcessUserPrompt_NonCommand(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	handler := NewPromptHandler("/nonexistent", "/test")
	handler.SetTestDBPath(":memory:")

	// Test with regular text that's not a command
	result, err := handler.ProcessUserPrompt(ctx, []byte(`{"prompt": "hello world"}`))

	// Regular text should pass through (not a command)
	require.NoError(t, err)
	require.Empty(t, result) // Not a command, pass through
}
