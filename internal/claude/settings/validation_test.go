package settings

import (
	"testing"
)

func TestSettings_Validate_ValidSettings(t *testing.T) {
	t.Parallel()
	settings := &Settings{
		OutputStyle: "default",
		Model:       "claude-3-opus",
		Permissions: &Permissions{
			Allow: []string{"Read", "Write"},
		},
	}

	result := settings.Validate()

	if !result.Valid {
		t.Error("Settings.Validate().Valid = false, want true")
	}

	if len(result.Errors) != 0 {
		t.Errorf("Settings.Validate() returned %d errors, want 0: %v", len(result.Errors), result.Errors)
	}
}

func TestSettings_Validate_InvalidOutputStyle(t *testing.T) {
	t.Parallel()
	settings := &Settings{
		OutputStyle: "invalid-style",
	}

	result := settings.Validate()

	if result.Valid {
		t.Error("Settings.Validate().Valid = true, want false for invalid output style")
	}

	if len(result.Errors) == 0 {
		t.Error("Settings.Validate() returned no errors, want at least one for invalid output style")
	}
}

func TestSettings_Validate_InvalidHookCommand(t *testing.T) {
	t.Parallel()
	settings := &Settings{
		Hooks: &Hooks{
			PreToolUse: []HookMatcher{
				{
					Matcher: "test-matcher",
					Hooks: []HookCommand{
						{
							Type:    "", // Invalid: empty type
							Command: "echo test",
						},
					},
				},
			},
		},
	}

	result := settings.Validate()

	if result.Valid {
		t.Error("Settings.Validate().Valid = true, want false for invalid hook command")
	}

	if len(result.Errors) == 0 {
		t.Error("Settings.Validate() returned no errors, want at least one for invalid hook command")
	}
}

func TestSettings_Validate_EmptyHookCommand(t *testing.T) {
	t.Parallel()
	settings := &Settings{
		Hooks: &Hooks{
			PostToolUse: []HookMatcher{
				{
					Matcher: "test-matcher",
					Hooks: []HookCommand{
						{
							Type:    "command",
							Command: "", // Invalid: empty command
						},
					},
				},
			},
		},
	}

	result := settings.Validate()

	if result.Valid {
		t.Error("Settings.Validate().Valid = true, want false for empty hook command")
	}

	if len(result.Errors) == 0 {
		t.Error("Settings.Validate() returned no errors, want at least one for empty hook command")
	}
}

// Table-driven test for settings validation
func getSettingsValidationTestCases() []struct {
	settings *Settings
	name     string
	wantErr  bool
} {
	return []struct {
		settings *Settings
		name     string
		wantErr  bool
	}{
		{
			name: "valid settings",
			settings: &Settings{
				OutputStyle: "default",
				Model:       "claude-3-opus",
				Permissions: &Permissions{
					Allow: []string{"Read", "Write"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid output style",
			settings: &Settings{
				OutputStyle: "invalid-style",
			},
			wantErr: true,
		},
		{
			name: "invalid hook command type",
			settings: &Settings{
				Hooks: &Hooks{
					PreToolUse: []HookMatcher{
						{
							Matcher: "test-matcher",
							Hooks: []HookCommand{
								{
									Type:    "",
									Command: "echo test",
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty hook command",
			settings: &Settings{
				Hooks: &Hooks{
					PostToolUse: []HookMatcher{
						{
							Matcher: "test-matcher",
							Hooks: []HookCommand{
								{
									Type:    "command",
									Command: "",
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}
}

func assertValidationResult(t *testing.T, result *ValidationResult, tc struct {
	settings *Settings
	name     string
	wantErr  bool
},
) {
	t.Helper()
	if tc.wantErr {
		assertValidationShouldFail(t, result, tc.name)
		return
	}
	assertValidationShouldPass(t, result, tc.name)
}

func assertValidationShouldFail(t *testing.T, result *ValidationResult, testName string) {
	t.Helper()
	if result.Valid {
		t.Errorf("Settings.Validate().Valid = true, want false for %s", testName)
	}
	if len(result.Errors) == 0 {
		t.Errorf("Settings.Validate() returned no errors, want at least one for %s", testName)
	}
}

func assertValidationShouldPass(t *testing.T, result *ValidationResult, testName string) {
	t.Helper()
	if !result.Valid {
		t.Errorf("Settings.Validate().Valid = false, want true for %s", testName)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Settings.Validate() returned %d errors, want 0 for %s: %v",
			len(result.Errors), testName, result.Errors)
	}
}

func TestSettingsValidate(t *testing.T) {
	t.Parallel()

	tests := getSettingsValidationTestCases()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.settings.Validate()
			assertValidationResult(t, result, tc)
		})
	}
}
