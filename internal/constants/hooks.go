package constants

// Hook event names
const (
	// SessionStartEvent is the hook event name for session start events
	SessionStartEvent = "SessionStart"

	// UserPromptSubmitEvent is the hook event name for user prompt submit events
	UserPromptSubmitEvent = "UserPromptSubmit"
)

// Session start sources
const (
	// SessionSourceStartup indicates a session started from application startup
	SessionSourceStartup = "startup"

	// SessionSourceClear indicates a session started from a clear command
	SessionSourceClear = "clear"
)
