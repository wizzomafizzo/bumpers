package settings

import (
	"testing"
)

func TestSettings_AddHook(t *testing.T) {
	t.Parallel()
	// Test adding a hook to an empty settings
	settings := &Settings{}

	command := HookCommand{
		Type:    "command",
		Command: "echo 'test'",
	}

	err := settings.AddHook(PreToolUseEvent, "Write|Edit", command)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify the hook was added
	if settings.Hooks == nil {
		t.Fatal("Expected hooks to be initialized")
	}

	if len(settings.Hooks.PreToolUse) != 1 {
		t.Errorf("Expected 1 PreToolUse hook, got %d", len(settings.Hooks.PreToolUse))
	}

	matcher := settings.Hooks.PreToolUse[0]
	if matcher.Matcher != "Write|Edit" {
		t.Errorf("Expected matcher 'Write|Edit', got %s", matcher.Matcher)
	}

	if len(matcher.Hooks) != 1 {
		t.Errorf("Expected 1 hook command, got %d", len(matcher.Hooks))
	}

	hook := matcher.Hooks[0]
	if hook.Type != "command" {
		t.Errorf("Expected hook type 'command', got %s", hook.Type)
	}
	if hook.Command != "echo 'test'" {
		t.Errorf("Expected hook command 'echo 'test'', got %s", hook.Command)
	}
}

func TestSettings_RemoveHook(t *testing.T) {
	t.Parallel()
	// Setup: Add a hook first
	settings := &Settings{}
	command := HookCommand{
		Type:    "command",
		Command: "echo 'test'",
	}

	err := settings.AddHook(PreToolUseEvent, "Write|Edit", command)
	if err != nil {
		t.Fatalf("Failed to add hook: %v", err)
	}

	// Test: Remove the hook
	err = settings.RemoveHook(PreToolUseEvent, "Write|Edit")
	if err != nil {
		t.Fatalf("Expected no error removing hook, got %v", err)
	}

	// Verify: Hook should be removed
	if len(settings.Hooks.PreToolUse) != 0 {
		t.Errorf("Expected 0 PreToolUse hooks after removal, got %d", len(settings.Hooks.PreToolUse))
	}
}

func TestSettings_AddHook_MultipleEvents(t *testing.T) {
	t.Parallel()
	// Test adding hooks to different events
	settings := &Settings{}

	preCommand := HookCommand{Type: "command", Command: "pre-tool"}
	postCommand := HookCommand{Type: "command", Command: "post-tool"}

	// Add to PreToolUse
	err := settings.AddHook(PreToolUseEvent, "Write", preCommand)
	if err != nil {
		t.Fatalf("Failed to add PreToolUse hook: %v", err)
	}

	// Add to PostToolUse
	err = settings.AddHook(PostToolUseEvent, "Bash", postCommand)
	if err != nil {
		t.Fatalf("Failed to add PostToolUse hook: %v", err)
	}

	// Verify both hooks exist
	if len(settings.Hooks.PreToolUse) != 1 {
		t.Errorf("Expected 1 PreToolUse hook, got %d", len(settings.Hooks.PreToolUse))
	}
	if len(settings.Hooks.PostToolUse) != 1 {
		t.Errorf("Expected 1 PostToolUse hook, got %d", len(settings.Hooks.PostToolUse))
	}
}

func TestSettings_AddHook_Validation(t *testing.T) {
	t.Parallel()
	settings := &Settings{}

	// Test empty matcher (now allowed - matches everything)
	err := settings.AddHook(PreToolUseEvent, "", HookCommand{Type: "command", Command: "test"})
	if err != nil {
		t.Errorf("Unexpected error for empty matcher (should be allowed): %v", err)
	}

	// Test empty command type
	err = settings.AddHook(PreToolUseEvent, ".*", HookCommand{Type: "", Command: "test"})
	if err == nil {
		t.Error("Expected error for empty command type")
	}

	// Test empty command
	err = settings.AddHook(PreToolUseEvent, ".*", HookCommand{Type: "command", Command: ""})
	if err == nil {
		t.Error("Expected error for empty command")
	}
}

