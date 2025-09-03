package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testConfigYml = "test-config.yml"

func TestNewAppFactory_ShouldCreateFactory(t *testing.T) {
	t.Parallel()
	// When
	factory := NewAppFactory()

	// Then
	assert.NotNil(t, factory)
}

func TestAppFactory_CreateApp_ShouldCreateApp(t *testing.T) {
	t.Parallel()
	// Given
	factory := NewAppFactory()
	configPath := testConfigYml

	// When
	app := factory.CreateApp(configPath)

	// Then
	assert.NotNil(t, app)
}

func TestAppFactory_CreateApp_ShouldCreateAllComponents(t *testing.T) {
	t.Parallel()
	// Given
	factory := NewAppFactory()
	configPath := testConfigYml

	// When
	app := factory.CreateApp(configPath)

	// Then
	assert.NotNil(t, app.hookProcessor)
	assert.NotNil(t, app.promptHandler)
	assert.NotNil(t, app.sessionManager)
	assert.NotNil(t, app.configValidator)
	assert.NotNil(t, app.installManager)
}

func TestAppFactory_CreateComponents_ShouldReturnComponentsStruct(t *testing.T) {
	t.Parallel()
	// Given
	factory := NewAppFactory()
	configPath := testConfigYml
	projectRoot := "/test/root"

	// When
	components := factory.CreateComponents(configPath, projectRoot, nil)

	// Then
	assert.NotNil(t, components)
	assert.NotNil(t, components.ConfigValidator)
	assert.NotNil(t, components.HookProcessor)
	assert.NotNil(t, components.PromptHandler)
	assert.NotNil(t, components.SessionManager)
	assert.NotNil(t, components.InstallManager)
}

func TestAppFactory_CreateAppWithComponentFactory_ShouldUseFactoryPattern(t *testing.T) {
	t.Parallel()
	// Given
	factory := NewAppFactory()
	ctx := context.Background()
	configPath := testConfigYml

	// When
	app := factory.CreateAppWithComponentFactory(ctx, configPath)

	// Then
	assert.NotNil(t, app)
	assert.NotNil(t, app.hookProcessor)
	assert.NotNil(t, app.promptHandler)
	assert.NotNil(t, app.sessionManager)
	assert.NotNil(t, app.configValidator)
	assert.NotNil(t, app.installManager)
}

func TestAppFactory_CreateAppWithComponentFactory_ShouldAcceptContext(t *testing.T) {
	t.Parallel()
	// Given
	factory := NewAppFactory()
	ctx := context.Background()
	configPath := testConfigYml

	// When
	app := factory.CreateAppWithComponentFactory(ctx, configPath)

	// Then
	assert.NotNil(t, app)
}

func TestAppFactory_CreateAppWithComponentFactory_ShouldSetConfigPath(t *testing.T) {
	t.Parallel()
	// Given
	factory := NewAppFactory()
	ctx := context.Background()
	configPath := testConfigYml

	// When
	app := factory.CreateAppWithComponentFactory(ctx, configPath)

	// Then
	assert.NotNil(t, app)
	assert.NotEmpty(t, app.configPath, "configPath should be set on the app")
}
