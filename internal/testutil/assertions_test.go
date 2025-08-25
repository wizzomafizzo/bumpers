package testutil

import (
	"testing"
)

func TestAssertNoError_WithNil_Passes(t *testing.T) {
	t.Parallel()
	// This should not fail the test
	AssertNoError(t, nil, "should pass with nil error")
}
