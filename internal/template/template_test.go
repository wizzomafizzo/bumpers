package template

import (
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
