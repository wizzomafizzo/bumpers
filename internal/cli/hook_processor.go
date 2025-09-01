package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/afero"
	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/core/engine/hooks"
	"github.com/wizzomafizzo/bumpers/internal/core/engine/matcher"
	"github.com/wizzomafizzo/bumpers/internal/core/engine/operation"
	"github.com/wizzomafizzo/bumpers/internal/core/logging"
	"github.com/wizzomafizzo/bumpers/internal/core/messaging/template"
	ai "github.com/wizzomafizzo/bumpers/internal/platform/claude/api"
	"github.com/wizzomafizzo/bumpers/internal/platform/claude/transcript"
	"github.com/wizzomafizzo/bumpers/internal/platform/state"
	"github.com/wizzomafizzo/bumpers/internal/platform/storage"
)

const intentFieldName = "#intent"

// HookProcessor handles all hook-related processing including pre/post tool use
type HookProcessor interface {
	ProcessHook(ctx context.Context, input io.Reader) (ProcessResult, error)
	ProcessPreToolUse(ctx context.Context, rawJSON json.RawMessage) (string, error)
	ProcessPostToolUse(ctx context.Context, rawJSON json.RawMessage) (string, error)
}

// DefaultHookProcessor implements HookProcessor
type DefaultHookProcessor struct {
	configValidator ConfigValidator
	aiGenerator     ai.MessageGenerator
	stateManager    *state.Manager
	projectRoot     string
}

// NewHookProcessor creates a new HookProcessor
func NewHookProcessor(
	configValidator ConfigValidator, projectRoot string, stateManager *state.Manager,
) *DefaultHookProcessor {
	return &DefaultHookProcessor{
		configValidator: configValidator,
		projectRoot:     projectRoot,
		stateManager:    stateManager,
	}
}

// SetMockAIGenerator sets a mock AI generator for testing
func (h *DefaultHookProcessor) SetMockAIGenerator(generator ai.MessageGenerator) {
	h.aiGenerator = generator
}

func (h *DefaultHookProcessor) ProcessHook(ctx context.Context, input io.Reader) (ProcessResult, error) {
	logger := logging.Get(ctx)

	if os.Getenv("BUMPERS_SKIP") == "1" {
		logger.Debug().Msg("BUMPERS_SKIP is set, skipping hook processing")
		return ProcessResult{Mode: ProcessModeAllow, Message: ""}, nil
	}

	logger.Debug().Msg("processing hook input")

	// Detect hook type and get raw JSON
	hookType, rawJSON, err := hooks.DetectHookType(input)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to detect hook type")
		return ProcessResult{}, fmt.Errorf("failed to detect hook type: %w", err)
	}
	logger.Debug().RawJSON("hook", rawJSON).Str("type", hookType.String()).Msg("received hook")

	// Route to appropriate handler based on hook type and convert response to ProcessResult
	var response string
	if hookType == hooks.PostToolUseHook {
		logger.Debug().Msg("processing PostToolUse hook")
		response, err = h.ProcessPostToolUse(ctx, rawJSON)
	} else {
		// Handle PreToolUse and other hooks
		response, err = h.ProcessPreToolUse(ctx, rawJSON)
	}

	if err != nil {
		return ProcessResult{}, err
	}

	// Convert string response to ProcessResult
	return convertResponseToProcessResult(response), nil
}

// shouldSkipProcessing checks state manager settings and returns true if processing should be skipped
func (h *DefaultHookProcessor) shouldSkipProcessing(ctx context.Context) bool {
	if h.stateManager == nil {
		return false
	}

	logger := logging.Get(ctx)

	// Check if rules are disabled
	rulesEnabled, err := h.stateManager.GetRulesEnabled(ctx)
	if err != nil {
		logger.Debug().Err(err).Msg("Failed to check rules enabled state, proceeding with normal processing")
	}
	if err == nil && !rulesEnabled {
		logger.Debug().Msg("Rules disabled via state manager, allowing command")
		return true
	}

	// Check and consume skip flag
	skipNext, err := h.stateManager.ConsumeSkipNext(ctx)
	if err != nil {
		logger.Debug().Err(err).Msg("Failed to check skip flag, proceeding with normal processing")
	}
	if err == nil && skipNext {
		logger.Debug().Msg("Skip flag consumed, allowing command")
		return true
	}

	return false
}

