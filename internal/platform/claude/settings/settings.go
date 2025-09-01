// Package settings provides programmatic access to Claude settings.json files.
package settings

// Schema: https://www.schemastore.org/claude-code-settings.json

// Settings represents the complete Claude settings.json structure.
type Settings struct {
	Permissions *Permissions `json:"permissions,omitempty"`
	Hooks       *Hooks       `json:"hooks,omitempty"`
	OutputStyle string       `json:"outputStyle,omitempty"` //nolint:tagliatelle // Claude settings.json format
	Model       string       `json:"model,omitempty"`
}

// Permissions defines the Claude permission system configuration.
type Permissions struct {
	Allow []string `json:"allow,omitempty"`
}

// Hooks defines the complete hooks configuration structure.
type Hooks struct {
	PreToolUse       []HookMatcher `json:"PreToolUse,omitempty"`       //nolint:tagliatelle // Claude uses PascalCase
	PostToolUse      []HookMatcher `json:"PostToolUse,omitempty"`      //nolint:tagliatelle // Claude uses PascalCase
	UserPromptSubmit []HookMatcher `json:"UserPromptSubmit,omitempty"` //nolint:tagliatelle // Claude uses PascalCase
	SessionStart     []HookMatcher `json:"SessionStart,omitempty"`     //nolint:tagliatelle // Claude uses PascalCase
	Stop             []HookMatcher `json:"Stop,omitempty"`             //nolint:tagliatelle // Claude uses PascalCase
	SubagentStop     []HookMatcher `json:"SubagentStop,omitempty"`     //nolint:tagliatelle // Claude uses PascalCase
	PreCompact       []HookMatcher `json:"PreCompact,omitempty"`       //nolint:tagliatelle // Claude uses PascalCase
	Notification     []HookMatcher `json:"Notification,omitempty"`     //nolint:tagliatelle // Claude uses PascalCase
}

// HookMatcher represents a single matcher within a hook event.
type HookMatcher struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []HookCommand `json:"hooks"`
}

// HookCommand represents a single command to execute when a hook matches.
type HookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// ValidationResult contains the results of a validation operation.
type ValidationResult struct {
	Errors []string
	Valid  bool
}

// Validate performs basic validation on the settings structure.
func (s *Settings) Validate() *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: []string{},
	}

	s.validateOutputStyle(result)
	s.validateHooks(result)

	return result
}

func (s *Settings) validateOutputStyle(result *ValidationResult) {
	if s.OutputStyle == "" {
		return
	}

	validStyles := []string{"default", "explanatory", "minimal", "creative"}
	for _, style := range validStyles {
		if s.OutputStyle == style {
			return
		}
	}

	result.Valid = false
	result.Errors = append(result.Errors, "invalid output style")
}

func (s *Settings) validateHooks(result *ValidationResult) {
	if s.Hooks == nil {
		return
	}

	hookEvents := [][]HookMatcher{
		s.Hooks.PreToolUse, s.Hooks.PostToolUse, s.Hooks.UserPromptSubmit,
		s.Hooks.SessionStart, s.Hooks.Stop, s.Hooks.SubagentStop,
		s.Hooks.PreCompact, s.Hooks.Notification,
	}

	for _, matchers := range hookEvents {
		for _, matcher := range matchers {
			s.validateHookCommands(matcher.Hooks, result)
		}
	}
}

func (*Settings) validateHookCommands(commands []HookCommand, result *ValidationResult) {
	for _, cmd := range commands {
		if cmd.Type == "" {
			result.Valid = false
			result.Errors = append(result.Errors, "hook command type cannot be empty")
		}
		if cmd.Command == "" {
			result.Valid = false
			result.Errors = append(result.Errors, "hook command cannot be empty")
		}
	}
}
