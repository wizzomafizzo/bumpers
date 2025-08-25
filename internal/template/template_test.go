package template

import (
	"os"
	"strings"
	"testing"
	"time"
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
	// Create a test file in the project root
	testContent := "File content from readFile"
	err := os.WriteFile("../../readfile-test.txt", []byte(testContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove("../../readfile-test.txt")
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
	// Create a test file in the project root
	testContent := "Test file for testPath"
	err := os.WriteFile("../../testpath-test.txt", []byte(testContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove("../../testpath-test.txt")
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
	// Create a test file in the project root
	err := os.WriteFile("../../config-test.yml", []byte("Config content"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove("../../config-test.yml")
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