func TestSettings_ListHooks(t *testing.T) {
	t.Parallel()
	settings := &Settings{}

	// Add some hooks
	cmd1 := HookCommand{Type: "command", Command: "test1"}
	cmd2 := HookCommand{Type: "command", Command: "test2"}

	_ = settings.AddHook(PreToolUseEvent, "Write", cmd1)
	_ = settings.AddHook(PostToolUseEvent, "Bash", cmd2)

	// List PreToolUse hooks
	hooks, err := settings.ListHooks(PreToolUseEvent)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(hooks) != 1 {
		t.Errorf("Expected 1 hook, got %d", len(hooks))
	}
	if hooks[0].Matcher != "Write" {
		t.Errorf("Expected matcher 'Write', got %s", hooks[0].Matcher)
	}
}

func TestSettings_FindHook(t *testing.T) {
	t.Parallel()
	settings := &Settings{}

	// Add a hook
	cmd := HookCommand{Type: "command", Command: "test"}
	_ = settings.AddHook(PreToolUseEvent, "Write|Edit", cmd)

	// Find the hook
	hook, err := settings.FindHook(PreToolUseEvent, "Write|Edit")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hook.Matcher != "Write|Edit" {
		t.Errorf("Expected matcher 'Write|Edit', got %s", hook.Matcher)
	}
	if len(hook.Hooks) != 1 {
		t.Errorf("Expected 1 command, got %d", len(hook.Hooks))
	}
	if hook.Hooks[0].Command != "test" {
		t.Errorf("Expected command 'test', got %s", hook.Hooks[0].Command)
	}
}

func TestSettings_UpdateHook(t *testing.T) {
	t.Parallel()
	settings := &Settings{}

	// Add initial hook
	initialCmd := HookCommand{Type: "command", Command: "initial-command"}
	err := settings.AddHook(PreToolUseEvent, "Write|Edit", initialCmd)
	if err != nil {
		t.Fatalf("Failed to add initial hook: %v", err)
	}

	// Update the hook with new command
	updatedCmd := HookCommand{Type: "command", Command: "updated-command", Timeout: 30}
	err = settings.UpdateHook(PreToolUseEvent, "Write|Edit", "Write|Edit", updatedCmd)
	if err != nil {
		t.Fatalf("Expected no error updating hook, got %v", err)
	}

	// Verify the hook was updated
	hook, err := settings.FindHook(PreToolUseEvent, "Write|Edit")
	if err != nil {
		t.Fatalf("Failed to find updated hook: %v", err)
	}

	if len(hook.Hooks) != 1 {
		t.Errorf("Expected 1 command after update, got %d", len(hook.Hooks))
	}

	updatedHook := hook.Hooks[0]
	if updatedHook.Command != "updated-command" {
		t.Errorf("Expected command 'updated-command', got %s", updatedHook.Command)
	}
	if updatedHook.Timeout != 30 {
		t.Errorf("Expected timeout 30, got %d", updatedHook.Timeout)
	}
}

func TestSettings_UpdateHook_PostToolUse(t *testing.T) {
	t.Parallel()
	settings := &Settings{}

	// Add initial hook to PostToolUse
	initialCmd := HookCommand{Type: "command", Command: "initial-post-command"}
	err := settings.AddHook(PostToolUseEvent, "Bash", initialCmd)
	if err != nil {
		t.Fatalf("Failed to add initial hook: %v", err)
	}

	// Update the hook
	updatedCmd := HookCommand{Type: "command", Command: "updated-post-command"}
	err = settings.UpdateHook(PostToolUseEvent, "Bash", "Bash", updatedCmd)
	if err != nil {
		t.Fatalf("Expected no error updating PostToolUse hook, got %v", err)
	}

	// Verify the hook was updated
	hook, err := settings.FindHook(PostToolUseEvent, "Bash")
	if err != nil {
		t.Fatalf("Failed to find updated hook: %v", err)
	}

	if hook.Hooks[0].Command != "updated-post-command" {
		t.Errorf("Expected command 'updated-post-command', got %s", hook.Hooks[0].Command)
	}
}

