package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LoadFromFile loads Claude settings from a JSON file.
func LoadFromFile(filename string) (*Settings, error) {
	data, err := os.ReadFile(filename) // #nosec G304
	if err != nil {
		return nil, err //nolint:wrapcheck // stdlib error
	}

	var settings Settings
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return nil, err //nolint:wrapcheck // stdlib error
	}

	return &settings, nil
}

// SaveToFile saves Claude settings to a JSON file.
func SaveToFile(settings *Settings, filename string) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err //nolint:wrapcheck // stdlib error
	}

	return os.WriteFile(filename, data, 0o600) //nolint:wrapcheck // stdlib error
}

// SaveToFileAtomically saves Claude settings to a JSON file using atomic operations.
func SaveToFileAtomically(settings *Settings, filename string) error {
	return SaveToFileAtomicallyWithTempFile(settings, filename)
}

// SaveToFileAtomicallyWithTempFile saves using actual atomic file operations.
func SaveToFileAtomicallyWithTempFile(settings *Settings, filename string) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err //nolint:wrapcheck // stdlib error
	}

	// Create temporary file in the same directory
	tmpFile := filename + ".tmp"

	// Write to temporary file first
	err = os.WriteFile(tmpFile, data, 0o600)
	if err != nil {
		return err //nolint:wrapcheck // stdlib error
	}

	// Atomically rename temporary file to target file
	return os.Rename(tmpFile, filename) //nolint:wrapcheck // stdlib error
}

// CreateBackup creates a timestamped backup of the settings file.
func CreateBackup(filename string) (string, error) {
	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	ext := filepath.Ext(filename)
	nameWithoutExt := base[:len(base)-len(ext)]

	backupFileName := fmt.Sprintf("%s_backup_%s%s", nameWithoutExt, timestamp, ext)
	backupPath := filepath.Join(dir, backupFileName)

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

// ListBackups returns a list of backup files for the given settings file.
func ListBackups(filename string) ([]string, error) {
	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	ext := filepath.Ext(filename)
	nameWithoutExt := base[:len(base)-len(ext)]

	// Pattern for backup files: <name>_backup_<timestamp><ext>
	backupPrefix := nameWithoutExt + "_backup_"

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, backupPrefix) && strings.HasSuffix(name, ext) {
			backups = append(backups, filepath.Join(dir, name))
		}
	}

	// Sort backups by filename (which includes timestamp)
	sort.Strings(backups)

	return backups, nil
}

// RestoreFromBackup restores settings from a backup file using atomic operations.
func RestoreFromBackup(backupPath, targetPath string) error {
	// Read backup file
	data, err := os.ReadFile(backupPath) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Write to target file atomically using temporary file
	tmpFile := targetPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Atomically rename temporary file to target file
	if err := os.Rename(tmpFile, targetPath); err != nil {
		// Clean up temporary file on failure
		_ = os.Remove(tmpFile) // Ignore cleanup errors as we're already in error state
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}
