package hooks

import (
	"strings"
	"testing"
)

func TestParseInput(t *testing.T) {
	t.Parallel()

	jsonInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		}
	}`

	event, err := ParseInput(strings.NewReader(jsonInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if event.ToolInput.Command != "go test ./..." {
		t.Errorf("Expected command 'go test ./...', got %s", event.ToolInput.Command)
	}
}

func TestDetectHookType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		jsonData string
		expected HookType
	}{
		{
			name: "PreToolUse hook",
			jsonData: `{
				"tool_input": {
					"command": "go test",
					"description": "Run tests"
				}
			}`,
			expected: PreToolUseHook,
		},
		{
			name: "UserPromptSubmit hook",
			jsonData: `{
				"prompt": "!help"
			}`,
			expected: UserPromptSubmitHook,
		},
		{
			name: "PostToolUse hook",
			jsonData: `{
				"tool_output": {
					"command": "go test",
					"status": "success"
				}
			}`,
			expected: PostToolUseHook,
		},
		{
			name: "SessionStart hook with startup source",
			jsonData: `{
				"session_id": "abc123",
				"hook_event_name": "SessionStart",
				"source": "startup"
			}`,
			expected: SessionStartHook,
		},
		{
			name: "SessionStart hook with clear source",
			jsonData: `{
				"session_id": "def456",
				"hook_event_name": "SessionStart",
				"source": "clear"
			}`,
			expected: SessionStartHook,
		},
		{
			name:     "Unknown hook",
			jsonData: `{"unknown_field": "value"}`,
			expected: UnknownHook,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hookType, rawJSON, err := DetectHookType(strings.NewReader(tt.jsonData))
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if hookType != tt.expected {
				t.Errorf("Expected hook type %v, got %v", tt.expected, hookType)
			}

			if rawJSON == nil {
				t.Error("Expected raw JSON to be returned")
			}
		})
	}
}

func TestParseInputWithToolName(t *testing.T) {
	t.Parallel()

	jsonInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash"
	}`

	event, err := ParseInput(strings.NewReader(jsonInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if event.ToolInput.Command != "go test ./..." {
		t.Errorf("Expected command 'go test ./...', got %s", event.ToolInput.Command)
	}

	if event.ToolName != "Bash" {
		t.Errorf("Expected tool name 'Bash', got '%s'", event.ToolName)
	}
}
