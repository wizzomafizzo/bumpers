package storage

import (
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/wizzomafizzo/bumpers/internal/constants"
	"github.com/wizzomafizzo/bumpers/internal/filesystem"
)

func TestGetDataDir(t *testing.T) {
	t.Parallel()
	fs := filesystem.NewMemoryFileSystem()
	manager := New(fs)

	dataDir, err := manager.GetDataDir()
	if err != nil {
		t.Fatalf("GetDataDir() failed: %v", err)
	}

	expectedPath := filepath.Join(xdg.DataHome, AppName)
	if dataDir != expectedPath {
		t.Errorf("GetDataDir() = %s, want %s", dataDir, expectedPath)
	}
}

func TestGetLogPath(t *testing.T) {
	t.Parallel()
	fs := filesystem.NewMemoryFileSystem()
	manager := New(fs)

	logPath, err := manager.GetLogPath()
	if err != nil {
		t.Fatalf("GetLogPath() failed: %v", err)
	}

	expectedPath := filepath.Join(xdg.DataHome, AppName, constants.LogFilename)
	if logPath != expectedPath {
		t.Errorf("GetLogPath() = %s, want %s", logPath, expectedPath)
	}
}

func TestGetCachePath(t *testing.T) {
	t.Parallel()
	fs := filesystem.NewMemoryFileSystem()
	manager := New(fs)

	cachePath, err := manager.GetCachePath()
	if err != nil {
		t.Fatalf("GetCachePath() failed: %v", err)
	}

	expectedPath := filepath.Join(xdg.DataHome, AppName, constants.CacheFilename)
	if cachePath != expectedPath {
		t.Errorf("GetCachePath() = %s, want %s", cachePath, expectedPath)
	}
}
