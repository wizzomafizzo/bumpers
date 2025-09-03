package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/wizzomafizzo/bumpers/internal/hooks"
)

// HookRouter routes hook events to appropriate handlers
type HookRouter struct {
	handlers map[hooks.HookType]HookHandler
}

// NewHookRouter creates a new HookRouter
func NewHookRouter() *HookRouter {
	return &HookRouter{
		handlers: make(map[hooks.HookType]HookHandler),
	}
}

// HookHandler defines the interface for processing specific hook types
type HookHandler interface {
	Handle(ctx context.Context, rawJSON json.RawMessage) (string, error)
}

// Register registers a handler for a specific hook type
func (r *HookRouter) Register(hookType hooks.HookType, handler HookHandler) {
	r.handlers[hookType] = handler
}

// Route processes the hook input and routes it to the appropriate handler
func (r *HookRouter) Route(ctx context.Context, input io.Reader) (string, error) {
	// Detect hook type and get raw JSON
	hookType, rawJSON, err := hooks.DetectHookType(input)
	if err != nil {
		return "", fmt.Errorf("failed to detect hook type: %w", err)
	}

	// Find and execute handler
	handler, exists := r.handlers[hookType]
	if !exists {
		return "", nil
	}

	result, err := handler.Handle(ctx, rawJSON)
	return result, err
}

// HandlerFunc is an adapter to allow regular functions to be used as HookHandler
type HandlerFunc func(ctx context.Context, rawJSON json.RawMessage) (string, error)

// Handle implements HookHandler interface
func (f HandlerFunc) Handle(ctx context.Context, rawJSON json.RawMessage) (string, error) {
	return f(ctx, rawJSON)
}
