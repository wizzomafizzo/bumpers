package operation

import "strings"

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

// DefaultState returns the default operation state (plan mode)
func DefaultState() *OperationState {
	return &OperationState{
		Mode:         PlanMode,
		TriggerCount: 0,
		UpdatedAt:    0,
	}
}

// TriggerPhrases that switch from plan to execute mode
var TriggerPhrases = []string{
	"make it so",
	"go ahead",
}

// EmergencyStopPhrases that immediately force plan mode
var EmergencyStopPhrases = []string{
	"STOP",
	"SILENCE",
}

// ReadOnlyBashCommands that are always allowed even in plan mode
var ReadOnlyBashCommands = []string{
	"ls", "cat", "git status",
}

// DetectTriggerPhrase checks if user message contains trigger phrases
func DetectTriggerPhrase(message string) bool {
	msgLower := strings.ToLower(message)
	for _, phrase := range TriggerPhrases {
		if strings.Contains(msgLower, phrase) {
			return true
		}
	}
	return false
}

// DetectEmergencyStop checks if user message contains emergency stop phrases
func DetectEmergencyStop(message string) bool {
	for _, phrase := range EmergencyStopPhrases {
		if strings.Contains(message, phrase) {
			return true
		}
	}
	return false
}

// IsReadOnlyBashCommand checks if a bash command is in the read-only whitelist
func IsReadOnlyBashCommand(command string) bool {
	command = strings.TrimSpace(command)

	// Check for exact matches and prefixes
	for _, readOnly := range ReadOnlyBashCommands {
		if command == readOnly || strings.HasPrefix(command, readOnly+" ") {
			return true
		}
	}

	return false
}
