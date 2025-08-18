package matcher

import (
	"testing"
	
	"github.com/wizzomafizzo/bumpers/internal/config"
)

func TestRuleMatcher(t *testing.T) {
	rule := config.Rule{
		Name:    "block-go-test",
		Pattern: "go test.*",
		Action:  "deny",
		Message: "Use make test instead",
	}
	
	matcher := NewRuleMatcher([]config.Rule{rule})
	
	match, err := matcher.Match("go test ./...")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if match == nil {
		t.Fatal("Expected match, got nil")
	}
	
	if match.Name != "block-go-test" {
		t.Errorf("Expected rule name 'block-go-test', got %s", match.Name)
	}
}

func TestRuleMatcherNoMatch(t *testing.T) {
	rule := config.Rule{
		Name:    "block-go-test", 
		Pattern: "go test.*",
		Action:  "deny",
		Message: "Use make test instead",
	}
	
	matcher := NewRuleMatcher([]config.Rule{rule})
	
	match, err := matcher.Match("make test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if match != nil {
		t.Errorf("Expected no match, got %v", match)
	}
}