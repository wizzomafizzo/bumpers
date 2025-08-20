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
	for i := range m.rules {
		matched := matchPattern(m.rules[i].Pattern, command)
		if matched {
			return &m.rules[i], nil
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
		if matchSinglePattern(p, command) {
			return true
		}
	}
	return false
}

// matchSinglePattern matches a single pattern against a command
func matchSinglePattern(pattern, command string) bool {
	// Check if pattern contains wildcards
	if strings.Contains(pattern, "*") {
		return matchWildcardPattern(pattern, command)
	}
	// Partial match (contains)
	return strings.Contains(command, pattern)
}

// matchWildcardPattern handles glob patterns with wildcards
func matchWildcardPattern(pattern, command string) bool {
	// Convert glob pattern to prefix matching for commands
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(command, prefix)
	}
	// Use filepath.Match for more complex glob patterns
	matched, err := filepath.Match(pattern, command)
	if err != nil {
		// If glob pattern is invalid, fall back to exact match
		return pattern == command
	}
	return matched
}
