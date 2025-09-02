package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldConstants_ShouldExist(t *testing.T) {
	t.Parallel()
	// Test that field constants exist and have expected values
	assert.Equal(t, "prompt", FieldPrompt)
	assert.Equal(t, "tool_response", FieldToolResponse)
	assert.Equal(t, "hook_event_name", FieldHookEventName)
	assert.Equal(t, "session_id", FieldSessionID)
	assert.Equal(t, "source", FieldSource)
}
