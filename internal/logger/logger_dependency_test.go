package logger

import (
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/context"
	"github.com/wizzomafizzo/bumpers/internal/filesystem"
)

// TestInitWithProjectContextRequiresDependencyInjection demonstrates that we need
// to inject the filesystem dependency to allow proper testing with different filesystems
func TestInitWithProjectContextRequiresDependencyInjection(t *testing.T) {
	t.Parallel()

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
