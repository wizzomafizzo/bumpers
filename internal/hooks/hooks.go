package hooks

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/wizzomafizzo/bumpers/internal/constants"
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
		return constants.UserPromptSubmitEvent
	case PostToolUseHook:
		return "PostToolUse"
	case SessionStartHook:
		return constants.SessionStartEvent
	default:
		return "Unknown"
	}
}

type HookEvent struct {
	ToolInput      map[string]any `json:"tool_input"`
	ToolName       string         `json:"tool_name"`
	TranscriptPath string         `json:"transcript_path"`
	ToolUseID      string         `json:"tool_use_id"`
	ToolResponse   any            `json:"tool_response"`
	SessionID      string         `json:"session_id"`
	HookEventName  string         `json:"hook_event_name"`
	CWD            string         `json:"cwd"`
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

	// Check for hook_event_name first for more reliable detection
	if hookEventName, ok := generic["hook_event_name"]; ok {
		if eventName, ok := hookEventName.(string); ok {
			switch eventName {
			case "PreToolUse":
				return PreToolUseHook, json.RawMessage(data), nil
			case "PostToolUse":
				return PostToolUseHook, json.RawMessage(data), nil
			case constants.UserPromptSubmitEvent:
				return UserPromptSubmitHook, json.RawMessage(data), nil
			case constants.SessionStartEvent:
				return SessionStartHook, json.RawMessage(data), nil
			}
		}
	}

	// Fallback to field presence detection for compatibility
	if _, hasInput := generic["tool_input"]; hasInput {
		if _, hasResponse := generic[constants.FieldToolResponse]; hasResponse {
			return PostToolUseHook, json.RawMessage(data), nil
		}
		return PreToolUseHook, json.RawMessage(data), nil
	}
	if _, ok := generic[constants.FieldPrompt]; ok {
		return UserPromptSubmitHook, json.RawMessage(data), nil
	}
	if _, ok := generic[constants.FieldToolResponse]; ok {
		return PostToolUseHook, json.RawMessage(data), nil
	}

	return UnknownHook, json.RawMessage(data), nil
}
