package hooks

import (
	"strings"
	"testing"

	testutil "github.com/wizzomafizzo/bumpers/internal/testing"
)

const (
	testPostToolUseEvent = "PostToolUse"
)

func TestParseInput(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
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

	if command, ok := event.ToolInput["command"].(string); !ok || command != "go test ./..." {
		t.Errorf("Expected command 'go test ./...', got %v", event.ToolInput["command"])
	}
}

func TestDetectHookType(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
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
				"tool_response": {
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
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
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

	if command, ok := event.ToolInput["command"].(string); !ok || command != "go test ./..." {
		t.Errorf("Expected command 'go test ./...', got %v", event.ToolInput["command"])
	}

	if event.ToolName != "Bash" {
		t.Errorf("Expected tool name 'Bash', got '%s'", event.ToolName)
	}
}

func TestParseInputWithToolUseID(t *testing.T) {
	_, _ = testutil.NewTestContext(t)
	t.Parallel()

	jsonInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash",
		"tool_use_id": "toolu_01KTePc3uLq34eriLmSLbgnx"
	}`

	event, err := ParseInput(strings.NewReader(jsonInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if event.ToolUseID != "toolu_01KTePc3uLq34eriLmSLbgnx" {
		t.Errorf("Expected tool use ID 'toolu_01KTePc3uLq34eriLmSLbgnx', got '%s'", event.ToolUseID)
	}
}

func TestParseInputWithPostToolUseFields(t *testing.T) {
	_, _ = testutil.NewTestContext(t)
	t.Parallel()

	// Test PostToolUse hook event with all required fields
	jsonInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash",
		"tool_use_id": "toolu_01KTePc3uLq34eriLmSLbgnx",
		"tool_response": {
			"stdout": "test output",
			"stderr": "",
			"exit_code": 0
		},
		"session_id": "sess_abc123",
		"hook_event_name": "PostToolUse",
		"cwd": "/home/user/project"
	}`

	event, err := ParseInput(strings.NewReader(jsonInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify tool_response field is populated
	if event.ToolResponse == nil {
		t.Error("Expected tool_response to be populated")
	}
	if response, ok := event.ToolResponse.(map[string]any); ok {
		if stdout, ok := response["stdout"].(string); !ok || stdout != "test output" {
			t.Errorf("Expected stdout 'test output', got %v", response["stdout"])
		}
	}

	// Verify session_id field is populated
	if event.SessionID != "sess_abc123" {
		t.Errorf("Expected session_id 'sess_abc123', got '%s'", event.SessionID)
	}

	// Verify hook_event_name field is populated
	if event.HookEventName != testPostToolUseEvent {
		t.Errorf("Expected hook_event_name '%s', got '%s'", testPostToolUseEvent, event.HookEventName)
	}

	// Verify cwd field is populated
	if event.CWD != "/home/user/project" {
		t.Errorf("Expected cwd '/home/user/project', got '%s'", event.CWD)
	}
}

func TestDetectHookTypeWithSpecificEventName(t *testing.T) {
	_, _ = testutil.NewTestContext(t)
	t.Parallel()

	// Test that hook_event_name is used for reliable detection
	jsonData := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		},
		"tool_name": "Bash",
		"hook_event_name": "PostToolUse",
		"tool_response": {
			"stdout": "test output"
		}
	}`

	hookType, rawJSON, err := DetectHookType(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should detect as PostToolUse based on hook_event_name, not field presence
	if hookType != PostToolUseHook {
		t.Errorf("Expected hook type %v, got %v", PostToolUseHook, hookType)
	}

	if rawJSON == nil {
		t.Error("Expected raw JSON to be returned")
	}
}

// Fuzz test for hook JSON parsing
func FuzzParseInput(f *testing.F) {
	// Add valid seed inputs
	f.Add(`{"tool_input": {"command": "test"}}`)
	f.Add(`{"tool_input": {"command": "go test"}, "tool_name": "Bash"}`)
	f.Add(`{"tool_input": {"description": "test desc"}}`)

	f.Fuzz(func(_ *testing.T, input string) {
		reader := strings.NewReader(input)
		_, _ = ParseInput(reader)
		// No assertions - just ensuring no panics occur
	})
}
