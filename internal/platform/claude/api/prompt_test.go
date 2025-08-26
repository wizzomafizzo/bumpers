package ai

import "testing"

func TestBuildDefaultPrompt(t *testing.T) {
	t.Parallel()
	message := "Use 'just test' instead of 'go test'"

	prompt := BuildDefaultPrompt(message)

	if prompt == "" {
		t.Error("BuildDefaultPrompt should return a non-empty prompt")
	}

	// The prompt should contain the original message
	if !contains(prompt, message) {
		t.Errorf("BuildDefaultPrompt should contain the original message %q", message)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
