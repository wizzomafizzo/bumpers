package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleWithToolsField(t *testing.T) {
	t.Parallel()

	yamlData := `
rules:
  - match:
      pattern: "^rm -rf"
    tool: "^(Bash|Task)$"
    send: "Dangerous command"
  - match:
      pattern: "password"
    tool: "^(Write|Edit)$"
    send: "No hardcoded secrets"
  - match:
      pattern: "test"
    send: "Bash only rule (no tool field)"
`

	config, err := LoadFromYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(config.Rules) != 3 {
		t.Fatalf("Expected 3 rules, got %d", len(config.Rules))
	}

	// First rule with tools field
	if config.Rules[0].GetMatch().Pattern != "^rm -rf" {
		t.Errorf("Expected pattern '^rm -rf', got '%s'", config.Rules[0].GetMatch().Pattern)
	}
	if config.Rules[0].Tool != "^(Bash|Task)$" {
		t.Errorf("Expected tools '^(Bash|Task)$', got '%s'", config.Rules[0].Tool)
	}

	// Second rule with tools field
	if config.Rules[1].Tool != "^(Write|Edit)$" {
		t.Errorf("Expected tools '^(Write|Edit)$', got '%s'", config.Rules[1].Tool)
	}

	// Third rule without tools field (should be empty string)
	if config.Rules[2].Tool != "" {
		t.Errorf("Expected empty tools field, got '%s'", config.Rules[2].Tool)
	}
}

func TestRuleValidationWithInvalidToolsRegex(t *testing.T) {
	t.Parallel()

	yamlData := `
rules:
  - match:
      pattern: "test"
    tool: "[invalid regex"
    send: "Test message"
`

	_, err := LoadFromYAML([]byte(yamlData))
	if err == nil {
		t.Fatal("Expected error for invalid tools regex, got nil")
	}

	expectedError := "invalid tools regex pattern"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
	}
}

func TestRuleValidationWithGenerateField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		yamlData    string
		errorText   string
		expectError bool
	}{
		{
			name: "valid generate field - once",
			yamlData: `
rules:
  - match:
      pattern: "test"
    send: "Test message"
    generate:
      mode: "once"
`,
			expectError: false,
		},
		{
			name: "valid generate field - session",
			yamlData: `
rules:
  - match:
      pattern: "test"
    send: "Test message"
    generate:
      mode: "session"
`,
			expectError: false,
		},
		{
			name: "valid generate field - always",
			yamlData: `
rules:
  - match:
      pattern: "test"
    send: "Test message"
    generate:
      mode: "always"
`,
			expectError: false,
		},
		{
			name: "valid generate field - off",
			yamlData: `
rules:
  - match:
      pattern: "test"  
    send: "Test message"
    generate:
      mode: "off"
`,
			expectError: false,
		},
		{
			name: "invalid generate field",
			yamlData: `
rules:
  - match:
      pattern: "test"
    send: "Test message"
    generate:
      mode: "invalid"
`,
			expectError: true,
			errorText:   "invalid generate mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testRuleGenerateValidation(t, tt)
		})
	}
}

// testRuleGenerateValidation is a helper function to reduce complexity
func testRuleGenerateValidation(t *testing.T, tt struct {
	name        string
	yamlData    string
	errorText   string
	expectError bool
},
) {
	_, err := LoadFromYAML([]byte(tt.yamlData))

	if tt.expectError {
		if err == nil {
			t.Fatalf("Expected error for %s, got nil", tt.name)
		}
		if !strings.Contains(err.Error(), tt.errorText) {
			t.Errorf("Expected error to contain '%s', got '%s'", tt.errorText, err.Error())
		}
		return
	}
	if err != nil {
		t.Errorf("Expected no error for %s, got %v", tt.name, err)
	}
}

