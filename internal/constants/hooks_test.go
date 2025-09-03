package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPreToolUseEvent_ShouldHaveConstant(t *testing.T) {
	t.Parallel()
	// When
	hookType := PreToolUseEvent

	// Then
	assert.Equal(t, "PreToolUse", hookType)
}

func TestPostToolUseEvent_ShouldHaveConstant(t *testing.T) {
	t.Parallel()
	// When
	hookType := PostToolUseEvent

	// Then
	assert.Equal(t, "PostToolUse", hookType)
}
