package matcher

import (
	"errors"
	"regexp"

	"github.com/wizzomafizzo/bumpers/internal/config"
)

var ErrNoRuleMatch = errors.New("no rule matched the command")

func NewRuleMatcher(rules []config.Rule) *RuleMatcher {
	return &RuleMatcher{rules: rules}
}

type RuleMatcher struct {
	rules []config.Rule
}

func (m *RuleMatcher) Match(command string) (*config.Rule, error) {
	for _, rule := range m.rules {
		matched, err := regexp.MatchString(rule.Pattern, command)
		if err != nil {
			return nil, err //nolint:wrapcheck // Regex errors include pattern context
		}
		if matched {
			return &rule, nil
		}
	}
	return nil, ErrNoRuleMatch
}