func getEventSourcesTestCases() []struct {
	name            string
	yamlContent     string
	expectedEvent   string
	expectedSources []string
} {
	return []struct {
		name            string
		yamlContent     string
		expectedEvent   string
		expectedSources []string
	}{
		{
			name: "post-tool intent matching",
			yamlContent: `rules:
  - match:
      pattern: "not related to my changes"
      event: "post"
      sources: ["#intent"]
    send: "AI claiming unrelated"`,
			expectedEvent:   "post",
			expectedSources: []string{"#intent"},
		},
		{
			name: "pre-tool command matching",
			yamlContent: `rules:
  - match:
      pattern: "^rm -rf"
      event: "pre"
      sources: ["command"]
    send: "Dangerous deletion"`,
			expectedEvent:   "pre",
			expectedSources: []string{"command"},
		},
		{
			name: "tool output matching",
			yamlContent: `rules:
  - match:
      pattern: "error|failed"
      event: "post"
      sources: ["tool_response"]
    send: "Command failed"`,
			expectedEvent:   "post",
			expectedSources: []string{"tool_response"},
		},
		{
			name: "multiple field matching",
			yamlContent: `rules:
  - match:
      pattern: "password|secret"
      event: "pre"
      sources: ["command", "content"]
    send: "Avoid secrets"`,
			expectedEvent:   "pre",
			expectedSources: []string{"command", "content"},
		},
		{
			name: "defaults to pre event when omitted",
			yamlContent: `rules:
  - match:
      pattern: "dangerous"
      sources: ["command"]
    send: "Be careful"`,
			expectedEvent:   "pre",
			expectedSources: []string{"command"},
		},
		{
			name: "no default sources when omitted",
			yamlContent: `rules:
  - match:
      pattern: "unrelated"
      event: "post"
    send: "AI deflection"`,
			expectedEvent:   "post",
			expectedSources: []string{},
		},
	}
}

func runEventSourcesConfigurationTest(t *testing.T, tc struct {
	name            string
	yamlContent     string
	expectedEvent   string
	expectedSources []string
},
) {
	config, err := LoadFromYAML([]byte(tc.yamlContent))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if len(config.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(config.Rules))
	}

	rule := config.Rules[0]
	// Test the new GetMatch() method instead of old Event/Sources fields
	match := rule.GetMatch()
	assert.Equal(t, tc.expectedEvent, match.Event)
	assert.Equal(t, tc.expectedSources, match.Sources)
}

// TestEventSourcesConfiguration tests the new Event and Sources configuration syntax
func TestEventSourcesConfiguration(t *testing.T) {
	t.Parallel()

	testCases := getEventSourcesTestCases()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runEventSourcesConfigurationTest(t, tc)
		})
	}
}

// TestIntentSourceNoValidation tests that no validation occurs on source names
func TestIntentSourceNoValidation(t *testing.T) {
	t.Parallel()

	// Test that all sources are accepted without validation using new match format
	yamlContent := `rules:
  - match:
      pattern: "test"
      event: "pre"
      sources: ["#intent", "any_arbitrary_field", "command"]
    send: "Test message"`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(config.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(config.Rules))
	}

	rule := config.Rules[0]
	match := rule.GetMatch()
	if len(match.Sources) != 3 {
		t.Fatalf("Expected 3 sources, got %d", len(match.Sources))
	}

	expectedSources := []string{"#intent", "any_arbitrary_field", "command"}
	for i, expected := range expectedSources {
		if match.Sources[i] != expected {
			t.Errorf("Expected source %d to be %q, got %q", i, expected, match.Sources[i])
		}
	}
}

func TestStringMatchFormatAccepted(t *testing.T) {
	t.Parallel()

	// Test that string format is accepted (simple form)
	yamlWithStringFormat := `rules:
  - match: "test-pattern"
    send: "Test message"`

	config, err := LoadFromYAML([]byte(yamlWithStringFormat))
	if err != nil {
		t.Fatalf("Expected no error for string match format, got: %v", err)
	}

	if len(config.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(config.Rules))
	}

	match := config.Rules[0].GetMatch()
	if match.Pattern != "test-pattern" {
		t.Errorf("Expected pattern 'test-pattern', got '%s'", match.Pattern)
	}
	const expectedEvent = "pre"
	if match.Event != expectedEvent {
		t.Errorf("Expected default event '%s', got '%s'", expectedEvent, match.Event)
	}
	if len(match.Sources) != 0 {
		t.Errorf("Expected empty sources, got %v", match.Sources)
	}
}

