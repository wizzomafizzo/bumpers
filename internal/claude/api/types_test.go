package ai

import (
	"testing"
	"time"
)

func TestIsValidGenerateMode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		mode string
		want bool
	}{
		{"off", true},
		{"once", true},
		{"session", true},
		{"always", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			t.Parallel()
			if got := IsValidGenerateMode(tt.mode); got != tt.want {
				t.Errorf("IsValidGenerateMode(%q) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}

func TestCacheEntryIsExpired(t *testing.T) {
	t.Parallel()
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		entry CacheEntry
		name  string
		want  bool
	}{
		{
			name:  "no expiration",
			entry: CacheEntry{ExpiresAt: nil},
			want:  false,
		},
		{
			name:  "expired",
			entry: CacheEntry{ExpiresAt: &past},
			want:  true,
		},
		{
			name:  "not expired",
			entry: CacheEntry{ExpiresAt: &future},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.entry.IsExpired(); got != tt.want {
				t.Errorf("CacheEntry.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateRequestShouldCache(t *testing.T) {
	t.Parallel()
	tests := []struct {
		mode string
		want bool
	}{
		{"off", false},
		{"once", true},
		{"session", true},
		{"always", false},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			t.Parallel()
			req := GenerateRequest{GenerateMode: tt.mode}
			if got := req.ShouldCache(); got != tt.want {
				t.Errorf("GenerateRequest.ShouldCache() with mode %q = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}
