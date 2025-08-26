//go:build integration

package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	t.Parallel()
	// Create a temporary settings file
	tempDir := t.TempDir()
	settingsFile := filepath.Join(tempDir, "settings.json")

	// Write test settings content
	content := `{
		"permissions": {
			"allow": ["WebSearch", "Bash(find:*)"],
			"deny": [],
			"ask": []
		},
		"hooks": {
			"PreToolUse": [
				{
					"matcher": "Write|Edit",
					"hooks": [
						{
							"type": "command",
							"command": "tdd-guard"
						}
					]
				}
			]
		},
		"outputStyle": "Explanatory"
	}`

	err := os.WriteFile(settingsFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test settings file: %v", err)
	}

	// Test loading the file
	settings, err := LoadFromFile(settingsFile)
	if err != nil {
		t.Fatalf("Failed to load settings from file: %v", err)
	}

	// Verify the loaded settings
	if settings == nil {
		t.Fatal("LoadFromFile returned nil settings")
	}

	if settings.OutputStyle != "Explanatory" {
		t.Errorf("Expected output style 'Explanatory', got %s", settings.OutputStyle)
	}

	if settings.Permissions == nil {
		t.Fatal("Permissions should not be nil")
	}

	if len(settings.Permissions.Allow) != 2 {
		t.Errorf("Expected 2 allow permissions, got %d", len(settings.Permissions.Allow))
	}
}

func TestLoadFromFile_DifferentContent(t *testing.T) {
	t.Parallel()
	// Create a temporary settings file with different content
	tempDir := t.TempDir()
	settingsFile := filepath.Join(tempDir, "settings.json")

	// Write different test settings content
	content := `{
		"outputStyle": "Concise",
		"model": "claude-3-5-sonnet-20241022"
	}`

	err := os.WriteFile(settingsFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test settings file: %v", err)
	}

	// Test loading the file
	settings, err := LoadFromFile(settingsFile)
	if err != nil {
		t.Fatalf("Failed to load settings from file: %v", err)
	}

	// Verify the loaded settings match the file content
	if settings.OutputStyle != "Concise" {
		t.Errorf("Expected output style 'Concise', got %s", settings.OutputStyle)
	}

	if settings.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected model 'claude-3-5-sonnet-20241022', got %s", settings.Model)
	}
}

func TestCreateBackup(t *testing.T) {
	t.Parallel()
	// Create a temporary directory for test
	tempDir := t.TempDir()
	settingsFile := filepath.Join(tempDir, "settings.json")

	// Create original settings file
	originalSettings := &Settings{
		OutputStyle: "explanatory",
		Model:       "claude-3-5-sonnet-20241022",
	}

	err := SaveToFile(originalSettings, settingsFile)
	if err != nil {
		t.Fatalf("Failed to create original settings file: %v", err)
	}

	// Test creating a backup
	backupPath, err := CreateBackup(settingsFile)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify backup file exists
	if _, statErr := os.Stat(backupPath); os.IsNotExist(statErr) {
		t.Fatalf("Backup file was not created at %s", backupPath)
	}

	// Verify backup contains the same content
	backupSettings, err := LoadFromFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to load backup file: %v", err)
	}

	if backupSettings.OutputStyle != originalSettings.OutputStyle {
		t.Errorf("Backup output style mismatch: expected %s, got %s",
			originalSettings.OutputStyle, backupSettings.OutputStyle)
	}

	if backupSettings.Model != originalSettings.Model {
		t.Errorf("Backup model mismatch: expected %s, got %s",
			originalSettings.Model, backupSettings.Model)
	}
}

func TestBackupHelpers(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	settingsFile := filepath.Join(tempDir, "settings.json")

	// Initially no backup should exist
	if HasBackup(settingsFile) {
		t.Error("HasBackup should return false when no backup exists")
	}

	// Create original settings file
	settings := &Settings{OutputStyle: "explanatory"}
	err := SaveToFile(settings, settingsFile)
	if err != nil {
		t.Fatalf("Failed to create settings file: %v", err)
	}

	// Create backup
	backupPath, err := CreateBackup(settingsFile)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Check backup path is correct
	expectedPath := GetBackupPath(settingsFile)
	if backupPath != expectedPath {
		t.Errorf("Backup path mismatch: expected %s, got %s", expectedPath, backupPath)
	}

	// Now backup should exist
	if !HasBackup(settingsFile) {
		t.Error("HasBackup should return true when backup exists")
	}
}

