package ai

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wizzomafizzo/bumpers/internal/claude"
)

// MessageGenerator interface for Claude launcher
type MessageGenerator interface {
	GenerateMessage(prompt string) (string, error)
}

// Generator handles AI message generation with caching
type Generator struct {
	cache    *Cache
	launcher MessageGenerator
}

// NewGenerator creates a new AI message generator with project context
func NewGenerator(dbPath, projectID string) (*Generator, error) {
	cache, err := NewCacheWithProject(dbPath, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	return &Generator{
		cache:    cache,
		launcher: claude.NewLauncher(nil),
	}, nil
}

// Close closes the generator
func (g *Generator) Close() error {
	return g.cache.Close()
}

// GenerateMessage generates an AI-enhanced message or returns original on error
func (g *Generator) GenerateMessage(req *GenerateRequest) (string, error) {
	// Skip if generation is off
	if req.GenerateMode == "off" || req.GenerateMode == "" {
		return req.OriginalMessage, nil
	}

	// Generate cache key
	cacheKey := g.generateCacheKey(req)

	// Try to get from cache first (except for "always" mode)
	if req.GenerateMode != "always" {
		if cached, err := g.cache.Get(cacheKey); err == nil && cached != nil {
			if !cached.IsExpired() {
				log.Debug().
					Str("mode", req.GenerateMode).
					Str("original", req.OriginalMessage).
					Msg("AI generation from cache")
				return cached.GeneratedMessage, nil
			}
		}
	}

	// Generate new message using Claude
	prompt := BuildDefaultPrompt(req.OriginalMessage)
	if req.CustomPrompt != "" {
		prompt = req.CustomPrompt + "\n\nMessage: " + req.OriginalMessage
	}

	result, err := g.launcher.GenerateMessage(prompt)
	if err != nil {
		// Return original message with error for caller to handle
		return req.OriginalMessage, fmt.Errorf("claude generation failed: %w", err)
	}

	log.Debug().
		Str("mode", req.GenerateMode).
		Str("original", req.OriginalMessage).
		Msg("AI generation from fresh Claude call")

	// Cache the result if mode supports caching
	if req.ShouldCache() {
		cacheEntry := &CacheEntry{
			GeneratedMessage: result,
			OriginalMessage:  req.OriginalMessage,
			Timestamp:        time.Now(),
			ExpiresAt:        g.calculateExpiry(req.GenerateMode),
		}

		// Store in cache (ignore cache errors)
		_ = g.cache.Put(cacheKey, cacheEntry)
	}

	return result, nil
}

// generateCacheKey creates a unique cache key for the request
func (*Generator) generateCacheKey(req *GenerateRequest) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(req.OriginalMessage))
	_, _ = hash.Write([]byte(req.CustomPrompt))
	_, _ = hash.Write([]byte(req.Pattern))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// calculateExpiry returns expiry time based on mode
func (*Generator) calculateExpiry(mode string) *time.Time {
	switch mode {
	case "session":
		expiry := time.Now().Add(24 * time.Hour)
		return &expiry
	case "once":
		// No expiry for "once" mode
		return nil
	default:
		return nil
	}
}
