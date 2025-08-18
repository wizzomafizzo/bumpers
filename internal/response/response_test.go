package response

import (
	"testing"
	
	"github.com/wizzomafizzo/bumpers/internal/config"
)

func TestFormatResponse(t *testing.T) {
	rule := &config.Rule{
		Name:    "block-go-test",
		Pattern: "go test.*",
		Action:  "deny",
		Message: "Use make test instead for better TDD integration",
		Alternatives: []string{
			"make test          # Run all tests",
			"make test-unit     # Run unit tests only",
		},
		UseClaude: false,
	}
	
	response := FormatResponse(rule)
	
	if response == "" {
		t.Fatal("Expected non-empty response")
	}
	
	// Should contain the message
	if !contains(response, "Use make test instead") {
		t.Error("Response should contain the rule message")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[0:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}