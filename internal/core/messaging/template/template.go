package template

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/wizzomafizzo/bumpers/internal/platform/filesystem"
)

func Execute(templateStr string, data any) (string, error) {
	if err := ValidateTemplate(templateStr); err != nil {
		return "", err
	}

	tmpl, err := template.New("message").Funcs(createFuncMap(filesystem.NewOSFileSystem(), nil)).Parse(templateStr)
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

// ExecuteWithCommandContext processes a template with command context for argc/argv functions
func ExecuteWithCommandContext(templateStr string, data any, commandCtx *CommandContext) (string, error) {
	if err := ValidateTemplate(templateStr); err != nil {
		return "", err
	}

	tmpl, err := template.New("message").
		Funcs(createFuncMap(filesystem.NewOSFileSystem(), commandCtx)).
		Parse(templateStr)
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

// ExecuteCommandTemplateWithArgs processes a command message template with arguments
func ExecuteCommandTemplateWithArgs(message, commandName, args string, argv []string) (string, error) {
	commandCtx := &CommandContext{
		Name: commandName,
		Args: args,
		Argv: argv,
	}
	context := MergeContexts(NewSharedContext(), *commandCtx)
	return ExecuteWithCommandContext(message, context, commandCtx)
}

// ExecuteNoteTemplate processes a note message template
func ExecuteNoteTemplate(message string) (string, error) {
	context := BuildNoteContext()
	return Execute(message, context)
}