func TestRestoreFromBackup(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	settingsFile := filepath.Join(tempDir, "settings.json")

	// Create original settings
	originalSettings := &Settings{
		OutputStyle: "explanatory",
		Model:       "claude-3-5-sonnet-20241022",
	}
	err := SaveToFile(originalSettings, settingsFile)
	if err != nil {
		t.Fatalf("Failed to create settings file: %v", err)
	}

	// Create backup
	backupPath, err := CreateBackup(settingsFile)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Modify the original file
	modifiedSettings := &Settings{
		OutputStyle: "minimal",
		Model:       "claude-3-opus-20240229",
	}
	err = SaveToFile(modifiedSettings, settingsFile)
	if err != nil {
		t.Fatalf("Failed to modify settings file: %v", err)
	}

	// Restore from backup
	err = RestoreFromBackup(backupPath, settingsFile)
	if err != nil {
		t.Fatalf("RestoreFromBackup failed: %v", err)
	}

	// Verify restoration
	restoredSettings, err := LoadFromFile(settingsFile)
	if err != nil {
		t.Fatalf("Failed to load restored settings: %v", err)
	}

	if restoredSettings.OutputStyle != originalSettings.OutputStyle {
		t.Errorf("Restored output style mismatch: expected %s, got %s",
			originalSettings.OutputStyle, restoredSettings.OutputStyle)
	}

	if restoredSettings.Model != originalSettings.Model {
		t.Errorf("Restored model mismatch: expected %s, got %s",
			originalSettings.Model, restoredSettings.Model)
	}
}

func TestRestoreFromBackup_Simple(t *testing.T) {
	t.Parallel()
	// Test that RestoreFromBackup correctly restores from backup
	tempDir := t.TempDir()
	settingsFile := filepath.Join(tempDir, "settings.json")

	// Create original settings
	originalSettings := &Settings{
		OutputStyle: "explanatory",
		Model:       "claude-3-5-sonnet-20241022",
	}
	err := SaveToFile(originalSettings, settingsFile)
	if err != nil {
		t.Fatalf("Failed to create settings file: %v", err)
	}

	// Create backup
	backupPath, err := CreateBackup(settingsFile)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Modify original file to create different content
	modifiedSettings := &Settings{
		OutputStyle: "minimal",
		Model:       "claude-3-opus-20240229",
	}
	err = SaveToFile(modifiedSettings, settingsFile)
	if err != nil {
		t.Fatalf("Failed to modify settings file: %v", err)
	}

	// Test atomic restore - should not leave temp files
	err = RestoreFromBackup(backupPath, settingsFile)
	if err != nil {
		t.Fatalf("RestoreFromBackup failed: %v", err)
	}

	// Test restore operation completed successfully

	// Verify content was restored correctly
	restoredSettings, err := LoadFromFile(settingsFile)
	if err != nil {
		t.Fatalf("Failed to load restored settings: %v", err)
	}

	if restoredSettings.OutputStyle != originalSettings.OutputStyle {
		t.Errorf("Restored output style mismatch: expected %s, got %s",
			originalSettings.OutputStyle, restoredSettings.OutputStyle)
	}
}

func TestLoadInitializesHooksField(t *testing.T) {
	t.Parallel()

	// Create JSON with hooks field that might not properly initialize pointers
	jsonContent := `{"hooks":{"PreToolUse":[]},"permissions":{"allow":["test"]}}`

	tempFile := filepath.Join(t.TempDir(), "test-hooks.json")
	err := os.WriteFile(tempFile, []byte(jsonContent), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// Load settings
	settings, err := LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("Expected no error loading settings, got: %v", err)
	}

	// Hooks field should be initialized and ready for modifications
	if settings.Hooks == nil {
		t.Error("Expected Hooks field to be initialized after loading")
	}

	// Should be able to add a hook without issues
	hookCmd := HookCommand{
		Type:    "command",
		Command: "test-command",
		Timeout: 30,
	}

	err = settings.AddHook(PreToolUseEvent, "TestMatcher", hookCmd)
	if err != nil {
		t.Fatalf("Expected no error adding hook, got: %v", err)
	}

	// Hook should be present
	if len(settings.Hooks.PreToolUse) != 1 {
		t.Errorf("Expected 1 hook in PreToolUse, got %d", len(settings.Hooks.PreToolUse))
	}

	// Now save and reload to ensure persistence
	tempSaveFile := filepath.Join(t.TempDir(), "test-save.json")
	err = SaveToFile(settings, tempSaveFile)
	if err != nil {
		t.Fatalf("Expected no error saving settings, got: %v", err)
	}

	// Reload and verify hook persisted
	reloadedSettings, err := LoadFromFile(tempSaveFile)
	if err != nil {
		t.Fatalf("Expected no error reloading settings, got: %v", err)
	}

	if len(reloadedSettings.Hooks.PreToolUse) != 1 {
		t.Errorf("Expected 1 hook after reload, got %d", len(reloadedSettings.Hooks.PreToolUse))
	}
}
