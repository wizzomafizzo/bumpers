package patterns

import (
	"testing"
)

func TestGeneratePattern(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "simple command",
			command:  "rm -rf /",
			expected: "^rm\\s+-rf\\s+/$",
		},
		{
			name:     "basic command",
			command:  "go test",
			expected: "^go\\s+test$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := GeneratePattern(tt.command)
			if result != tt.expected {
				t.Errorf("GeneratePattern(%s) = %s, want %s", tt.command, result, tt.expected)
			}
		})
	}
}
