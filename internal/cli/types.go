package cli

import "github.com/wizzomafizzo/bumpers/internal/config"

// UserPromptEvent represents a user prompt submission event
type UserPromptEvent struct {
	Prompt string `json:"prompt"`
}

// HookSpecificOutput represents the hook-specific output structure
type HookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`     //nolint:tagliatelle // Claude Code API format
	AdditionalContext string `json:"additionalContext"` //nolint:tagliatelle // Claude Code API format
}

// HookResponse wraps the hook specific output
type HookResponse struct {
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput"` //nolint:tagliatelle // Claude Code API format
}

// Decision represents the validation decision type
type Decision string

const (
	DecisionBlock Decision = "block"
	DecisionAllow Decision = "allow"
)

// ValidationResult represents a validation decision result
type ValidationResult struct {
	Decision   Decision `json:"decision,omitempty"`
	Reason     string   `json:"reason"`
	StopReason string   `json:"stopReason,omitempty"` //nolint:tagliatelle // Claude Code API format
	Continue   bool     `json:"continue,omitempty"`
}

// SessionStartEvent represents a session start event
type SessionStartEvent struct {
	SessionID     string `json:"session_id"`
	HookEventName string `json:"hook_event_name"`
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

// ProcessResult represents the result of processing a hook event
type ProcessResult struct {
	Mode    ProcessMode `json:"mode"`
	Message string      `json:"message"`
}

// ProcessMode defines how the CLI should respond to a hook event
type ProcessMode string

const (
	ProcessModeAllow         ProcessMode = "allow"         // Exit 0, no output
	ProcessModeInformational ProcessMode = "informational" // Exit 0, print message
	ProcessModeBlock         ProcessMode = "block"         // Exit 2, print message
)
