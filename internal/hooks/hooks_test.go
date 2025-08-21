package hooks

import (
	"strings"
	"testing"
)

func TestParseInput(t *testing.T) {
	t.Parallel()

	jsonInput := `{
		"tool_input": {
			"command": "go test ./...",
			"description": "Run tests"
		}
	}`

	event, err := ParseInput(strings.NewReader(jsonInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if event.ToolInput.Command != "go test ./..." {
		t.Errorf("Expected command 'go test ./...', got %s", event.ToolInput.Command)
	}
}
