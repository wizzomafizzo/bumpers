package claude

import "testing"

func TestCommonTestResponses_HasExpectedKeys(t *testing.T) {
	t.Parallel()
	expectedKeys := []string{
		".*just test.*",
		".*go test.*",
		".*rm -rf.*",
		".*password.*",
		".*secret.*",
		".*npm.*",
	}

	if len(CommonTestResponses) != len(expectedKeys) {
		t.Errorf("Expected %d common test responses, got %d", len(expectedKeys), len(CommonTestResponses))
	}

	for _, key := range expectedKeys {
		if _, exists := CommonTestResponses[key]; !exists {
			t.Errorf("Expected key %q not found in CommonTestResponses", key)
		}
	}
}

func TestCommonTestErrors_AreNotNil(t *testing.T) {
	t.Parallel()
	if ErrMockNetworkTimeout == nil {
		t.Error("ErrMockNetworkTimeout should not be nil")
	}
	if ErrMockAPIError == nil {
		t.Error("ErrMockAPIError should not be nil")
	}
	if ErrMockInvalidJSON == nil {
		t.Error("ErrMockInvalidJSON should not be nil")
	}
}

func TestSetupMockLauncherWithDefaults_ReturnsValidMock(t *testing.T) {
	t.Parallel()
	mock := SetupMockLauncherWithDefaults()

	if mock == nil {
		t.Fatal("SetupMockLauncherWithDefaults returned nil")
	}

	if mock.GetCallCount() != 0 {
		t.Errorf("Expected new mock to have 0 calls, got %d", mock.GetCallCount())
	}
}

func TestAssertMockCalled_WithMatchingCalls(t *testing.T) {
	t.Parallel()
	mock := NewMockLauncher()

	// Add some mock calls manually
	mock.Calls = []MockCall{
		{Prompt: "test1"},
		{Prompt: "test2"},
	}

	// This should not fail
	AssertMockCalled(t, mock, 2)
}

func TestAssertMockNotCalled_WithNoCalls(t *testing.T) {
	t.Parallel()
	mock := NewMockLauncher()

	// This should not fail
	AssertMockNotCalled(t, mock)
}
