package cli

import "github.com/wizzomafizzo/bumpers/internal/config"

// UserPromptEvent represents a user prompt submission event
type UserPromptEvent struct {
	Prompt string `json:"prompt"`
}

// HookSpecificOutput represents the hook-specific output structure
type HookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// HookResponse wraps the hook specific output
type HookResponse struct {
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput"`
}

// ValidationResult represents a validation decision result
type ValidationResult struct {
	Decision   any    `json:"decision,omitempty"`
	Reason     string `json:"reason"`
	StopReason string `json:"stopReason,omitempty"`
	Continue   bool   `json:"continue,omitempty"`
}

// SessionStartEvent represents a session start event
type SessionStartEvent struct {
	SessionID     string `json:"session_id"`      //nolint:tagliatelle // API uses snake_case
	HookEventName string `json:"hook_event_name"` //nolint:tagliatelle // API uses snake_case
	Source        string `json:"source"`
}

// GenerateConfig interface for types that have GetGenerate method
type GenerateConfig interface {
	GetGenerate() config.Generate
}

// postToolContent contains the content extracted from post-tool-use events
type postToolContent struct {
	intent        string
	toolOutputMap map[string]any
	toolName      string
}
