package cli

import (
	"context"
	"fmt"

	"github.com/spf13/afero"
	"github.com/wizzomafizzo/bumpers/internal/core/logging"
	ai "github.com/wizzomafizzo/bumpers/internal/platform/claude/api"
	"github.com/wizzomafizzo/bumpers/internal/platform/storage"
)

// AIHelper provides shared AI generation functionality
type AIHelper struct {
	cachePath   string // Optional cache path injection for tests
	aiGenerator ai.MessageGenerator
	fileSystem  afero.Fs
	projectRoot string
}

// AIHelperOptions holds configuration options for AIHelper
type AIHelperOptions struct {
	Generator   ai.MessageGenerator
	FileSystem  afero.Fs
	CachePath   string
	ProjectRoot string
}

// NewAIHelper creates a new AI helper with options
func NewAIHelper(opts AIHelperOptions) *AIHelper {
	return &AIHelper{
		cachePath:   opts.CachePath,
		projectRoot: opts.ProjectRoot,
		aiGenerator: opts.Generator,
		fileSystem:  opts.FileSystem,
	}
}

// getFileSystem returns the filesystem to use - either injected or defaults to OS
func (h *AIHelper) getFileSystem() afero.Fs {
	if h.fileSystem != nil {
		return h.fileSystem
	}
	return afero.NewOsFs()
}

// ProcessAIGenerationGeneric method that accepts any type with GetGenerate()
func (h *AIHelper) ProcessAIGenerationGeneric(
	ctx context.Context,
	generateConfig GenerateConfig,
	message, pattern string,
) (string, error) {
	generate := generateConfig.GetGenerate()
	// Skip if generation mode is "off"
	if generate.Mode == "off" {
		return message, nil
	}

	// Use injected cache path (for tests) or XDG-compliant cache path (production)
	var cachePath string
	var err error

	if h.cachePath != "" {
		// Use injected cache path (for tests)
		cachePath = h.cachePath
	} else {
		// Use XDG-compliant database path (production)
		storageManager := storage.New(h.getFileSystem())
		cachePath, err = storageManager.GetDatabasePath()
		if err != nil {
			return message, fmt.Errorf("failed to get database path: %w", err)
		}
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
			logging.Get(ctx).Error().Err(closeErr).Msg("failed to close AI generator")
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
	result, err := generator.GenerateMessage(ctx, req)
	if err != nil {
		return message, fmt.Errorf("failed to generate AI message: %w", err)
	}

	return result, nil
}
