package testutil

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// Legacy race condition tests removed - these functions demonstrated race conditions
// and have been replaced with context-aware logging. The TestContextAwareLogging
// test below validates the new race-free approach.

func TestContextAwareLogging(t *testing.T) {
	t.Parallel()

	ctx, getLogs := NewTestContext(t)

	// Test that context logger works
	zerolog.Ctx(ctx).Debug().Msg("Test message")

	output := getLogs()
	assert.Contains(t, output, "Test message")
}
