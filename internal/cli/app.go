package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/hooks"
	"github.com/wizzomafizzo/bumpers/internal/logger"
	"github.com/wizzomafizzo/bumpers/internal/matcher"
)

func NewApp(configPath string) *App {
	return &App{configPath: configPath}
}

// NewAppWithWorkDir creates a new App instance with an injectable working directory.
// This is primarily used for testing to avoid global state dependencies.
func NewAppWithWorkDir(configPath, workDir string) *App {
	return &App{configPath: configPath, workDir: workDir}
}

type App struct {
	logger     *logger.Logger
	configPath string
	workDir    string
}

// loadConfigAndMatcher loads configuration and creates a rule matcher
func (a *App) loadConfigAndMatcher() (*config.Config, *matcher.RuleMatcher, error) {
	cfg, err := config.Load(a.configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config from %s: %w", a.configPath, err)
	}

	ruleMatcher, err := matcher.NewRuleMatcher(cfg.Rules)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create rule matcher: %w", err)
	}

	return cfg, ruleMatcher, nil
}

func (a *App) ProcessHook(input io.Reader) (string, error) {
	// Parse hook input to get command
	event, err := hooks.ParseInput(input)
	if err != nil {
		return "", fmt.Errorf("failed to parse hook input: %w", err)
	}

	// Load config and match rules
	_, ruleMatcher, err := a.loadConfigAndMatcher()
	if err != nil {
		return "", err
	}

	rule, err := ruleMatcher.Match(event.ToolInput.Command)
	if err != nil {
		if errors.Is(err, matcher.ErrNoRuleMatch) {
			// No rule matched, command is allowed
			return "", nil
		}
		return "", fmt.Errorf("failed to match rule for command '%s': %w", event.ToolInput.Command, err)
	}

	if rule != nil {
		return rule.Response, nil
	}

	// This should never happen based on matcher logic, but Go requires a return
	return "", nil
}

func (a *App) TestCommand(command string) (string, error) {
	// Load config and match rules
	_, ruleMatcher, err := a.loadConfigAndMatcher()
	if err != nil {
		return "", err
	}

	rule, err := ruleMatcher.Match(command)
	if err != nil {
		if errors.Is(err, matcher.ErrNoRuleMatch) {
			// No rule matched, command is allowed
			return "Command allowed", nil
		}
		return "", fmt.Errorf("failed to match rule for command '%s': %w", command, err)
	}

	if rule != nil {
		return rule.Response, nil
	}

	// This should never happen based on matcher logic, but Go requires a return
	return "Command allowed", nil
}

func (a *App) ValidateConfig() (string, error) {
	// Load config file
	cfg, err := config.Load(a.configPath)
	if err != nil {
		return "", fmt.Errorf("failed to load config from %s: %w", a.configPath, err)
	}

	// Validate regex patterns by trying to create matcher
	_, err = matcher.NewRuleMatcher(cfg.Rules)
	if err != nil {
		return "", fmt.Errorf("failed to validate config rules: %w", err)
	}

	return "Configuration is valid", nil
}
