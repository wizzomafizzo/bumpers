// Package storage provides XDG-compliant storage path management for bumpers.
package storage

import (
	"fmt"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/wizzomafizzo/bumpers/internal/constants"
	"github.com/wizzomafizzo/bumpers/internal/filesystem"
)

const (
	// AppName is the application name used for XDG directory paths
	AppName = "bumpers"
)

// Manager handles storage operations with filesystem abstraction
type Manager struct {
	fs filesystem.FileSystem
}

// New creates a new storage manager with the given filesystem
func New(fs filesystem.FileSystem) *Manager {
	return &Manager{fs: fs}
}

// GetDataDir returns the XDG data directory for bumpers, creating it if necessary
func (m *Manager) GetDataDir() (string, error) {
	dataDir := filepath.Join(xdg.DataHome, AppName)
	err := m.fs.MkdirAll(dataDir, 0o750)
	if err != nil {
		return "", fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
	}
	return dataDir, nil
}

// GetLogPath returns the full path to the bumpers log file
func (m *Manager) GetLogPath() (string, error) {
	dataDir, err := m.GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, constants.LogFilename), nil
}

// GetCachePath returns the full path to the bumpers cache database
func (m *Manager) GetCachePath() (string, error) {
	dataDir, err := m.GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, constants.CacheFilename), nil
}
