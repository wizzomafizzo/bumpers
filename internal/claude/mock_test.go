package claude

import (
	"context"
	"testing"
)

const (
	testPrompt          = "test prompt"
	testResponse        = "Custom response"
	defaultMockResponse = "Mock response"
)

func createMockResponses() map[string]string {
	return map[string]string{
		".*test.*": "Test response",
		".*help.*": "Help response",
	}
}

func assertMockInitialState(t *testing.T, mock *MockLauncher) {
	t.Helper()

	if mock == nil {
		t.Fatal("MockLauncher should not be nil")
	}

	if mock.Response != "" {
		t.Errorf("Expected empty response, got %q", mock.Response)
	}

	if len(mock.Calls) != 0 {
		t.Errorf("Expected no calls, got %d", len(mock.Calls))
	}
}

func assertCallCount(t *testing.T, mock *MockLauncher, expected int) {
	t.Helper()

	if actual := mock.GetCallCount(); actual != expected {
		t.Errorf("Expected %d calls, got %d", expected, actual)
	}
}

func assertResponseEquals(t *testing.T, expected, actual string) {
	t.Helper()

	if actual != expected {
		t.Errorf("Expected response %q, got %q", expected, actual)
	}
}

func assertCallRecorded(t *testing.T, mock *MockLauncher, expectedPrompt string) {
	t.Helper()

	if len(mock.Calls) != 1 {
		t.Fatalf("Expected 1 call in Calls slice, got %d", len(mock.Calls))
	}

	if mock.Calls[0].Prompt != expectedPrompt {
		t.Errorf("Expected prompt %q, got %q", expectedPrompt, mock.Calls[0].Prompt)
	}
}

func TestNewMockLauncher_ReturnsValidInstance(t *testing.T) {
	t.Parallel()
	mock := NewMockLauncher()
	assertMockInitialState(t, mock)
}

func TestNewMockLauncherWithResponses_IgnoresResponses(t *testing.T) {
	t.Parallel()
	responses := createMockResponses()
	mock := NewMockLauncherWithResponses(responses)
	assertMockInitialState(t, mock)
}

func TestGetCallCount_InitiallyZero(t *testing.T) {
	t.Parallel()
	mock := NewMockLauncher()
	assertCallCount(t, mock, 0)
}

func TestSetResponseForPattern_SetsResponse(t *testing.T) {
	t.Parallel()
	mock := NewMockLauncher()

	mock.SetResponseForPattern(".*test.*", testResponse)

	if mock.Response != testResponse {
		t.Errorf("Expected response %q, got %q", testResponse, mock.Response)
	}
}

func TestGenerateMessage_DefaultResponse(t *testing.T) {
	t.Parallel()
	mock := NewMockLauncher()

	response, err := mock.GenerateMessage(context.TODO(), testPrompt)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	assertResponseEquals(t, defaultMockResponse, response)
	assertCallCount(t, mock, 1)
	assertCallRecorded(t, mock, testPrompt)
}
