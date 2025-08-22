package template

import "time"

// SharedContext contains variables available to all template types
type SharedContext struct {
	Today string
}

// RuleContext contains variables specific to rule templates
type RuleContext struct {
	Command string
}

// CommandContext contains variables specific to command templates
type CommandContext struct {
	Name string
}

// NoteContext contains variables specific to note templates
// Currently empty but provided for consistency and future expansion
type NoteContext struct{}

// NewSharedContext creates a new shared context with current date
func NewSharedContext() SharedContext {
	return SharedContext{
		Today: time.Now().Format("2006-01-02"),
	}
}

// MergeContexts combines shared context with type-specific context
// Returns a map that can be used with template.Execute
func MergeContexts(shared SharedContext, specific any) map[string]any {
	result := make(map[string]any)
	result["Today"] = shared.Today

	if ruleCtx, ok := specific.(RuleContext); ok {
		result["Command"] = ruleCtx.Command
	}

	if cmdCtx, ok := specific.(CommandContext); ok {
		result["Name"] = cmdCtx.Name
	}

	return result
}

// BuildRuleContext creates a complete context for rule templates
func BuildRuleContext(command string) map[string]any {
	shared := NewSharedContext()
	specific := RuleContext{Command: command}
	return MergeContexts(shared, specific)
}

// BuildCommandContext creates a complete context for command templates
func BuildCommandContext(name string) map[string]any {
	shared := NewSharedContext()
	specific := CommandContext{Name: name}
	return MergeContexts(shared, specific)
}

// BuildNoteContext creates a complete context for note templates
func BuildNoteContext() map[string]any {
	shared := NewSharedContext()
	specific := NoteContext{}
	return MergeContexts(shared, specific)
}
