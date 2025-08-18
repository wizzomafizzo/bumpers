package matcher

import (
	"regexp"
	
	"github.com/wizzomafizzo/bumpers/internal/config"
)

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
			return nil, err
		}
		if matched {
			return &rule, nil
		}
	}
	return nil, nil
}