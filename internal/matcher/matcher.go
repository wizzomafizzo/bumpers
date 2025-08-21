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

func (m *RuleMatcher) Match(command string) (*config.Rule, error) {
	for i := range m.rules {
		re, err := regexp.Compile(m.rules[i].Pattern)
		if err != nil {
			continue
		}
		if re.MatchString(command) {
			return &m.rules[i], nil
		}
	}
	return nil, ErrNoRuleMatch
}
