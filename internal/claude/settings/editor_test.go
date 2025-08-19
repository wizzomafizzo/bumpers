package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEditor_Load(t *testing.T) {
	t.Parallel()
	// Create a temporary settings file
	tempDir := t.TempDir()
	settingsFile := filepath.Join(tempDir, "settings.json")

	// Write test settings content
	content := `{
		"outputStyle": "Explanatory",
		"permissions": {
			"allow": ["WebSearch"]
		}
	}`

	err := os.WriteFile(settingsFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test settings file: %v", err)
	}

	// Test loading through editor
	editor := NewEditor()
	settings, err := editor.Load(settingsFile)
	if err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}

	if settings.OutputStyle != "Explanatory" {
		t.Errorf("Expected output style 'Explanatory', got %s", settings.OutputStyle)
	}
}

func TestEditor_Save(t *testing.T) {
	t.Parallel()
	// Create a test settings object
	settings := &Settings{
		OutputStyle: "Concise",
		Model:       "claude-3-5-sonnet-20241022",
	}

	// Create a temporary file path
	tempDir := t.TempDir()
	settingsFile := filepath.Join(tempDir, "settings.json")

	// Test saving through editor
	editor := NewEditor()
	err := editor.Save(settings, settingsFile)
	if err != nil {
		t.Fatalf("Failed to save settings: %v", err)
	}

	// Verify the file was written correctly by loading it back
	loadedSettings, err := editor.Load(settingsFile)
	if err != nil {
		t.Fatalf("Failed to load saved settings: %v", err)
	}

	if loadedSettings.OutputStyle != "Concise" {
		t.Errorf("Expected output style 'Concise', got %s", loadedSettings.OutputStyle)
	}

	if loadedSettings.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected model 'claude-3-5-sonnet-20241022', got %s", loadedSettings.Model)
	}
}

func TestEditor_SaveAtomic(t *testing.T) {
	t.Parallel()
	// Test that Editor.Save uses atomic operations like other atomic save functions
	settings := &Settings{
		OutputStyle: "explanatory",
		Model:       "claude-3-5-sonnet-20241022",
	}

	tempDir := t.TempDir()
	settingsFile := filepath.Join(tempDir, "settings.json")

	// Test saving through editor
	editor := NewEditor()
	err := editor.Save(settings, settingsFile)
	if err != nil {
		t.Fatalf("Editor.Save failed: %v", err)
	}

	// Verify the file exists and no temp file remains
	if _, statErr := os.Stat(settingsFile); os.IsNotExist(statErr) {
		t.Fatal("Settings file was not created")
	}

	// Verify atomic behavior: no temporary file should remain
	tmpFile := settingsFile + ".tmp"
	if _, tmpErr := os.Stat(tmpFile); !os.IsNotExist(tmpErr) {
		t.Error("Temporary file was not cleaned up - Save operation may not be atomic")
	}

	// Verify content correctness
	loadedSettings, err := editor.Load(settingsFile)
	if err != nil {
		t.Fatalf("Failed to load saved settings: %v", err)
	}

	if loadedSettings.OutputStyle != settings.OutputStyle {
		t.Errorf("OutputStyle mismatch. Expected: %s, Got: %s", settings.OutputStyle, loadedSettings.OutputStyle)
	}
}
