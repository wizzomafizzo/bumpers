package hooks

import (
	"encoding/json"
	"fmt"
	"io"
)

type HookType int

const (
	UnknownHook HookType = iota
	PreToolUseHook
	UserPromptSubmitHook
	PostToolUseHook
	SessionStartHook
)

type HookEvent struct {
	ToolInput map[string]any `json:"tool_input"` //nolint:tagliatelle // API uses snake_case
	ToolName  string         `json:"tool_name"`  //nolint:tagliatelle // API uses snake_case
}

func ParseInput(reader io.Reader) (*HookEvent, error) {
	var event HookEvent
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&event)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hook input JSON: %w", err)
	}
	return &event, nil
}

func DetectHookType(reader io.Reader) (HookType, json.RawMessage, error) {
	// Read all data from reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return UnknownHook, nil, fmt.Errorf("failed to read hook input: %w", err)
	}

	// Parse JSON to check for distinctive fields
	var generic map[string]any
	if err := json.Unmarshal(data, &generic); err != nil {
		return UnknownHook, nil, fmt.Errorf("failed to parse hook input JSON: %w", err)
	}

	// Check for distinctive fields to determine hook type
	// PostToolUse has both tool_input and tool_output, so check for it first
	if _, hasInput := generic["tool_input"]; hasInput {
		if _, hasOutput := generic["tool_output"]; hasOutput {
			return PostToolUseHook, json.RawMessage(data), nil
		}
		return PreToolUseHook, json.RawMessage(data), nil
	}
	if _, ok := generic["prompt"]; ok {
		return UserPromptSubmitHook, json.RawMessage(data), nil
	}
	if _, ok := generic["tool_output"]; ok {
		return PostToolUseHook, json.RawMessage(data), nil
	}
	if hookEventName, ok := generic["hook_event_name"]; ok {
		if eventName, ok := hookEventName.(string); ok && eventName == "SessionStart" {
			return SessionStartHook, json.RawMessage(data), nil
		}
	}

	return UnknownHook, json.RawMessage(data), nil
}
