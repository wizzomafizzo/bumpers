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
  - match: "go test"
    send: "Use just test instead for better TDD integration"`

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
	event, err := hooks.ParseInput(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Failed to parse hook input: %v", err)
	}

	// Match rule - get command from the map
	command, ok := event.ToolInput["command"].(string)
	if !ok {
		t.Fatal("Expected command field in tool input")
	}
	rule, err := ruleMatcher.Match(command, "Bash")
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
	resp := rule.Send
	if resp == "" {
		t.Fatal("Expected non-empty message")
	}

	if !strings.Contains(resp, "just test") {
		t.Error("Add should suggest just test alternative")
	}
}
