package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/wizzomafizzo/bumpers/internal/ai"
	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/filesystem"
	"github.com/wizzomafizzo/bumpers/internal/hooks"
	"github.com/wizzomafizzo/bumpers/internal/matcher"
	"github.com/wizzomafizzo/bumpers/internal/project"
	"github.com/wizzomafizzo/bumpers/internal/storage"
	"github.com/wizzomafizzo/bumpers/internal/template"
)

func NewApp(configPath string) *App {
	// Detect project root
	projectRoot, err := project.FindRoot()
	if err != nil {
		// Fall back to current working directory if project root detection fails
		projectRoot = ""
	}

	// Resolve config path relative to project root if it's relative
	resolvedConfigPath := configPath
	shouldResolve := projectRoot != "" && !filepath.IsAbs(configPath)
	if shouldResolve {
		resolvedConfigPath = filepath.Join(projectRoot, configPath)
	}

	// If using default config name, try different extensions in order
	if shouldResolve && configPath == "bumpers.yml" {
		if _, err := os.Stat(resolvedConfigPath); os.IsNotExist(err) {
			resolvedConfigPath = findAlternativeConfig(projectRoot)
		}
	}

	app := &App{
		configPath:  resolvedConfigPath,
		projectRoot: projectRoot,
	}

	return app
}

func findAlternativeConfig(projectRoot string) string {
	extensions := []string{"yaml", "toml", "json"}
	for _, ext := range extensions {
		candidatePath := filepath.Join(projectRoot, "bumpers."+ext)
		if _, err := os.Stat(candidatePath); err == nil {
			return candidatePath
		}
	}
	return filepath.Join(projectRoot, "bumpers.yml") // fallback to original
}

// NewAppWithWorkDir creates a new App instance with an injectable working directory.
// This is primarily used for testing to avoid global state dependencies.
func NewAppWithWorkDir(configPath, workDir string) *App {
	return &App{configPath: configPath, workDir: workDir}
}

// NewAppWithFileSystem creates a new App instance with injectable filesystem.
// This enables parallel testing by using in-memory filesystem instead of real I/O.
func NewAppWithFileSystem(configPath, workDir string, fs filesystem.FileSystem) *App {
	return &App{
		configPath: configPath,
		workDir:    workDir,
		fileSystem: fs,
	}
}

type App struct {
	fileSystem   filesystem.FileSystem
	mockLauncher ai.MessageGenerator
	configPath   string
	workDir      string
	projectRoot  string
}

// loadConfigAndMatcher loads configuration and creates a rule matcher
func (a *App) loadConfigAndMatcher() (*config.Config, *matcher.RuleMatcher, error) {
	// Read config file content
	data, err := os.ReadFile(a.configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config from %s: %w", a.configPath, err)
	}

	// Use partial loading to handle invalid rules
	partialCfg, err := config.LoadPartial(data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config from %s: %w", a.configPath, err)
	}

	// Log warnings for invalid rules
	for i := range partialCfg.ValidationWarnings {
		warning := &partialCfg.ValidationWarnings[i]
		log.Warn().
			Int("ruleIndex", warning.RuleIndex).
			Str("pattern", warning.Rule.Match).
			Err(warning.Error).
			Msg("Invalid rule skipped")
	}

	ruleMatcher, err := matcher.NewRuleMatcher(partialCfg.Rules)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create rule matcher: %w", err)
	}

	return &partialCfg.Config, ruleMatcher, nil
}

