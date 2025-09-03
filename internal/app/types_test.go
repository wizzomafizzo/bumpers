package app

import (
	"encoding/json"
	"testing"
)

func TestUserPromptEvent_JSONMarshaling(t *testing.T) {
	t.Parallel()
	testPrompt := "test prompt content"
	event := UserPromptEvent{
		Prompt: testPrompt,
	}

	// Test marshaling
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal UserPromptEvent: %v", err)
	}

	// Test unmarshaling
	var unmarshaled UserPromptEvent
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal UserPromptEvent: %v", err)
	}

	if unmarshaled.Prompt != testPrompt {
		t.Errorf("Expected Prompt %q, got %q", testPrompt, unmarshaled.Prompt)
	}
}

func TestDecision_Constants(t *testing.T) {
	t.Parallel()
	// Test that decision constants have expected values
	if DecisionBlock != "block" {
		t.Errorf("Expected DecisionBlock to be 'block', got %q", DecisionBlock)
	}

	if DecisionAllow != "allow" {
		t.Errorf("Expected DecisionAllow to be 'allow', got %q", DecisionAllow)
	}
}

func TestProcessMode_Constants(t *testing.T) {
	t.Parallel()
	// Test that ProcessMode constants have expected values
	if ProcessModeAllow != "allow" {
		t.Errorf("Expected ProcessModeAllow to be 'allow', got %q", ProcessModeAllow)
	}

	if ProcessModeInformational != "informational" {
		t.Errorf("Expected ProcessModeInformational to be 'informational', got %q", ProcessModeInformational)
	}

	if ProcessModeBlock != "block" {
		t.Errorf("Expected ProcessModeBlock to be 'block', got %q", ProcessModeBlock)
	}
}
