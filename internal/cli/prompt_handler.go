package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/core/logging"
	"github.com/wizzomafizzo/bumpers/internal/core/messaging/template"
	"github.com/wizzomafizzo/bumpers/internal/infrastructure/constants"
	ai "github.com/wizzomafizzo/bumpers/internal/platform/claude/api"
)

// PromptHandler handles user prompt processing and command generation
type PromptHandler interface {
	ProcessUserPrompt(ctx context.Context, rawJSON json.RawMessage) (string, error)
}

// DefaultPromptHandler implements PromptHandler
type DefaultPromptHandler struct {
	aiHelper   *AIHelper
	configPath string
}

// NewPromptHandler creates a new PromptHandler
func NewPromptHandler(configPath, projectRoot string) *DefaultPromptHandler {
	return &DefaultPromptHandler{
		configPath: configPath,
		aiHelper:   NewAIHelper(projectRoot, nil, nil),
	}
}

// SetMockAIGenerator sets a mock AI generator for testing
func (p *DefaultPromptHandler) SetMockAIGenerator(generator ai.MessageGenerator) {
	p.aiHelper.aiGenerator = generator
}

func (p *DefaultPromptHandler) ProcessUserPrompt(ctx context.Context, rawJSON json.RawMessage) (string, error) {
	logger := logging.Get(ctx)

	// Parse the UserPromptSubmit JSON
	var event UserPromptEvent
	if err := json.Unmarshal(rawJSON, &event); err != nil {
		logger.Error().Err(err).Msg("Failed to parse UserPromptSubmit event")
		return "", fmt.Errorf("failed to parse UserPromptSubmit event: %w", err)
	}

	logger.Debug().Str("prompt", event.Prompt).Msg("processing UserPromptSubmit with prompt")

	// Check if prompt starts with command prefix
	if !strings.HasPrefix(event.Prompt, constants.CommandPrefix) {
		return "", nil // Not a command, pass through
	}

	// Extract command string and parse arguments
	commandStr := strings.TrimPrefix(event.Prompt, constants.CommandPrefix)
	logger.Debug().Str("command_str", commandStr).Msg("extracted command string")

	// Parse command name and arguments
	commandName, args, argv := ParseCommandArgs(commandStr)
	logger.Debug().
		Str("command_name", commandName).
		Str("args", args).
		Int("argc", len(argv)-1).
		Msg("parsed command arguments")

	// Load config to get commands
	cfg, err := config.Load(p.configPath)
	if err != nil {
		logger.Error().Err(err).Str("config_path", p.configPath).Msg("Failed to load config")
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	// Find command by name
	var commandMessage string
	var foundCommand bool
	var matchedCommand *config.Command

	for _, cmd := range cfg.Commands {
		if cmd.Name != commandName {
			continue
		}
		commandMessage = cmd.Send
		foundCommand = true
		matchedCommand = &cmd
		logger.Debug().Str("commandName", commandName).Str("message", commandMessage).Msg("found valid command")
		break
	}

	if !foundCommand {
		return "", nil // Command not found, pass through
	}

	// Process template with command context including shared variables and arguments
	processedMessage, err := template.ExecuteCommandTemplateWithArgs(commandMessage, commandName, args, argv)
	if err != nil {
		logger.Error().Err(err).Str("commandName", commandName).Msg("Failed to process command template")
		return "", fmt.Errorf("failed to process command template: %w", err)
	}

	// Apply AI generation if configured
	finalMessage, err := p.aiHelper.ProcessAIGenerationGeneric(ctx, matchedCommand, processedMessage, commandStr)
	if err != nil {
		// Log error but don't fail the hook - fallback to original message
		logger.Error().Err(err).Msg("AI generation failed, using original message")
		finalMessage = processedMessage
	}

	// Create hook response that replaces the prompt and continues processing
	response := HookSpecificOutput{
		HookEventName:     constants.UserPromptSubmitEvent,
		AdditionalContext: finalMessage,
	}

	// Wrap in hookSpecificOutput structure as required by Claude Code hook specification
	responseWrapper := map[string]any{
		"hookSpecificOutput": response,
	}

	// Convert to JSON
	responseJSON, err := json.Marshal(responseWrapper)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to marshal response")
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	logger.Info().Str("response", string(responseJSON)).Msg("Returning ValidationResult response")
	return string(responseJSON), nil
}
