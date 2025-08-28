# Detailed Logging Fix List - Race Condition Resolution

## Root Cause
The logging refactor introduced mixed usage of global `log` and context `logger` throughout the codebase. Methods that accept `context.Context` get a context logger with `logger := zerolog.Ctx(ctx)` but then inconsistently use both `logger` and `log`. This creates race conditions in parallel tests.

## Files to Fix

### 1. internal/cli/commands.go

**Context**: This file has `logger := zerolog.Ctx(ctx)` on line 37 but uses global `log` in 9 places.

**Changes needed:**
- Line 42: `log.Error().Err(err).Msg("Failed to parse UserPromptSubmit event")` → `logger.Error().Err(err).Msg("Failed to parse UserPromptSubmit event")`
- Line 55: `log.Debug().Str("commandStr", commandStr).Msg("extracted command string")` → `logger.Debug().Str("commandStr", commandStr).Msg("extracted command string")`
- Lines 59-63: 
  ```go
  log.Debug().
      Str("commandName", commandName).
      Str("args", args).
      Int("argc", len(argv)-1).
      Msg("parsed command arguments")
  ```
  →
  ```go
  logger.Debug().
      Str("commandName", commandName).
      Str("args", args).
      Int("argc", len(argv)-1).
      Msg("parsed command arguments")
  ```
- Line 68: `log.Error().Err(err).Str("configPath", a.configPath).Msg("Failed to load config")` → `logger.Error().Err(err).Str("configPath", a.configPath).Msg("Failed to load config")`
- Line 84: `log.Debug().Str("commandName", commandName).Str("message", commandMessage).Msg("found valid command")` → `logger.Debug().Str("commandName", commandName).Str("message", commandMessage).Msg("found valid command")`
- Line 95: `log.Error().Err(err).Str("commandName", commandName).Msg("Failed to process command template")` → `logger.Error().Err(err).Str("commandName", commandName).Msg("Failed to process command template")`
- Line 103: `log.Error().Err(err).Msg("AI generation failed, using original message")` → `logger.Error().Err(err).Msg("AI generation failed, using original message")`
- Line 121: `log.Error().Err(err).Msg("Failed to marshal response")` → `logger.Error().Err(err).Msg("Failed to marshal response")`
- Line 125: `log.Info().Str("response", string(responseJSON)).Msg("Returning ValidationResult response")` → `logger.Info().Str("response", string(responseJSON)).Msg("Returning ValidationResult response")`

### 2. internal/cli/sessionstart.go

**Context**: This file has `logger := zerolog.Ctx(ctx)` on line 23 but uses global `log` in 2 places.

**Changes needed:**
- Line 40: `log.Warn().Err(cacheErr).Msg("failed to clear session cache")` → `logger.Warn().Err(cacheErr).Msg("failed to clear session cache")`
- Line 67: `log.Error().Err(genErr).Msg("AI generation failed, using original message")` → `logger.Error().Err(genErr).Msg("AI generation failed, using original message")`

### 3. internal/cli/app.go

**Context**: This file has multiple context-aware methods but also has methods without context. Need to identify which methods should use context logging vs global logging.

**Methods with context that need fixing:**
- `processHookWithContext` (has `logger := zerolog.Ctx(ctx)` on line 165) - needs context logging
- `ProcessPostToolUse` (has `logger := zerolog.Ctx(ctx)` on line 505) - needs context logging

**Changes needed:**

#### In `processHookWithContext` method (around line 165+):
- No immediate changes needed in the main method as it already uses `logger` correctly

#### In `ProcessPostToolUse` method (around line 505+):
- Line 519: `log.Debug().` (starts multi-line debug statement) → `logger.Debug().`
- Line 527: `log.Debug().Msg("No content to match against - skipping PostToolUse processing")` → `logger.Debug().Msg("No content to match against - skipping PostToolUse processing")`

#### In methods without context (these are OK to keep using global `log`):
- Line 54: `log.Debug().` in `NewApp()` - OK (no context available)
- Line 101: `log.Debug().Str("configPath", a.configPath).Msg("loading config file")` in `loadConfigAndMatcher()` - OK (no context available)
- Line 116: `log.Warn().` in `loadConfigAndMatcher()` - OK (no context available)
- Line 364: `log.Error().Err(err).Msg("AI generation failed, using original message")` in `processPreToolUse()` - OK (no context available)
- Line 396: `log.Debug().` in `extractPostToolContent()` - OK (static method, no context)
- Line 402: `log.Debug().Err(err).Str("transcriptPath", transcriptPath).Msg("Failed to extract intent")` in `extractPostToolContent()` - OK (static method, no context)
- Line 408: `log.Debug().` in `extractPostToolContent()` - OK (static method, no context)
- Line 556: `log.Debug().Err(err).Str("pattern", toolPattern).Msg("Invalid tool pattern")` in method without context - OK
- Line 568: `log.Debug().Err(err).Str("pattern", match.Pattern).Msg("Invalid content pattern")` in method without context - OK
- Line 693: `log.Debug().` in method without context - OK

## Summary

**Total changes needed: 13 replacements across 3 files**
- internal/cli/commands.go: 9 changes (all `log.` → `logger.`)
- internal/cli/sessionstart.go: 2 changes (all `log.` → `logger.`)
- internal/cli/app.go: 2 changes in context-aware methods (all `log.` → `logger.`)

## Validation Steps After Fix

1. Run `just test-unit ./internal/cli -timeout=10s` to verify tests no longer hang
2. Run `just test` to ensure all tests pass
3. Check for any remaining race conditions with `go test -race ./internal/cli`

## Why These Changes Fix The Issue

The race condition occurs because:
1. Tests call methods with context and expect context-aware logging
2. These methods get context loggers but then mix global and context logging
3. Parallel tests create race conditions when multiple goroutines access global logger
4. Context loggers are isolated per test and avoid races

By consistently using context loggers in context-aware methods, we eliminate the race condition while preserving global logging where no context is available.