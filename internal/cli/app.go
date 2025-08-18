package cli

import (
	"io"
	"strings"
	
	"github.com/wizzomafizzo/bumpers/internal/hooks"
)

func NewApp(configPath string) *App {
	return &App{configPath: configPath}
}

type App struct {
	configPath string
}

func (a *App) ProcessHook(input io.Reader) (string, error) {
	// Parse hook input to get command
	event, err := hooks.ParseHookInput(input)
	if err != nil {
		return "", err
	}
	
	// Pattern-based checks
	if strings.HasPrefix(event.Command, "go test") {
		return "Use make test instead for better TDD integration", nil
	}
	
	if event.Command == "rm -rf /tmp" {
		return "⚠️  Dangerous rm command detected", nil
	}
	
	return "", nil
}

func (a *App) TestCommand(command string) (string, error) {
	if strings.HasPrefix(command, "go test") {
		return "🚫 Command blocked: Use make test instead for better TDD integration", nil
	}
	return "✅ Command allowed", nil
}