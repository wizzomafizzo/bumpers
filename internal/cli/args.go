package cli

import (
	"strings"
	"unicode"
)

// ParseCommandArgs parses a command string into command name and arguments.
func ParseCommandArgs(input string) (commandName, args string, argv []string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", []string{}
	}

	// Find the first space to separate command name from arguments
	firstSpace := strings.IndexFunc(input, unicode.IsSpace)
	if firstSpace == -1 {
		// No arguments, just command name
		return input, "", []string{input}
	}

	commandName = input[:firstSpace]
	args = strings.TrimSpace(input[firstSpace+1:])

	// Parse arguments into array
	argv = []string{commandName}
	if args != "" {
		parsedArgs := parseArguments(args)
		argv = append(argv, parsedArgs...)
	}

	return commandName, args, argv
}

// parseArguments parses space-separated arguments respecting quoted strings.
func parseArguments(input string) []string {
	if input == "" {
		return []string{}
	}

	var args []string
	var current strings.Builder
	state := &parseState{}

	runes := []rune(input)
	for i := 0; i < len(runes); i++ {
		char := runes[i]

		if state.handleQuoteToggle(char) {
			continue
		}

		if state.handleSpaceSeparator(char, &current, &args, runes, &i) {
			continue
		}

		// Regular character or quoted content
		_, _ = current.WriteRune(char)
	}

	// Add final argument if any
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

type parseState struct {
	inQuotes  bool
	quoteChar rune
}

func (s *parseState) handleQuoteToggle(char rune) bool {
	if !s.inQuotes && (char == '"' || char == '\'') {
		s.inQuotes = true
		s.quoteChar = char
		return true
	}
	if s.inQuotes && char == s.quoteChar {
		s.inQuotes = false
		s.quoteChar = 0
		return true
	}
	return false
}

func (s *parseState) handleSpaceSeparator(
	char rune, current *strings.Builder, args *[]string, runes []rune, i *int,
) bool {
	if !s.inQuotes && unicode.IsSpace(char) {
		if current.Len() > 0 {
			*args = append(*args, current.String())
			current.Reset()
		}
		// Skip additional whitespace
		for *i+1 < len(runes) && unicode.IsSpace(runes[*i+1]) {
			*i++
		}
		return true
	}
	return false
}
