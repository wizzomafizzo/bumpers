package apptypes

// PostToolContent contains the content extracted from post-tool-use events
type PostToolContent struct {
	Intent        string
	ToolOutputMap map[string]any
	ToolName      string
}
