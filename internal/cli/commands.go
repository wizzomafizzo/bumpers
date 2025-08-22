package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/constants"
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

type ValidationResult struct {
	Decision   interface{} `json:"decision,omitempty"`
	Reason     string      `json:"reason"`
	StopReason string      `json:"stopReason,omitempty"`
	Continue   bool        `json:"continue,omitempty"`
}

func (a *App) ProcessUserPrompt(rawJSON json.RawMessage) (string, error) {
	// Parse the UserPromptSubmit JSON
	var event UserPromptEvent
	if err := json.Unmarshal(rawJSON, &event); err != nil {
		log.Error().Err(err).Msg("Failed to parse UserPromptSubmit event")
		return "", fmt.Errorf("failed to parse UserPromptSubmit event: %w", err)
	}

	log.Info().Str("prompt", event.Prompt).Msg("Processing UserPromptSubmit with prompt")

	// Check if prompt starts with command prefix
	if !strings.HasPrefix(event.Prompt, constants.CommandPrefix) {
		log.Info().Str("prompt", event.Prompt).Msg("Prompt does not start with command prefix, passing through")
		return "", nil // Not a command, pass through
	}

	// Extract command index
	commandStr := strings.TrimPrefix(event.Prompt, constants.CommandPrefix)
	log.Info().Str("commandStr", commandStr).Msg("Extracted command string")

	// Load config to get commands
	cfg, err := config.Load(a.configPath)
	if err != nil {
		log.Error().Err(err).Str("configPath", a.configPath).Msg("Failed to load config")
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	log.Info().Int("totalCommands", len(cfg.Commands)).Msg("Loaded config")

	// Find command by name
	var commandMessage string
	var foundCommand bool

	for _, cmd := range cfg.Commands {
		if cmd.Name == commandStr {
			commandMessage = cmd.Message
			foundCommand = true
			log.Info().Str("commandName", commandStr).Str("message", commandMessage).Msg("Found valid command")
			break
		}
	}

	if !foundCommand {
		log.Info().Str("commandStr", commandStr).Msg("Command not found, passing through")
		return "", nil // Command not found, pass through
	}

	// Create hook response that replaces the prompt and continues processing
	response := HookSpecificOutput{
		HookEventName:     "UserPromptSubmit",
		AdditionalContext: commandMessage,
	}

	// Wrap in hookSpecificOutput structure as required by Claude Code hook specification
	responseWrapper := map[string]interface{}{
		"hookSpecificOutput": response,
	}

	// Convert to JSON
	responseJSON, err := json.Marshal(responseWrapper)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal response")
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	log.Info().Str("response", string(responseJSON)).Msg("Returning ValidationResult response")
	return string(responseJSON), nil
}
