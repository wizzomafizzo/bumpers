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
)

type SessionStartEvent struct {
	SessionID     string `json:"session_id"`      //nolint:tagliatelle // API uses snake_case
	HookEventName string `json:"hook_event_name"` //nolint:tagliatelle // API uses snake_case
	Source        string `json:"source"`
}

func (a *App) ProcessSessionStart(ctx context.Context, rawJSON json.RawMessage) (string, error) {
	logger := logging.Get(ctx)
	logger.Debug().Msg("processing SessionStart hook")

	// Parse the SessionStart JSON
	var event SessionStartEvent
	if err := json.Unmarshal(rawJSON, &event); err != nil {
		return "", fmt.Errorf("failed to parse SessionStart event: %w", err)
	}

	// Only process startup and clear sources
	if event.Source != constants.SessionSourceStartup && event.Source != constants.SessionSourceClear {
		return "", nil
	}

	// Clear session-based cache entries when a new session starts
	if cacheErr := a.clearSessionCache(ctx); cacheErr != nil {
		// Log error but don't fail the hook - cache clearing is non-critical
		logger.Warn().Err(cacheErr).Msg("failed to clear session cache")
	}

	// Load config to get notes
	cfg, err := config.Load(a.configPath)
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	// If no notes, return empty
	if len(cfg.Session) == 0 {
		return "", nil
	}

	// Process and concatenate all note messages
	messages := make([]string, 0, len(cfg.Session))
	for _, note := range cfg.Session {
		// Process template with note context including shared variables
		processedMessage, templateErr := template.ExecuteNoteTemplate(note.Add)
		if templateErr != nil {
			return "", fmt.Errorf("failed to process note template: %w", templateErr)
		}

		// Apply AI generation if configured
		finalMessage, genErr := a.processAIGenerationGeneric(ctx, &note, processedMessage, "")
		if genErr != nil {
			// Log error but don't fail the hook - fallback to original message
			logger.Error().Err(genErr).Msg("AI generation failed, using original message")
			finalMessage = processedMessage
		}

		messages = append(messages, finalMessage)
	}

	additionalContext := strings.Join(messages, "\n")

	// Create hook response that adds context
	response := HookSpecificOutput{
		HookEventName:     constants.SessionStartEvent,
		AdditionalContext: additionalContext,
	}

	// Wrap in hookSpecificOutput structure
	responseWrapper := map[string]any{
		"hookSpecificOutput": response,
	}

	// Convert to JSON
	responseJSON, err := json.Marshal(responseWrapper)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(responseJSON), nil
}