// TestOldFormatIgnored tests that old event/sources format at rule level is ignored (breaking backward compatibility)
func TestOldFormatIgnored(t *testing.T) {
	t.Parallel()

	// Test that old format with event at rule level is ignored (defaults used instead)
	yamlWithOldEvent := `rules:
  - match:
      pattern: "test"
    send: "Test message"
    event: "post"`

	config, err := LoadFromYAML([]byte(yamlWithOldEvent))
	if err != nil {
		t.Fatalf("Expected no error for ignored old event format, got: %v", err)
	}

	// Verify the old event field was ignored and defaults are used (breaking backward compatibility)
	rule := config.Rules[0]
	match := rule.GetMatch()
	if match.Event != "pre" {
		t.Errorf("Expected default event 'pre' (old format ignored), got %q", match.Event)
	}

	// Test that old format with sources at rule level is ignored
	yamlWithOldSources := `rules:
  - match:
      pattern: "test"  
    send: "Test message"
    sources: ["command"]`

	config, err = LoadFromYAML([]byte(yamlWithOldSources))
	if err != nil {
		t.Fatalf("Expected no error for ignored old sources format, got: %v", err)
	}

	// Verify the old sources field was ignored and defaults are used (breaking backward compatibility)
	rule = config.Rules[0]
	match = rule.GetMatch()
	if len(match.Sources) != 0 {
		t.Errorf("Expected default empty sources (old format ignored), got %v", match.Sources)
	}
}

// TestMatchFieldParsing tests the new match field structure supporting both string and struct forms
func TestMatchFieldParsing(t *testing.T) {
	t.Parallel()

	t.Run("struct form with pattern only", func(t *testing.T) {
		t.Parallel()
		testMatchFieldCase(t, `rules:
  - match:
      pattern: "rm -rf"
    send: "Use safer deletion"`, "rm -rf", "pre", []string{})
	})

	t.Run("struct form with all fields", func(t *testing.T) {
		t.Parallel()
		testMatchFieldCase(t, `rules:
  - match:
      pattern: "not related to my changes"
      event: "post"
      sources: ["#intent"]
    send: "AI claiming unrelated"`, "not related to my changes", "post", []string{"#intent"})
	})

	t.Run("struct form with defaults", func(t *testing.T) {
		t.Parallel()
		testMatchFieldCase(t, `rules:
  - match:
      pattern: "production"
      sources: ["url"]
    send: "Production URL detected"`, "production", "pre", []string{"url"})
	})

	t.Run("struct form minimal", func(t *testing.T) {
		t.Parallel()
		testMatchFieldCase(t, `rules:
  - match:
      pattern: "password"
    send: "Avoid hardcoding secrets"`, "password", "pre", []string{})
	})

	t.Run("invalid - nil match field", func(t *testing.T) {
		t.Parallel()
		_, err := LoadFromYAML([]byte(`rules:
  - send: "Test message"`))
		assert.Error(t, err, "Expected error for invalid config")
	})
}

// testMatchFieldCase helper function for testing match field parsing
func testMatchFieldCase(t *testing.T, yamlContent, expectedPattern, expectedEvent string, expectedSources []string) {
	t.Helper()

	config, err := LoadFromYAML([]byte(yamlContent))
	require.NoError(t, err, "Config loading should not error")
	require.Len(t, config.Rules, 1, "Should have exactly 1 rule")

	rule := config.Rules[0]
	matchResult := rule.GetMatch()

	assert.Equal(t, expectedPattern, matchResult.Pattern)
	assert.Equal(t, expectedEvent, matchResult.Event)
	assert.Equal(t, expectedSources, matchResult.Sources)
}

