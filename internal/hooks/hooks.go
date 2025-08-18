package hooks

import (
	"encoding/json"
	"io"
)

type HookEvent struct {
	Command string   `json:"command"`
	Cwd     string   `json:"cwd"`
	Args    []string `json:"args"`
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
