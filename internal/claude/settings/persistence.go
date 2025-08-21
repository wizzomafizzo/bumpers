package settings

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadFromFile loads Claude settings from a JSON file.
func LoadFromFile(filename string) (*Settings, error) {
	data, err := os.ReadFile(filename) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file %s: %w", filename, err)
	}

	var settings Settings
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse settings JSON from %s: %w", filename, err)
	}

	return &settings, nil
}

// SaveToFile saves Claude settings to a JSON file.
func SaveToFile(settings *Settings, filename string) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings to JSON: %w", err)
	}

	err = os.WriteFile(filename, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write settings to file %s: %w", filename, err)
	}
	return nil
}

// CreateBackup creates a simple .bak backup of the settings file.
func CreateBackup(filename string) (string, error) {
	backupPath := filename + ".bak"

	// Read original file
	data, err := os.ReadFile(filename) // #nosec G304
	if err != nil {
		return "", fmt.Errorf("failed to read original file: %w", err)
	}

	// Write backup file
	err = os.WriteFile(backupPath, data, 0o600)
	if err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return backupPath, nil
}

// HasBackup checks if a .bak backup exists for the given settings file.
func HasBackup(filename string) bool {
	backupPath := filename + ".bak"
	_, err := os.Stat(backupPath)
	return err == nil
}

// GetBackupPath returns the backup file path for the given settings file.
func GetBackupPath(filename string) string {
	return filename + ".bak"
}

// RestoreFromBackup restores settings from a backup file.
func RestoreFromBackup(backupPath, targetPath string) error {
	// Read backup file
	data, err := os.ReadFile(backupPath) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Write to target file
	err = os.WriteFile(targetPath, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write target file: %w", err)
	}

	return nil
}
