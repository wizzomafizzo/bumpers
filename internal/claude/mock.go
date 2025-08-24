package claude

// MockCall represents a single call to the mock launcher
type MockCall struct {
	Prompt string
}

// MockLauncher provides a mock implementation for testing
type MockLauncher struct {
	Response string
	Calls    []MockCall
}

// NewMockLauncher creates a new mock launcher
func NewMockLauncher() *MockLauncher {
	return &MockLauncher{}
}

// NewMockLauncherWithResponses creates a mock with responses
func NewMockLauncherWithResponses(_ map[string]string) *MockLauncher {
	return &MockLauncher{}
}

// GetCallCount returns number of calls
func (m *MockLauncher) GetCallCount() int {
	return len(m.Calls)
}

// WasCalledWithPattern checks if called with pattern
func (*MockLauncher) WasCalledWithPattern(_ string) bool {
	return false // TODO: implement
}

// SetResponseForPattern sets response for pattern
func (m *MockLauncher) SetResponseForPattern(_, response string) {
	m.Response = response
}

// GenerateMessage implements MessageGenerator interface
func (m *MockLauncher) GenerateMessage(prompt string) (string, error) {
	m.Calls = append(m.Calls, MockCall{Prompt: prompt})
	if m.Response != "" {
		return m.Response, nil
	}
	return "Mock response", nil
}
