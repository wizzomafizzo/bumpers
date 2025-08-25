package hooks

import "testing"

func TestHookTypeString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expected string
		hookType HookType
	}{
		{"Unknown", UnknownHook},
		{"PreToolUse", PreToolUseHook},
		{"UserPromptSubmit", UserPromptSubmitHook},
		{"PostToolUse", PostToolUseHook},
		{"SessionStart", SessionStartHook},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			result := tt.hookType.String()
			if result != tt.expected {
				t.Errorf("HookType.String() = %q, want %q", result, tt.expected)
			}
		})
	}

	// Test invalid hook type
	t.Run("invalid hook type", func(t *testing.T) {
		invalidType := HookType(999)
		result := invalidType.String()
		if result != "Unknown" {
			t.Errorf("HookType.String() for invalid type = %q, want %q", result, "Unknown")
		}
	})
}
