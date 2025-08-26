package settings

import (
	"encoding/json"
	"fmt"

	"github.com/wizzomafizzo/bumpers/internal/platform/filesystem"
)

// LoadFromFile loads Claude settings from a JSON file.
func LoadFromFile(filename string) (*Settings, error) {
	return LoadFromFileWithFS(filesystem.NewOSFileSystem(), filename)
}

// LoadFromFileWithFS loads Claude settings from a JSON file using the provided filesystem.
func LoadFromFileWithFS(fs filesystem.FileSystem, filename string) (*Settings, error) {
	data, err := fs.ReadFile(filename)
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
	return SaveToFileWithFS(filesystem.NewOSFileSystem(), settings, filename)
}

// SaveToFileWithFS saves Claude settings to a JSON file using the provided filesystem.
func SaveToFileWithFS(fs filesystem.FileSystem, settings *Settings, filename string) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings to JSON: %w", err)
	}

	err = fs.WriteFile(filename, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write settings to file %s: %w", filename, err)
	}
	return nil
}

// CreateBackup creates a simple .bak backup of the settings file.
func CreateBackup(filename string) (string, error) {
	return CreateBackupWithFS(filesystem.NewOSFileSystem(), filename)
}

// CreateBackupWithFS creates a simple .bak backup of the settings file using the provided filesystem.
func CreateBackupWithFS(fs filesystem.FileSystem, filename string) (string, error) {
	backupPath := filename + ".bak"

	// Read original file
	data, err := fs.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read original file: %w", err)
	}

	// Write backup file
	err = fs.WriteFile(backupPath, data, 0o600)
	if err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return backupPath, nil
}

// HasBackup checks if a .bak backup exists for the given settings file.
func HasBackup(filename string) bool {
	return HasBackupWithFS(filesystem.NewOSFileSystem(), filename)
}

// HasBackupWithFS checks if a .bak backup exists for the given settings file using the provided filesystem.
func HasBackupWithFS(fs filesystem.FileSystem, filename string) bool {
	backupPath := filename + ".bak"
	_, err := fs.Stat(backupPath)
	return err == nil
}

// GetBackupPath returns the backup file path for the given settings file.
func GetBackupPath(filename string) string {
	return filename + ".bak"
}

// RestoreFromBackup restores settings from a backup file.
func RestoreFromBackup(backupPath, targetPath string) error {
	return RestoreFromBackupWithFS(filesystem.NewOSFileSystem(), backupPath, targetPath)
}

// RestoreFromBackupWithFS restores settings from a backup file using the provided filesystem.
func RestoreFromBackupWithFS(fs filesystem.FileSystem, backupPath, targetPath string) error {
	// Read backup file
	data, err := fs.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Write to target file
	err = fs.WriteFile(targetPath, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write target file: %w", err)
	}

	return nil
}
