package hooks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestHookProcessor_DefaultToolFieldsBehavior(t *testing.T) {
	t.Parallel()

	// This test verifies that when no sources are specified, default tool fields are used
	// instead of checking all fields. This prevents false positives from description fields.

	// For Bash tool: should check "command" field, not "description"
	// Current behavior: checks all fields (leads to false positives)
	// Expected behavior: checks only default fields for known tools

	processor := NewHookProcessor(&MockConfigValidator{}, testProjectRoot, nil)

	// Test data - rule without sources should use default fields for Bash
	rule := &config.Rule{
		Match: "rm -rf", // Simple pattern match
		Send:  "Dangerous command detected",
	}

	// Create event with both command and description containing the pattern
	event := &hooks.HookEvent{
		ToolName: "Bash",
		ToolInput: map[string]any{
			"command":     "ls",               // Doesn't match pattern
			"description": "rm -rf /tmp/test", // Matches pattern but should be ignored for Bash
		},
	}

	// Test the checkOriginalBehavior method - this test will fail until we implement default fields
	ctx := context.Background()
	matchedRule, matchedValue := processor.checkOriginalBehavior(ctx, rule, event)

	// Expected behavior: Bash should only check command field, not description
	// Since command="ls" doesn't match pattern "rm -rf", there should be no match
	// This test will fail initially because current behavior checks all fields
	assert.Nil(t, matchedRule, "EXPECTED: Should not match because command field doesn't match pattern")
	assert.Empty(t, matchedValue, "EXPECTED: Should return empty string when no match")
}

func TestHookProcessor_UnknownToolUsesAllFields(t *testing.T) {
	t.Parallel()

	// Debug: First test that a known tool works
	processor := &DefaultHookProcessor{}

	// Test event with Bash tool (known tool)
	bashEvent := &hooks.HookEvent{
		ToolName: "Bash",
		ToolInput: map[string]any{
			"command": "test-pattern found here", // Should match because Bash checks command
		},
	}

	// Test rule that matches any tool
	rule := &config.Rule{
		Match: "test-pattern found here",
		Tool:  ".*", // Match any tool
		Send:  "Found pattern",
	}

	// Create matcher
	ruleMatcher, err := matcher.NewRuleMatcher([]config.Rule{*rule})
	require.NoError(t, err)

	// Test Bash first to make sure matcher works
	ctx := context.Background()
	matchedRule, matchedValue, err := processor.findMatchingRule(ctx, ruleMatcher, bashEvent)
	require.NoError(t, err)
	assert.NotNil(t, matchedRule, "Bash should match in command field")
	assert.Equal(t, "test-pattern found here", matchedValue, "Should match command field")

	// Now test unknown tool - should still check all fields
	unknownEvent := &hooks.HookEvent{
		ToolName: "UnknownTool",
		ToolInput: map[string]any{
			"field1":       "no match",
			"custom_field": "test-pattern found here", // This should match for unknown tools
		},
	}

	matchedRule2, matchedValue2, err := processor.findMatchingRule(ctx, ruleMatcher, unknownEvent)
	require.NoError(t, err)
	assert.NotNil(t, matchedRule2, "Unknown tool should check all fields")
	assert.Equal(t, "test-pattern found here", matchedValue2, "Should match the custom field")
}

func TestHookProcessor_EditToolUsesDefaultFields(t *testing.T) {
	t.Parallel()

	processor := &DefaultHookProcessor{}

	// Test rule that matches any tool
	rule := &config.Rule{
		Match: "sensitive-data",
		Tool:  ".*", // Match any tool
		Send:  "Avoid hardcoding secrets",
	}

	// Create matcher
	ruleMatcher, err := matcher.NewRuleMatcher([]config.Rule{*rule})
	require.NoError(t, err)

	// Test event with Edit tool - should only check file_path and new_string, not old_string
	editEvent := &hooks.HookEvent{
		ToolName: "Edit",
		ToolInput: map[string]any{
			"file_path":  "/path/to/config.js",
			"old_string": "sensitive-data in old code", // Should NOT match (Edit ignores old_string)
			"new_string": "clean new code",             // Doesn't match pattern
		},
	}

	ctx := context.Background()
	matchedRule, matchedValue, err := processor.findMatchingRule(ctx, ruleMatcher, editEvent)
	require.NoError(t, err)
	assert.Nil(t, matchedRule, "Edit should ignore old_string field")
	assert.Empty(t, matchedValue, "Should not match since only default fields are checked")
}

func TestHookProcessor_LogsWarningForUnmappedTools(t *testing.T) {
	t.Parallel()

	processor := &DefaultHookProcessor{}

	// Create a test rule that matches any tool
	rule := &config.Rule{
		Match: "test-pattern",
		Tool:  ".*",
		Send:  "Found pattern",
	}

	ruleMatcher, err := matcher.NewRuleMatcher([]config.Rule{*rule})
	require.NoError(t, err)

	// Test with an unmapped tool - this should eventually log a warning
	// For now, just test that the fallback behavior works
	unmappedEvent := &hooks.HookEvent{
		ToolName: "CompletelyUnknownTool",
		ToolInput: map[string]any{
			"custom_field": "test-pattern",
		},
	}

	// This should work (fall back to checking all fields) but will eventually log a warning
	ctx := context.Background()
	matchedRule, matchedValue, err := processor.findMatchingRule(ctx, ruleMatcher, unmappedEvent)
	require.NoError(t, err)
	assert.NotNil(t, matchedRule, "Should still match by falling back to all fields")
	assert.Equal(t, "test-pattern", matchedValue, "Should find match in custom field")

	// TODO: Once logging is implemented, verify that a warning was logged
}