func TestSettings_RemoveHook_PostToolUse(t *testing.T) {
	t.Parallel()
	settings := &Settings{}

	// Add hooks to both events
	preCmd := HookCommand{Type: "command", Command: "pre-tool"}
	postCmd := HookCommand{Type: "command", Command: "post-tool"}

	_ = settings.AddHook(PreToolUseEvent, "Write", preCmd)
	_ = settings.AddHook(PostToolUseEvent, "Bash", postCmd)

	// Remove only the PostToolUse hook
	err := settings.RemoveHook(PostToolUseEvent, "Bash")
	if err != nil {
		t.Fatalf("Expected no error removing PostToolUse hook, got %v", err)
	}

	// Verify PostToolUse hook is removed
	if len(settings.Hooks.PostToolUse) != 0 {
		t.Errorf("Expected 0 PostToolUse hooks after removal, got %d", len(settings.Hooks.PostToolUse))
	}

	// Verify PreToolUse hook remains
	if len(settings.Hooks.PreToolUse) != 1 {
		t.Errorf("Expected 1 PreToolUse hook to remain, got %d", len(settings.Hooks.PreToolUse))
	}
}

func TestSettings_AddHook_UserPromptSubmit(t *testing.T) {
	t.Parallel()
	settings := &Settings{}

	command := HookCommand{
		Type:    "command",
		Command: "echo 'user prompt submitted'",
	}

	err := settings.AddHook(UserPromptSubmitEvent, ".*", command)
	if err != nil {
		t.Fatalf("Expected no error adding UserPromptSubmit hook, got %v", err)
	}

	// Verify the hook was added
	if len(settings.Hooks.UserPromptSubmit) != 1 {
		t.Errorf("Expected 1 UserPromptSubmit hook, got %d", len(settings.Hooks.UserPromptSubmit))
	}
}

func TestSettings_RemoveHook_UserPromptSubmit(t *testing.T) {
	t.Parallel()
	settings := &Settings{}

	// Add UserPromptSubmit hook
	cmd := HookCommand{Type: "command", Command: "test"}
	_ = settings.AddHook(UserPromptSubmitEvent, ".*", cmd)

	// Remove it
	err := settings.RemoveHook(UserPromptSubmitEvent, ".*")
	if err != nil {
		t.Fatalf("Expected no error removing UserPromptSubmit hook, got %v", err)
	}

	// Verify it's removed
	if len(settings.Hooks.UserPromptSubmit) != 0 {
		t.Errorf("Expected 0 UserPromptSubmit hooks after removal, got %d", len(settings.Hooks.UserPromptSubmit))
	}
}

func TestSettings_AddHook_SessionStart(t *testing.T) {
	t.Parallel()
	settings := &Settings{}

	command := HookCommand{
		Type:    "command",
		Command: "echo 'session started'",
		Timeout: 10,
	}

	err := settings.AddHook(SessionStartEvent, ".*", command)
	if err != nil {
		t.Fatalf("Expected no error adding SessionStart hook, got %v", err)
	}

	// Verify the hook was added with timeout
	if len(settings.Hooks.SessionStart) != 1 {
		t.Errorf("Expected 1 SessionStart hook, got %d", len(settings.Hooks.SessionStart))
	}

	hook := settings.Hooks.SessionStart[0].Hooks[0]
	if hook.Timeout != 10 {
		t.Errorf("Expected timeout 10, got %d", hook.Timeout)
	}
}

func TestSettings_AddHook_Stop(t *testing.T) {
	t.Parallel()
	settings := &Settings{}

	command := HookCommand{
		Type:    "command",
		Command: "echo 'stopping'",
	}

	err := settings.AddHook(StopEvent, ".*", command)
	if err != nil {
		t.Fatalf("Expected no error adding Stop hook, got %v", err)
	}

	// Verify the hook was added
	if len(settings.Hooks.Stop) != 1 {
		t.Errorf("Expected 1 Stop hook, got %d", len(settings.Hooks.Stop))
	}
}

func TestSettings_AddHook_SubagentStop(t *testing.T) {
	t.Parallel()
	settings := &Settings{}

	command := HookCommand{
		Type:    "command",
		Command: "echo 'subagent stopping'",
	}

	err := settings.AddHook(SubagentStopEvent, ".*", command)
	if err != nil {
		t.Fatalf("Expected no error adding SubagentStop hook, got %v", err)
	}

	// Verify the hook was added
	if len(settings.Hooks.SubagentStop) != 1 {
		t.Errorf("Expected 1 SubagentStop hook, got %d", len(settings.Hooks.SubagentStop))
	}
}

