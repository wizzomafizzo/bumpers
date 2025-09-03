package hooks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/hooks"
	"github.com/wizzomafizzo/bumpers/internal/matcher"
	"github.com/wizzomafizzo/bumpers/internal/storage"
)

// Test helpers and constants
const testProjectRoot = "/test/project"

func TestNewHookProcessor(t *testing.T) {
	t.Parallel()

	configValidator := &MockConfigValidator{}
	var stateManager *storage.StateManager

	processor := NewHookProcessor(configValidator, testProjectRoot, stateManager)

	assert.NotNil(t, processor)
	assert.Equal(t, configValidator, processor.configValidator)
	assert.Equal(t, testProjectRoot, processor.projectRoot)
	assert.Equal(t, stateManager, processor.stateManager)
}

func TestHookProcessor_SetMockAIGenerator(t *testing.T) {
	t.Parallel()

	processor := NewHookProcessor(&MockConfigValidator{}, testProjectRoot, nil)

	// This test just verifies the setter works - we'll need to import claude when we actually test AI functionality
	processor.SetMockAIGenerator(nil)

	assert.Nil(t, processor.aiGenerator)
}

func TestHookProcessor_isEditingTool(t *testing.T) {
	t.Parallel()

	processor := &DefaultHookProcessor{}

	// Test editing tools
	assert.True(t, processor.isEditingTool("Edit"))
	assert.True(t, processor.isEditingTool("Write"))
	assert.True(t, processor.isEditingTool("MultiEdit"))
	assert.True(t, processor.isEditingTool("NotebookEdit"))

	// Test non-editing tools
	assert.False(t, processor.isEditingTool("Read"))
	assert.False(t, processor.isEditingTool("Bash"))
	assert.False(t, processor.isEditingTool("Search"))
	assert.False(t, processor.isEditingTool(""))
}

func TestHookProcessor_shouldSkipProcessing_NoStateManager(t *testing.T) {
	t.Parallel()

	processor := NewHookProcessor(&MockConfigValidator{}, testProjectRoot, nil)

	result := processor.shouldSkipProcessing(context.Background())

	assert.False(t, result)
}

func TestHookProcessor_ExtractAndLogIntent_EmptyTranscriptPath(t *testing.T) {
	t.Parallel()

	processor := &DefaultHookProcessor{}
	event := &hooks.HookEvent{
		TranscriptPath: "",
	}

	result := processor.ExtractAndLogIntent(context.Background(), event)

	assert.Empty(t, result)
}

// Mock implementation for testing
type MockConfigValidator struct{}

func (*MockConfigValidator) LoadConfigAndMatcher(_ context.Context) (*config.Config, *matcher.RuleMatcher, error) {
	return nil, nil, nil
}

func (*MockConfigValidator) ValidateConfig() (string, error) {
	return "", nil
}

func (*MockConfigValidator) TestCommand(_ context.Context, _ string) (string, error) {
	return "", nil
}