func (*App) findMatchingRule(ruleMatcher *matcher.RuleMatcher, event hooks.HookEvent) (*config.Rule, string, error) {
	for key, value := range event.ToolInput {
		strValue, ok := value.(string)
		if !ok {
			continue
		}

		rule, err := ruleMatcher.Match(strValue, event.ToolName)
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

func (a *App) ProcessHook(input io.Reader) (string, error) {
	if os.Getenv("BUMPERS_SKIP") == "1" {
		log.Debug().Msg("BUMPERS_SKIP is set, skipping hook processing")
		return "", nil
	}

	log.Debug().Msg("processing hook input")

	// Detect hook type and get raw JSON
	hookType, rawJSON, err := hooks.DetectHookType(input)
	if err != nil {
		log.Error().Err(err).Msg("Failed to detect hook type")
		return "", fmt.Errorf("failed to detect hook type: %w", err)
	}
	log.Debug().RawJSON("hook", rawJSON).Msg("hook JSON")

	log.Info().Int("hookType", int(hookType)).Msg("Detected hook type")

	// Handle UserPromptSubmit hooks
	if hookType == hooks.UserPromptSubmitHook {
		log.Info().Msg("Processing UserPromptSubmit hook")
		return a.ProcessUserPrompt(rawJSON)
	}

	// Handle SessionStart hooks
	if hookType == hooks.SessionStartHook {
		log.Info().Msg("Processing SessionStart hook")
		return a.ProcessSessionStart(rawJSON)
	}

	// Handle PostToolUse hooks
	if hookType == hooks.PostToolUseHook {
		log.Info().Msg("Processing PostToolUse hook")
		return a.ProcessPostToolUse(rawJSON)
	}

	// Handle PreToolUse hooks (existing logic)
	var event hooks.HookEvent
	if unmarshalErr := json.Unmarshal(rawJSON, &event); unmarshalErr != nil {
		return "", fmt.Errorf("failed to parse hook input: %w", unmarshalErr)
	}

	// Load config and match rules
	_, ruleMatcher, err := a.loadConfigAndMatcher()
	if err != nil {
		return "", err
	}

	// Try matching against all string fields in tool_input
	matchedRule, matchedValue, err := a.findMatchingRule(ruleMatcher, event)
	if err != nil {
		return "", err
	}

	if matchedRule != nil {
		// Process template with rule context including shared variables
		processedMessage, err := template.ExecuteRuleTemplate(matchedRule.Send, matchedValue)
		if err != nil {
			return "", fmt.Errorf("failed to process rule template: %w", err)
		}

		// Apply AI generation if configured
		finalMessage, err := a.processAIGeneration(matchedRule, processedMessage, matchedValue)
		if err != nil {
			// Log error but don't fail the hook - fallback to original message
			log.Error().Err(err).Msg("AI generation failed, using original message")
			return processedMessage, nil
		}

		return finalMessage, nil
	}

	// This should never happen based on matcher logic, but Go requires a return
	return "", nil
}

// ProcessPostToolUse processes post-tool-use hook events
type postToolContent struct {
	reasoning  string
	toolOutput string
	toolName   string
}

func (*App) extractPostToolContent(rawJSON json.RawMessage) (*postToolContent, error) {
	// Parse the JSON to get transcript path and tool info
	var event map[string]any
	if err := json.Unmarshal(rawJSON, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal post-tool-use event: %w", err)
	}

	transcriptPath, _ := event["transcript_path"].(string) //nolint:revive // intentionally ignoring ok value
	toolName, _ := event["tool_name"].(string)             //nolint:revive // intentionally ignoring ok value
	toolResponse := event["tool_response"]

	content := &postToolContent{toolName: toolName}

	// Read transcript content for reasoning matching
	if transcriptPath != "" {
		reasoningBytes, err := os.ReadFile(transcriptPath) //nolint:gosec // from Claude Code hook event
		if err != nil {
			log.Debug().Str("path", transcriptPath).Msg("Could not read transcript, continuing without reasoning")
		} else {
			content.reasoning = string(reasoningBytes)
		}
	}

	// Extract tool output content for tool_output field matching
	if toolResponse != nil {
		if str, ok := toolResponse.(string); ok {
			content.toolOutput = str
		}
	}

	return content, nil
}

func (*App) determineRuleContentMatch(rule *config.Rule, content *postToolContent) (string, bool) {
	rule.ValidateEventFields()

	matchesReasoning := false
	matchesToolOutput := false

	// New syntax: event="post" + fields
	if rule.Event == "post" {
		if containsString(rule.Fields, "reasoning") {
			matchesReasoning = true
		}
		if containsString(rule.Fields, "tool_output") {
			matchesToolOutput = true
		}
	}

	// Backward compatibility: when includes "reasoning"
	if len(rule.When) > 0 {
		expandedWhen := rule.ExpandWhen()
		if containsString(expandedWhen, "reasoning") {
			matchesReasoning = true
		}
	}

	// Skip if rule doesn't match any available content
	if !matchesReasoning && !matchesToolOutput {
		return "", false
	}

	// Choose content to match against (prioritize reasoning for backward compatibility)
	switch {
	case matchesReasoning && content.reasoning != "":
		return content.reasoning, true
	case matchesToolOutput && content.toolOutput != "":
		return content.toolOutput, true
	default:
		return "", false // No matching content available
	}
}

func (a *App) ProcessPostToolUse(rawJSON json.RawMessage) (string, error) {
	log.Debug().Msg("ProcessPostToolUse called")

	// Load config for rule matching
	cfg, _, err := a.loadConfigAndMatcher()
	if err != nil {
		return "", err
	}

	content, err := a.extractPostToolContent(rawJSON)
	if err != nil {
		return "", err
	}

	// Skip if no content to match against (neither reasoning nor tool output)
	if content.reasoning == "" && content.toolOutput == "" {
		return "", nil
	}

	// Check each rule for post-tool-use matching
	for i := range cfg.Rules {
		rule := &cfg.Rules[i]
		contentToMatch, hasMatch := a.determineRuleContentMatch(rule, content)
		if !hasMatch {
			continue
		}

		// Check if pattern matches the selected content
		if matched, err := a.matchRulePattern(rule, contentToMatch, content.toolName); err == nil && matched {
			// Process and return the rule's message using existing template system
			return template.ExecuteRuleTemplate(rule.Send, contentToMatch) //nolint:wrapcheck // preserve behavior
		}
	}

	return "", nil
}

// containsString checks if slice contains the given string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// matchRulePattern checks if a rule's pattern matches the given content
func (*App) matchRulePattern(rule *config.Rule, content, toolName string) (bool, error) {
	// Check tool pattern if specified (similar to existing matcher logic)
	toolPattern := rule.Tool
	if toolPattern != "" {
		toolRe, err := regexp.Compile("(?i)" + toolPattern)
		if err != nil {
			log.Debug().Err(err).Str("pattern", toolPattern).Msg("Invalid tool pattern")
			return false, err //nolint:wrapcheck // preserving existing behavior
		}
		if !toolRe.MatchString(toolName) {
			return false, nil
		}
	}

	// Check content pattern
	contentRe, err := regexp.Compile(rule.Match)
	if err != nil {
		log.Debug().Err(err).Str("pattern", rule.Match).Msg("Invalid content pattern")
		return false, err //nolint:wrapcheck // preserving existing behavior
	}

	return contentRe.MatchString(content), nil
}

func (a *App) TestCommand(command string) (string, error) {
	// Load config and match rules
	_, ruleMatcher, err := a.loadConfigAndMatcher()
	if err != nil {
		return "", err
	}

	rule, err := ruleMatcher.Match(command, "Bash")
	if err != nil {
		if errors.Is(err, matcher.ErrNoRuleMatch) {
			// No rule matched, command is allowed
			return "Command allowed", nil
		}
		return "", fmt.Errorf("failed to match rule for command '%s': %w", command, err)
	}

	if rule != nil {
		// Process template with rule context including shared variables
		processedMessage, err := template.ExecuteRuleTemplate(rule.Send, command)
		if err != nil {
			return "", fmt.Errorf("failed to process rule template: %w", err)
		}

		return processedMessage, nil
	}

	// This should never happen based on matcher logic, but Go requires a return
	return "Command allowed", nil
}

func (a *App) ValidateConfig() (string, error) {
	// Read config file content
	data, err := os.ReadFile(a.configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config from %s: %w", a.configPath, err)
	}

	// Use partial loading to get validation results
	partialCfg, err := config.LoadPartial(data)
	if err != nil {
		return "", fmt.Errorf("failed to load config from %s: %w", a.configPath, err)
	}

	// Build validation result message
	validCount := len(partialCfg.Rules)
	invalidCount := len(partialCfg.ValidationWarnings)

	var result strings.Builder
	if invalidCount == 0 {
		_, _ = result.WriteString("Configuration is valid")
	} else {
		_, _ = result.WriteString(fmt.Sprintf(
			"Configuration partially valid: %d valid rules, %d invalid rules\n\nInvalid rules:\n",
			validCount, invalidCount))
		for i := range partialCfg.ValidationWarnings {
			warning := &partialCfg.ValidationWarnings[i]
			_, _ = result.WriteString(fmt.Sprintf("  Rule %d: %s (pattern: '%s')\n",
				warning.RuleIndex+1, warning.Error.Error(), warning.Rule.Match))
		}
	}

	// Validate that valid rules can create matcher
	if validCount > 0 {
		_, err = matcher.NewRuleMatcher(partialCfg.Rules)
		if err != nil {
			return "", fmt.Errorf("failed to validate valid config rules: %w", err)
		}
	}

	return result.String(), nil
}

// getFileSystem returns the filesystem to use - either injected or defaults to OS
func (a *App) getFileSystem() filesystem.FileSystem {
	if a.fileSystem != nil {
		return a.fileSystem
	}
	return filesystem.NewOSFileSystem()
}

// SetMockLauncher sets the mock launcher for testing
func (a *App) SetMockLauncher(launcher ai.MessageGenerator) {
	a.mockLauncher = launcher
}

// clearSessionCache clears all session-based cached AI generation entries
func (a *App) clearSessionCache() error {
	// Use XDG-compliant cache path
	storageManager := storage.New(filesystem.NewOSFileSystem())
	cachePath, err := storageManager.GetCachePath()
	if err != nil {
		return fmt.Errorf("failed to get cache path: %w", err)
	}

	// Create cache instance with project context
	cache, err := ai.NewCacheWithProject(cachePath, a.projectRoot)
	if err != nil {
		return fmt.Errorf("failed to create cache: %w", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			// Log error but don't fail the function - cache close is non-critical
			_ = closeErr
		}
	}()

	// Clear session cache entries
	err = cache.ClearSessionCache()
	if err != nil {
		return fmt.Errorf("failed to clear session cache: %w", err)
	}

	log.Debug().
		Str("project", a.projectRoot).
		Msg("Session cache cleared on session start")

	return nil
}

// processAIGeneration applies AI generation to a message if configured
func (a *App) processAIGeneration(rule *config.Rule, message, _ string) (string, error) {
	generate := rule.GetGenerate()
	// Skip if generation mode is "off"
	if generate.Mode == "off" {
		return message, nil
	}

	// Use XDG-compliant cache path
	storageManager := storage.New(filesystem.NewOSFileSystem())
	cachePath, err := storageManager.GetCachePath()
	if err != nil {
		return message, fmt.Errorf("failed to get cache path: %w", err)
	}

	// Create AI generator with mock launcher if available
	var generator *ai.Generator
	if a.mockLauncher != nil {
		generator, err = ai.NewGeneratorWithLauncher(cachePath, a.projectRoot, a.mockLauncher)
	} else {
		generator, err = ai.NewGenerator(cachePath, a.projectRoot)
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
	req := &ai.GenerateRequest{
		OriginalMessage: message,
		CustomPrompt:    generate.Prompt,
		GenerateMode:    generate.Mode,
		Pattern:         rule.Match,
	}

	// Generate message
	result, err := generator.GenerateMessage(req)
	if err != nil {
		return message, fmt.Errorf("failed to generate AI message: %w", err)
	}

	return result, nil
}

// GenerateConfig interface for types that have GetGenerate method
type GenerateConfig interface {
	GetGenerate() config.Generate
}

// processAIGenerationGeneric method that accepts any type with GetGenerate()
func (a *App) processAIGenerationGeneric(generateConfig GenerateConfig, message, pattern string) (string, error) {
	generate := generateConfig.GetGenerate()
	// Skip if generation mode is "off"
	if generate.Mode == "off" {
		return message, nil
	}

	// Use XDG-compliant cache path
	storageManager := storage.New(filesystem.NewOSFileSystem())
	cachePath, err := storageManager.GetCachePath()
	if err != nil {
		return message, fmt.Errorf("failed to get cache path: %w", err)
	}

	// Create AI generator with mock launcher if available
	var generator *ai.Generator
	if a.mockLauncher != nil {
		generator, err = ai.NewGeneratorWithLauncher(cachePath, a.projectRoot, a.mockLauncher)
	} else {
		generator, err = ai.NewGenerator(cachePath, a.projectRoot)
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
	req := &ai.GenerateRequest{
		OriginalMessage: message,
		CustomPrompt:    generate.Prompt,
		GenerateMode:    generate.Mode,
		Pattern:         pattern,
	}

	// Generate message
	result, err := generator.GenerateMessage(req)
	if err != nil {
		return message, fmt.Errorf("failed to generate AI message: %w", err)
	}

	return result, nil
}
