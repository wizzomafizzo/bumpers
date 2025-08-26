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
