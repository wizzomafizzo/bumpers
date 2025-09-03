package apptypes

import "strings"

// ProcessResult represents the result of processing a hook event
type ProcessResult struct {
	Mode    ProcessMode `json:"mode"`
	Message string      `json:"message"`
}

// ProcessMode defines how the CLI should respond to a hook event
type ProcessMode string

const (
	ProcessModeAllow         ProcessMode = "allow"         // Exit 0, no output
	ProcessModeInformational ProcessMode = "informational" // Exit 0, print message
	ProcessModeBlock         ProcessMode = "block"         // Exit 2, print message
)

// ConvertResponseToProcessResult converts legacy string responses to structured ProcessResult
func ConvertResponseToProcessResult(response string) ProcessResult {
	if response == "" {
		return ProcessResult{Mode: ProcessModeAllow, Message: ""}
	}

	// Check if response is hookSpecificOutput format (should be informational)
	if strings.Contains(response, "hookEventName") {
		return ProcessResult{Mode: ProcessModeInformational, Message: response}
	}

	// Otherwise it's a blocking response
	return ProcessResult{Mode: ProcessModeBlock, Message: response}
}
