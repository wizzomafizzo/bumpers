package internal

import (
	"errors"
	"strings"
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/hooks"
	"github.com/wizzomafizzo/bumpers/internal/matcher"
)

func TestEndToEndHookProcessing(t *testing.T) {
	t.Parallel()

	// Create config in memory
	configContent := `rules:
  - pattern: "go test"
    response: "Use make test instead for better TDD integration"
    alternatives:
      - "make test          # Run all tests"
      - "make test-unit     # Run unit tests only"
    use_claude: false`

	cfg, err := config.LoadFromYAML([]byte(configContent))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create matcher
	ruleMatcher, err := matcher.NewRuleMatcher(cfg.Rules)
	if err != nil {
		t.Fatalf("Failed to create rule matcher: %v", err)
	}

	// Simulate hook input
	hookInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		}
	}`

	// Parse hook event
	event, err := hooks.ParseHookInput(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Failed to parse hook input: %v", err)
	}

	// Match rule
	rule, err := ruleMatcher.Match(event.ToolInput.Command)
	if err != nil {
		if errors.Is(err, matcher.ErrNoRuleMatch) {
			t.Fatal("Expected rule match for go test command, but got no match")
		}
		t.Fatalf("Failed to match rule: %v", err)
	}

	if rule == nil {
		t.Fatal("Expected rule match for go test command")
	}

	// Generate response
	resp := rule.Response
	if resp == "" {
		t.Fatal("Expected non-empty response")
	}

	if !strings.Contains(resp, "make test") {
		t.Error("Response should suggest make test alternative")
	}
}
