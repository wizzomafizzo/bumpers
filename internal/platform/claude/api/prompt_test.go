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

func TestBuildRegexGenerationPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "literal command",
			input: "rm -rf /",
		},
		{
			name:  "descriptive command",
			input: "commands that delete files",
		},
		{
			name:  "git command",
			input: "git push origin main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			prompt := BuildRegexGenerationPrompt(tt.input)

			if prompt == "" {
				t.Error("BuildRegexGenerationPrompt should return a non-empty prompt")
			}

			// The prompt should contain the input
			if !contains(prompt, tt.input) {
				t.Errorf("BuildRegexGenerationPrompt should contain the input %q", tt.input)
			}

			// The prompt should mention regex generation
			if !contains(prompt, "regex") {
				t.Error("BuildRegexGenerationPrompt should mention regex generation")
			}
		})
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
