package app

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAppWithOptions(t *testing.T) {
	t.Parallel()
	// Create in-memory filesystem for testing
	fs := afero.NewMemMapFs()
	tempDir := "/test"
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Use NewAppWithFileSystem which is designed for testing
	app := NewAppWithFileSystem(configPath, tempDir, fs)

	require.NotNil(t, app)
	assert.Equal(t, configPath, app.configPath)
	assert.Equal(t, tempDir, app.workDir)
	// Verify components are properly initialized with working managers
	assert.NotNil(t, app.dbManager)
	assert.NotNil(t, app.hookProcessor)
	assert.NotNil(t, app.promptHandler)
	assert.NotNil(t, app.sessionManager)
	assert.NotNil(t, app.configValidator)
	assert.NotNil(t, app.installManager)
}

func TestNewAppWithOptions_WorkDir(t *testing.T) {
	t.Parallel()
	// Create in-memory filesystem for testing
	fs := afero.NewMemMapFs()
	tempDir := "/test/dir"
	configPath := filepath.Join(tempDir, "config.yml")

	// Use NewAppWithFileSystem which is designed for testing
	app := NewAppWithFileSystem(configPath, tempDir, fs)

	require.NotNil(t, app)
	assert.Equal(t, configPath, app.configPath)
	assert.Equal(t, tempDir, app.workDir)
	// Verify components are properly initialized with working managers
	assert.NotNil(t, app.dbManager)
}

func TestNewAppWithOptions_ShouldInitializeAllComponents(t *testing.T) {
	t.Parallel()
	// Create in-memory filesystem for testing
	fs := afero.NewMemMapFs()
	tempDir := "/test"
	configPath := filepath.Join(tempDir, "bumpers.yml")

	// Use NewAppWithFileSystem which is designed for testing
	app := NewAppWithFileSystem(configPath, tempDir, fs)

	require.NotNil(t, app)

	// Debug what we're actually getting
	t.Logf("projectRoot: %s, workDir: %s", app.projectRoot, app.workDir)

	// These assertions should pass with proper implementation
	assert.NotNil(t, app.hookProcessor, "hookProcessor should be initialized")
	assert.NotNil(t, app.promptHandler, "promptHandler should be initialized")
	assert.NotNil(t, app.sessionManager, "sessionManager should be initialized")
	assert.NotNil(t, app.configValidator, "configValidator should be initialized")
	assert.NotNil(t, app.installManager, "installManager should be initialized")
	assert.NotNil(t, app.dbManager, "dbManager should be initialized")
}

func TestNewAppWithOptions_ErrorWhenManagersCannotBeCreated(t *testing.T) {
	t.Parallel()

	// Test error path by using NewAppWithFileSystem with empty projectRoot
	// This follows the same pattern as other tests but tests the failure condition
	fs := afero.NewMemMapFs()
	configPath := filepath.Join("test", "bumpers.yml")

	// Empty workDir will cause createDatabaseAndStateManagerWithFS to return nil, nil
	app := NewAppWithFileSystem(configPath, "", fs)

	// Should handle the case where managers cannot be created gracefully
	// In this case, the function still returns an app but with nil managers
	require.NotNil(t, app)
	assert.Equal(t, configPath, app.configPath)
	assert.Empty(t, app.workDir)
	// When projectRoot is empty, managers should be nil
	assert.Nil(t, app.dbManager)
}
