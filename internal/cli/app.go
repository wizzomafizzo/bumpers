package cli

import (
	"errors"
	"io"

	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/hooks"
	"github.com/wizzomafizzo/bumpers/internal/logger"
	"github.com/wizzomafizzo/bumpers/internal/matcher"
	"github.com/wizzomafizzo/bumpers/internal/response"
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
		return "", err //nolint:wrapcheck // Error context is clear from function name
	}

	// Load config and match rules
	cfg, err := config.LoadFromFile(a.configPath)
	if err != nil {
		logger.Error("Failed to load config file", "path", a.configPath, "error", err)
		return "", err //nolint:wrapcheck // Config file path is known from app context
	}

	ruleMatcher := matcher.NewRuleMatcher(cfg.Rules)
	rule, err := ruleMatcher.Match(event.ToolInput.Command)
	if err != nil {
		if errors.Is(err, matcher.ErrNoRuleMatch) {
			// No rule matched, command is allowed
			return "", nil
		}
		return "", err //nolint:wrapcheck // Rule matching errors are internal
	}

	if rule != nil {
		return response.FormatResponse(rule), nil
	}

	// This should never happen based on matcher logic, but Go requires a return
	return "", nil
}

func (a *App) TestCommand(command string) (string, error) {
	// Load config and match rules
	cfg, err := config.LoadFromFile(a.configPath)
	if err != nil {
		return "", err //nolint:wrapcheck // Config file path is known from app context
	}

	ruleMatcher := matcher.NewRuleMatcher(cfg.Rules)
	rule, err := ruleMatcher.Match(command)
	if err != nil {
		if errors.Is(err, matcher.ErrNoRuleMatch) {
			// No rule matched, command is allowed
			return "Command allowed", nil
		}
		return "", err //nolint:wrapcheck // Rule matching errors are internal
	}

	if rule != nil {
		return response.FormatResponse(rule), nil
	}

	// This should never happen based on matcher logic, but Go requires a return
	return "Command allowed", nil
}
