package ai

import "time"

// CacheEntry represents a cached AI-generated message
type CacheEntry struct {
	Timestamp        time.Time  `json:"timestamp"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	GeneratedMessage string     `json:"generated_message"`
	OriginalMessage  string     `json:"original_message"`
}

// GenerateRequest contains the parameters for AI message generation
type GenerateRequest struct {
	OriginalMessage string
	CustomPrompt    string
	GenerateMode    string
	Pattern         string
}

// IsExpired checks if a cache entry has expired based on its mode
func (e *CacheEntry) IsExpired() bool {
	if e.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*e.ExpiresAt)
}

// ShouldCache returns true if the entry should be cached based on mode
func (r *GenerateRequest) ShouldCache() bool {
	const (
		modeOnce    = "once"
		modeSession = "session"
	)
	return r.GenerateMode == modeOnce || r.GenerateMode == modeSession
}

// IsValidGenerateMode checks if the given generate mode is valid
func IsValidGenerateMode(mode string) bool {
	switch mode {
	case "off", "once", "session", "always":
		return true
	default:
		return false
	}
}
