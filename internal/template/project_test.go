package template

import (
	"testing"
)

func TestGenerateProjectID(t *testing.T) {
	t.Parallel()
	// Test basic project ID generation
	projectName := "my-app"
	projectPath := "/home/user/dev/my-app"

	id := GenerateProjectID(projectName, projectPath)

	// Should start with sanitized project name
	if id == "" {
		t.Error("GenerateProjectID() returned empty string")
	}

	// Should be in format: name-hash (at least 6 characters for "x-1234")
	if len(id) < 6 {
		t.Errorf("GenerateProjectID() = %s, expected format like 'name-hash'", id)
	}

	// Should contain a dash separator
	expectedPrefix := "myapp-"
	if id[:6] != expectedPrefix {
		t.Errorf("GenerateProjectID() = %s, expected to start with %s", id, expectedPrefix)
	}
}

func TestGenerateProjectIDDeterministic(t *testing.T) {
	t.Parallel()
	// Test that the same inputs always produce the same ID
	projectName := "test-project"
	projectPath := "/some/path/test-project"

	id1 := GenerateProjectID(projectName, projectPath)
	id2 := GenerateProjectID(projectName, projectPath)

	if id1 != id2 {
		t.Errorf("GenerateProjectID() not deterministic: %s != %s", id1, id2)
	}
}

func TestNewProjectContext(t *testing.T) {
	t.Parallel()
	projectPath := "/home/user/projects/my-app"

	ctx := New(projectPath)

	if ctx.Path != projectPath {
		t.Errorf("New() path = %s, want %s", ctx.Path, projectPath)
	}

	if ctx.Name != "my-app" {
		t.Errorf("New() name = %s, want %s", ctx.Name, "my-app")
	}

	expectedID := GenerateProjectID("my-app", projectPath)
	if ctx.ID != expectedID {
		t.Errorf("New() ID = %s, want %s", ctx.ID, expectedID)
	}
}
