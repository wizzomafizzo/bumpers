package transcript

import (
	"bufio"
	"os"
	"strings"
)

func ExtractReasoningContent(transcriptPath string) (string, error) {
	file, err := os.Open(transcriptPath) // #nosec G304 - path is validated by caller
	if err != nil {
		return "", nil
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	result := make([]string, 0, 10) // Pre-allocate with reasonable capacity

	for scanner.Scan() {
		line := scanner.Text()
		// Only include lines from assistant messages
		hasAssistantType := strings.Contains(line, `"type":"assistant"`)
		hasReasoningPattern := strings.Contains(line, "I need to") ||
			strings.Contains(line, "I can see") ||
			strings.Contains(line, "not related") ||
			strings.Contains(line, "timed out") ||
			strings.Contains(line, "timeout")

		if !hasAssistantType || !hasReasoningPattern {
			continue
		}

		// Extract text content from the JSONL line
		start := strings.Index(line, `"text":"`)
		if start < 0 {
			continue
		}
		start += 8 // Skip `"text":"`
		end := strings.Index(line[start:], `"`)
		if end < 0 {
			continue
		}
		text := line[start : start+end]
		result = append(result, text)
	}

	return strings.Join(result, " "), nil
}
