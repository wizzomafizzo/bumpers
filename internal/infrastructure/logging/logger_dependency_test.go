package logger

import (
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/core/messaging/context"
	"github.com/wizzomafizzo/bumpers/internal/platform/filesystem"
)

// TestInitWithProjectContextRequiresDependencyInjection demonstrates that we need
// to inject the filesystem dependency to allow proper testing with different filesystems
// TestInitWithProjectContextRequiresDependencyInjection verifies dependency injection requirement
//
//nolint:paralleltest // modifies global logger state
func TestInitWithProjectContextRequiresDependencyInjection(t *testing.T) {
	// Initialize test logger to prevent race conditions
	InitTest()

	projectCtx := &context.ProjectContext{
		ID:   "test-project",
		Name: "Test Project",
	}

	// Now we can test with MemoryFileSystem using dependency injection
	memoryFS := filesystem.NewMemoryFileSystem()

	// Test with injected MemoryFileSystem
	err := InitWithProjectContextAndFS(projectCtx, memoryFS)
	if err != nil {
		t.Fatalf("InitWithProjectContextAndFS should succeed with MemoryFileSystem, got: %v", err)
	}

	// Test backward compatibility - original function should still work
	err = InitWithProjectContext(projectCtx)
	if err != nil {
		t.Fatalf("InitWithProjectContext should succeed with OSFileSystem, got: %v", err)
	}
}
