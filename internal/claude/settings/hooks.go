package settings

import (
	"errors"
	"fmt"
)

// HookEvent represents the valid hook event types.
type HookEvent string

const (
	PreToolUseEvent       HookEvent = "PreToolUse"
	PostToolUseEvent      HookEvent = "PostToolUse"
	UserPromptSubmitEvent HookEvent = "UserPromptSubmit"
	SessionStartEvent     HookEvent = "SessionStart"
	StopEvent             HookEvent = "Stop"
	SubagentStopEvent     HookEvent = "SubagentStop"
	PreCompactEvent       HookEvent = "PreCompact"
	NotificationEvent     HookEvent = "Notification"
)

// AddHook adds a new hook to the specified event in the settings.
func (s *Settings) AddHook(event HookEvent, matcher string, command HookCommand) error {
	// Validation
	if matcher == "" {
		return errors.New("matcher cannot be empty")
	}
	if command.Type == "" {
		return errors.New("command type cannot be empty")
	}
	if command.Command == "" {
		return errors.New("command cannot be empty")
	}

	// Initialize hooks if nil
	if s.Hooks == nil {
		s.Hooks = &Hooks{}
	}

	// Check for existing hook to prevent duplicates
	if existing, _ := s.FindHook(event, matcher); existing != nil {
		return fmt.Errorf("hook with matcher '%s' already exists for event '%s'", matcher, event)
	}

	// Create new matcher
	newMatcher := HookMatcher{
		Matcher: matcher,
		Hooks:   []HookCommand{command},
	}

	// Add to the appropriate event
	switch event {
	case PreToolUseEvent:
		s.Hooks.PreToolUse = append(s.Hooks.PreToolUse, newMatcher)
	case PostToolUseEvent:
		s.Hooks.PostToolUse = append(s.Hooks.PostToolUse, newMatcher)
	case UserPromptSubmitEvent:
		s.Hooks.UserPromptSubmit = append(s.Hooks.UserPromptSubmit, newMatcher)
	case SessionStartEvent:
		s.Hooks.SessionStart = append(s.Hooks.SessionStart, newMatcher)
	case StopEvent:
		s.Hooks.Stop = append(s.Hooks.Stop, newMatcher)
	case SubagentStopEvent:
		s.Hooks.SubagentStop = append(s.Hooks.SubagentStop, newMatcher)
	case PreCompactEvent:
		s.Hooks.PreCompact = append(s.Hooks.PreCompact, newMatcher)
	case NotificationEvent:
		s.Hooks.Notification = append(s.Hooks.Notification, newMatcher)
	}

	return nil
}

// RemoveHook removes a hook from the specified event and matcher.
func (s *Settings) RemoveHook(event HookEvent, matcher string) error {
	if s.Hooks == nil {
		return nil
	}

	// Find and remove the matcher from the appropriate event
	var hookMatchers *[]HookMatcher
	switch event {
	case PreToolUseEvent:
		hookMatchers = &s.Hooks.PreToolUse
	case PostToolUseEvent:
		hookMatchers = &s.Hooks.PostToolUse
	case UserPromptSubmitEvent:
		hookMatchers = &s.Hooks.UserPromptSubmit
	case SessionStartEvent:
		hookMatchers = &s.Hooks.SessionStart
	case StopEvent:
		hookMatchers = &s.Hooks.Stop
	case SubagentStopEvent:
		hookMatchers = &s.Hooks.SubagentStop
	case PreCompactEvent:
		hookMatchers = &s.Hooks.PreCompact
	case NotificationEvent:
		hookMatchers = &s.Hooks.Notification
	default:
		return fmt.Errorf("unsupported event: %s", event)
	}

	for i, hookMatcher := range *hookMatchers {
		if hookMatcher.Matcher == matcher {
			// Remove entire matcher
			*hookMatchers = append((*hookMatchers)[:i], (*hookMatchers)[i+1:]...)
			return nil
		}
	}

	return nil
}

// ListHooks returns all hooks for the specified event.
func (s *Settings) ListHooks(event HookEvent) ([]HookMatcher, error) {
	if s.Hooks == nil {
		return []HookMatcher{}, nil
	}

	switch event {
	case PreToolUseEvent:
		return s.Hooks.PreToolUse, nil
	case PostToolUseEvent:
		return s.Hooks.PostToolUse, nil
	case UserPromptSubmitEvent:
		return s.Hooks.UserPromptSubmit, nil
	case SessionStartEvent:
		return s.Hooks.SessionStart, nil
	case StopEvent:
		return s.Hooks.Stop, nil
	case SubagentStopEvent:
		return s.Hooks.SubagentStop, nil
	case PreCompactEvent:
		return s.Hooks.PreCompact, nil
	case NotificationEvent:
		return s.Hooks.Notification, nil
	}

	return []HookMatcher{}, nil
}

// FindHook finds a specific hook matcher for the given event and matcher pattern.
func (s *Settings) FindHook(event HookEvent, matcher string) (*HookMatcher, error) {
	if s.Hooks == nil {
		return nil, errors.New("no hooks configured")
	}

	var hookMatchers []HookMatcher
	switch event {
	case PreToolUseEvent:
		hookMatchers = s.Hooks.PreToolUse
	case PostToolUseEvent:
		hookMatchers = s.Hooks.PostToolUse
	case UserPromptSubmitEvent:
		hookMatchers = s.Hooks.UserPromptSubmit
	case SessionStartEvent:
		hookMatchers = s.Hooks.SessionStart
	case StopEvent:
		hookMatchers = s.Hooks.Stop
	case SubagentStopEvent:
		hookMatchers = s.Hooks.SubagentStop
	case PreCompactEvent:
		hookMatchers = s.Hooks.PreCompact
	case NotificationEvent:
		hookMatchers = s.Hooks.Notification
	default:
		return nil, fmt.Errorf("unsupported event: %s", event)
	}

	for _, hookMatcher := range hookMatchers {
		if hookMatcher.Matcher == matcher {
			return &hookMatcher, nil
		}
	}

	return nil, fmt.Errorf("hook not found for event %s with matcher %s", event, matcher)
}

// UpdateHook updates an existing hook by replacing the old matcher with a new one and command.
func (s *Settings) UpdateHook(event HookEvent, oldMatcher, newMatcher string, command HookCommand) error {
	if s.Hooks == nil {
		return errors.New("no hooks configured")
	}

	// Find and update the hook in the appropriate event
	var hookMatchers *[]HookMatcher
	switch event {
	case PreToolUseEvent:
		hookMatchers = &s.Hooks.PreToolUse
	case PostToolUseEvent:
		hookMatchers = &s.Hooks.PostToolUse
	case UserPromptSubmitEvent:
		hookMatchers = &s.Hooks.UserPromptSubmit
	case SessionStartEvent:
		hookMatchers = &s.Hooks.SessionStart
	case StopEvent:
		hookMatchers = &s.Hooks.Stop
	case SubagentStopEvent:
		hookMatchers = &s.Hooks.SubagentStop
	case PreCompactEvent:
		hookMatchers = &s.Hooks.PreCompact
	case NotificationEvent:
		hookMatchers = &s.Hooks.Notification
	default:
		return fmt.Errorf("unsupported event: %s", event)
	}

	for i, hookMatcher := range *hookMatchers {
		if hookMatcher.Matcher == oldMatcher {
			(*hookMatchers)[i] = HookMatcher{
				Matcher: newMatcher,
				Hooks:   []HookCommand{command},
			}
			return nil
		}
	}

	return errors.New("hook not found")
}
