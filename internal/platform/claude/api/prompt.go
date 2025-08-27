package ai

// BuildDefaultPrompt creates a default prompt for AI message generation
func BuildDefaultPrompt(message string) string {
	return `You are reviewing tool usage in this project and providing
alternative guidance.

The original message below is a hook guard message intended to be sent in
response to matched rules. I want you to rephrase the message to be more
positive and educational while maintaining the same guidance.

If necessary, you can review items in this project to provide additional
context and give improved guidance, but you may only read and search, DO NOT
edit the project or call any long running tools. You are on a strict time limit
to perform this task of only 30 seconds.

When you respond, only include the rephrased message by itself, no additional
context about your own work to rephrase the message or things like "here's 
the message:" or similar.

ORIGINAL MESSAGE:
` + message
}

// BuildRegexGenerationPrompt creates a prompt for generating regex patterns from commands or descriptions
func BuildRegexGenerationPrompt(input string) string {
	return `You are a regex pattern generator for command matching in a security hook system.

Your task is to analyze the input and generate an optimal regex pattern:

1. If the input is a LITERAL COMMAND (e.g., "rm -rf /", "git push origin main"):
   - Create a precise regex that matches this specific command
   - Escape special regex characters appropriately
   - Allow for flexible whitespace between arguments
   - Add anchors (^ and $) to match the full line

2. If the input is a DESCRIPTION (e.g., "commands that delete files", "git operations that modify remote"):
   - Create a broader regex that matches the category of commands described
   - Consider common command variations and aliases
   - Focus on the core intent of the description

Requirements:
- Return ONLY the regex pattern, no explanations or additional text
- The pattern must be valid for Go's regexp package
- Use word boundaries and anchoring appropriately
- Consider common command variations (e.g., ls vs ll, rm vs del)

Examples:
Input: "rm -rf /"
Output: ^rm\s+-rf\s+/$

Input: "commands that delete files"
Output: ^(rm|rmdir|unlink|del)(\s+.*)?$

Input: "git push"
Output: ^git\s+push(\s+.*)?$

INPUT: ` + input
}
