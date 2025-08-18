package hooks

import (
	"strings"
	"testing"
)

func TestParseHookInput(t *testing.T) {
	t.Parallel()

	jsonInput := `{
		"command": "go test ./...",
		"args": ["go", "test", "./..."],
		"cwd": "/path/to/project"
	}`

	event, err := ParseHookInput(strings.NewReader(jsonInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if event.Command != "go test ./..." {
		t.Errorf("Expected command 'go test ./...', got %s", event.Command)
	}

	if len(event.Args) != 3 {
		t.Fatalf("Expected 3 args, got %d", len(event.Args))
	}

	if event.Args[0] != "go" {
		t.Errorf("Expected first arg 'go', got %s", event.Args[0])
	}
}
