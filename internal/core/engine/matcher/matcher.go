package matcher

import (
	"errors"
	"regexp"

	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/core/messaging/template"
)

var (
	ErrNoRuleMatch  = errors.New("no rule matched the command")
	ErrInvalidRegex = errors.New("invalid regex pattern")
)

func NewRuleMatcher(rules []config.Rule) (*RuleMatcher, error) {
	// Validate all patterns can be compiled as regex
	for i := range rules {
		if err := validatePattern(rules[i].GetMatch().Pattern); err != nil {
			return nil, err
		}
	}

	return &RuleMatcher{rules: rules}, nil
}

// validatePattern checks if a pattern can be used for matching
func validatePattern(pattern string) error {
	// Try to compile as regex to validate syntax
	if _, err := regexp.Compile(pattern); err != nil {
		return ErrInvalidRegex
	}
	return nil
}

type RuleMatcher struct {
	rules []config.Rule
}

func (m *RuleMatcher) Match(command, toolName string) (*config.Rule, error) {
	return m.MatchWithContext(command, toolName, nil)
}

func (m *RuleMatcher) MatchWithContext(command, toolName string, context map[string]any) (*config.Rule, error) {
	for i := range m.rules {
		if m.matchesRule(command, toolName, context, &m.rules[i]) {
			return &m.rules[i], nil
		}
	}
	return nil, ErrNoRuleMatch
}

// matchesRule checks if a single rule matches the given command and tool
func (*RuleMatcher) matchesRule(command, toolName string, context map[string]any, rule *config.Rule) bool {
	// Filter rules by tool first
	toolPattern := rule.Tool
	if toolPattern == "" {
		toolPattern = "^Bash$" // Default to Bash only when empty
	}

	// Compile tools pattern with case-insensitive flag
	toolRe, err := regexp.Compile("(?i)" + toolPattern)
	if err != nil {
		return false // Skip rules with invalid tool patterns
	}

	// Skip rule if tool doesn't match
	if !toolRe.MatchString(toolName) {
		return false
	}

	// Now check if command matches
	pattern := rule.GetMatch().Pattern

	// Process template if context provided
	if context != nil {
		if processedPattern, templateErr := template.Execute(pattern, context); templateErr == nil {
			pattern = processedPattern
		}
	}

	cmdRe, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return cmdRe.MatchString(command)
}
