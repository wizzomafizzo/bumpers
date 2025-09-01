package operation

import (
	"testing"
)

func TestDetectTriggerPhrase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		{
			name:     "detects make it so",
			message:  "make it so",
			expected: true,
		},
		{
			name:     "detects go ahead",
			message:  "go ahead with the changes",
			expected: true,
		},
		{
			name:     "case insensitive",
			message:  "MAKE IT SO",
			expected: true,
		},
		{
			name:     "no trigger phrase",
			message:  "let me think about this",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := DetectTriggerPhrase(tt.message)
			if result != tt.expected {
				t.Errorf("DetectTriggerPhrase(%q) = %v, want %v", tt.message, result, tt.expected)
			}
		})
	}
}

func TestDetectEmergencyStop(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		{
			name:     "detects STOP",
			message:  "STOP",
			expected: true,
		},
		{
			name:     "detects SILENCE",
			message:  "please SILENCE the tools",
			expected: true,
		},
		{
			name:     "no emergency stop",
			message:  "continue with the work",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := DetectEmergencyStop(tt.message)
			if result != tt.expected {
				t.Errorf("DetectEmergencyStop(%q) = %v, want %v", tt.message, result, tt.expected)
			}
		})
	}
}

func TestIsReadOnlyBashCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "ls command",
			command:  "ls",
			expected: true,
		},
		{
			name:     "ls with arguments",
			command:  "ls -la",
			expected: true,
		},
		{
			name:     "git status",
			command:  "git status",
			expected: true,
		},
		{
			name:     "cat file",
			command:  "cat README.md",
			expected: true,
		},
		{
			name:     "rm command (not read-only)",
			command:  "rm file.txt",
			expected: false,
		},
		{
			name:     "echo with redirection (not read-only)",
			command:  "echo 'test' > file.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IsReadOnlyBashCommand(tt.command)
			if result != tt.expected {
				t.Errorf("IsReadOnlyBashCommand(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}
