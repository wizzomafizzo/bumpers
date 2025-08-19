package settings

import (
	"encoding/json"
	"testing"
)

func TestSettings_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	// Test that we can unmarshal the example settings.local.json structure
	jsonData := `{
		"permissions": {
			"allow": ["WebSearch", "Bash(find:*)"],
			"deny": [],
			"ask": []
		},
		"hooks": {
			"PreToolUse": [
				{
					"matcher": "Write|Edit|MultiEdit|TodoWrite",
					"hooks": [
						{
							"type": "command",
							"command": "tdd-guard"
						}
					]
				}
			]
		},
		"outputStyle": "Explanatory"
	}`

	var settings Settings
	err := json.Unmarshal([]byte(jsonData), &settings)
	if err != nil {
		t.Fatalf("Failed to unmarshal settings: %v", err)
	}

	// Verify permissions
	if settings.Permissions == nil {
		t.Fatal("Permissions should not be nil")
	}
	if len(settings.Permissions.Allow) != 2 {
		t.Errorf("Expected 2 allow permissions, got %d", len(settings.Permissions.Allow))
	}
	if settings.Permissions.Allow[0] != "WebSearch" {
		t.Errorf("Expected first allow permission to be 'WebSearch', got %s", settings.Permissions.Allow[0])
	}

	// Verify hooks
	if settings.Hooks == nil {
		t.Fatal("Hooks should not be nil")
	}
	if len(settings.Hooks.PreToolUse) != 1 {
		t.Errorf("Expected 1 PreToolUse hook matcher, got %d", len(settings.Hooks.PreToolUse))
	}

	matcher := settings.Hooks.PreToolUse[0]
	if matcher.Matcher != "Write|Edit|MultiEdit|TodoWrite" {
		t.Errorf("Expected matcher to be 'Write|Edit|MultiEdit|TodoWrite', got %s", matcher.Matcher)
	}

	if len(matcher.Hooks) != 1 {
		t.Errorf("Expected 1 hook command, got %d", len(matcher.Hooks))
	}

	hook := matcher.Hooks[0]
	if hook.Type != "command" {
		t.Errorf("Expected hook type to be 'command', got %s", hook.Type)
	}
	if hook.Command != "tdd-guard" {
		t.Errorf("Expected hook command to be 'tdd-guard', got %s", hook.Command)
	}

	// Verify output style
	if settings.OutputStyle != "Explanatory" {
		t.Errorf("Expected output style to be 'Explanatory', got %s", settings.OutputStyle)
	}
}

func TestSettings_PostToolUseHooks(t *testing.T) {
	t.Parallel()
	// Test that we can handle PostToolUse hooks
	jsonData := `{
		"hooks": {
			"PostToolUse": [
				{
					"matcher": "Bash",
					"hooks": [
						{
							"type": "command",
							"command": "echo 'Command executed'"
						}
					]
				}
			]
		}
	}`

	var settings Settings
	err := json.Unmarshal([]byte(jsonData), &settings)
	if err != nil {
		t.Fatalf("Failed to unmarshal settings: %v", err)
	}

	// Verify PostToolUse hooks
	if settings.Hooks == nil {
		t.Fatal("Hooks should not be nil")
	}
	if len(settings.Hooks.PostToolUse) != 1 {
		t.Errorf("Expected 1 PostToolUse hook matcher, got %d", len(settings.Hooks.PostToolUse))
	}

	matcher := settings.Hooks.PostToolUse[0]
	if matcher.Matcher != "Bash" {
		t.Errorf("Expected matcher to be 'Bash', got %s", matcher.Matcher)
	}
}

func TestSettings_UserPromptSubmitHooks(t *testing.T) {
	t.Parallel()
	// Test that we can handle UserPromptSubmit hooks
	jsonData := `{
		"hooks": {
			"UserPromptSubmit": [
				{
					"matcher": ".*",
					"hooks": [
						{
							"type": "command",
							"command": "echo 'User prompt submitted'"
						}
					]
				}
			]
		}
	}`

	var settings Settings
	err := json.Unmarshal([]byte(jsonData), &settings)
	if err != nil {
		t.Fatalf("Failed to unmarshal settings: %v", err)
	}

	// Verify UserPromptSubmit hooks
	if settings.Hooks == nil {
		t.Fatal("Hooks should not be nil")
	}
	if len(settings.Hooks.UserPromptSubmit) != 1 {
		t.Errorf("Expected 1 UserPromptSubmit hook matcher, got %d", len(settings.Hooks.UserPromptSubmit))
	}
}

func TestSettings_AllHookEvents(t *testing.T) {
	t.Parallel()
	// Test all hook events to ensure complete coverage
	jsonData := `{
		"hooks": {
			"SessionStart": [
				{
					"matcher": ".*",
					"hooks": [{"type": "command", "command": "session-start"}]
				}
			],
			"Stop": [
				{
					"matcher": ".*",
					"hooks": [{"type": "command", "command": "stop"}]
				}
			],
			"SubagentStop": [
				{
					"matcher": ".*",
					"hooks": [{"type": "command", "command": "subagent-stop"}]
				}
			],
			"PreCompact": [
				{
					"matcher": ".*",
					"hooks": [{"type": "command", "command": "pre-compact"}]
				}
			],
			"Notification": [
				{
					"matcher": ".*",
					"hooks": [{"type": "command", "command": "notification"}]
				}
			]
		}
	}`

	var settings Settings
	err := json.Unmarshal([]byte(jsonData), &settings)
	if err != nil {
		t.Fatalf("Failed to unmarshal settings: %v", err)
	}

	// Verify all hook events exist
	if settings.Hooks == nil {
		t.Fatal("Hooks should not be nil")
	}

	if len(settings.Hooks.SessionStart) != 1 {
		t.Errorf("Expected 1 SessionStart hook matcher, got %d", len(settings.Hooks.SessionStart))
	}
	if len(settings.Hooks.Stop) != 1 {
		t.Errorf("Expected 1 Stop hook matcher, got %d", len(settings.Hooks.Stop))
	}
	if len(settings.Hooks.SubagentStop) != 1 {
		t.Errorf("Expected 1 SubagentStop hook matcher, got %d", len(settings.Hooks.SubagentStop))
	}
	if len(settings.Hooks.PreCompact) != 1 {
		t.Errorf("Expected 1 PreCompact hook matcher, got %d", len(settings.Hooks.PreCompact))
	}
	if len(settings.Hooks.Notification) != 1 {
		t.Errorf("Expected 1 Notification hook matcher, got %d", len(settings.Hooks.Notification))
	}
}

func TestHookCommand_Timeout(t *testing.T) {
	t.Parallel()
	// Test that HookCommand supports timeout field
	jsonData := `{
		"hooks": {
			"PreToolUse": [
				{
					"matcher": ".*",
					"hooks": [
						{
							"type": "command",
							"command": "slow-command",
							"timeout": 30
						}
					]
				}
			]
		}
	}`

	var settings Settings
	err := json.Unmarshal([]byte(jsonData), &settings)
	if err != nil {
		t.Fatalf("Failed to unmarshal settings: %v", err)
	}

	// Verify timeout is preserved
	hook := settings.Hooks.PreToolUse[0].Hooks[0]
	if hook.Timeout != 30 {
		t.Errorf("Expected timeout to be 30, got %d", hook.Timeout)
	}
}
