package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/afero"
	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/core/logging"
	"github.com/wizzomafizzo/bumpers/internal/core/messaging/template"
	"github.com/wizzomafizzo/bumpers/internal/infrastructure/constants"
	ai "github.com/wizzomafizzo/bumpers/internal/platform/claude/api"
	"github.com/wizzomafizzo/bumpers/internal/platform/storage"
)

// SessionManager handles session start events and session-based operations
type SessionManager interface {
	ProcessSessionStart(ctx context.Context, rawJSON json.RawMessage) (string, error)
	ClearSessionCache(ctx context.Context) error
}

// DefaultSessionManager implements SessionManager
type DefaultSessionManager struct {
	fileSystem afero.Fs
	aiHelper   *AIHelper
	cache      *ai.Cache
	configPath string
}

// SessionManagerOptions configures SessionManager construction
type SessionManagerOptions struct {
	FileSystem  afero.Fs
	Cache       *ai.Cache
	ConfigPath  string
	ProjectRoot string
}

// NewSessionManager creates a new SessionManager (maintains backward compatibility)
func NewSessionManager(configPath, projectRoot string, fileSystem afero.Fs) *DefaultSessionManager {
	return NewSessionManagerFromOptions(SessionManagerOptions{
		ConfigPath:  configPath,
		ProjectRoot: projectRoot,
		FileSystem:  fileSystem,
		Cache:       nil,
	})
}

// NewSessionManagerFromOptions creates a new SessionManager with options pattern
func NewSessionManagerFromOptions(opts SessionManagerOptions) *DefaultSessionManager {
	return &DefaultSessionManager{
		configPath: opts.ConfigPath,
		fileSystem: opts.FileSystem,
		aiHelper:   NewAIHelper(AIHelperOptions{ProjectRoot: opts.ProjectRoot, FileSystem: opts.FileSystem}),
		cache:      opts.Cache,
	}
}

// NewSessionManagerWithCache creates a new SessionManager with shared cache instance
func NewSessionManagerWithCache(
	configPath, projectRoot string, fileSystem afero.Fs, cache *ai.Cache,
) *DefaultSessionManager {
	return NewSessionManagerFromOptions(SessionManagerOptions{
		ConfigPath:  configPath,
		ProjectRoot: projectRoot,
		FileSystem:  fileSystem,
		Cache:       cache,
	})
}

// SetMockAIGenerator sets a mock AI generator for testing
func (s *DefaultSessionManager) SetMockAIGenerator(generator ai.MessageGenerator) {
	s.aiHelper.aiGenerator = generator
}

// SetCacheForTesting sets the cache instance for testing
func (s *DefaultSessionManager) SetCacheForTesting(cache *ai.Cache) {
	s.cache = cache
}

// getFileSystem returns the filesystem to use - either injected or defaults to OS
func (s *DefaultSessionManager) getFileSystem() afero.Fs {
	if s.fileSystem != nil {
		return s.fileSystem
	}
	return afero.NewOsFs()
}

func (s *DefaultSessionManager) ProcessSessionStart(ctx context.Context, rawJSON json.RawMessage) (string, error) {
	logger := logging.Get(ctx)
	logger.Debug().Msg("processing SessionStart hook")

	// Parse the SessionStart JSON
	var event SessionStartEvent
	if err := json.Unmarshal(rawJSON, &event); err != nil {
		return "", fmt.Errorf("failed to parse SessionStart event: %w", err)
	}

	// Only process startup and clear sources
	if event.Source != constants.SessionSourceStartup && event.Source != constants.SessionSourceClear {
		return "", nil
	}

	// Clear session-based cache entries when a new session starts
	if cacheErr := s.ClearSessionCache(ctx); cacheErr != nil {
		// Log error but don't fail the hook - cache clearing is non-critical
		logger.Warn().Err(cacheErr).Msg("failed to clear session cache")
	}

	// Load config to get notes
	cfg, err := config.Load(s.configPath)
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	// If no notes, return empty
	if len(cfg.Session) == 0 {
		return "", nil
	}

	// Process and concatenate all note messages
	messages := make([]string, 0, len(cfg.Session))
	for _, note := range cfg.Session {
		// Process template with note context including shared variables
		processedMessage, templateErr := template.ExecuteNoteTemplate(note.Add)
		if templateErr != nil {
			return "", fmt.Errorf("failed to process note template: %w", templateErr)
		}

		// Apply AI generation if configured
		finalMessage, genErr := s.aiHelper.ProcessAIGenerationGeneric(ctx, &note, processedMessage, "")
		if genErr != nil {
			// Log error but don't fail the hook - fallback to original message
			logger.Error().Err(genErr).Msg("AI generation failed, using original message")
			finalMessage = processedMessage
		}

		messages = append(messages, finalMessage)
	}

	additionalContext := strings.Join(messages, "\n")

	// Create hook response that adds context
	response := HookSpecificOutput{
		HookEventName:     constants.SessionStartEvent,
		AdditionalContext: additionalContext,
	}

	// Wrap in hookSpecificOutput structure
	responseWrapper := map[string]any{
		"hookSpecificOutput": response,
	}

	// Convert to JSON
	responseJSON, err := json.Marshal(responseWrapper)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(responseJSON), nil
}

// ClearSessionCache clears all session-based cached AI generation entries
func (s *DefaultSessionManager) ClearSessionCache(ctx context.Context) error {
	// Use shared cache instance if available
	if s.cache != nil {
		err := s.cache.ClearSessionCache(ctx)
		if err != nil {
			return fmt.Errorf("failed to clear session cache: %w", err)
		}

		logging.Get(ctx).Debug().
			Str("project_root", s.aiHelper.projectRoot).
			Msg("session cache cleared on session start using shared cache")

		return nil
	}

	// Fallback to creating temporary cache instance (for backward compatibility)
	storageManager := storage.New(s.getFileSystem())
	cachePath, err := storageManager.GetDatabasePath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	// Create cache instance with project context
	cache, err := ai.NewCacheWithProject(ctx, cachePath, s.aiHelper.projectRoot)
	if err != nil {
		return fmt.Errorf("failed to create cache: %w", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			// Log error but don't fail the function - cache close is non-critical
			logging.Get(ctx).Error().Err(closeErr).Msg("failed to close cache")
		}
	}()

	// Clear session cache entries
	err = cache.ClearSessionCache(ctx)
	if err != nil {
		return fmt.Errorf("failed to clear session cache: %w", err)
	}

	logging.Get(ctx).Debug().
		Str("project_root", s.aiHelper.projectRoot).
		Msg("session cache cleared on session start using fallback cache")

	return nil
}
