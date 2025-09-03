package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test helper constants for prompt handler tests
const (
	promptTestConfigPath  = "/test/bumpers.yml"
	promptTestProjectRoot = "/test/project"
	promptTestDBPath      = "/test/db.sqlite"
)

// createPromptHandler creates a prompt handler for testing
func createPromptHandler() *DefaultPromptHandler {
	handler := NewPromptHandler(promptTestConfigPath, promptTestProjectRoot)
	handler.SetTestDBPath(promptTestDBPath)
	return handler
}

func TestNewPromptHandler(t *testing.T) {
	t.Parallel()

	handler := NewPromptHandler(promptTestConfigPath, promptTestProjectRoot)

	assert.NotNil(t, handler)
	assert.Equal(t, promptTestConfigPath, handler.configPath)
	assert.Equal(t, promptTestProjectRoot, handler.projectRoot)
	assert.NotNil(t, handler.aiHelper)
	assert.Nil(t, handler.stateManager)
}

func TestDefaultPromptHandler_SetTestDBPath(t *testing.T) {
	t.Parallel()

	handler := NewPromptHandler(promptTestConfigPath, promptTestProjectRoot)

	handler.SetTestDBPath(promptTestDBPath)

	assert.Equal(t, promptTestDBPath, handler.testDBPath)
}

func TestDefaultPromptHandler_ExtractCommand(t *testing.T) {
	t.Parallel()

	handler := createPromptHandler()

	// Test command with prefix
	cmd, isCmd := handler.extractCommand("$test command")
	assert.True(t, isCmd)
	assert.Equal(t, "test command", cmd)

	// Test without prefix
	cmd, isCmd = handler.extractCommand("regular text")
	assert.False(t, isCmd)
	assert.Empty(t, cmd)
}
