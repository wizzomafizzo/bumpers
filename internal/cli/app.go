package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/wizzomafizzo/bumpers/internal/core/engine/hooks"
	"github.com/wizzomafizzo/bumpers/internal/core/logging"
	"github.com/wizzomafizzo/bumpers/internal/infrastructure/project"
	ai "github.com/wizzomafizzo/bumpers/internal/platform/claude/api"
)

// App represents the main application with composed components
type App struct {
	// Core components
	hookProcessor   HookProcessor
	promptHandler   PromptHandler
	sessionManager  SessionManager
	configValidator ConfigValidator
	installManager  InstallManager

	// Configuration
	fileSystem   afero.Fs
	mockLauncher ai.MessageGenerator
	configPath   string
	workDir      string
	projectRoot  string
}

func NewApp(ctx context.Context, configPath string) *App {
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

	// Create specialized components
	configValidator := NewConfigValidator(resolvedConfigPath, projectRoot)
	hookProcessor := NewHookProcessor(configValidator, projectRoot)
	promptHandler := NewPromptHandler(resolvedConfigPath, projectRoot)
	sessionManager := NewSessionManager(resolvedConfigPath, projectRoot, nil)
	installManager := NewInstallManager(resolvedConfigPath, "", projectRoot, nil)

	app := &App{
		hookProcessor:   hookProcessor,
		promptHandler:   promptHandler,
		sessionManager:  sessionManager,
		configValidator: configValidator,
		installManager:  installManager,
		configPath:      resolvedConfigPath,
		projectRoot:     projectRoot,
	}

	logging.Get(ctx).Debug().
		Str("original_config_path", configPath).
		Str("resolved_config_path", resolvedConfigPath).
		Str("project_root", projectRoot).
		Msg("created new app instance")

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
	// Detect project root starting from workDir (similar to NewApp)
	projectRoot := workDir
	if root, found := project.FindProjectMarkerFrom(workDir); found {
		projectRoot = root
	}

	// Create specialized components with consistent projectRoot
	configValidator := NewConfigValidator(configPath, projectRoot)
	hookProcessor := NewHookProcessor(configValidator, projectRoot)
	promptHandler := NewPromptHandler(configPath, projectRoot)
	sessionManager := NewSessionManager(configPath, projectRoot, nil)
	installManager := NewInstallManager(configPath, projectRoot, projectRoot, nil)

	return &App{
		hookProcessor:   hookProcessor,
		promptHandler:   promptHandler,
		sessionManager:  sessionManager,
		configValidator: configValidator,
		installManager:  installManager,
		configPath:      configPath,
		workDir:         workDir,
		projectRoot:     projectRoot, // Use detected project root
	}
}

// NewAppWithFileSystem creates a new App instance with injectable filesystem.
// This enables parallel testing by using in-memory filesystem instead of real I/O.
func NewAppWithFileSystem(configPath, workDir string, fs afero.Fs) *App {
	// Create specialized components with consistent workDir as projectRoot and injected filesystem
	configValidator := NewConfigValidator(configPath, workDir)
	hookProcessor := NewHookProcessor(configValidator, workDir)
	promptHandler := NewPromptHandler(configPath, workDir)
	sessionManager := NewSessionManager(configPath, workDir, fs)
	installManager := NewInstallManager(configPath, workDir, workDir, fs)

	return &App{
		hookProcessor:   hookProcessor,
		promptHandler:   promptHandler,
		sessionManager:  sessionManager,
		configValidator: configValidator,
		installManager:  installManager,
		configPath:      configPath,
		workDir:         workDir,
		projectRoot:     workDir, // Ensure projectRoot is set consistently
		fileSystem:      fs,
	}
}

// ProcessHook delegates to HookProcessor
func (a *App) ProcessHook(ctx context.Context, input io.Reader) (ProcessResult, error) {
	response, err := a.processHookWithContext(ctx, input)
	if err != nil {
		return ProcessResult{}, err
	}
	return a.convertResponseToProcessResult(response), nil
}

// convertResponseToProcessResult converts legacy string responses to structured ProcessResult
func (*App) convertResponseToProcessResult(response string) ProcessResult {
	if response == "" {
		return ProcessResult{Mode: ProcessModeAllow, Message: ""}
	}

	// Check if response is hookSpecificOutput format (should be informational)
	if strings.Contains(response, "hookEventName") {
		return ProcessResult{Mode: ProcessModeInformational, Message: response}
	}

	// Otherwise it's a blocking response
	return ProcessResult{Mode: ProcessModeBlock, Message: response}
}

