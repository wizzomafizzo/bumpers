package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/hooks"
)

func TestHookRouter_RouteToCorrectHandler(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	router := NewHookRouter()

	// Track which handler was called
	var calledHandler string

	// Register handler for UserPromptSubmit hook
	router.Register(hooks.UserPromptSubmitHook, HandlerFunc(func(_ context.Context, _ json.RawMessage) (string, error) {
		calledHandler = "user-prompt"
		return "user-prompt-response", nil
	}))

	// Test with UserPromptSubmit hook
	input := bytes.NewBufferString(`{"prompt": "test prompt"}`)
	result, err := router.Route(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, "user-prompt-response", result)
	assert.Equal(t, "user-prompt", calledHandler)
}
