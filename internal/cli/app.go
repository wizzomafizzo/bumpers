package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/filesystem"
	"github.com/wizzomafizzo/bumpers/internal/hooks"
	"github.com/wizzomafizzo/bumpers/internal/matcher"
	"github.com/wizzomafizzo/bumpers/internal/project"
	"github.com/wizzomafizzo/bumpers/internal/template"
)

func NewApp(configPath string) *App {
	// Detect project root
	projectRoot, err := project.FindRoot()
	if err != nil {
		// Fall back to current working directory if project root detection fails
		projectRoot = ""
	}

	// Resolve config path relative to project root if it's relative
	resolvedConfigPath := configPath
	shouldResolve := projectRoot != "" && !filepath.IsAbs(configPath)
	if shouldResolve {
		resolvedConfigPath = filepath.Join(projectRoot, configPath)
	}

	// If using default config name, try different extensions in order
	if shouldResolve && configPath == "bumpers.yml" {
		if _, err := os.Stat(resolvedConfigPath); os.IsNotExist(err) {
			resolvedConfigPath = findAlternativeConfig(projectRoot)
		}
	}

	app := &App{
		configPath:  resolvedConfigPath,
		projectRoot: projectRoot,
	}

	return app
}

func findAlternativeConfig(projectRoot string) string {
	extensions := []string{"yaml", "toml", "json"}
	for _, ext := range extensions {
		candidatePath := filepath.Join(projectRoot, "bumpers."+ext)
		if _, err := os.Stat(candidatePath); err == nil {
			return candidatePath
		}
	}
	return filepath.Join(projectRoot, "bumpers.yml") // fallback to original
}

// NewAppWithWorkDir creates a new App instance with an injectable working directory.
// This is primarily used for testing to avoid global state dependencies.
func NewAppWithWorkDir(configPath, workDir string) *App {
	return &App{configPath: configPath, workDir: workDir}
}

// NewAppWithFileSystem creates a new App instance with injectable filesystem.
// This enables parallel testing by using in-memory filesystem instead of real I/O.
func NewAppWithFileSystem(configPath, workDir string, fs filesystem.FileSystem) *App {
	return &App{
		configPath: configPath,
		workDir:    workDir,
		fileSystem: fs,
	}
}

type App struct {
	fileSystem  filesystem.FileSystem
	configPath  string
	workDir     string
	projectRoot string
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
	// Log that we're processing a hook
	log.Info().Msg("Processing hook input")

	// Detect hook type and get raw JSON
	hookType, rawJSON, err := hooks.DetectHookType(input)
	if err != nil {
		log.Error().Err(err).Msg("Failed to detect hook type")
		return "", fmt.Errorf("failed to detect hook type: %w", err)
	}

	log.Info().Int("hookType", int(hookType)).Msg("Detected hook type")

	// Handle UserPromptSubmit hooks
	if hookType == hooks.UserPromptSubmitHook {
		log.Info().Msg("Processing UserPromptSubmit hook")
		return a.ProcessUserPrompt(rawJSON)
	}

	// Handle SessionStart hooks
	if hookType == hooks.SessionStartHook {
		log.Info().Msg("Processing SessionStart hook")
		return a.ProcessSessionStart(rawJSON)
	}

	// Handle PreToolUse hooks (existing logic)
	var event hooks.HookEvent
	if unmarshalErr := json.Unmarshal(rawJSON, &event); unmarshalErr != nil {
		return "", fmt.Errorf("failed to parse hook input: %w", unmarshalErr)
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
		// Process template with rule context including shared variables
		context := template.BuildRuleContext(event.ToolInput.Command)

		processedMessage, err := template.Execute(rule.Message, context)
		if err != nil {
			return "", fmt.Errorf("failed to process rule template: %w", err)
		}

		return processedMessage, nil
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
		// Process template with rule context including shared variables
		context := template.BuildRuleContext(command)

		processedMessage, err := template.Execute(rule.Message, context)
		if err != nil {
			return "", fmt.Errorf("failed to process rule template: %w", err)
		}

		return processedMessage, nil
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

// getFileSystem returns the filesystem to use - either injected or defaults to OS
func (a *App) getFileSystem() filesystem.FileSystem {
	if a.fileSystem != nil {
		return a.fileSystem
	}
	return filesystem.NewOSFileSystem()
}
