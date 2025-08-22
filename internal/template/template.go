package template

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/wizzomafizzo/bumpers/internal/filesystem"
)

func Execute(templateStr string, data any) (string, error) {
	if err := ValidateTemplate(templateStr); err != nil {
		return "", err
	}

	tmpl, err := template.New("message").Funcs(createFuncMap(filesystem.NewOSFileSystem())).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var result strings.Builder
	err = tmpl.Execute(&result, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return result.String(), nil
}

// ExecuteRuleTemplate processes a rule message template with the given command
func ExecuteRuleTemplate(message, command string) (string, error) {
	context := BuildRuleContext(command)
	return Execute(message, context)
}

// ExecuteCommandTemplate processes a command message template with the given command name
func ExecuteCommandTemplate(message, commandName string) (string, error) {
	context := BuildCommandContext(commandName)
	return Execute(message, context)
}

// ExecuteNoteTemplate processes a note message template
func ExecuteNoteTemplate(message string) (string, error) {
	context := BuildNoteContext()
	return Execute(message, context)
}
