package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRuleConfig = `rules:
  - match: "go test.*"
    send: "Use just test instead"
`

func TestNewConfigValidator(t *testing.T) {
	t.Parallel()

	configPath := "/test/bumpers.yml"
	projectRoot := promptTestProjectRoot

	validator := NewConfigValidator(configPath, projectRoot)

	assert.NotNil(t, validator)
	assert.Equal(t, configPath, validator.configPath)
	assert.Equal(t, projectRoot, validator.projectRoot)
}

func TestDefaultConfigValidator_LoadPartialConfig_Success(t *testing.T) {
	t.Parallel()

	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	configContent := testRuleConfig

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	validator := NewConfigValidator(configPath, "/test/project")
	partialConfig, err := validator.loadPartialConfig(context.Background())

	require.NoError(t, err)
	assert.NotNil(t, partialConfig)
	assert.Len(t, partialConfig.Rules, 1)
	assert.Equal(t, "go test.*", partialConfig.Rules[0].Match)
	assert.Equal(t, "Use just test instead", partialConfig.Rules[0].Send)
}

func TestDefaultConfigValidator_LoadPartialConfig_FileNotFound(t *testing.T) {
	t.Parallel()

	validator := NewConfigValidator("/nonexistent/config.yml", "/test/project")
	_, err := validator.loadPartialConfig(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config from")
	assert.Contains(t, err.Error(), "/nonexistent/config.yml")
}

func TestDefaultConfigValidator_LoadConfigAndMatcher_Success(t *testing.T) {
	t.Parallel()

	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	configContent := testRuleConfig

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	validator := NewConfigValidator(configPath, "/test/project")
	config, matcher, err := validator.LoadConfigAndMatcher(context.Background())

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.NotNil(t, matcher)
}

func TestDefaultConfigValidator_TestCommand_NoMatch(t *testing.T) {
	t.Parallel()

	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	configContent := testRuleConfig

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	validator := NewConfigValidator(configPath, "/test/project")

	result, err := validator.TestCommand(context.Background(), "npm install")

	require.NoError(t, err)
	assert.Equal(t, "Command allowed", result)
}

func TestDefaultConfigValidator_TestCommand_WithMatch(t *testing.T) {
	t.Parallel()

	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	configContent := testRuleConfig

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	validator := NewConfigValidator(configPath, "/test/project")

	result, err := validator.TestCommand(context.Background(), "go test ./...")

	require.NoError(t, err)
	assert.Equal(t, "Use just test instead", result)
}

func TestDefaultConfigValidator_ValidateConfig_ValidConfig(t *testing.T) {
	t.Parallel()

	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bumpers.yml")

	configContent := testRuleConfig

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	validator := NewConfigValidator(configPath, "/test/project")

	result, err := validator.ValidateConfig()

	require.NoError(t, err)
	assert.Equal(t, "Configuration is valid", result)
}