func (a *App) processHookWithContext(ctx context.Context, input io.Reader) (string, error) {
	logger := logging.Get(ctx)

	if os.Getenv("BUMPERS_SKIP") == "1" {
		logger.Debug().Msg("BUMPERS_SKIP is set, skipping hook processing")
		return "", nil
	}

	logger.Debug().Msg("processing hook input")

	// Detect hook type and get raw JSON
	hookType, rawJSON, err := hooks.DetectHookType(input)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to detect hook type")
		return "", fmt.Errorf("failed to detect hook type: %w", err)
	}
	logger.Debug().RawJSON("hook", rawJSON).Str("type", hookType.String()).Msg("received hook")

	// Route to appropriate handler based on hook type
	if hookType == hooks.UserPromptSubmitHook {
		logger.Debug().Msg("processing UserPromptSubmit hook")
		return a.ProcessUserPrompt(ctx, rawJSON)
	}
	if hookType == hooks.SessionStartHook {
		logger.Debug().Msg("processing SessionStart hook")
		return a.ProcessSessionStart(ctx, rawJSON)
	}
	if hookType == hooks.PostToolUseHook {
		logger.Debug().Msg("processing PostToolUse hook")
		return a.ProcessPostToolUse(ctx, rawJSON)
	}
	// Handle PreToolUse and other hooks
	return a.processPreToolUse(ctx, rawJSON)
}

// processPreToolUse delegates to HookProcessor
func (a *App) processPreToolUse(ctx context.Context, rawJSON json.RawMessage) (string, error) {
	result, err := a.hookProcessor.ProcessPreToolUse(ctx, rawJSON)
	if err != nil {
		return "", fmt.Errorf("hook processor failed: %w", err)
	}
	return result, nil
}

// ProcessPostToolUse delegates to HookProcessor
func (a *App) ProcessPostToolUse(ctx context.Context, rawJSON json.RawMessage) (string, error) {
	result, err := a.hookProcessor.ProcessPostToolUse(ctx, rawJSON)
	if err != nil {
		return "", fmt.Errorf("hook processor failed: %w", err)
	}
	return result, nil
}

// ProcessUserPrompt delegates to PromptHandler
func (a *App) ProcessUserPrompt(ctx context.Context, rawJSON json.RawMessage) (string, error) {
	result, err := a.promptHandler.ProcessUserPrompt(ctx, rawJSON)
	if err != nil {
		return "", fmt.Errorf("prompt handler failed: %w", err)
	}
	return result, nil
}

// ProcessSessionStart delegates to SessionManager
func (a *App) ProcessSessionStart(ctx context.Context, rawJSON json.RawMessage) (string, error) {
	result, err := a.sessionManager.ProcessSessionStart(ctx, rawJSON)
	if err != nil {
		return "", fmt.Errorf("session manager failed: %w", err)
	}
	return result, nil
}

// TestCommand delegates to ConfigValidator
func (a *App) TestCommand(ctx context.Context, command string) (string, error) {
	result, err := a.configValidator.TestCommand(ctx, command)
	if err != nil {
		return "", fmt.Errorf("config validator failed: %w", err)
	}
	return result, nil
}

// ValidateConfig delegates to ConfigValidator
func (a *App) ValidateConfig() (string, error) {
	result, err := a.configValidator.ValidateConfig()
	if err != nil {
		return "", fmt.Errorf("config validation failed: %w", err)
	}
	return result, nil
}

// Initialize delegates to InstallManager
func (a *App) Initialize() error {
	err := a.installManager.Initialize()
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}
	return nil
}

// Status delegates to InstallManager
func (a *App) Status() (string, error) {
	result, err := a.installManager.Status()
	if err != nil {
		return "", fmt.Errorf("status check failed: %w", err)
	}
	return result, nil
}

// InstallClaudeHooks delegates to InstallManager - needed for tests
func (a *App) installClaudeHooks() error {
	err := a.installManager.InstallClaudeHooks()
	if err != nil {
		return fmt.Errorf("hook installation failed: %w", err)
	}
	return nil
}

// extractAndLogIntent delegates to HookProcessor - needed for tests
func (a *App) extractAndLogIntent(ctx context.Context, event hooks.HookEvent) string {
	if defaultHookProcessor, ok := a.hookProcessor.(*DefaultHookProcessor); ok {
		return defaultHookProcessor.ExtractAndLogIntent(ctx, event)
	}
	return ""
}

// SetMockLauncher sets the mock launcher for testing
func (a *App) SetMockLauncher(launcher ai.MessageGenerator) {
	a.mockLauncher = launcher

	// Also set on components that need it
	if defaultHookProcessor, ok := a.hookProcessor.(*DefaultHookProcessor); ok {
		defaultHookProcessor.SetMockAIGenerator(launcher)
	}
	if defaultPromptHandler, ok := a.promptHandler.(*DefaultPromptHandler); ok {
		defaultPromptHandler.SetMockAIGenerator(launcher)
	}
	if defaultSessionManager, ok := a.sessionManager.(*DefaultSessionManager); ok {
		defaultSessionManager.SetMockAIGenerator(launcher)
	}
}
