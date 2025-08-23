package matcher

import (
	"errors"
	"regexp"

	"github.com/wizzomafizzo/bumpers/internal/config"
)

var (
	ErrNoRuleMatch  = errors.New("no rule matched the command")
	ErrInvalidRegex = errors.New("invalid regex pattern")
)

func NewRuleMatcher(rules []config.Rule) (*RuleMatcher, error) {
	// Validate all patterns can be compiled as regex
	for _, rule := range rules {
		if err := validatePattern(rule.Pattern); err != nil {
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
	for i := range m.rules {
		// Filter rules by tool first
		toolPattern := m.rules[i].Tools
		if toolPattern == "" {
			toolPattern = "^Bash$" // Default to Bash only when empty
		}

		// Compile tools pattern with case-insensitive flag
		toolRe, err := regexp.Compile("(?i)" + toolPattern)
		if err != nil {
			continue // Skip rules with invalid tool patterns
		}

		// Skip rule if tool doesn't match
		if !toolRe.MatchString(toolName) {
			continue
		}

		// Now check if command matches
		cmdRe, err := regexp.Compile(m.rules[i].Pattern)
		if err != nil {
			continue
		}
		if cmdRe.MatchString(command) {
			return &m.rules[i], nil
		}
	}
	return nil, ErrNoRuleMatch
}