func TestSettings_AddHook_PreCompact(t *testing.T) {
	t.Parallel()
	settings := &Settings{}

	command := HookCommand{
		Type:    "command",
		Command: "echo 'pre compact'",
	}

	err := settings.AddHook(PreCompactEvent, ".*", command)
	if err != nil {
		t.Fatalf("Expected no error adding PreCompact hook, got %v", err)
	}

	// Verify the hook was added
	if len(settings.Hooks.PreCompact) != 1 {
		t.Errorf("Expected 1 PreCompact hook, got %d", len(settings.Hooks.PreCompact))
	}
}

func TestSettings_AddHook_Notification(t *testing.T) {
	t.Parallel()
	settings := &Settings{}

	command := HookCommand{
		Type:    "command",
		Command: "echo 'notification'",
	}

	err := settings.AddHook(NotificationEvent, ".*", command)
	if err != nil {
		t.Fatalf("Expected no error adding Notification hook, got %v", err)
	}

	// Verify the hook was added
	if len(settings.Hooks.Notification) != 1 {
		t.Errorf("Expected 1 Notification hook, got %d", len(settings.Hooks.Notification))
	}
}

func TestSettings_All8HookEvents(t *testing.T) {
	t.Parallel()
	settings := &Settings{}

	// Test all 8 hook events
	events := []HookEvent{
		PreToolUseEvent,
		PostToolUseEvent,
		UserPromptSubmitEvent,
		SessionStartEvent,
		StopEvent,
		SubagentStopEvent,
		PreCompactEvent,
		NotificationEvent,
	}

	for _, event := range events {
		command := HookCommand{
			Type:    "command",
			Command: "echo 'test'",
		}

		err := settings.AddHook(event, ".*", command)
		if err != nil {
			t.Fatalf("Expected no error adding hook for event %s, got %v", event, err)
		}

		// Verify the hook can be found
		hooks, err := settings.ListHooks(event)
		if err != nil {
			t.Fatalf("Expected no error listing hooks for event %s, got %v", event, err)
		}

		if len(hooks) != 1 {
			t.Errorf("Expected 1 hook for event %s, got %d", event, len(hooks))
		}
	}
}

func TestSettings_AddHook_PreventsDuplicates(t *testing.T) {
	t.Parallel()
	// Test that AddHook prevents duplicate matchers for the same event
	settings := &Settings{}

	command1 := HookCommand{
		Type:    "command",
		Command: "echo 'first'",
	}

	command2 := HookCommand{
		Type:    "command",
		Command: "echo 'second'",
	}

	// Add first hook
	err := settings.AddHook(PreToolUseEvent, "Write|Edit", command1)
	if err != nil {
		t.Fatalf("Expected no error adding first hook, got %v", err)
	}

	// Try to add duplicate hook with same matcher - this should fail
	err = settings.AddHook(PreToolUseEvent, "Write|Edit", command2)
	if err == nil {
		t.Error("Expected error when adding duplicate matcher, got nil")
	}

	// Verify only the first hook exists
	if len(settings.Hooks.PreToolUse) != 1 {
		t.Errorf("Expected 1 hook after attempting duplicate, got %d", len(settings.Hooks.PreToolUse))
	}

	// Verify the original hook is unchanged
	if settings.Hooks.PreToolUse[0].Hooks[0].Command != "echo 'first'" {
		t.Error("Original hook was modified when duplicate was rejected")
	}
}

func TestRemoveHook_UnusedParameterBehavior(t *testing.T) {
	t.Parallel()
	// Test that RemoveHook removes entire matchers regardless of commandToRemove parameter
	settings := &Settings{}

	command := HookCommand{
		Type:    "command",
		Command: "echo 'test'",
	}

	// Add a hook
	err := settings.AddHook(PreToolUseEvent, "Write|Edit", command)
	if err != nil {
		t.Fatalf("Failed to add hook: %v", err)
	}

	// Remove hook using the corrected signature (no unused parameter)
	err = settings.RemoveHook(PreToolUseEvent, "Write|Edit")
	if err != nil {
		t.Fatalf("RemoveHook failed: %v", err)
	}

	// Verify hook was removed
	if len(settings.Hooks.PreToolUse) != 0 {
		t.Errorf("Expected 0 hooks after removal, got %d", len(settings.Hooks.PreToolUse))
	}

	// Test removing again to ensure it works consistently
	_ = settings.AddHook(PreToolUseEvent, "Write|Edit", command)
	err = settings.RemoveHook(PreToolUseEvent, "Write|Edit")
	if err != nil {
		t.Fatalf("RemoveHook failed on second removal: %v", err)
	}

	if len(settings.Hooks.PreToolUse) != 0 {
		t.Errorf("Expected 0 hooks after second removal, got %d", len(settings.Hooks.PreToolUse))
	}
}

