package hooks

import (
	"encoding/json"
	"io"
)

type ToolInput struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type HookEvent struct {
	ToolInput ToolInput `json:"tool_input"` //nolint:tagliatelle // API uses snake_case
}

func ParseHookInput(reader io.Reader) (*HookEvent, error) {
	var event HookEvent
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&event)
	if err != nil {
		return nil, err //nolint:wrapcheck // JSON decode errors are self-descriptive
	}
	return &event, nil
}