// TestAddRule tests adding rules to existing configuration
func TestAddRule(t *testing.T) {
	t.Parallel()

	config := &Config{
		Rules: []Rule{
			{
				Match: "existing.*",
				Send:  "Existing rule",
			},
		},
	}

	// Add a new rule
	newRule := Rule{
		Match: "new.*",
		Send:  "New rule",
		Tool:  "^Bash$",
	}
	config.AddRule(newRule)

	// Verify the rule was added
	require.Len(t, config.Rules, 2, "Should have 2 rules after adding")

	// Check the new rule
	addedRule := config.Rules[1]
	assert.Equal(t, "new.*", addedRule.GetMatch().Pattern)
	assert.Equal(t, "New rule", addedRule.Send)
	assert.Equal(t, "^Bash$", addedRule.Tool)

	// Check the existing rule is still there
	existingRule := config.Rules[0]
	assert.Equal(t, "existing.*", existingRule.GetMatch().Pattern)
	assert.Equal(t, "Existing rule", existingRule.Send)
}

// TestDeleteRule tests removing rules from configuration by index
func TestDeleteRule(t *testing.T) {
	t.Parallel()

	config := &Config{
		Rules: []Rule{
			{
				Match: "rule1.*",
				Send:  "Rule 1",
			},
			{
				Match: "rule2.*",
				Send:  "Rule 2",
			},
			{
				Match: "rule3.*",
				Send:  "Rule 3",
			},
		},
	}

	// Test deleting middle rule
	err := config.DeleteRule(1)
	require.NoError(t, err, "Should be able to delete valid rule")
	require.Len(t, config.Rules, 2, "Should have 2 rules after deletion")

	// Verify correct rule was deleted
	assert.Equal(t, "rule1.*", config.Rules[0].GetMatch().Pattern)
	assert.Equal(t, "rule3.*", config.Rules[1].GetMatch().Pattern)

	// Test deleting invalid index (negative)
	err = config.DeleteRule(-1)
	require.Error(t, err, "Should error on negative index")
	require.Contains(t, err.Error(), "must be between 1 and", "Error should show 1-indexed ranges")

	// Test deleting invalid index (too large)
	err = config.DeleteRule(10)
	require.Error(t, err, "Should error on index too large")
	require.Contains(t, err.Error(), "must be between 1 and", "Error should show 1-indexed ranges")

	// Test deleting from empty config
	emptyConfig := &Config{Rules: []Rule{}}
	err = emptyConfig.DeleteRule(0)
	require.Error(t, err, "Should error when deleting from empty config")
}

// TestUpdateRule tests updating rules at specific indices
func TestUpdateRule(t *testing.T) {
	t.Parallel()

	config := &Config{
		Rules: []Rule{
			{
				Match: "original.*",
				Send:  "Original rule",
			},
			{
				Match: "another.*",
				Send:  "Another rule",
			},
		},
	}

	// Test updating first rule
	updatedRule := Rule{
		Match: "updated.*",
		Send:  "Updated rule",
		Tool:  "^Bash$",
	}

	err := config.UpdateRule(0, updatedRule)
	require.NoError(t, err, "Should be able to update valid rule")
	require.Len(t, config.Rules, 2, "Should still have 2 rules after update")

	// Verify rule was updated
	assert.Equal(t, "updated.*", config.Rules[0].GetMatch().Pattern)
	assert.Equal(t, "Updated rule", config.Rules[0].Send)
	assert.Equal(t, "^Bash$", config.Rules[0].Tool)

	// Verify other rule unchanged
	assert.Equal(t, "another.*", config.Rules[1].GetMatch().Pattern)

	// Test updating invalid index (negative)
	err = config.UpdateRule(-1, updatedRule)
	require.Error(t, err, "Should error on negative index")
	require.Contains(t, err.Error(), "must be between 1 and", "Error should show 1-indexed ranges")

	// Test updating invalid index (too large)
	err = config.UpdateRule(10, updatedRule)
	require.Error(t, err, "Should error on index too large")
	require.Contains(t, err.Error(), "must be between 1 and", "Error should show 1-indexed ranges")
}
