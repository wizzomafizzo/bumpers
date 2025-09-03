package rules

import (
	"strings"
	"time"
)

// OperationMode represents the current operation state
type OperationMode string

const (
	PlanMode    OperationMode = "plan"
	ExecuteMode OperationMode = "execute"
)

// OperationState tracks the current operation mode and metadata
type OperationState struct {
	Mode         OperationMode `json:"mode"`
	TriggerCount int           `json:"trigger_count"`
	UpdatedAt    int64         `json:"updated_at"`
}

// DefaultState returns the default operation state (execute mode - plan mode temporarily disabled)
func DefaultState() *OperationState {
	return &OperationState{
		Mode:         ExecuteMode,
		TriggerCount: 0,
		UpdatedAt:    time.Now().Unix(),
	}
}

// triggerPhrases that switch from plan to execute mode
var triggerPhrases = []string{
	"make it so",
	"go ahead",
}

// GetTriggerPhrases returns a copy of the trigger phrases
func GetTriggerPhrases() []string {
	return append([]string(nil), triggerPhrases...)
}

// emergencyStopPhrases that immediately force plan mode
var emergencyStopPhrases = []string{
	"STOP",
	"SILENCE",
}

// GetEmergencyStopPhrases returns a copy of the emergency stop phrases
func GetEmergencyStopPhrases() []string {
	return append([]string(nil), emergencyStopPhrases...)
}

// readOnlyBashCommands that are always allowed even in plan mode
var readOnlyBashCommands = []string{
	"ls", "cat", "git status",
}

// GetReadOnlyBashCommands returns a copy of the read-only bash commands
func GetReadOnlyBashCommands() []string {
	return append([]string(nil), readOnlyBashCommands...)
}

// DetectTriggerPhrase checks if user message contains trigger phrases
func DetectTriggerPhrase(message string) bool {
	msgLower := strings.ToLower(message)
	for _, phrase := range triggerPhrases {
		if strings.Contains(msgLower, phrase) {
			return true
		}
	}
	return false
}

// DetectEmergencyStop checks if user message contains emergency stop phrases
func DetectEmergencyStop(message string) bool {
	msgUpper := strings.ToUpper(message)
	for _, phrase := range emergencyStopPhrases {
		if strings.Contains(msgUpper, strings.ToUpper(phrase)) {
			return true
		}
	}
	return false
}

// IsReadOnlyBashCommand checks if a bash command is in the read-only whitelist
func IsReadOnlyBashCommand(command string) bool {
	command = strings.TrimSpace(command)

	// Check for exact matches and prefixes
	for _, readOnly := range readOnlyBashCommands {
		if command == readOnly || strings.HasPrefix(command, readOnly+" ") {
			return true
		}
	}

	return false
}
