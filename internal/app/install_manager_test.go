package app

import (
	"testing"

	"github.com/spf13/afero"
)

func TestNewInstallManager(t *testing.T) {
	t.Parallel()

	configPath := "/test/config.yml"
	workDir := "/test/work"
	projectRoot := "/test/project"
	fs := afero.NewMemMapFs()

	manager := NewInstallManager(configPath, workDir, projectRoot, fs)

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.configPath != configPath {
		t.Errorf("Expected configPath '%s', got '%s'", configPath, manager.configPath)
	}

	if manager.workDir != workDir {
		t.Errorf("Expected workDir '%s', got '%s'", workDir, manager.workDir)
	}

	if manager.projectRoot != projectRoot {
		t.Errorf("Expected projectRoot '%s', got '%s'", projectRoot, manager.projectRoot)
	}

	if manager.fileSystem != fs {
		t.Error("Expected filesystem to be set correctly")
	}
}

func TestGetFileSystem_WithInjectedFS(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	manager := NewInstallManager("/config", "/work", "/project", fs)

	result := manager.getFileSystem()

	if result != fs {
		t.Error("Expected getFileSystem to return injected filesystem")
	}
}

func TestGetFileSystem_DefaultFS(t *testing.T) {
	t.Parallel()

	manager := NewInstallManager("/config", "/work", "/project", nil)

	result := manager.getFileSystem()

	if result == nil {
		t.Error("Expected getFileSystem to return non-nil filesystem")
	}
}
