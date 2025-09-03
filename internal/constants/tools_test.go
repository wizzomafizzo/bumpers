package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultToolFieldsExist(t *testing.T) {
	t.Parallel()

	// Test that DefaultToolFields contains expected tools
	assert.NotNil(t, DefaultToolFields, "DefaultToolFields should be defined")

	// Test that common tools have default fields defined
	bashFields, exists := DefaultToolFields["Bash"]
	assert.True(t, exists, "Bash should have default fields")
	assert.Contains(t, bashFields, "command", "Bash should check command field by default")
	assert.NotContains(t, bashFields, "description", "Bash should not check description by default")

	editFields, exists := DefaultToolFields["Edit"]
	assert.True(t, exists, "Edit should have default fields")
	assert.Contains(t, editFields, "file_path", "Edit should check file_path by default")
	assert.Contains(t, editFields, "new_string", "Edit should check new_string by default")
	assert.NotContains(t, editFields, "old_string", "Edit should not check old_string by default")
}

func TestSpecialSourceAllConstant(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "#all", SpecialSourceAll, "SpecialSourceAll should be '#all'")
}

func TestDefaultToolFieldsIncludeMoreCommonTools(t *testing.T) {
	t.Parallel()

	// Test that we have mappings for more common tools
	readFields, exists := DefaultToolFields["Read"]
	assert.True(t, exists, "Read should have default fields")
	assert.Contains(t, readFields, "file_path", "Read should check file_path by default")

	grepFields, exists := DefaultToolFields["Grep"]
	assert.True(t, exists, "Grep should have default fields")
	assert.Contains(t, grepFields, "pattern", "Grep should check pattern by default")
}

func TestDefaultToolFieldsIncludeAllCommonlyUsedTools(t *testing.T) {
	t.Parallel()

	// Based on analysis of bumpers logs, these tools are commonly used and should have default fields
	commonTools := map[string][]string{
		// Core file tools
		"Write":     {"file_path", "content"},
		"MultiEdit": {"file_path", "edits"},

		// Search and discovery
		"Glob": {"pattern", "path"},
		"Task": {"subagent_type", "prompt"},

		// Web tools
		"WebFetch":  {"url", "prompt"},
		"WebSearch": {"query"},

		// Session management
		"TodoWrite":    {"todos"},
		"ExitPlanMode": {"plan"},

		// Shell tools
		"BashOutput": {"bash_id"},
		"KillBash":   {"shell_id"},

		// MCP tools found in logs
		"mcp__zen__chat":                      {"prompt", "model"},
		"mcp__zen__debug":                     {"step", "hypothesis"},
		"mcp__octocode__githubSearchCode":     {"queries"},
		"mcp__octocode__githubGetFileContent": {"queries"},
	}

	for toolName, expectedFields := range commonTools {
		fields, exists := DefaultToolFields[toolName]
		assert.True(t, exists, "Tool %s should have default fields defined", toolName)

		if exists {
			for _, expectedField := range expectedFields {
				assert.Contains(t, fields, expectedField,
					"Tool %s should include field %s in default fields", toolName, expectedField)
			}
		}
	}
}
