package hooks

import (
	"encoding/json"
	"io"
)

type HookEvent struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Cwd     string   `json:"cwd"`
}

func ParseHookInput(reader io.Reader) (*HookEvent, error) {
	var event HookEvent
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}