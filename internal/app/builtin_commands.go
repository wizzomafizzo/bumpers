package app

import (
	"context"
	"fmt"

	"github.com/wizzomafizzo/bumpers/internal/database"
	"github.com/wizzomafizzo/bumpers/internal/storage"
)

// IsBuiltinCommand returns true if the input starts with "bumpers "
func IsBuiltinCommand(input string) bool {
	return len(input) >= 8 && input[:8] == "bumpers "
}

// ProcessBuiltinCommand processes a built-in bumpers command
func ProcessBuiltinCommand(ctx context.Context, input, dbPath, projectID string) (any, error) {
	// Create database connection
	dbManager, err := database.NewManager(ctx, dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database manager: %w", err)
	}
	defer func() {
		_ = dbManager.Close() // Best effort cleanup - errors during cleanup are not actionable
	}()

	// Create state manager
	stateManager, err := storage.NewSQLManager(dbManager.DB(), projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	switch input {
	case "bumpers disable":
		if err := stateManager.SetRulesEnabled(ctx, false); err != nil {
			return nil, fmt.Errorf("failed to disable rules: %w", err)
		}
		return "Rules disabled", nil

	case "bumpers enable":
		if err := stateManager.SetRulesEnabled(ctx, true); err != nil {
			return nil, fmt.Errorf("failed to enable rules: %w", err)
		}
		return "Rules enabled", nil

	case "bumpers status":
		enabled, err := stateManager.GetRulesEnabled(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get rules status: %w", err)
		}

		if enabled {
			return "Rules are currently enabled", nil
		}
		return "Rules are currently disabled", nil

	default:
		return "success", nil
	}
}
