package matcher

import (
	"errors"
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/config"
)

func TestRuleMatcher(t *testing.T) {
	t.Parallel()

	rule := config.Rule{
		Pattern:  "^go test",
		Response: "Use make test instead",
	}

	matcher, err := NewRuleMatcher([]config.Rule{rule})
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	match, err := matcher.Match("go test ./...")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if match == nil {
		t.Fatal("Expected match, got nil")
	}

	if match.Pattern != "^go test" {
		t.Errorf("Expected rule pattern '^go test', got %s", match.Pattern)
	}
}

func TestRuleMatcherNoMatch(t *testing.T) {
	t.Parallel()

	rule := config.Rule{
		Pattern:  "^go test",
		Response: "Use make test instead",
	}

	matcher, err := NewRuleMatcher([]config.Rule{rule})
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

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

func TestRegexPatternMatching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		command string
		matches bool
	}{
		// Regex exact patterns
		{"regex exact match", "^make test$", "make test", true},
		{"regex substring match", "make test", "run make test now", true},
		{"regex prefix with args", "^go test", "go test ./...", true},
		{"regex contains match", "go test", "make go test", true},
		{"regex no match", "make test", "npm install", false},

		// Regex prefix patterns
		{"regex prefix exact", "^go test", "go test", true},
		{"regex prefix with args", "^go test", "go test ./...", true},
		{"regex prefix with flags", "^go test", "go test -v", true},
		{"regex prefix no match", "^go test", "make go test", false},

		// OR operator with regex
		{"regex or match first", "(npm|yarn|pnpm)", "run npm install", true},
		{"regex or match npm", "(npm|yarn|pnpm)", "npm", true},
		{"regex or match yarn", "(npm|yarn|pnpm)", "yarn", true},
		{"regex or match pnpm", "(npm|yarn|pnpm)", "pnpm", true},
		{"regex or no match", "(npm|yarn|pnpm)", "bun", false},

		// Combined regex patterns with alternation
		{"regex prefix or npm", "^(npm|yarn|pnpm) ", "npm install", true},
		{"regex prefix or yarn", "^(npm|yarn|pnpm) ", "yarn add", true},
		{"regex prefix or pnpm", "^(npm|pnpm) ", "pnpm update", true},
		{"regex prefix or no match bare", "^(npm|yarn|pnpm) ", "npm", false},
		{"regex prefix or no match other", "^(npm|yarn|pnpm) ", "bun install", false},

		// Dangerous rm patterns with regex
		{"dangerous rm prefix match 1", "^rm -rf /", "rm -rf /", true},
		{"dangerous rm prefix match 2", "^rm -rf /", "rm -rf /usr", true},
		{"dangerous rm prefix match 3", "^rm -fr /", "rm -fr /home", true},
		{"dangerous rm no match relative", "^rm -rf /", "rm -rf ./", false},
		{"dangerous rm no match simple", "^rm -rf /", "rm /tmp/file", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rule := config.Rule{
				Pattern:  tc.pattern,
				Response: "Test response",
			}

			matcher, matcherErr := NewRuleMatcher([]config.Rule{rule})
			if matcherErr != nil {
				t.Fatalf("Failed to create matcher: %v", matcherErr)
			}
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

func TestNewRuleMatcherInvalidRegex(t *testing.T) {
	t.Parallel()

	rule := config.Rule{
		Pattern:  "[invalid-regex",
		Response: "Test response",
	}

	_, err := NewRuleMatcher([]config.Rule{rule})
	if err == nil {
		t.Fatal("Expected error for invalid regex pattern")
	}

	if !errors.Is(err, ErrInvalidRegex) {
		t.Errorf("Expected ErrInvalidRegex, got %v", err)
	}
}
