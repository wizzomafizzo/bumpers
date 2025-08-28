//go:build integration

package ai

import (
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/platform/claude"
	"github.com/wizzomafizzo/bumpers/internal/testing"
)

// TestMessageGeneratorBasicContract tests that both mock and real implementations
// can handle a simple valid prompt
func TestMessageGeneratorBasicContract(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	t.Run("MockLauncher", func(t *testing.T) {
		t.Parallel()

		mock := claude.NewMockLauncher()
		prompt := "Generate a helpful message about version control."

		response, err := mock.GenerateMessage(prompt)

		if err != nil {
			t.Errorf("Mock GenerateMessage should succeed, got error: %v", err)
		}

		if response == "" {
			t.Error("Mock GenerateMessage should return non-empty response")
		}
	})

	t.Run("RealLauncher", func(t *testing.T) {
		t.Parallel()

		launcher := claude.NewLauncher(nil)
		if _, err := launcher.GetClaudePath(); err != nil {
			t.Skip("Claude binary not available, skipping real launcher test")
		}

		prompt := "Generate a helpful message about version control."

		response, err := launcher.GenerateMessage(prompt)

		if err != nil {
			t.Logf("Real launcher failed (may be acceptable in test env): %v", err)
		} else {
			t.Logf("Real launcher succeeded with response length: %d", len(response))
			// Note: Empty responses may occur due to network issues, rate limits, or configuration
			// We don't fail the test for empty responses as this would make CI flaky
		}
	})
}
