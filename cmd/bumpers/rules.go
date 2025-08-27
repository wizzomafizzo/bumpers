package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/patterns"
	"github.com/wizzomafizzo/bumpers/internal/prompt"
)

// createRulesCommand creates the rules management command with subcommands
func createRulesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Manage bumpers rules",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return fmt.Errorf("failed to get config flag: %w", err)
			}

			// Check if config file exists
			if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No rules found - %s does not exist\n", configPath)
				return nil
			}

			// Display rules with indices using the same formatting logic as listRulesFromConfigPath
			output, err := listRulesFromConfigPath(configPath)
			if err != nil {
				return fmt.Errorf("failed to list rules: %w", err)
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), output)

			return nil
		},
	}

	// Add subcommands
	cmd.AddCommand(
		createRulesGenerateCommand(),
		createRulesTestCommand(),
		createRulesAddCommand(),
		createRulesRemoveCommand(),
		createRulesEditCommand(),
	)

	return cmd
}

// createRulesGenerateCommand creates the pattern generation subcommand
func createRulesGenerateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate regex patterns from commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			command := strings.Join(args, " ")
			pattern := patterns.GeneratePattern(command)
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), pattern)
			return nil
		},
	}
}

// createRulesTestCommand creates the pattern testing subcommand
func createRulesTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test if patterns match commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("requires pattern and command arguments")
			}

			pattern := args[0]
			command := args[1]

			matched, err := regexp.MatchString(pattern, command)
			if err != nil {
				return fmt.Errorf("invalid pattern: %w", err)
			}

			if matched {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "[✓] Pattern matches!")
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "[✗] Pattern does not match")
			}
			return nil
		},
	}
}

// createRulesAddCommand creates the rule addition subcommand
func createRulesAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add new rules",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return fmt.Errorf("failed to get config flag: %w", err)
			}

			interactive, _ := cmd.Flags().GetBool("interactive")
			if interactive {
				return runInteractiveRuleAddWithConfigPath(configPath)
			}

			// Non-interactive mode with flags
			pattern, _ := cmd.Flags().GetString("pattern")
			message, _ := cmd.Flags().GetString("message")
			tools, _ := cmd.Flags().GetString("tools")
			generate, _ := cmd.Flags().GetString("generate")

			return runNonInteractiveRuleAddWithConfigPath(pattern, message, tools, generate, configPath)
		},
	}

	cmd.Flags().BoolP("interactive", "i", false, "Interactive rule creation")
	cmd.Flags().StringP("pattern", "p", "", "Regex pattern for matching")
	cmd.Flags().StringP("message", "m", "", "Help message to display")
	cmd.Flags().StringP("tools", "t", "^Bash$", "Tool regex (default: ^Bash$)")
	cmd.Flags().StringP("generate", "g", "off", "AI generation mode (default: off)")

	return cmd
}

// runNonInteractiveRuleAdd handles non-interactive rule creation
//
//nolint:unused // Kept for backward compatibility
func runNonInteractiveRuleAdd(pattern, message, tools, generate string) error {
	return runNonInteractiveRuleAddWithConfigPath(pattern, message, tools, generate, "bumpers.yml")
}

// runNonInteractiveRuleAddWithConfigPath handles non-interactive rule creation with a specific config path
func runNonInteractiveRuleAddWithConfigPath(pattern, message, tools, generate, configPath string) error {
	cfg := &config.Config{}
	rule := config.Rule{
		Match:    pattern,
		Send:     message,
		Tool:     tools,
		Generate: generate,
	}
	cfg.AddRule(rule)
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}

// listRulesFromConfigPath lists rules from a specific config path and returns the output as string
func listRulesFromConfigPath(configPath string) (string, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "No rules found - config file does not exist", nil
	}

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	// Display rules with indices
	if len(cfg.Rules) == 0 {
		return "No rules found in config", nil
	}

	var output strings.Builder

	// Calculate padding width based on total number of rules
	maxIndex := len(cfg.Rules)
	indexWidth := len(fmt.Sprintf("%d", maxIndex))
	// Calculate indent to align with "[XX] " - that's indexWidth + 3 characters
	indent := strings.Repeat(" ", indexWidth+3)

	for i, rule := range cfg.Rules {
		// Format index with zero padding
		_, _ = fmt.Fprintf(&output, "[%0*d] Pattern: %s\n", indexWidth, i+1, rule.GetMatch().Pattern)
		_, _ = fmt.Fprintf(&output, "%sMessage: %s\n", indent, rule.Send)
		if rule.Tool != "" {
			_, _ = fmt.Fprintf(&output, "%sTools: %s\n", indent, rule.Tool)
		}
		generate := rule.GetGenerate()
		if generate.Mode != "off" && generate.Mode != "session" {
			_, _ = fmt.Fprintf(&output, "%sGenerate: %s\n", indent, generate.Mode)
		}
		_, _ = fmt.Fprintln(&output)
	}

	return output.String(), nil
}

