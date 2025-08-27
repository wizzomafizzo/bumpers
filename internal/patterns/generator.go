package patterns

import (
	"regexp"
	"strings"
)

// GeneratePattern converts a command string into a regex pattern
func GeneratePattern(command string) string {
	// Start with escaped literal
	pattern := regexp.QuoteMeta(command)

	// Replace spaces with flexible whitespace (spaces don't need escaping by QuoteMeta)
	pattern = strings.ReplaceAll(pattern, " ", "\\s+")

	// Add anchors for simple commands
	return "^" + pattern + "$"
}
