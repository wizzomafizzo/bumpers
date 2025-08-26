package matcher

import (
	"errors"
	"fmt"
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/testing"
)

func TestRuleMatcher(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	rule := config.Rule{
		Match: "^go test",
		Send:  "Use just test instead",
	}

	matcher, err := NewRuleMatcher([]config.Rule{rule})
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	match, err := matcher.Match("go test ./...", "Bash")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if match == nil {
		t.Fatal("Expected match, got nil")
	}

	if match.Match != "^go test" {
		t.Errorf("Expected rule pattern '^go test', got %s", match.Match)
	}
}

func TestRuleMatcherNoMatch(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	rule := config.Rule{
		Match: "^go test",
		Send:  "Use just test instead",
	}

	matcher, err := NewRuleMatcher([]config.Rule{rule})
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	match, err := matcher.Match("make test", "Bash")
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

func TestRuleMatcherWithPartialConfig(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	yamlContent := `rules:
  - match: "go test.*"
    send: "Use just test instead"
  - match: "[invalid regex"
    send: "This rule has invalid regex"
  - match: "rm -rf"
    send: "Dangerous command - use safer alternatives"
`

	// Load partial config which filters out invalid rules
	partialConfig, err := config.LoadPartial([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Failed to load partial config: %v", err)
	}

	// Create matcher with the valid rules from partial config
	matcher, err := NewRuleMatcher(partialConfig.Rules)
	if err != nil {
		t.Fatalf("Expected NewRuleMatcher to work with valid rules from partial config, got %v", err)
	}

	// Test that matching works with filtered rules
	match, err := matcher.Match("go test ./...", "Bash")
	if err != nil {
		t.Fatalf("Expected no error matching, got %v", err)
	}

	if match == nil {
		t.Fatal("Expected match, got nil")
	}

	if match.Match != "go test.*" {
		t.Errorf("Expected rule pattern 'go test.*', got %s", match.Match)
	}

	// Test the second valid rule
	match, err = matcher.Match("rm -rf /tmp", "Bash")
	if err != nil {
		t.Fatalf("Expected no error matching rm, got %v", err)
	}

	if match == nil {
		t.Fatal("Expected rm match, got nil")
	}

	if match.Match != "rm -rf" {
		t.Errorf("Expected rule pattern 'rm -rf', got %s", match.Match)
	}
}

func TestRegexPatternMatching(t *testing.T) {
	testutil.InitTestLogger(t)
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
				Match: tc.pattern,
				Send:  "Test response",
			}

			matcher, matcherErr := NewRuleMatcher([]config.Rule{rule})
			if matcherErr != nil {
				t.Fatalf("Failed to create matcher: %v", matcherErr)
			}
			match, err := matcher.Match(tc.command, "Bash")

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
	testutil.InitTestLogger(t)
	t.Parallel()

	rule := config.Rule{
		Match: "[invalid-regex",
		Send:  "Test response",
	}

	_, err := NewRuleMatcher([]config.Rule{rule})
	if err == nil {
		t.Fatal("Expected error for invalid regex pattern")
	}

	if !errors.Is(err, ErrInvalidRegex) {
		t.Errorf("Expected ErrInvalidRegex, got %v", err)
	}
}

func TestRuleMatcherWithToolFiltering(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	rules := []config.Rule{
		{
			Match: "^rm -rf",
			Tool:  "^(Bash|Task)$",
			Send:  "Dangerous command",
		},
		{
			Match: "password",
			Tool:  "^(Write|Edit)$",
			Send:  "No secrets",
		},
		{
			Match: "test",
			Tool:  "", // Empty = defaults to Bash only
			Send:  "Bash test command",
		},
	}

	matcher, err := NewRuleMatcher(rules)
	if err != nil {
		t.Fatalf("Expected no error creating matcher, got %v", err)
	}

	// Test rm command matches Bash tool
	rule, err := matcher.Match("rm -rf /tmp", "Bash")
	if err != nil {
		t.Errorf("Expected match for Bash, got error: %v", err)
	}
	if rule == nil {
		t.Error("Expected rule but got nil")
	}

	// Test rm command does not match Write tool
	_, err = matcher.Match("rm -rf /tmp", "Write")
	if err == nil {
		t.Error("Expected no match for Write tool")
	}
	if !errors.Is(err, ErrNoRuleMatch) {
		t.Errorf("Expected ErrNoRuleMatch, got %v", err)
	}

	// Test empty tools field defaults to Bash only
	_, err = matcher.Match("test file", "Bash")
	if err != nil {
		t.Errorf("Expected match for Bash with empty tools field, got error: %v", err)
	}

	_, err = matcher.Match("test file", "Write")
	if err == nil {
		t.Error("Expected no match for Write with empty tools field")
	}
}

func TestBumpersYmlToolMatching(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	// This test specifically covers the bumpers.yml rule from the config
	rule := config.Rule{
		Match: "bumpers.yml",
		Tool:  "Read|Edit|Grep", // Pipe-separated OR pattern
		Send:  "Bumpers configuration file should not be accessed directly.",
	}

	matcher, err := NewRuleMatcher([]config.Rule{rule})
	if err != nil {
		t.Fatalf("Expected no error creating matcher, got %v", err)
	}

	// Test cases that should match
	shouldMatch := []struct {
		command  string
		toolName string
	}{
		{"bumpers.yml", "Read"},
		{"bumpers.yml", "Edit"},
		{"bumpers.yml", "Grep"},
		{"path/to/bumpers.yml", "Read"}, // substring match
		{"cat bumpers.yml", "Read"},     // command containing file
	}

	for _, tc := range shouldMatch {
		result, err := matcher.Match(tc.command, tc.toolName)
		if err != nil {
			t.Errorf("Expected match for command=%q tool=%q, got error: %v", tc.command, tc.toolName, err)
		}
		if result == nil {
			t.Errorf("Expected rule match for command=%q tool=%q, got nil", tc.command, tc.toolName)
		}
	}

	// Test cases that should NOT match
	shouldNotMatch := []struct {
		command  string
		toolName string
	}{
		{"bumpers.yml", "Bash"},  // Wrong tool
		{"bumpers.yml", "Write"}, // Wrong tool
		{"other.yml", "Read"},    // Wrong command
	}

	for _, tc := range shouldNotMatch {
		result, err := matcher.Match(tc.command, tc.toolName)
		if err == nil {
			t.Errorf("Expected no match for command=%q tool=%q, but got rule: %v", tc.command, tc.toolName, result)
		}
		if !errors.Is(err, ErrNoRuleMatch) {
			t.Errorf("Expected ErrNoRuleMatch for command=%q tool=%q, got: %v", tc.command, tc.toolName, err)
		}
	}
}

// Benchmark tests for critical performance paths
func BenchmarkRuleMatcherMatch(b *testing.B) {
	rule := config.Rule{
		Match: "^go test",
		Send:  "Use just test instead",
	}

	matcher, err := NewRuleMatcher([]config.Rule{rule})
	if err != nil {
		b.Fatalf("Failed to create matcher: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = matcher.Match("go test ./...", "Bash")
	}
}

func BenchmarkRuleMatcherMatchComplex(b *testing.B) {
	rules := []config.Rule{
		{Match: "^rm -rf", Tool: "^(Bash|Task)$", Send: "Dangerous command"},
		{Match: "password", Tool: "^(Write|Edit)$", Send: "No secrets"},
		{Match: "^go test", Tool: "", Send: "Use just test"},
		{Match: "(npm|yarn|pnpm)", Tool: "^Bash$", Send: "Use package manager"},
		{Match: "^git (add|commit)", Tool: "^Bash$", Send: "Review changes first"},
	}

	matcher, err := NewRuleMatcher(rules)
	if err != nil {
		b.Fatalf("Failed to create matcher: %v", err)
	}

	commands := []string{
		"go test ./...",
		"rm -rf /tmp",
		"npm install",
		"git add .",
		"echo hello",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := commands[i%len(commands)]
		_, _ = matcher.Match(cmd, "Bash")
	}
}

func BenchmarkNewRuleMatcher(b *testing.B) {
	rules := []config.Rule{
		{Match: "^rm -rf", Tool: "^(Bash|Task)$", Send: "Dangerous command"},
		{Match: "password", Tool: "^(Write|Edit)$", Send: "No secrets"},
		{Match: "^go test", Tool: "", Send: "Use just test"},
		{Match: "(npm|yarn|pnpm)", Tool: "^Bash$", Send: "Use package manager"},
		{Match: "^git (add|commit)", Tool: "^Bash$", Send: "Review changes first"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewRuleMatcher(rules)
	}
}

// Fuzz test for pattern matching
func FuzzRuleMatcherMatch(f *testing.F) {
	// Create a matcher with common patterns
	rules := []config.Rule{
		{Match: "^go test", Send: "Use just test"},
		{Match: "rm -rf", Send: "Dangerous command"},
		{Match: "(npm|yarn|pnpm)", Send: "Package manager"},
	}

	matcher, err := NewRuleMatcher(rules)
	if err != nil {
		f.Fatalf("Failed to create matcher: %v", err)
	}

	// Add seed inputs
	f.Add("go test ./...")
	f.Add("rm -rf /tmp")
	f.Add("npm install")
	f.Add("invalid command")

	f.Fuzz(func(_ *testing.T, command string) {
		_, _ = matcher.Match(command, "Bash")
		// No assertions - just ensuring no panics occur
	})
}

// TestMatcherBehaviorWithInvalidPatterns tests what happens when regex compilation fails
// This addresses mutation testing findings about missing continue vs break logic
func TestMatcherBehaviorWithInvalidPatterns(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	// Create rules where some have invalid patterns that would fail during matching
	rules := []config.Rule{
		{
			Match: "valid_pattern",
			Tool:  "[invalid-tool-regex", // Invalid tool regex - should skip this rule
			Send:  "Should be skipped",
		},
		{
			Match: "[invalid-cmd-regex", // Invalid command regex - should skip this rule
			Tool:  "^Bash$",
			Send:  "Should also be skipped",
		},
		{
			Match: "working_pattern",
			Tool:  "^Bash$",
			Send:  "Should match this one",
		},
	}

	// Test that invalid rules are properly skipped during matching
	// We need to manually create a matcher that bypasses validation to test runtime behavior
	testMatcher := &RuleMatcher{rules: rules}

	// This should still find the valid rule despite invalid ones being skipped
	match, err := testMatcher.Match("working_pattern test", "Bash")
	if err != nil {
		t.Fatalf("Expected match despite invalid patterns being skipped, got error: %v", err)
	}
	if match == nil {
		t.Fatal("Expected match despite invalid patterns, got nil")
	}
	if match.Send != "Should match this one" {
		t.Errorf("Expected 'Should match this one', got %q", match.Send)
	}
}

// Example demonstrates how to create a rule matcher and check commands
func ExampleNewRuleMatcher() {
	rules := []config.Rule{
		{Match: "^go test", Send: "Use 'just test' for better TDD integration"},
		{Match: "rm -rf", Tool: "^Bash$", Send: "Dangerous command - be more specific"},
		{Match: "password", Tool: "^(Write|Edit)$", Send: "Avoid hardcoding secrets"},
	}

	matcher, err := NewRuleMatcher(rules)
	if err != nil {
		_, _ = fmt.Printf("Error: %v\n", err)
		return
	}

	// Check a go test command
	match, err := matcher.Match("go test ./...", "Bash")
	if err != nil {
		_, _ = fmt.Printf("No match: %v\n", err)
	} else {
		_, _ = fmt.Printf("Rule matched: %s\n", match.Send)
	}

	// Check a command that doesn't match
	_, err = matcher.Match("echo hello", "Bash")
	_, _ = fmt.Printf("No rule match: %v\n", errors.Is(err, ErrNoRuleMatch))

	// Output:
	// Rule matched: Use 'just test' for better TDD integration
	// No rule match: true
}

func TestRuleMatcherWithTemplatePattern(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	rule := config.Rule{
		Match: "^{{.ProjectRoot}}/bumpers\\.yml$",
		Tool:  "Read|Edit|Grep",
		Send:  "Bumpers configuration file should not be accessed.",
	}

	matcher, err := NewRuleMatcher([]config.Rule{rule})
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	// Create template context with project root
	context := map[string]any{
		"ProjectRoot": "/home/user/project",
	}

	// Should match the templated path
	match, err := matcher.MatchWithContext("/home/user/project/bumpers.yml", "Read", context)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if match == nil {
		t.Fatal("Expected match, got nil")
	}

	if match.Send != "Bumpers configuration file should not be accessed." {
		t.Errorf("Expected correct message, got %s", match.Send)
	}

	// Should not match different paths
	_, err = matcher.MatchWithContext("/home/user/project/testdata/bumpers.yml", "Read", context)
	if !errors.Is(err, ErrNoRuleMatch) {
		t.Errorf("Expected no match for test file, got %v", err)
	}

	// Should not match without context (template processing should fail gracefully)
	_, err = matcher.MatchWithContext("/home/user/project/bumpers.yml", "Read", nil)
	if !errors.Is(err, ErrNoRuleMatch) {
		t.Errorf("Expected no match without context, got %v", err)
	}
}