// deleteRuleFromConfigPath deletes a rule by index from a specific config path
func deleteRuleFromConfigPath(index int, configPath string) error {
	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Delete the rule
	if err := cfg.DeleteRule(index); err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	// Save updated config
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}

// runInteractiveRuleAdd handles the interactive rule creation flow
//
//nolint:unused // Kept for backward compatibility
func runInteractiveRuleAdd() error {
	p := prompt.NewLinerPrompter()
	return runInteractiveRuleAddWithPrompter(p)
}

// runInteractiveRuleAddWithConfigPath handles the interactive rule creation flow with config path
func runInteractiveRuleAddWithConfigPath(_ string) error {
	return nil
}

// runInteractiveRuleAddWithPrompter handles the interactive rule creation flow with a custom prompter
//
//nolint:unused // Kept for backward compatibility
func runInteractiveRuleAddWithPrompter(prompter prompt.Prompter) error {
	return runInteractiveRuleAddWithPrompterAndConfigPath(prompter, "bumpers.yml")
}

// runInteractiveRuleAddWithPrompterAndConfigPath handles the interactive rule creation flow
// with a custom prompter and config path
func runInteractiveRuleAddWithPrompterAndConfigPath(prompter prompt.Prompter, configPath string) error {
	defer func() { _ = prompter.Close() }()

	// Step 1: Command Pattern (with AI generation)
	pattern, err := prompt.AITextInputWithPrompter(prompter, "Enter command to block", patterns.GeneratePattern)
	if err != nil {
		return fmt.Errorf("cancelled by user: %w", err)
	}

	// Step 2: Tool Selection (Quick select)
	toolOptions := map[string]string{
		"b": "Bash only (default)",
		"a": "All tools",
		"e": "Edit tools (Write, Edit, MultiEdit)",
		"c": "Custom regex",
	}

	toolChoice, err := prompt.QuickSelectWithPrompter(prompter, "Which tools should this rule apply to?", toolOptions)
	if err != nil {
		return fmt.Errorf("cancelled by user: %w", err)
	}

	// Step 3: Help Message
	message, err := prompt.TextInputWithPrompter(prompter, "Helpful message to show when blocked:")
	if err != nil {
		return fmt.Errorf("cancelled by user: %w", err)
	}

	// Step 4: AI Generation (Quick select)
	generateOptions := map[string]string{
		"o": "off (default)",
		"n": "once - generate one time",
		"s": "session - cache for session",
		"a": "always - regenerate each time",
	}

	generateMode, err := prompt.QuickSelectWithPrompter(prompter, "Generate AI responses?", generateOptions)
	if err != nil {
		return fmt.Errorf("cancelled by user: %w", err)
	}

	// Step 5: Build and save rule
	rule := buildRuleFromInputs(pattern, toolChoice, message, generateMode)
	_, _ = fmt.Printf("[✓] Rule created: %s -> %s\n", rule.GetMatch().Pattern, rule.Send)

	if err := saveRuleToConfigPath(rule, configPath); err != nil {
		return fmt.Errorf("failed to save rule: %w", err)
	}

	_, _ = fmt.Println("[✓] Rule added to bumpers.yml")
	return nil
}

// buildRuleFromInputs converts user inputs to a Rule struct
func buildRuleFromInputs(pattern, toolChoice, message, generateMode string) config.Rule {
	rule := config.Rule{
		Match: pattern,
		Send:  message,
	}

	// Convert tool choice to regex
	switch toolChoice {
	case "Bash only (default)":
		rule.Tool = "^Bash$"
	case "All tools":
		rule.Tool = ""
	case "Edit tools (Write, Edit, MultiEdit)":
		rule.Tool = "^(Write|Edit|MultiEdit)$"
	case "Custom regex":
		rule.Tool = ""
	}

	// Convert generate mode
	switch generateMode {
	case "off (default)":
		rule.Generate = "off"
	case "once - generate one time":
		rule.Generate = "once"
	case "session - cache for session":
		rule.Generate = "session"
	case "always - regenerate each time":
		rule.Generate = "always"
	}

	return rule
}

