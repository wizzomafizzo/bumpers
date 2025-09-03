package app

import (
	"context"
	"errors"

	apphooks "github.com/wizzomafizzo/bumpers/internal/app/hooks"
	"github.com/wizzomafizzo/bumpers/internal/database"
	"github.com/wizzomafizzo/bumpers/internal/storage"
)

// AppOptions contains configuration options for creating an App
type AppOptions struct {
	ConfigPath string
	WorkDir    string
}

// AppOptionsKey is a custom type for context keys to avoid collisions
type AppOptionsKey struct{}

// NewAppWithOptions creates a new App with the given options
func NewAppWithOptions(ctx context.Context, opts AppOptions) (*App, error) {
	ctx = context.WithValue(ctx, AppOptionsKey{}, opts)

	// Set projectRoot - ensure it's never empty
	projectRoot := opts.WorkDir
	if projectRoot == "" {
		projectRoot = "."
	}

	// Create required components
	configValidator := NewConfigValidator(opts.ConfigPath, projectRoot)
	var dbManager *database.Manager
	var stateManager *storage.StateManager
	dbManager, stateManager = createDatabaseAndStateManager(ctx, projectRoot)

	// Database and state managers are required dependencies
	if dbManager == nil {
		return nil, errors.New("failed to create database manager: project root may be " +
			"invalid or database inaccessible")
	}
	if stateManager == nil {
		return nil, errors.New("failed to create state manager: database may be inaccessible")
	}

	hookProcessor := apphooks.NewHookProcessor(configValidator, projectRoot, stateManager)
	promptHandler := NewPromptHandler(opts.ConfigPath, projectRoot, stateManager)
	sessionManager := NewSessionManager(opts.ConfigPath, projectRoot, nil)
	installManager := NewInstallManager(opts.ConfigPath, opts.WorkDir, projectRoot, nil)

	return &App{
		hookProcessor:   hookProcessor,
		promptHandler:   promptHandler,
		sessionManager:  sessionManager,
		configValidator: configValidator,
		installManager:  installManager,
		dbManager:       dbManager,
		configPath:      opts.ConfigPath,
		workDir:         opts.WorkDir,
		projectRoot:     projectRoot,
	}, nil
}
