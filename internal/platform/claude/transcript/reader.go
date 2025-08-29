package transcript

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/wizzomafizzo/bumpers/internal/core/logging"
)

func ExtractReasoningContent(transcriptPath string) (string, error) {
	file, err := os.Open(transcriptPath) // #nosec G304 - path is validated by caller
	if err != nil {
		return "", nil
	}
	defer func() {
		_ = file.Close()
	}()

	reader := bufio.NewReader(file)
	result := make([]string, 0, 10) // Pre-allocate with reasonable capacity

	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return "", fmt.Errorf("failed to read transcript file %s: %w", transcriptPath, readErr)
		}
		if line == "" && readErr == io.EOF {
			break
		}

		if text := processReasoningLine(line); text != "" {
			result = append(result, text)
		}

		if readErr == io.EOF {
			break
		}
	}

	return strings.Join(result, " "), nil
}

// processReasoningLine extracts reasoning content from a single line
func processReasoningLine(line string) string {
	line = strings.TrimSuffix(line, "\n")

	// Check if it's an assistant message and extract text content
	if !strings.Contains(line, `"type":"assistant"`) {
		return ""
	}

	return extractTextFromLine(line)
}

// extractTextFromLine extracts text content from JSONL line
func extractTextFromLine(line string) string {
	start := strings.Index(line, `"text":"`)
	if start < 0 {
		return ""
	}
	start += 8 // Skip `"text":"`
	end := strings.Index(line[start:], `"`)
	if end < 0 {
		return ""
	}
	return line[start : start+end]
}

// TranscriptEntry represents a single entry in the Claude Code transcript
type TranscriptEntry struct {
	Type       string         `json:"type"`
	UUID       string         `json:"uuid"`
	ParentUUID string         `json:"parentUuid"`
	Message    MessageContent `json:"message"`
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
	Thinking string `json:"thinking,omitempty"` // For thinking content
	ID       string `json:"id,omitempty"`       // For tool_use content
}

func ExtractIntentContent(ctx context.Context, transcriptPath string) (string, error) {
	file, err := os.Open(transcriptPath) // #nosec G304 - path is validated by caller
	if err != nil {
		return "", fmt.Errorf("failed to open transcript file %s: %w", transcriptPath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logging.Get(ctx).Debug().Err(closeErr).
				Str("transcript_path", transcriptPath).
				Msg("Failed to close transcript file")
		}
	}()

	intentParts, err := readIntentFromFile(file, transcriptPath)
	if err != nil {
		return "", err
	}

	result := strings.Join(intentParts, " ")
	logExtractedIntent(ctx, transcriptPath, intentParts, result)

	return result, nil
}

// readIntentFromFile reads and processes all lines from the file
func readIntentFromFile(file *os.File, transcriptPath string) ([]string, error) {
	reader := bufio.NewReader(file)
	var intentParts []string

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read transcript file %s: %w", transcriptPath, err)
		}

		if line == "" && err == io.EOF {
			break
		}

		if processedLine := processIntentLine(line); len(processedLine) > 0 {
			intentParts = append(intentParts, processedLine...)
		}

		if err == io.EOF {
			break
		}
	}

	return intentParts, nil
}

// processIntentLine processes a single line for intent content
func processIntentLine(line string) []string {
	line = strings.TrimSuffix(line, "\n")
	if strings.TrimSpace(line) == "" {
		return nil
	}
	return extractIntentFromLine(line)
}

// logExtractedIntent logs the extraction results for debugging
func logExtractedIntent(ctx context.Context, transcriptPath string, intentParts []string, result string) {
	logging.Get(ctx).Debug().
		Str("transcript_path", transcriptPath).
		Int("intent_parts_count", len(intentParts)).
		Str("extracted_intent", result).
		Msg("ExtractIntentContent extracted content from transcript")
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
		intentParts = append(intentParts, extractContentFromContentItem(content)...)
	}
	return intentParts
}

// extractContentFromContentItem extracts text content from a single content item
func extractContentFromContentItem(content ContentItem) []string {
	var parts []string
	if content.Type == "text" && strings.TrimSpace(content.Text) != "" {
		parts = append(parts, strings.TrimSpace(content.Text))
	}
	if content.Type == "thinking" && strings.TrimSpace(content.Thinking) != "" {
		parts = append(parts, strings.TrimSpace(content.Thinking))
	}
	return parts
}

