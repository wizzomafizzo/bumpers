package internal

import (
	"strings"
	"testing"
	
	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/hooks"
	"github.com/wizzomafizzo/bumpers/internal/matcher"
	"github.com/wizzomafizzo/bumpers/internal/response"
)

func TestEndToEndHookProcessing(t *testing.T) {
	// Load config
	cfg, err := config.LoadFromFile("../configs/bumpers.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// Create matcher
	m := matcher.NewRuleMatcher(cfg.Rules)
	
	// Simulate hook input
	hookInput := `{
		"command": "go test ./...",
		"args": ["go", "test", "./..."],
		"cwd": "/path/to/project"
	}`
	
	// Parse hook event
	event, err := hooks.ParseHookInput(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Failed to parse hook input: %v", err)
	}
	
	// Match rule
	rule, err := m.Match(event.Command)
	if err != nil {
		t.Fatalf("Failed to match rule: %v", err)
	}
	
	if rule == nil {
		t.Fatal("Expected rule match for go test command")
	}
	
	// Generate response
	resp := response.FormatResponse(rule)
	if resp == "" {
		t.Fatal("Expected non-empty response")
	}
	
	if !strings.Contains(resp, "make test") {
		t.Error("Response should suggest make test alternative")
	}
}