// ProcessPreToolUse handles PreToolUse hook events
func (h *DefaultHookProcessor) ProcessPreToolUse(ctx context.Context, rawJSON json.RawMessage) (string, error) {
	logger := logging.Get(ctx)
	logger.Debug().Msg("processing PreToolUse hook")

	// Check state manager for rules enabled/skip state
	if h.shouldSkipProcessing(ctx) {
		return "", nil
	}

	var event hooks.HookEvent
	if unmarshalErr := json.Unmarshal(rawJSON, &event); unmarshalErr != nil {
		return "", fmt.Errorf("failed to parse hook input: %w", unmarshalErr)
	}

	// Check operation state - block editing tools if in plan mode
	if h.stateManager != nil {
		operationState, err := h.stateManager.GetOperationMode(ctx)
		if err != nil {
			logger.Debug().Err(err).Msg("Failed to get operation state, proceeding with normal processing")
		} else if operationState != nil && operationState.Mode == operation.PlanMode &&
			h.isEditingTool(event.ToolName) {
			message := "You're currently in plan mode. Please discuss your planned changes first, " +
				"then use a trigger phrase like 'make it so', 'go ahead', or 'proceed' to enter " +
				"execute mode."
			return message, nil
		}
	}

	// Extract intent from transcript if available
	var intentContent string
	if event.TranscriptPath != "" {
		intentContent = h.ExtractAndLogIntent(ctx, &event)
	}

	// Log summary of available sources for rule matching
	logger.Debug().
		Str("hook_type", "PreToolUse").
		Str("tool_name", event.ToolName).
		Str("extracted_intent", intentContent).
		Interface("tool_input", event.ToolInput).
		Msg("Hook processing summary - sources available for rule matching")

	// Load config and create matcher
	cfg, _, err := h.configValidator.LoadConfigAndMatcher(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	// Filter and process pre-event rules
	preRules := h.filterPreEventRules(cfg.Rules)
	ruleMatcher, err := matcher.NewRuleMatcher(preRules)
	if err != nil {
		return "", fmt.Errorf("failed to create rule matcher: %w", err)
	}

	// Find matching rule
	matchedRule, matchedValue := h.findMatchingPreRule(ctx, preRules, ruleMatcher, &event)
	if matchedRule == nil {
		return "", nil
	}

	// Process and return response
	return h.processMatchedRule(ctx, matchedRule, matchedValue)
}

// filterPreEventRules filters rules for pre events
func (*DefaultHookProcessor) filterPreEventRules(rules []config.Rule) []config.Rule {
	var preRules []config.Rule
	for i := range rules {
		rule := &rules[i]
		match := rule.GetMatch()

		// Check if rule applies to pre events (default is "pre")
		if match.Event == "pre" || match.Event == "" {
			preRules = append(preRules, *rule)
		}
	}
	return preRules
}

// ExtractAndLogIntent extracts and logs intent content from transcript (public for testing)
func (*DefaultHookProcessor) ExtractAndLogIntent(ctx context.Context, event *hooks.HookEvent) string {
	if event.TranscriptPath == "" {
		return ""
	}

	intentContent, err := transcript.FindRecentToolUseAndExtractIntent(ctx, event.TranscriptPath)
	if err != nil {
		logging.Get(ctx).Debug().Err(err).
			Str("transcript_path", event.TranscriptPath).
			Msg("Failed to extract intent from transcript")
		return ""
	}

	logging.Get(ctx).Debug().
		Str("transcript_path", event.TranscriptPath).
		Str("extracted_intent", intentContent).
		Msg("Intent extracted from transcript for hook processing")

	return intentContent
}

// findMatchingPreRule finds the first rule that matches the event
func (h *DefaultHookProcessor) findMatchingPreRule(
	ctx context.Context, preRules []config.Rule, ruleMatcher *matcher.RuleMatcher, event *hooks.HookEvent,
) (rule *config.Rule, matchedField string) {
	for i := range preRules {
		rule := &preRules[i]

		if matchedRule, matchedValue := h.checkRuleSources(ctx, rule, ruleMatcher, event); matchedRule != nil {
			return matchedRule, matchedValue
		}
	}
	return nil, ""
}

// checkRuleSources checks if rule matches using sources or fallback behavior
func (h *DefaultHookProcessor) checkRuleSources(
	ctx context.Context, rule *config.Rule, ruleMatcher *matcher.RuleMatcher, event *hooks.HookEvent,
) (matchedRule *config.Rule, matchedField string) {
	match := rule.GetMatch()
	if len(match.Sources) > 0 {
		return h.checkSpecificSources(ctx, rule, ruleMatcher, event)
	}
	return h.checkOriginalBehavior(rule, event)
}

// checkSpecificSources checks only specified source fields
func (h *DefaultHookProcessor) checkSpecificSources(
	ctx context.Context, rule *config.Rule, ruleMatcher *matcher.RuleMatcher, event *hooks.HookEvent,
) (matchedRule *config.Rule, matchedField string) {
	match := rule.GetMatch()
	for _, fieldName := range match.Sources {
		if matched, content := h.checkIntentSource(ctx, fieldName, rule, ruleMatcher, event); matched {
			return rule, content
		}
		if matched, content := h.checkToolInputSource(fieldName, rule, ruleMatcher, event); matched {
			return rule, content
		}
	}
	return nil, ""
}

// checkIntentSource handles #intent source field
func (h *DefaultHookProcessor) checkIntentSource(
	ctx context.Context, fieldName string, rule *config.Rule, ruleMatcher *matcher.RuleMatcher, event *hooks.HookEvent,
) (matched bool, content string) {
	if fieldName != "#intent" {
		return false, ""
	}
	if event.TranscriptPath == "" {
		return false, ""
	}

	var intentContent string
	var err error

	if event.ToolUseID != "" {
		// Use precise tool-use-ID based extraction
		intentContent, err = transcript.ExtractIntentByToolUseIDWithContext(
			ctx, event.TranscriptPath, event.ToolUseID)
	} else {
		// Use new reliable method that scans backwards for recent tool use
		intentContent, err = transcript.FindRecentToolUseAndExtractIntent(ctx, event.TranscriptPath)
	}

	if err != nil || strings.TrimSpace(intentContent) == "" {
		return false, ""
	}
	return h.matchRuleContent(intentContent, rule, ruleMatcher, event.ToolName)
}

// checkToolInputSource handles regular ToolInput fields
func (h *DefaultHookProcessor) checkToolInputSource(
	fieldName string, rule *config.Rule, ruleMatcher *matcher.RuleMatcher, event *hooks.HookEvent,
) (matched bool, content string) {
	value, exists := event.ToolInput[fieldName]
	if !exists {
		return false, ""
	}
	strValue, ok := value.(string)
	if !ok {
		return false, ""
	}
	return h.matchRuleContent(strValue, rule, ruleMatcher, event.ToolName)
}

// matchRuleContent checks if content matches rule pattern
func (h *DefaultHookProcessor) matchRuleContent(
	content string, rule *config.Rule, _ *matcher.RuleMatcher, toolName string,
) (matched bool, matchedContent string) {
	// Create template context with project information
	templateContext := make(map[string]any)
	if h.projectRoot != "" {
		templateContext["ProjectRoot"] = h.projectRoot
	}

	// Create a temporary matcher with just this single rule to test if content matches
	tempMatcher, err := matcher.NewRuleMatcher([]config.Rule{*rule})
	if err != nil {
		return false, ""
	}

	foundRule, err := tempMatcher.MatchWithContext(content, toolName, templateContext)
	isMatch := err == nil && foundRule != nil
	if isMatch {
		return true, content
	}
	return false, ""
}

// checkOriginalBehavior uses original matching behavior for backward compatibility
func (h *DefaultHookProcessor) checkOriginalBehavior(rule *config.Rule, event *hooks.HookEvent) (
	matchedRule *config.Rule, matchedField string,
) {
	tempMatcher, err := matcher.NewRuleMatcher([]config.Rule{*rule})
	if err != nil {
		return nil, ""
	}

	foundRule, foundValue, err := h.findMatchingRule(tempMatcher, event)
	if err == nil && foundRule != nil {
		return foundRule, foundValue
	}
	return nil, ""
}

func (h *DefaultHookProcessor) findMatchingRule(
	ruleMatcher *matcher.RuleMatcher, event *hooks.HookEvent,
) (*config.Rule, string, error) {
	for key, value := range event.ToolInput {
		strValue, ok := value.(string)
		if !ok {
			continue
		}

		// Create template context with project information
		templateContext := make(map[string]any)
		if h.projectRoot != "" {
			templateContext["ProjectRoot"] = h.projectRoot
		}

		rule, err := ruleMatcher.MatchWithContext(strValue, event.ToolName, templateContext)
		if err != nil {
			if errors.Is(err, matcher.ErrNoRuleMatch) {
				continue // Try next field
			}
			return nil, "", fmt.Errorf("failed to match rule for %s '%s': %w", key, strValue, err)
		}

		if rule != nil {
			return rule, strValue, nil
		}
	}

	return nil, "", nil
}

// processMatchedRule processes template and AI generation for matched rule
func (h *DefaultHookProcessor) processMatchedRule(
	ctx context.Context, matchedRule *config.Rule, matchedValue string,
) (string, error) {
	// Process template with rule context including shared variables
	processedMessage, err := template.ExecuteRuleTemplate(matchedRule.Send, matchedValue)
	if err != nil {
		return "", fmt.Errorf("failed to process rule template: %w", err)
	}

	// Apply AI generation if configured
	finalMessage, err := h.processAIGeneration(ctx, matchedRule, processedMessage, matchedValue)
	if err != nil {
		// Log error but don't fail the hook - fallback to original message
		logging.Get(ctx).Error().Err(err).Msg("AI generation failed, using original message")
		return processedMessage, nil
	}

	return finalMessage, nil
}

// processAIGeneration applies AI generation to a message if configured
func (h *DefaultHookProcessor) processAIGeneration(
	ctx context.Context, rule *config.Rule, message, _ string,
) (string, error) {
	generate := rule.GetGenerate()
	// Skip if generation mode is "off"
	if generate.Mode == "off" {
		return message, nil
	}

	// Use XDG-compliant database path
	storageManager := storage.New(afero.NewOsFs())
	cachePath, err := storageManager.GetDatabasePath()
	if err != nil {
		return message, fmt.Errorf("failed to get database path: %w", err)
	}

	// Create AI generator with mock launcher if available
	var generator *ai.Generator
	if h.aiGenerator != nil {
		generator, err = ai.NewGeneratorWithLauncher(ctx, cachePath, h.projectRoot, h.aiGenerator)
	} else {
		generator, err = ai.NewGenerator(ctx, cachePath, h.projectRoot)
	}
	if err != nil {
		return message, fmt.Errorf("failed to create AI generator: %w", err)
	}
	defer func() {
		if closeErr := generator.Close(); closeErr != nil {
			// Log error but don't fail the hook - generator.Close() error is non-critical
			_ = closeErr // Silence linter about empty block
		}
	}()

	// Create request
	match := rule.GetMatch()
	req := &ai.GenerateRequest{
		OriginalMessage: message,
		CustomPrompt:    generate.Prompt,
		GenerateMode:    generate.Mode,
		Pattern:         match.Pattern,
	}

	// Generate message
	result, err := generator.GenerateMessage(ctx, req)
	if err != nil {
		return message, fmt.Errorf("failed to generate AI message: %w", err)
	}

	return result, nil
}

// ProcessPostToolUse processes post-tool-use hook events

func (*DefaultHookProcessor) extractPostToolContent(
	ctx context.Context, rawJSON json.RawMessage,
) (*postToolContent, error) {
	logger := logging.Get(ctx)

	// Parse the JSON to get transcript path and tool info
	var event map[string]any
	if err := json.Unmarshal(rawJSON, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal post-tool-use event: %w", err)
	}

	transcriptPath, _ := event["transcript_path"].(string) //nolint:revive // intentionally ignoring ok value
	toolName, _ := event["tool_name"].(string)             //nolint:revive // intentionally ignoring ok value
	toolUseID, _ := event["tool_use_id"].(string)          //nolint:revive // intentionally ignoring ok value
	toolResponse := event["tool_response"]

	content := &postToolContent{
		toolName:      toolName,
		toolOutputMap: make(map[string]any),
	}

	// Read transcript content for intent matching using efficient parser
	if transcriptPath != "" { //nolint:nestif // intent extraction logic complexity is acceptable
		logger.Debug().
			Str("tool_name", toolName).
			Str("transcript_path", transcriptPath).
			Msg("PostToolUse hook triggered, extracting intent")
		var intent string
		var err error

		if toolUseID != "" {
			intent, err = transcript.ExtractIntentByToolUseIDWithContext(ctx, transcriptPath, toolUseID)
		} else {
			intent, err = transcript.FindRecentToolUseAndExtractIntent(ctx, transcriptPath)
		}
		if err != nil {
			logger.Debug().Err(err).Str("transcript_path", transcriptPath).Msg("Failed to extract intent")
		} else {
			logger.Debug().
				Str("transcript_path", transcriptPath).
				Str("extracted_intent", intent).
				Msg("FindRecentToolUseAndExtractIntent extracted content from transcript")
		}
		content.intent = intent
	}

	// Extract tool output fields from structured response
	if toolResponse != nil {
		switch v := toolResponse.(type) {
		case map[string]any:
			content.toolOutputMap = v
		case string:
			// Handle simple string responses for backward compatibility
			content.toolOutputMap["tool_response"] = v
		}
	}

	return content, nil
}

func (*DefaultHookProcessor) determineRuleContentMatch(rule *config.Rule, content *postToolContent) (string, bool) {
	match := rule.GetMatch()

	if match.Event != "post" {
		return "", false
	}

	return determinePostEventMatch(rule, content)
}

func (h *DefaultHookProcessor) ProcessPostToolUse(ctx context.Context, rawJSON json.RawMessage) (string, error) {
	logger := logging.Get(ctx)
	logger.Debug().Msg("processing PostToolUse hook")

	// Check state manager for rules enabled state
	if h.shouldSkipProcessing(ctx) {
		return "", nil
	}

	// Load config for rule matching
	cfg, _, err := h.configValidator.LoadConfigAndMatcher(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	content, err := h.extractPostToolContent(ctx, rawJSON)
	if err != nil {
		return "", err
	}

	logger.Debug().
		Str("hook_type", "PostToolUse").
		Str("tool_name", content.toolName).
		Str("extracted_intent", content.intent).
		Int("tool_response_field_count", len(content.toolOutputMap)).
		Interface("tool_response_sources", func() []string {
			sources := make([]string, 0, len(content.toolOutputMap))
			for key := range content.toolOutputMap {
				sources = append(sources, key)
			}
			return sources
		}()).
		Msg("Hook processing summary - sources available for rule matching")

	// Skip if no content to match against (neither intent nor tool response)
	if content.intent == "" && len(content.toolOutputMap) == 0 {
		logger.Debug().Msg("No content to match against - skipping PostToolUse processing")
		return "", nil
	}

	// Check each rule for post-tool-use matching
	for i := range cfg.Rules {
		rule := &cfg.Rules[i]
		contentToMatch, hasMatch := h.determineRuleContentMatch(rule, content)
		if !hasMatch {
			continue
		}

		// Check if pattern matches the selected content
		if matched, err := h.matchRulePattern(ctx, rule, contentToMatch, content.toolName); err == nil && matched {
			// Process and return the rule's message using existing template system
			return template.ExecuteRuleTemplate(rule.Send, contentToMatch) //nolint:wrapcheck // preserve behavior
		}
	}

	return "", nil
}

// matchRulePattern checks if a rule's pattern matches the given content
func (*DefaultHookProcessor) matchRulePattern(
	ctx context.Context, rule *config.Rule, content, toolName string,
) (bool, error) {
	// Check tool pattern if specified (similar to existing matcher logic)
	toolPattern := rule.Tool
	if toolPattern != "" {
		toolRe, err := regexp.Compile("(?i)" + toolPattern)
		if err != nil {
			logging.Get(ctx).Debug().Err(err).Str("pattern", toolPattern).Msg("invalid tool pattern")
			return false, err //nolint:wrapcheck // preserving existing behavior
		}
		if !toolRe.MatchString(toolName) {
			return false, nil
		}
	}

	// Check content pattern
	match := rule.GetMatch()
	contentRe, err := regexp.Compile(match.Pattern)
	if err != nil {
		logging.Get(ctx).Debug().Err(err).Str("pattern", match.Pattern).Msg("invalid content pattern")
		return false, err //nolint:wrapcheck // preserving existing behavior
	}

	return contentRe.MatchString(content), nil
}

// isEditingTool checks if the given tool name is an editing tool that should be blocked in discussion mode
func (*DefaultHookProcessor) isEditingTool(toolName string) bool {
	editingTools := []string{
		"Edit", "Write", "MultiEdit", "NotebookEdit",
	}

	for _, tool := range editingTools {
		if toolName == tool {
			return true
		}
	}
	return false
}

// Helper functions from original implementation

func determinePostEventMatch(rule *config.Rule, content *postToolContent) (string, bool) {
	match := rule.GetMatch()
	// If no sources specified, match against all tool output fields by default
	if len(match.Sources) == 0 {
		return findFirstToolOutputValue(content.toolOutputMap)
	}

	matchesIntent, matchesToolOutput := analyzeSourceMatches(match.Sources)

	// Skip if rule doesn't match any available content
	if !matchesIntent && !matchesToolOutput {
		return "", false
	}

	// Choose content to match against (prioritize intent for backward compatibility)
	if matchesIntent && content.intent != "" {
		return content.intent, true
	}

	if matchesToolOutput {
		return findMatchingToolOutputField(match.Sources, content.toolOutputMap)
	}

	return "", false
}

func findFirstToolOutputValue(toolOutputMap map[string]any) (string, bool) {
	for _, value := range toolOutputMap {
		if strValue, ok := value.(string); ok && strValue != "" {
			return strValue, true
		}
	}
	return "", false
}

func analyzeSourceMatches(sources []string) (matchesIntent, matchesToolOutput bool) {
	for _, source := range sources {
		if source == intentFieldName {
			matchesIntent = true
		} else {
			matchesToolOutput = true
		}

		// Early exit if both are found
		if matchesIntent && matchesToolOutput {
			break
		}
	}
	return matchesIntent, matchesToolOutput
}

func findMatchingToolOutputField(sources []string, toolOutputMap map[string]any) (string, bool) {
	for _, source := range sources {
		if source != intentFieldName {
			if value, exists := toolOutputMap[source]; exists {
				if strValue, ok := value.(string); ok && strValue != "" {
					return strValue, true
				}
			}
		}
	}
	return "", false
}
