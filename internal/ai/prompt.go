package ai

// BuildDefaultPrompt creates a default prompt for AI message generation
func BuildDefaultPrompt(message string) string {
	const basePrompt = "Rephrase the following hook guard message to be more positive, " +
		"encouraging, and educational while maintaining the same guidance:\n\nMessage: "
	const suffix = "\n\nProvide only the rephrased message without explanation."
	return basePrompt + message + suffix
}
