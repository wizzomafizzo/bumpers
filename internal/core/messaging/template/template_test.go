package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wizzomafizzo/bumpers/internal/infrastructure/project"
)

func TestExecute_SimpleTemplate(t *testing.T) {
	t.Parallel()
	result, err := Execute("Hello {{.Name}}", map[string]string{"Name": "World"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := "Hello World"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestExecute_NoTemplate(t *testing.T) {
	t.Parallel()
	result, err := Execute("Hello World", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := "Hello World"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestExecute_TemplateTooLarge(t *testing.T) {
	t.Parallel()
	largeTemplate := strings.Repeat("{{.Name}}", 5000) // ~35KB
	_, err := Execute(largeTemplate, map[string]string{"Name": "Test"})
	if err == nil {
		t.Fatal("Expected error for large template, got nil")
	}
}

func TestExecute_InvalidTemplate(t *testing.T) {
	t.Parallel()
	_, err := Execute("{{.InvalidSyntax", nil)
	if err == nil {
		t.Fatal("Expected error for invalid template syntax, got nil")
	}

	// Check that error message mentions parsing
	if !strings.Contains(err.Error(), "failed to parse template") {
		t.Errorf("Expected error to mention template parsing, got: %v", err)
	}
}

func TestExecute_ExecutionError(t *testing.T) {
	t.Parallel()
	_, err := Execute("{{.MissingField}}", struct{}{})
	if err == nil {
		t.Fatal("Expected error for missing field, got nil")
	}

	// Check that error message mentions execution
	if !strings.Contains(err.Error(), "failed to execute template") {
		t.Errorf("Expected error to mention template execution, got: %v", err)
	}
}

func TestExecute_WithTodayVariable(t *testing.T) {
	t.Parallel()

	context := BuildNoteContext()
	result, err := Execute("Today is {{.Today}}", context)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedDate := time.Now().Format("2006-01-02")
	expected := "Today is " + expectedDate
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestExecute_WithReadFileFunction(t *testing.T) {
	t.Parallel()
	// Get project root and create test file there
	projectRoot, err := project.FindRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	// Create a test file in the project root
	testContent := "File content from readFile"
	testFile := filepath.Join(projectRoot, "readfile-test.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(testFile)
	})

	// Template that uses the readFile function
	templateStr := "Content: {{readFile \"readfile-test.txt\"}}"
	data := map[string]any{}

	result, err := Execute(templateStr, data)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	expected := "Content: " + testContent
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestExecute_WithTestPathFunction_NonExistentFile(t *testing.T) {
	t.Parallel()

	// Template that uses the testPath function to check a nonexistent file
	templateStr := "File exists: {{testPath \"nonexistent.txt\"}}"
	data := map[string]any{}

	result, err := Execute(templateStr, data)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	expected := "File exists: false"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestExecute_WithTestPathFunction_ExistingFile(t *testing.T) {
	t.Parallel()
	// Get project root and create test file there
	projectRoot, err := project.FindRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	// Create a test file in the project root
	testContent := "Test file for testPath"
	testFile := filepath.Join(projectRoot, "testpath-test.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(testFile)
	})

	// Template that uses the testPath function to check an existing file
	templateStr := "File exists: {{testPath \"testpath-test.txt\"}}"
	data := map[string]any{}

	result, err := Execute(templateStr, data)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	expected := "File exists: true"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestExecute_WithTestPathFunction_InConditional(t *testing.T) {
	t.Parallel()
	// Get project root and create test file there
	projectRoot, err := project.FindRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	// Create a test file in the project root
	testFile := filepath.Join(projectRoot, "config-test.yml")
	err = os.WriteFile(testFile, []byte("Config content"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(testFile)
	})

	// Template that uses testPath in a conditional statement - different from ExistingFile test
	templateStr := "{{if testPath \"config-test.yml\"}}Config found{{else}}No config{{end}}"

	result, err := Execute(templateStr, nil)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	expected := "Config found"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// Benchmark tests for template rendering performance
func BenchmarkExecuteSimple(b *testing.B) {
	template := "Hello {{.Name}}"
	data := map[string]string{"Name": "World"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Execute(template, data)
	}
}

func BenchmarkExecuteComplex(b *testing.B) {
	template := `
	{{if .HasConfig}}Config: {{.ConfigName}}{{else}}No config{{end}}
	Today: {{.Today}}
	{{range .Items}}
	- Item: {{.Name}} ({{.Type}})
	{{end}}
	Total: {{len .Items}} items
	`

	context := BuildNoteContext()
	context["HasConfig"] = true
	context["ConfigName"] = "test.yml"
	context["Items"] = []map[string]string{
		{"Name": "test1", "Type": "unit"},
		{"Name": "test2", "Type": "integration"},
		{"Name": "test3", "Type": "e2e"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Execute(template, context)
	}
}

func TestExecuteWithCommandContext_WithArgcFunction(t *testing.T) {
	t.Parallel()

	templateStr := "Argument count: {{argc}}"
	ctx := &CommandContext{
		Name: "test",
		Args: "arg1 arg2",
		Argv: []string{"test", "arg1", "arg2"},
	}
	data := MergeContexts(NewSharedContext(), *ctx)

	result, err := ExecuteWithCommandContext(templateStr, data, ctx)
	if err != nil {
		t.Fatalf("ExecuteWithCommandContext() failed: %v", err)
	}

	expected := "Argument count: 2"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestExecuteWithCommandContext_WithArgvFunction(t *testing.T) {
	t.Parallel()

	templateStr := "Command: {{argv 0}}, First arg: {{argv 1}}"
	ctx := &CommandContext{
		Name: "test",
		Args: "arg1 arg2",
		Argv: []string{"test", "arg1", "arg2"},
	}
	data := MergeContexts(NewSharedContext(), *ctx)

	result, err := ExecuteWithCommandContext(templateStr, data, ctx)
	if err != nil {
		t.Fatalf("ExecuteWithCommandContext() failed: %v", err)
	}

	expected := "Command: test, First arg: arg1"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}
