package app

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewSessionManager(t *testing.T) {
	t.Parallel()

	configPath := "/test/bumpers.yml"
	projectRoot := "/test/project"
	fs := afero.NewMemMapFs()

	manager := NewSessionManager(configPath, projectRoot, fs)

	assert.NotNil(t, manager)
	assert.Equal(t, configPath, manager.configPath)
	assert.Equal(t, fs, manager.fileSystem)
	assert.NotNil(t, manager.aiHelper)
	assert.Nil(t, manager.cache)
}

func TestNewSessionManagerFromOptions(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	opts := SessionManagerOptions{
		ConfigPath:  "/test/config.yml",
		ProjectRoot: "/test/project",
		FileSystem:  fs,
		Cache:       nil,
	}

	manager := NewSessionManagerFromOptions(opts)

	assert.NotNil(t, manager)
	assert.Equal(t, opts.ConfigPath, manager.configPath)
	assert.Equal(t, fs, manager.fileSystem)
	assert.NotNil(t, manager.aiHelper)
	assert.Nil(t, manager.cache)
}

func TestDefaultSessionManager_GetFileSystem_WithInjected(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	manager := NewSessionManager("/test/config.yml", "/test/project", fs)

	result := manager.getFileSystem()

	assert.Equal(t, fs, result)
}
