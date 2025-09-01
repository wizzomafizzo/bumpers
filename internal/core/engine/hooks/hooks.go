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

// String returns a human-readable string representation of the hook type
func (h HookType) String() string {
	switch h {
	case UnknownHook:
		const unknownType = "Unknown"
		return unknownType
	case PreToolUseHook:
		return "PreToolUse"
	case UserPromptSubmitHook:
		return "UserPromptSubmit"
	case PostToolUseHook:
		return "PostToolUse"
	case SessionStartHook:
		return "SessionStart"
	default:
		return "Unknown"
	}
}

type HookEvent struct {
	ToolInput      map[string]any `json:"tool_input"`
	ToolName       string         `json:"tool_name"`
	TranscriptPath string         `json:"transcript_path"`
	ToolUseID      string         `json:"tool_use_id"`
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
	// PostToolUse has both tool_input and tool_response, so check for it first
	if _, hasInput := generic["tool_input"]; hasInput {
		if _, hasResponse := generic["tool_response"]; hasResponse {
			return PostToolUseHook, json.RawMessage(data), nil
		}
		return PreToolUseHook, json.RawMessage(data), nil
	}
	if _, ok := generic["prompt"]; ok {
		return UserPromptSubmitHook, json.RawMessage(data), nil
	}
	if _, ok := generic["tool_response"]; ok {
		return PostToolUseHook, json.RawMessage(data), nil
	}
	if hookEventName, ok := generic["hook_event_name"]; ok {
		if eventName, ok := hookEventName.(string); ok && eventName == "SessionStart" {
			return SessionStartHook, json.RawMessage(data), nil
		}
	}

	return UnknownHook, json.RawMessage(data), nil
}