func TestSettings_AddOrAppendHook(t *testing.T) {
	t.Parallel()

	// Test adding a hook to a new matcher
	settings := &Settings{}

	command1 := HookCommand{
		Type:    "command",
		Command: "echo 'first'",
	}

	err := settings.AddOrAppendHook(PreToolUseEvent, "Bash", command1)
	if err != nil {
		t.Fatalf("Failed to add first hook: %v", err)
	}

	// Verify hook was added
	if len(settings.Hooks.PreToolUse) != 1 {
		t.Errorf("Expected 1 matcher, got %d", len(settings.Hooks.PreToolUse))
	}
	if len(settings.Hooks.PreToolUse[0].Hooks) != 1 {
		t.Errorf("Expected 1 command, got %d", len(settings.Hooks.PreToolUse[0].Hooks))
	}
	if settings.Hooks.PreToolUse[0].Hooks[0].Command != "echo 'first'" {
		t.Errorf("Expected 'echo 'first'', got %s", settings.Hooks.PreToolUse[0].Hooks[0].Command)
	}

	// Test appending to existing matcher
	command2 := HookCommand{
		Type:    "command",
		Command: "echo 'second'",
	}

	err = settings.AddOrAppendHook(PreToolUseEvent, "Bash", command2)
	if err != nil {
		t.Fatalf("Failed to append second hook: %v", err)
	}

	// Verify both commands exist
	if len(settings.Hooks.PreToolUse) != 1 {
		t.Errorf("Expected still 1 matcher, got %d", len(settings.Hooks.PreToolUse))
	}
	if len(settings.Hooks.PreToolUse[0].Hooks) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(settings.Hooks.PreToolUse[0].Hooks))
	}

	// Verify commands are preserved
	commands := settings.Hooks.PreToolUse[0].Hooks
	if commands[0].Command != "echo 'first'" {
		t.Errorf("Expected first command 'echo 'first'', got %s", commands[0].Command)
	}
	if commands[1].Command != "echo 'second'" {
		t.Errorf("Expected second command 'echo 'second'', got %s", commands[1].Command)
	}
}

func TestSettings_AddOrAppendHook_PreventsDuplicates(t *testing.T) {
	t.Parallel()

	settings := &Settings{}

	command := HookCommand{
		Type:    "command",
		Command: "bumpers",
	}

	// Add command first time
	err := settings.AddOrAppendHook(PreToolUseEvent, "Bash", command)
	if err != nil {
		t.Fatalf("Failed to add hook: %v", err)
	}

	// Try to add same command again
	err = settings.AddOrAppendHook(PreToolUseEvent, "Bash", command)
	if err != nil {
		t.Fatalf("AddOrAppendHook should not error on duplicate: %v", err)
	}

	// Verify only one command exists
	if len(settings.Hooks.PreToolUse[0].Hooks) != 1 {
		t.Errorf("Expected 1 command after duplicate attempt, got %d", len(settings.Hooks.PreToolUse[0].Hooks))
	}
}

func TestSettings_AddOrAppendHook_UserPromptSubmit(t *testing.T) {
	t.Parallel()

	// Test with empty matcher (UserPromptSubmit case)
	settings := &Settings{}

	command1 := HookCommand{
		Type:    "command",
		Command: "tdd-guard-go",
	}

	command2 := HookCommand{
		Type:    "command",
		Command: "bumpers",
	}

	// Add first command with empty matcher
	err := settings.AddOrAppendHook(UserPromptSubmitEvent, "", command1)
	if err != nil {
		t.Fatalf("Failed to add first hook: %v", err)
	}

	// Add second command to same empty matcher
	err = settings.AddOrAppendHook(UserPromptSubmitEvent, "", command2)
	if err != nil {
		t.Fatalf("Failed to append second hook: %v", err)
	}

	// Verify both commands exist
	if len(settings.Hooks.UserPromptSubmit) != 1 {
		t.Errorf("Expected 1 matcher, got %d", len(settings.Hooks.UserPromptSubmit))
	}
	if len(settings.Hooks.UserPromptSubmit[0].Hooks) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(settings.Hooks.UserPromptSubmit[0].Hooks))
	}
}
