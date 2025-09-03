package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	testutil "github.com/wizzomafizzo/bumpers/internal/testing"
)

func TestRun_Success(t *testing.T) {
	t.Parallel()
	_, _ = testutil.NewTestContext(t) // Context-aware logging

	// This test verifies that run() can complete successfully in a basic scenario
	// The function should find project root, initialize logger, and create the root command
	err := run()

	// Should complete without error
	assert.NoError(t, err)
}
