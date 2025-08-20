package response

import (
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/config"
)

func TestFormatResponse(t *testing.T) {
	t.Parallel()

	rule := &config.Rule{
		Name:    "block-go-test",
		Pattern: "go test",
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

func TestFormatResponseWithNewStructure(t *testing.T) {
	t.Parallel()

	rule := &config.Rule{
		Name:    "block-go-test",
		Pattern: "go test*",
		Action:  "deny",
		Response: "Use make test instead for better TDD integration\n\n" +
			"Try one of these alternatives:\n• make test          # Run all tests\n" +
			"• make test-unit     # Run unit tests only",
	}

	response := FormatResponse(rule)

	if response == "" {
		t.Fatal("Expected non-empty response")
	}

	// Should contain the response
	if !contains(response, "Use make test instead") {
		t.Error("Response should contain the rule response")
	}

	if !contains(response, "alternatives") {
		t.Error("Response should contain alternatives within the response text")
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
