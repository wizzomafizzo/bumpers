package hooks

import (
	"encoding/json"
	"fmt"
	"io"
)

type ToolInput struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type HookEvent struct {
	ToolInput ToolInput `json:"tool_input"` //nolint:tagliatelle // API uses snake_case
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
