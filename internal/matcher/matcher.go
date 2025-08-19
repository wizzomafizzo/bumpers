package matcher

import (
	"errors"
	"path/filepath"
	"strings"

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
		matched := matchPattern(rule.Pattern, command)
		if matched {
			return &rule, nil
		}
	}
	return nil, ErrNoRuleMatch
}

// matchPattern handles glob-style patterns with OR operations
func matchPattern(pattern, command string) bool {
	// Split pattern by OR operator
	orPatterns := strings.Split(pattern, "|")
	
	for _, p := range orPatterns {
		p = strings.TrimSpace(p)
		
		// Check if pattern contains wildcards
		if strings.Contains(p, "*") {
			// Convert glob pattern to prefix matching for commands
			if strings.HasSuffix(p, "*") {
				prefix := strings.TrimSuffix(p, "*")
				if strings.HasPrefix(command, prefix) {
					return true
				}
			} else {
				// Use filepath.Match for more complex glob patterns
				matched, err := filepath.Match(p, command)
				if err != nil {
					// If glob pattern is invalid, fall back to exact match
					return p == command
				}
				if matched {
					return true
				}
			}
		} else {
			// Partial match (contains)
			if strings.Contains(command, p) {
				return true
			}
		}
	}
	return false
}