// ExtractIntentContentOptimized reads transcript files efficiently by reading from the end backwards
// This is optimized for large files where recent content is most relevant
func ExtractIntentContentOptimized(ctx context.Context, transcriptPath string, maxLines int) (string, error) {
	if maxLines <= 0 {
		maxLines = 100 // Default reasonable limit for recent content
	}

	file, err := os.Open(transcriptPath) // #nosec G304 - path is validated by caller
	if err != nil {
		return "", fmt.Errorf("failed to open transcript file %s: %w", transcriptPath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logging.Get(ctx).Debug().Err(closeErr).
				Str("transcript_path", transcriptPath).
				Msg("Failed to close transcript file")
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
		err = fmt.Errorf("failed to read file at offset %d: %w", newOffset, err)
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

// ExtractIntentByToolUseID extracts the intent for a specific tool use ID
// by finding the assistant message that precedes the tool_use message
func ExtractIntentByToolUseID(ctx context.Context, transcriptPath, toolUseID string) (string, error) {
	return ExtractIntentByToolUseIDWithContext(ctx, transcriptPath, toolUseID)
}

func ExtractIntentByToolUseIDWithContext(ctx context.Context, transcriptPath, toolUseID string) (string, error) {
	file, err := os.Open(transcriptPath) // #nosec G304 - path is validated by caller
	if err != nil {
		return "", fmt.Errorf("failed to open transcript file %s: %w", transcriptPath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logging.Get(ctx).Debug().Err(closeErr).
				Str("transcript_path", transcriptPath).
				Msg("Failed to close transcript file")
		}
	}()

	result, err := findIntentByToolUseID(file, toolUseID)
	if err == nil {
		logging.Get(ctx).Debug().
			Str("transcript_path", transcriptPath).
			Str("tool_use_id", toolUseID).
			Str("extracted_intent", result).
			Msg("ExtractIntentByToolUseID extracted content from transcript")
	}
	return result, err
}

// findIntentByToolUseID searches for intent message associated with tool use ID
func findIntentByToolUseID(file *os.File, toolUseID string) (string, error) {
	reader := bufio.NewReader(file)
	processedEntries := make([]TranscriptEntry, 0, 100)

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("failed to read transcript: %w", err)
		}
		if line == "" && err == io.EOF {
			break
		}

		entry, valid := parseTranscriptEntry(line)
		if !valid {
			continue
		}

		processedEntries = append(processedEntries, entry)

		if result := checkForToolUseMatch(&entry, toolUseID, processedEntries); result != "" {
			return result, nil
		}

		if err == io.EOF {
			break
		}
	}

	return "", nil // No intent found for this tool use ID
}

// parseTranscriptEntry parses a transcript line into an entry
func parseTranscriptEntry(line string) (TranscriptEntry, bool) {
	var entry TranscriptEntry
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &entry); err != nil {
		return entry, false
	}
	return entry, true
}

// checkForToolUseMatch checks if entry contains matching tool use and returns intent
func checkForToolUseMatch(entry *TranscriptEntry, toolUseID string, processedEntries []TranscriptEntry) string {
	if entry.Type != "assistant" {
		return ""
	}

	if content := findToolUseWithID(entry, toolUseID); content != nil {
		return findParentIntent(entry, processedEntries)
	}

	return ""
}

// findParentIntent finds the parent intent message for the given entry
func findParentIntent(entry *TranscriptEntry, processedEntries []TranscriptEntry) string {
	for _, processed := range processedEntries {
		if processed.Type == "assistant" && processed.UUID == entry.ParentUUID {
			return extractTextFromEntry(&processed)
		}
	}
	return ""
}

// findToolUseWithID searches for tool_use content with matching ID
func findToolUseWithID(entry *TranscriptEntry, toolUseID string) *ContentItem {
	for i := range entry.Message.Content {
		content := &entry.Message.Content[i]
		if content.Type == "tool_use" && content.ID == toolUseID {
			return content
		}
	}
	return nil
}

// extractTextFromEntry extracts text content from a transcript entry
func extractTextFromEntry(entry *TranscriptEntry) string {
	parts := extractTextPartsFromEntry(entry)
	return strings.Join(parts, " ")
}

// extractTextPartsFromEntry extracts text content parts from a transcript entry
func extractTextPartsFromEntry(entry *TranscriptEntry) []string {
	var parts []string
	for _, content := range entry.Message.Content {
		if content.Type == "text" && strings.TrimSpace(content.Text) != "" {
			parts = append(parts, strings.TrimSpace(content.Text))
		}
	}
	return parts
}

// findMostRecentToolUseParentUUID scans backwards to find the most recent tool_use and returns its parentUuid
func findMostRecentToolUseParentUUID(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		if parentUUID := extractParentUUIDFromLine(lines[i]); parentUUID != "" {
			return parentUUID
		}
	}
	return ""
}

