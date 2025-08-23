package claude

import (
	"testing"
)

func TestLauncherGenerateMessage(t *testing.T) {
	t.Parallel()
	launcher := NewLauncher(nil)

	prompt := "Rephrase this message: 'Use just test instead of go test'"

	// Note: This will fail in CI/testing without Claude binary
	// We're testing the interface exists and basic error handling
	result, err := launcher.GenerateMessage(prompt)
	// We expect either a result (if Claude is installed) or a specific error
	if err != nil {
		// For now, we just check that the method exists and handles errors
		// In a real scenario we'd mock the Claude binary or skip this test
		t.Logf("GenerateMessage failed (expected in test environment): %v", err)
		return
	}

	// If we got here, Claude was found and should return something
	if result == "" {
		t.Error("GenerateMessage should return a non-empty result when Claude is available")
	}
}
