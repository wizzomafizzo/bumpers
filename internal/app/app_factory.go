package app

import (
	"context"

	apphooks "github.com/wizzomafizzo/bumpers/internal/app/hooks"
	apptypes "github.com/wizzomafizzo/bumpers/internal/app/types"
	"github.com/wizzomafizzo/bumpers/internal/storage"
)

// AppFactory handles the creation and initialization of App instances
type AppFactory struct{}

// NewAppFactory creates a new instance of AppFactory
func NewAppFactory() *AppFactory {
	return &AppFactory{}
}

// AppComponents holds all the specialized components needed by App
type AppComponents struct {
	ConfigValidator apptypes.ConfigValidator
	HookProcessor   apphooks.HookProcessor
	PromptHandler   PromptHandler
	SessionManager  SessionManager
	InstallManager  InstallManager
}

// CreateApp creates a new App instance using the factory pattern
func (*AppFactory) CreateApp(configPath string) *App {
	// Use the existing NewApp constructor for now
	ctx := context.Background()
	return NewApp(ctx, configPath)
}

// CreateComponents creates all the specialized components needed by App
func (*AppFactory) CreateComponents(
	configPath, projectRoot string,
	stateManager *storage.StateManager,
) AppComponents {
	configValidator := NewConfigValidator(configPath, projectRoot)
	return AppComponents{
		ConfigValidator: configValidator,
		HookProcessor:   apphooks.NewHookProcessor(configValidator, projectRoot, stateManager),
		PromptHandler:   NewPromptHandler(configPath, projectRoot, stateManager),
		SessionManager:  NewSessionManager(configPath, projectRoot, nil),
		InstallManager:  NewInstallManager(configPath, "", projectRoot, nil),
	}
}

// CreateAppWithComponentFactory creates a new App using the component factory pattern
func (f *AppFactory) CreateAppWithComponentFactory(_ context.Context, configPath string) *App {
	components := f.CreateComponents(configPath, "", nil)
	return &App{
		hookProcessor:   components.HookProcessor,
		promptHandler:   components.PromptHandler,
		sessionManager:  components.SessionManager,
		configValidator: components.ConfigValidator,
		installManager:  components.InstallManager,
		configPath:      configPath,
	}
}