// extractParentUUIDFromLine extracts parent UUID from a single line if it contains tool use
func extractParentUUIDFromLine(line string) string {
	var entry map[string]any
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		return ""
	}

	if !isAssistantEntry(entry) {
		return ""
	}

	return findToolUseParentUUID(entry)
}

// isAssistantEntry checks if the entry is an assistant type
func isAssistantEntry(entry map[string]any) bool {
	entryType, ok := entry["type"].(string)
	return ok && entryType == "assistant"
}

// findToolUseParentUUID finds the parent UUID for a tool use entry
func findToolUseParentUUID(entry map[string]any) string {
	message, ok := entry["message"].(map[string]any)
	if !ok {
		return ""
	}

	content, ok := message["content"].([]any)
	if !ok {
		return ""
	}

	if hasToolUseContent(content) {
		if parentUUID, ok := entry["parentUuid"].(string); ok && parentUUID != "" {
			return parentUUID
		}
	}
	return ""
}

// hasToolUseContent checks if content array contains a tool_use item
func hasToolUseContent(content []any) bool {
	for _, contentItem := range content {
		if item, ok := contentItem.(map[string]any); ok {
			if itemType, ok := item["type"].(string); ok && itemType == "tool_use" {
				return true
			}
		}
	}
	return false
}

// FindRecentToolUseAndExtractIntent scans backwards through transcript to find recent tool uses
// and extracts the associated intent content within a 1-minute time window
func FindRecentToolUseAndExtractIntent(ctx context.Context, transcriptPath string) (string, error) {
	lines, err := readTranscriptLines(ctx, transcriptPath)
	if err != nil {
		return "", err
	}

	// First pass: find the most recent tool_use message and get its parent UUID
	mostRecentToolUseParentUUID := findMostRecentToolUseParentUUID(lines)

	// Second pass: collect text content, prioritizing the tool_use parent message
	allContentParts := extractPrioritizedContent(lines, mostRecentToolUseParentUUID)

	if len(allContentParts) > 0 {
		result := strings.Join(allContentParts, " ")
		logging.Get(ctx).Debug().
			Str("transcript_path", transcriptPath).
			Int("content_parts_count", len(allContentParts)).
			Str("extracted_intent", result).
			Msg("FindRecentToolUseAndExtractIntent extracted content from transcript")
		return result, nil
	}

	return "", errors.New("no recent tool use intent found")
}

// readTranscriptLines reads all lines from the transcript file
func readTranscriptLines(ctx context.Context, transcriptPath string) ([]string, error) {
	file, err := os.Open(transcriptPath) // #nosec G304 - path is validated by caller
	if err != nil {
		return nil, fmt.Errorf("failed to open transcript file %s: %w", transcriptPath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logging.Get(ctx).Debug().Err(closeErr).
				Str("transcript_path", transcriptPath).
				Msg("Failed to close transcript file")
		}
	}()

	var lines []string
	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read line: %w", err)
		}

		if line != "" {
			lines = append(lines, strings.TrimSuffix(line, "\n"))
		}

		if err == io.EOF {
			break
		}
	}

	return lines, nil
}

// extractPrioritizedContent extracts content from lines, prioritizing tool use parent message
func extractPrioritizedContent(lines []string, mostRecentToolUseParentUUID string) []string {
	var allContentParts []string
	assistantCount := 0
	maxRecentAssistants := 2

	for i := len(lines) - 1; i >= 0 && assistantCount < maxRecentAssistants; i-- {
		contentParts := extractContentFromLine(lines[i])
		if len(contentParts) == 0 {
			continue
		}

		if shouldUseContent(lines[i], mostRecentToolUseParentUUID, &allContentParts, contentParts, &assistantCount) {
			break
		}
	}

	return allContentParts
}

// extractContentFromLine extracts text content from a single line
func extractContentFromLine(line string) []string {
	entry, valid := parseTranscriptEntry(line)
	if !valid || entry.Type != "assistant" {
		return nil
	}

	return extractTextPartsFromEntry(&entry)
}

// shouldUseContent determines if content should be used and updates collections
func shouldUseContent(line, parentUUID string, allParts *[]string, contentParts []string, assistantCount *int) bool {
	if parentUUID != "" {
		// Check if this line has the specific parent UUID we want
		entry, valid := parseTranscriptEntry(line)
		if !valid || entry.UUID != parentUUID {
			return false
		}

		*allParts = append(*allParts, contentParts...)
		return true // Found the parent intent message, we're done
	}

	// Fallback: collect recent assistant messages
	*allParts = append(contentParts, *allParts...)
	*assistantCount++
	return false // Continue collecting
}
