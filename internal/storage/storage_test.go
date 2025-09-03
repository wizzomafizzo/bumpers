package storage

import (
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/spf13/afero"
	"github.com/wizzomafizzo/bumpers/internal/constants"
)

func TestStorageManagerPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		methodCall   func(*Manager) (string, error)
		expectedPath func() string
		name         string
	}{
		{
			name: "GetDataDir returns correct path",
			methodCall: func(m *Manager) (string, error) {
				return m.GetDataDir()
			},
			expectedPath: func() string {
				return filepath.Join(xdg.DataHome, AppName)
			},
		},
		{
			name: "GetLogPath returns correct path",
			methodCall: func(m *Manager) (string, error) {
				return m.GetLogPath()
			},
			expectedPath: func() string {
				return filepath.Join(xdg.DataHome, AppName, constants.LogFilename)
			},
		},
		{
			name: "GetDatabasePath returns correct path",
			methodCall: func(m *Manager) (string, error) {
				return m.GetDatabasePath()
			},
			expectedPath: func() string {
				return filepath.Join(xdg.DataHome, AppName, constants.DatabaseFilename)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			manager := New(afero.NewMemMapFs())

			actualPath, err := tt.methodCall(manager)
			if err != nil {
				t.Fatalf("method call failed: %v", err)
			}

			expectedPath := tt.expectedPath()
			if actualPath != expectedPath {
				t.Errorf("got %s, want %s", actualPath, expectedPath)
			}
		})
	}
}
