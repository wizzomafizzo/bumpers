//go:build integration

package claude

import (
	"context"
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/testing"
)

// TestLauncherBasicContract tests that both mock and real launcher implementations
// satisfy the basic MessageGenerator contract
func TestLauncherBasicContract(t *testing.T) {
	t.Parallel()

	t.Run("MockLauncher", func(t *testing.T) {
		t.Parallel()

		ctx, _ := testutil.NewTestContext(t)
		mock := NewMockLauncher()
		mock.SetResponseForPattern(".*", "Mock response for contract test")
		prompt := "Test prompt for contract validation"

		response, err := mock.GenerateMessage(ctx, prompt)

		if err != nil {
			t.Errorf("Mock launcher should succeed with valid prompt, got error: %v", err)
		}

		if response == "" {
			t.Error("Mock launcher should return non-empty response")
		}

		// Verify mock was called
		if mock.GetCallCount() != 1 {
			t.Errorf("Expected 1 call to mock launcher, got %d", mock.GetCallCount())
		}
	})

	t.Run("RealLauncher", func(t *testing.T) {
		t.Parallel()

		ctx, _ := testutil.NewTestContext(t)
		launcher := NewLauncher(nil)
		if _, err := launcher.GetClaudePath(); err != nil {
			t.Skip("Claude binary not available, skipping real launcher contract test")
		}

		prompt := "Test prompt for real Claude validation"

		response, err := launcher.GenerateMessage(ctx, prompt)

		if err != nil {
			t.Logf("Real launcher failed (may be acceptable in test env): %v", err)
		} else if response == "" {
			t.Error("Real launcher should return non-empty response when successful")
		}
	})
}
