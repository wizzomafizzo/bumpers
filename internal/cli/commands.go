package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/wizzomafizzo/bumpers/internal/config"
)

type UserPromptEvent struct {
	Prompt string `json:"prompt"`
}

type HookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

type HookResponse struct {
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput"`
}

func (a *App) ProcessUserPrompt(rawJSON json.RawMessage) (string, error) {
	// Parse the UserPromptSubmit JSON
	var event UserPromptEvent
	if err := json.Unmarshal(rawJSON, &event); err != nil {
		return "", fmt.Errorf("failed to parse UserPromptSubmit event: %w", err)
	}

	// Check if prompt starts with ! prefix
	if !strings.HasPrefix(event.Prompt, "!") {
		return "", nil // Not a command, pass through
	}

	// Extract command index
	commandStr := strings.TrimPrefix(event.Prompt, "!")
	commandIndex, err := strconv.Atoi(commandStr)
	if err != nil {
		// Invalid command format, pass through without processing
		return "", nil //nolint:nilerr // Intentional pass-through for non-command prompts
	}

	// Load config to get commands
	cfg, err := config.Load(a.configPath)
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	// Check if command index is valid
	if commandIndex < 0 || commandIndex >= len(cfg.Commands) {
		return "", nil // Invalid command index, pass through
	}

	// Create hook response with command message
	response := HookResponse{}
	response.HookSpecificOutput.HookEventName = "UserPromptSubmit"
	response.HookSpecificOutput.AdditionalContext = cfg.Commands[commandIndex].Message

	// Convert to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(responseJSON), nil
}