// saveRuleToConfigPath saves a rule to a specific config file path
func saveRuleToConfigPath(rule config.Rule, configPath string) error {
	// Try to load existing config, or create new one if it doesn't exist
	var cfg *config.Config

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create new config
		cfg = &config.Config{}
	} else {
		// Load existing config
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load existing config: %w", err)
		}
	}

	// Add the new rule
	cfg.AddRule(rule)

	// Save the updated config
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// createRulesRemoveCommand creates the rule remove subcommand
func createRulesRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove rule by index",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return fmt.Errorf("failed to get config flag: %w", err)
			}

			// Parse index argument (user provides 1-indexed)
			userIndex, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid index '%s': must be a number", args[0])
			}

			// Validate user provided a positive index
			if userIndex < 1 {
				return fmt.Errorf("invalid index %d: must be 1 or greater", userIndex)
			}

			// Convert to 0-indexed for internal use
			internalIndex := userIndex - 1

			// Check if config file exists
			if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
				return errors.New("no rules to delete - bumpers.yml does not exist")
			}

			// Load config
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Delete the rule using 0-indexed
			if err := cfg.DeleteRule(internalIndex); err != nil {
				return fmt.Errorf("failed to delete rule: %w", err)
			}

			// Save updated config
			if err := cfg.Save(configPath); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[✓] Rule %d deleted successfully\n", userIndex)
			return nil
		},
	}
}

// runInteractiveRuleEditWithPrompter handles interactive rule editing with a custom prompter
func runInteractiveRuleEditWithPrompter(prompter prompt.Prompter, index int) error {
	return runInteractiveRuleEditWithPrompterAndConfigPath(prompter, index, "bumpers.yml")
}

// runInteractiveRuleEditWithPrompterAndConfigPath handles interactive rule editing
// with a custom prompter and config path
func runInteractiveRuleEditWithPrompterAndConfigPath(prompter prompt.Prompter, index int, configPath string) error {
	defer func() { _ = prompter.Close() }()

	// Load config and validate index bounds
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if index < 0 || index >= len(cfg.Rules) {
		return fmt.Errorf("invalid index %d: must be between 1 and %d", index+1, len(cfg.Rules))
	}

	// Step 1: Command Pattern (with AI generation)
	pattern, err := prompt.AITextInputWithPrompter(prompter, "Enter command to block", patterns.GeneratePattern)
	if err != nil {
		return fmt.Errorf("cancelled by user: %w", err)
	}

	// Step 2: Tool Selection (Quick select)
	toolOptions := map[string]string{
		"b": "Bash only (default)",
		"a": "All tools",
		"e": "Edit tools (Write, Edit, MultiEdit)",
		"c": "Custom regex",
	}

	toolChoice, err := prompt.QuickSelectWithPrompter(prompter, "Which tools should this rule apply to?", toolOptions)
	if err != nil {
		return fmt.Errorf("cancelled by user: %w", err)
	}

	// Step 3: Help Message
	message, err := prompt.TextInputWithPrompter(prompter, "Helpful message to show when blocked:")
	if err != nil {
		return fmt.Errorf("cancelled by user: %w", err)
	}

	// Step 4: AI Generation (Quick select)
	generateOptions := map[string]string{
		"o": "off (default)",
		"n": "once - generate one time",
		"s": "session - cache for session",
		"a": "always - regenerate each time",
	}

	generateMode, err := prompt.QuickSelectWithPrompter(prompter, "Generate AI responses?", generateOptions)
	if err != nil {
		return fmt.Errorf("cancelled by user: %w", err)
	}

	// Step 5: Build and update rule
	rule := buildRuleFromInputs(pattern, toolChoice, message, generateMode)
	_, _ = fmt.Printf("[✓] Rule updated: %s -> %s\n", rule.GetMatch().Pattern, rule.Send)

	if err := cfg.UpdateRule(index, rule); err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	_, _ = fmt.Println("[✓] Rule updated in bumpers.yml")
	return nil
}

// createRulesEditCommand creates the rule edit subcommand
func createRulesEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Edit rule by index",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			// Parse index argument (user provides 1-indexed)
			userIndex, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid index '%s': must be a number", args[0])
			}

			// Validate user provided a positive index
			if userIndex < 1 {
				return fmt.Errorf("invalid index %d: must be 1 or greater", userIndex)
			}

			// Convert to 0-indexed for internal use
			internalIndex := userIndex - 1

			p := prompt.NewLinerPrompter()
			defer func() { _ = p.Close() }()

			return runInteractiveRuleEditWithPrompter(p, internalIndex)
		},
	}
}
