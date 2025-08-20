package matcher

import (
	"errors"
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/config"
)

func TestRuleMatcher(t *testing.T) {
	t.Parallel()

	rule := config.Rule{
		Pattern:  "go test*",
		Response: "Use make test instead",
	}

	matcher := NewRuleMatcher([]config.Rule{rule})

	match, err := matcher.Match("go test ./...")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if match == nil {
		t.Fatal("Expected match, got nil")
	}

	if match.Pattern != "go test*" {
		t.Errorf("Expected rule pattern 'go test*', got %s", match.Pattern)
	}
}

func TestRuleMatcherNoMatch(t *testing.T) {
	t.Parallel()

	rule := config.Rule{
		Pattern:  "go test*",
		Response: "Use make test instead",
	}

	matcher := NewRuleMatcher([]config.Rule{rule})

	match, err := matcher.Match("make test")
	if err == nil {
		t.Fatal("Expected ErrNoRuleMatch error, got nil")
	}

	if !errors.Is(err, ErrNoRuleMatch) {
		t.Errorf("Expected ErrNoRuleMatch, got %v", err)
	}

	if match != nil {
		t.Errorf("Expected no match, got %v", match)
	}
}

func TestGlobPatternMatching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		command string
		matches bool
	}{
		// Partial match (default behavior)
		{"partial match exact", "make test", "make test", true},
		{"partial match within command", "make test", "run make test now", true},
		{"partial match with args", "go test", "go test ./...", true},
		{"partial match prefix", "go test", "make go test", true},
		{"partial no match", "make test", "npm install", false},

		// Glob wildcard pattern (explicit)
		{"glob wildcard start", "go test*", "go test", true},
		{"glob wildcard with args", "go test*", "go test ./...", true},
		{"glob wildcard with flags", "go test*", "go test -v", true},
		{"glob wildcard no match prefix", "go test*", "make go test", false},

		// OR operator
		{"or partial match within command", "npm|yarn|pnpm", "run npm install", true},
		{"or simple match first", "npm|yarn|pnpm", "npm", true},
		{"or simple match second", "npm|yarn|pnpm", "yarn", true},
		{"or simple match third", "npm|yarn|pnpm", "pnpm", true},
		{"or simple no match", "npm|yarn|pnpm", "bun", false},

		// Combined patterns
		{"or with wildcards first", "npm *|yarn *|pnpm *", "npm install", true},
		{"or with wildcards second", "npm *|yarn *|pnpm *", "yarn add", true},
		{"or with wildcards third", "npm *|yarn *|pnpm *", "pnpm update", true},
		{"or with wildcards no match simple", "npm *|yarn *|pnpm *", "npm", false},
		{"or with wildcards no match other", "npm *|yarn *|pnpm *", "bun install", false},

		// Dangerous rm patterns
		{"dangerous rm match 1", "rm -rf /*|rm -fr /*", "rm -rf /", true},
		{"dangerous rm match 2", "rm -rf /*|rm -fr /*", "rm -rf /usr", true},
		{"dangerous rm match 3", "rm -rf /*|rm -fr /*", "rm -fr /home", true},
		{"dangerous rm no match relative", "rm -rf /*|rm -fr /*", "rm -rf ./", false},
		{"dangerous rm no match simple", "rm -rf /*|rm -fr /*", "rm /tmp/file", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rule := config.Rule{
				Pattern:  tc.pattern,
				Response: "Test response",
			}

			matcher := NewRuleMatcher([]config.Rule{rule})
			match, err := matcher.Match(tc.command)

			if tc.matches {
				assertMatch(t, match, err)
			} else {
				assertNoMatch(t, match, err)
			}
		})
	}
}

func assertMatch(t *testing.T, match *config.Rule, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected match but got error: %v", err)
	}
	if match == nil {
		t.Fatal("Expected match but got nil")
	}
}

func assertNoMatch(t *testing.T, match *config.Rule, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected no match but got one")
	}
	if !errors.Is(err, ErrNoRuleMatch) {
		t.Errorf("Expected ErrNoRuleMatch, got %v", err)
	}
	if match != nil {
		t.Errorf("Expected no match, got %v", match)
	}
}
