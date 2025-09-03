package constants

// Hook event names
const (
	// SessionStartEvent is the hook event name for session start events
	SessionStartEvent = "SessionStart"

	// UserPromptSubmitEvent is the hook event name for user prompt submit events
	UserPromptSubmitEvent = "UserPromptSubmit"

	// PreToolUseEvent is the hook event name for pre-tool-use events
	PreToolUseEvent = "PreToolUse"

	// PostToolUseEvent is the hook event name for post-tool-use events
	PostToolUseEvent = "PostToolUse"
)

// Session start sources
const (
	// SessionSourceStartup indicates a session started from application startup
	SessionSourceStartup = "startup"

	// SessionSourceClear indicates a session started from a clear command
	SessionSourceClear = "clear"
)
