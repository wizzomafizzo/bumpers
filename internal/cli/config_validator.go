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

// ConfigLoader handles configuration loading and parsing
type ConfigLoader interface {
	LoadConfigAndMatcher(ctx context.Context) (*config.Config, *matcher.RuleMatcher, error)
}

// ConfigValidator handles configuration validation and testing
type ConfigValidator interface {
	ConfigLoader
	ValidateConfig() (string, error)
	TestCommand(ctx context.Context, command string) (string, error)
}

// CommandTester handles command testing against rules
type CommandTester interface {
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

// loadPartialConfig loads and parses the configuration file
func (c *DefaultConfigValidator) loadPartialConfig(ctx context.Context) (*config.PartialConfig, error) {
	logging.Get(ctx).Debug().Str("config_path", c.configPath).Msg("loading config file")
	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config from %s: %w", c.configPath, err)
	}

	partialCfg, err := config.LoadPartial(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", c.configPath, err)
	}

	return partialCfg, nil
}

// LoadConfigAndMatcher loads configuration and creates a rule matcher
func (c *DefaultConfigValidator) LoadConfigAndMatcher(
	ctx context.Context,
) (*config.Config, *matcher.RuleMatcher, error) {
	partialCfg, err := c.loadPartialConfig(ctx)
	if err != nil {
		return nil, nil, err
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

	// Process template with rule context including shared variables
	// rule is guaranteed to be non-nil here based on matcher logic
	processedMessage, err := template.ExecuteRuleTemplate(rule.Send, command)
	if err != nil {
		return "", fmt.Errorf("failed to process rule template: %w", err)
	}

	return processedMessage, nil
}

func (c *DefaultConfigValidator) ValidateConfig() (string, error) {
	// Use shared config loading method
	partialCfg, err := c.loadPartialConfig(context.Background())
	if err != nil {
		return "", err
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
