package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/core/engine/matcher"
	"github.com/wizzomafizzo/bumpers/internal/core/logging"
	"github.com/wizzomafizzo/bumpers/internal/core/messaging/template"
)

// ConfigValidator handles configuration loading, validation, and testing
type ConfigValidator interface {
	LoadConfigAndMatcher(ctx context.Context) (*config.Config, *matcher.RuleMatcher, error)
	ValidateConfig() (string, error)
	TestCommand(ctx context.Context, command string) (string, error)
}

// DefaultConfigValidator implements ConfigValidator
type DefaultConfigValidator struct {
	configPath  string
	projectRoot string
}

// NewConfigValidator creates a new ConfigValidator
func NewConfigValidator(configPath, projectRoot string) *DefaultConfigValidator {
	return &DefaultConfigValidator{
		configPath:  configPath,
		projectRoot: projectRoot,
	}
}

// LoadConfigAndMatcher loads configuration and creates a rule matcher
func (c *DefaultConfigValidator) LoadConfigAndMatcher(
	ctx context.Context,
) (*config.Config, *matcher.RuleMatcher, error) {
	logging.Get(ctx).Debug().Str("config_path", c.configPath).Msg("loading config file")
	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config from %s: %w", c.configPath, err)
	}

	// Use partial loading to handle invalid rules
	partialCfg, err := config.LoadPartial(data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config from %s: %w", c.configPath, err)
	}

	// Log warnings for invalid rules
	for i := range partialCfg.ValidationWarnings {
		warning := &partialCfg.ValidationWarnings[i]
		logging.Get(ctx).Warn().
			Int("rule_index", warning.RuleIndex).
			Str("pattern", warning.Rule.GetMatch().Pattern).
			Err(warning.Error).
			Msg("invalid rule skipped")
	}

	ruleMatcher, err := matcher.NewRuleMatcher(partialCfg.Rules)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create rule matcher: %w", err)
	}

	return &partialCfg.Config, ruleMatcher, nil
}

func (c *DefaultConfigValidator) TestCommand(ctx context.Context, command string) (string, error) {
	// Load config and match rules
	_, ruleMatcher, err := c.LoadConfigAndMatcher(ctx)
	if err != nil {
		return "", err
	}

	// Create template context with project information
	templateContext := make(map[string]any)
	if c.projectRoot != "" {
		templateContext["ProjectRoot"] = c.projectRoot
	}

	rule, err := ruleMatcher.MatchWithContext(command, "Bash", templateContext)
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

func (c *DefaultConfigValidator) ValidateConfig() (string, error) {
	// Read config file content
	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config from %s: %w", c.configPath, err)
	}

	// Use partial loading to get validation results
	partialCfg, err := config.LoadPartial(data)
	if err != nil {
		return "", fmt.Errorf("failed to load config from %s: %w", c.configPath, err)
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
