package apptypes

import "testing"

func TestProcessResult_String(t *testing.T) {
	t.Parallel()
	result := ProcessResult{
		Mode:    ProcessModeAllow,
		Message: "test message",
	}

	if result.Message != "test message" {
		t.Errorf("expected 'test message', got %s", result.Message)
	}
}
