package rules

import (
	"testing"
	"time"
)

const (
	testModifiedValue = "modified"
)

func TestEncapsulation_GlobalVariablesShouldBePrivate(t *testing.T) {
	t.Parallel()
	// Test that we can get copies but cannot modify originals directly
	originalTriggers := GetTriggerPhrases()
	originalStops := GetEmergencyStopPhrases()
	originalCommands := GetReadOnlyBashCommands()

	// Modify the copies
	originalTriggers[0] = testModifiedValue
	originalStops[0] = testModifiedValue
	originalCommands[0] = testModifiedValue

	// Get fresh copies - they should not be affected
	freshTriggers := GetTriggerPhrases()
	freshStops := GetEmergencyStopPhrases()
	freshCommands := GetReadOnlyBashCommands()

	// Verify original values are preserved
	if freshTriggers[0] == testModifiedValue {
		t.Error("triggerPhrases should return copies to prevent external modification")
	}
	if freshStops[0] == testModifiedValue {
		t.Error("emergencyStopPhrases should return copies to prevent external modification")
	}
	if freshCommands[0] == testModifiedValue {
		t.Error("readOnlyBashCommands should return copies to prevent external modification")
	}
}

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
		{
			name:     "case insensitive stop",
			message:  "please stop the process",
			expected: true,
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

func TestDefaultState(t *testing.T) {
	t.Parallel()

	beforeCall := time.Now().Unix()
	state := DefaultState()
	afterCall := time.Now().Unix()

	if state.Mode != ExecuteMode {
		t.Errorf("DefaultState().Mode = %v, want %v", state.Mode, ExecuteMode)
	}

	if state.TriggerCount != 0 {
		t.Errorf("DefaultState().TriggerCount = %v, want %v", state.TriggerCount, 0)
	}

	// UpdatedAt should be set to current time, not zero
	if state.UpdatedAt < beforeCall || state.UpdatedAt > afterCall {
		t.Errorf("DefaultState().UpdatedAt = %v, want between %v and %v", state.UpdatedAt, beforeCall, afterCall)
	}
}

func TestGetTriggerPhrases(t *testing.T) {
	t.Parallel()

	phrases := GetTriggerPhrases()

	// Should contain expected default phrases
	expected := []string{"make it so", "go ahead"}
	if len(phrases) != len(expected) {
		t.Errorf("GetTriggerPhrases() length = %v, want %v", len(phrases), len(expected))
	}

	for i, expected := range expected {
		if i >= len(phrases) || phrases[i] != expected {
			t.Errorf("GetTriggerPhrases()[%d] = %v, want %v", i, phrases[i], expected)
		}
	}

	// Modifying returned slice should not affect internal state
	originalLen := len(phrases)
	phrases[0] = testModifiedValue
	phrases2 := GetTriggerPhrases()
	if phrases2[0] == testModifiedValue {
		t.Error("GetTriggerPhrases() should return a copy, not the original slice")
	}
	if len(phrases2) != originalLen {
		t.Error("GetTriggerPhrases() should return consistent results")
	}
}

func TestGetEmergencyStopPhrases(t *testing.T) {
	t.Parallel()

	phrases := GetEmergencyStopPhrases()

	// Should contain expected default phrases
	expected := []string{"STOP", "SILENCE"}
	if len(phrases) != len(expected) {
		t.Errorf("GetEmergencyStopPhrases() length = %v, want %v", len(phrases), len(expected))
	}

	for i, expected := range expected {
		if i >= len(phrases) || phrases[i] != expected {
			t.Errorf("GetEmergencyStopPhrases()[%d] = %v, want %v", i, phrases[i], expected)
		}
	}

	// Modifying returned slice should not affect internal state
	phrases[0] = testModifiedValue
	phrases2 := GetEmergencyStopPhrases()
	if phrases2[0] == testModifiedValue {
		t.Error("GetEmergencyStopPhrases() should return a copy, not the original slice")
	}
}

func TestGetReadOnlyBashCommands(t *testing.T) {
	t.Parallel()

	commands := GetReadOnlyBashCommands()

	// Should contain expected default commands
	expected := []string{"ls", "cat", "git status"}
	if len(commands) != len(expected) {
		t.Errorf("GetReadOnlyBashCommands() length = %v, want %v", len(commands), len(expected))
	}

	for i, expected := range expected {
		if i >= len(commands) || commands[i] != expected {
			t.Errorf("GetReadOnlyBashCommands()[%d] = %v, want %v", i, commands[i], expected)
		}
	}

	// Modifying returned slice should not affect internal state
	commands[0] = testModifiedValue
	commands2 := GetReadOnlyBashCommands()
	if commands2[0] == testModifiedValue {
		t.Error("GetReadOnlyBashCommands() should return a copy, not the original slice")
	}
}
