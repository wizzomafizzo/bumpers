package constants

// DefaultToolFields maps tool names to their most useful fields for rule matching.
// When a rule doesn't specify sources, these fields will be checked instead of all fields.
// This prevents false positives from matching against description fields or other less useful data.
var DefaultToolFields = map[string][]string{
	// Core file and command tools
	"Bash":      {"command"},                 // Only check command, not description which can have junk
	"Edit":      {"file_path", "new_string"}, // file_path and new content, not old_string which might be unrelated
	"Read":      {"file_path"},               // file path being read
	"Write":     {"file_path", "content"},    // file being written and its content
	"MultiEdit": {"file_path", "edits"},      // file being edited and edit operations

	// Search and discovery tools
	"Grep": {"pattern", "path"},         // search pattern and path, not output which varies
	"Glob": {"pattern", "path"},         // file pattern and search path
	"Task": {"subagent_type", "prompt"}, // type of agent and task description

	// Web tools
	"WebFetch":  {"url", "prompt"}, // URL being fetched and processing instructions
	"WebSearch": {"query"},         // search terms

	// Session management tools
	"TodoWrite":    {"todos"}, // todo items being managed
	"ExitPlanMode": {"plan"},  // plan being proposed

	// Shell tools
	"BashOutput": {"bash_id"},  // shell session identifier
	"KillBash":   {"shell_id"}, // shell session to terminate

	// MCP (Model Context Protocol) tools - AI-powered tools
	"mcp__zen__chat":                      {"prompt", "model"},    // AI chat with context and model
	"mcp__zen__debug":                     {"step", "hypothesis"}, // debugging analysis step and theory
	"mcp__octocode__githubSearchCode":     {"queries"},            // GitHub code search queries
	"mcp__octocode__githubGetFileContent": {"queries"},            // GitHub file retrieval queries
}

// SpecialSourceAll is used to explicitly request checking all tool input fields
const SpecialSourceAll = "#all"
