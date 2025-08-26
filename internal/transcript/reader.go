package transcript

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
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

// TranscriptEntry represents a single entry in the Claude Code transcript
type TranscriptEntry struct {
	Type    string         `json:"type"`
	Message MessageContent `json:"message"`
}

// MessageContent contains the content for assistant messages
type MessageContent struct {
	Role    string        `json:"role"`
	Content []ContentItem `json:"content"`
}

// ContentItem represents individual content items in a message
type ContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Thinking string `json:"thinking,omitempty"`
}

func ExtractIntentContent(transcriptPath string) (string, error) {
	file, err := os.Open(transcriptPath) // #nosec G304 - path is validated by caller
	if err != nil {
		return "", fmt.Errorf("failed to open transcript file %s: %w", transcriptPath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Debug().Err(closeErr).Str("transcript_path", transcriptPath).Msg("Failed to close transcript file")
		}
	}()

	scanner := bufio.NewScanner(file)
	var intentParts []string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := extractIntentFromLine(line)
		intentParts = append(intentParts, parts...)
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading transcript file %s: %w", transcriptPath, err)
	}

	return strings.Join(intentParts, " "), nil
}

// extractIntentFromLine extracts intent content from a single transcript line
func extractIntentFromLine(line string) []string {
	var entry TranscriptEntry
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		// Skip lines that aren't valid JSON (like user messages or errors)
		return nil
	}

	// Only process assistant messages
	if entry.Type != "assistant" {
		return nil
	}

	var intentParts []string
	// Extract content from all assistant message content blocks
	for _, content := range entry.Message.Content {
		intentParts = append(intentParts, extractContentByType(content)...)
	}
	return intentParts
}

// extractContentByType extracts content based on the content item type
func extractContentByType(content ContentItem) []string {
	var parts []string
	switch content.Type {
	case "thinking":
		if strings.TrimSpace(content.Thinking) != "" {
			parts = append(parts, strings.TrimSpace(content.Thinking))
		}
	case "text":
		if strings.TrimSpace(content.Text) != "" {
			parts = append(parts, strings.TrimSpace(content.Text))
		}
	}
	return parts
}

// ExtractIntentContentOptimized reads transcript files efficiently by reading from the end backwards
// This is optimized for large files where recent content is most relevant
func ExtractIntentContentOptimized(transcriptPath string, maxLines int) (string, error) {
	if maxLines <= 0 {
		maxLines = 100 // Default reasonable limit for recent content
	}

	file, err := os.Open(transcriptPath) // #nosec G304 - path is validated by caller
	if err != nil {
		return "", fmt.Errorf("failed to open transcript file %s: %w", transcriptPath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Debug().Err(closeErr).Str("transcript_path", transcriptPath).Msg("Failed to close transcript file")
		}
	}()

	lines, err := readRecentLines(file, maxLines)
	if err != nil {
		return "", err
	}

	return extractIntentFromLines(lines), nil
}

// readRecentLines reads the most recent lines from a file efficiently
func readRecentLines(file *os.File, maxLines int) ([]string, error) {
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}

	lines := make([]string, 0, maxLines)
	var buf []byte
	const chunkSize = 8192

	offset := fileInfo.Size()
	for len(lines) < maxLines && offset > 0 {
		chunk, newOffset, err := readChunk(file, offset, chunkSize)
		if err != nil {
			return nil, err
		}
		offset = newOffset

		buf = prependToBuffer(chunk, buf)
		lines, buf = extractLinesFromBuffer(buf, lines, maxLines)
	}

	if offset == 0 && len(buf) > 0 {
		lines = addRemainingBuffer(buf, lines)
	}

	return reverseLines(lines), nil
}

// readChunk reads a chunk of data from the file at the specified offset
func readChunk(file *os.File, offset, chunkSize int64) (chunk []byte, newOffset int64, err error) {
	readSize := chunkSize
	if offset < chunkSize {
		readSize = offset
	}
	newOffset = offset - readSize

	chunk = make([]byte, readSize)
	_, err = file.ReadAt(chunk, newOffset)
	if err != nil && !errors.Is(err, io.EOF) {
		err = fmt.Errorf("error reading file at offset %d: %w", newOffset, err)
		return
	}

	return
}

// prependToBuffer prepends a chunk to the buffer
func prependToBuffer(chunk, buf []byte) []byte {
	newBuf := make([]byte, 0, len(chunk)+len(buf))
	newBuf = append(newBuf, chunk...)
	return append(newBuf, buf...)
}

// extractLinesFromBuffer extracts complete lines from the buffer
func extractLinesFromBuffer(buf []byte, lines []string, maxLines int) (updatedLines []string, remainingBuf []byte) {
	for len(lines) < maxLines {
		newlineIdx := findLastNewline(buf)
		if newlineIdx == -1 {
			break
		}

		line := string(buf[newlineIdx+1:])
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}

		buf = buf[:newlineIdx]
	}
	return lines, buf
}

// findLastNewline finds the last newline character in the buffer
func findLastNewline(buf []byte) int {
	for i := len(buf) - 1; i >= 0; i-- {
		if buf[i] == '\n' {
			return i
		}
	}
	return -1
}

// addRemainingBuffer adds any remaining buffer content as the first line
func addRemainingBuffer(buf []byte, lines []string) []string {
	line := string(buf)
	if strings.TrimSpace(line) != "" {
		lines = append(lines, line)
	}
	return lines
}

// reverseLines reverses the order of lines to get chronological order
func reverseLines(lines []string) []string {
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return lines
}

// extractIntentFromLines processes lines to extract intent content
func extractIntentFromLines(lines []string) string {
	var intentParts []string
	for _, line := range lines {
		parts := extractIntentFromLine(line)
		intentParts = append(intentParts, parts...)
	}
	return strings.Join(intentParts, " ")
}
