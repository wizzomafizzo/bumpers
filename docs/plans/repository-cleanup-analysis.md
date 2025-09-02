# Repository Cleanup TODO List

*Generated: 2025-09-01*  
*Analysis Scope: Complete repository scan for architecture issues, DRY/SOLID violations, stubs, TODOs, and bugs*

## Cleanup Tasks

### ✅ Completed Tasks (2025-09-02)

#### Architecture Improvements
- [x] Created `HookRouter` struct for cleaner hook routing (`internal/cli/hook_router.go`)
- [x] Replaced multiple if statements with switch statement for hook type routing
- [x] Implemented options pattern with `AppOptions` struct (`internal/cli/app_options.go`)
- [x] Added proper error wrapping in:
  - `internal/cli/hook_processor.go` (3 instances)
  - `internal/platform/state/manager.go` (2 instances)  
  - `internal/platform/claude/api/cache.go` (3 instances)

### Dead Code Removal
- [x] Remove unused `WasCalledWithPattern` method from `internal/platform/claude/mock.go:32-34`
- [x] Remove unused `AssertMockCalledWithPattern` function from `internal/platform/claude/testing_helpers.go:88-95`

### Test Improvements
- [ ] Fix or remove skipped test at `internal/prompt/input_test.go:46` - "Test replaced by TestAITextInputWithMockPrompter"
- [ ] Complete or remove skipped test at `internal/prompt/input_test.go:92` - "QuickSelect implementation with liner not yet complete"
- [ ] Handle conditional skip at `internal/platform/claude/launcher_integration_test.go:46` - "Claude binary not available"

### Architecture Refactoring

#### App Struct Cleanup (`internal/cli/app.go`)
- [x] Extract hook routing to a dedicated `HookRouter` struct with handler registry (Created `hook_router.go`)
- [ ] Reduce App struct to thin coordinator with only essential dependencies (router, dbManager, config)
- [ ] Move component creation logic to Factory/Builder pattern
- [ ] Let each handler manage its own specific dependencies

#### Routing Improvements (`internal/cli/app.go:256-299`)
- [x] Replace multiple if statements with switch statement for hook type routing (Completed in app.go lines 275-288)

#### Constructor Consolidation
- [x] Implement options pattern for App construction to eliminate 3 duplicate constructors (Created `app_options.go`)
- [x] Create single `NewAppWithOptions(opts AppOptions)` function with proper component initialization
- [ ] Remove `NewAppWithDatabase` and `NewAppWithFileSystem` in favor of options (can be done in future cleanup)

### Test Coverage - Add Tests for Critical Files
- [ ] `cmd/bumpers/hook.go`
- [ ] `internal/cli/install_manager.go`
- [ ] `internal/cli/config_validator.go`
- [ ] `internal/cli/prompt_handler.go`
- [ ] `internal/cli/session_manager.go`
- [ ] `internal/cli/app.go`
- [ ] `internal/cli/hook_processor.go`
- [ ] `internal/config/defaults.go`
- [ ] `internal/core/messaging/template/security.go`
- [ ] `internal/core/logging/logger.go`
- [ ] `internal/infrastructure/constants/commands.go`
- [ ] `internal/infrastructure/constants/hooks.go`
- [ ] `internal/infrastructure/constants/paths.go`
- [ ] `internal/platform/claude/settings/persistence.go`
- [ ] `internal/platform/claude/testing_helpers.go`
- [ ] `internal/platform/claude/mock.go`
- [ ] `internal/platform/database/migrations.go`
- [ ] `internal/testing/resources.go`
- [ ] `internal/cli/app_test_helpers.go`
- [ ] `internal/cli/types.go`

### Code Quality Improvements

#### Error Handling
- [x] Identified regex compilation failures in `internal/core/engine/matcher/matcher.go:62-64` and `82-84` - These are already handled gracefully and would benefit from debug logging in future iterations
- [x] Remove all `//nolint:wrapcheck` comments throughout codebase (8 instances removed)
- [x] Add proper error wrapping with context using `fmt.Errorf("context: %w", err)` (8 instances added)

#### Constants and Magic Strings
- [x] Extract hard-coded field names ("prompt", "tool_response", "SessionStart") to constants - Created `internal/infrastructure/constants/fields.go` with field constants
- [x] Started replacement of hard-coded strings with constants in `internal/cli/hook_processor.go`
- [ ] Continue replacing remaining hard-coded field names throughout codebase
- [x] Consolidate scattered constants into `internal/infrastructure/constants/` package (already well organized)
- [x] Organize constants by type: `hooks.go`, `paths.go`, `commands.go`, etc. (already done)

#### Encapsulation
- [x] Global slice variables in `internal/core/engine/operation/operation.go` are already private (lowercase names)
- [x] Getter functions that return copies are already implemented and working properly
- [x] Added test to verify encapsulation works correctly (`TestEncapsulation_GlobalVariablesShouldBePrivate`)

## Priority Levels

**High Priority:**
- Architecture refactoring (App struct, routing, constructors)
- Test coverage for critical components
- Error wrapping improvements

**Medium Priority:**
- Constants consolidation
- Debug logging for regex failures
- Encapsulation fixes

**Low Priority:**
- Dead code removal
- Skipped test cleanup

## Notes

- 35% of source files (20 out of 57) lack corresponding test files
- The codebase has good test-to-source ratio overall (65 test files for 57 source files)
- Focus on maintainability and design improvements rather than bug fixes

## Package Structure Reorganization (Domain-Focused)

### Current Problems
- **Too deep nesting**: 4 levels deep (`internal/core/engine/hooks/`)
- **Single-file packages**: Many directories with just 1 file
- **Unclear categorization**: "core" vs "platform" vs "infrastructure" distinction is confusing
- **Navigation overhead**: Too many directories to traverse for related code

### Proposed New Structure
```
internal/
├── app/           # Main application orchestration (from cli/)
│                  # Contains: app.go, hook_processor.go, prompt_handler.go, 
│                  # session_manager.go, install_manager.go, config_validator.go
├── config/        # Configuration management (unchanged)
├── hooks/         # Hook type detection and processing (from core/engine/hooks/)
├── matcher/       # Pattern matching logic (from core/engine/matcher/)
├── rules/         # Rule/operation processing (from core/engine/operation/)
├── template/      # Template engine (from core/messaging/template/)
│                  # Merge in context.go from messaging/context/
├── claude/        # Claude integration (from platform/claude/)
│   ├── api/       # API client and generator
│   ├── settings/  # Settings management
│   └── transcript/# Transcript reader
├── database/      # Database layer (from platform/database/)
├── storage/       # File storage + state management 
│                  # (merge platform/storage/ + platform/state/)
├── constants/     # All constants (from infrastructure/constants/)
├── project/       # Project detection (from infrastructure/project/)
├── logging/       # Logging utilities (from core/logging/)
├── prompt/        # User prompt utilities (unchanged)
└── testing/       # Test utilities (unchanged)
```

### Migration Checklist
- [ ] **Phase 1: Flatten deep nesting**
  - [ ] Move `internal/core/engine/hooks/` → `internal/hooks/`
  - [ ] Move `internal/core/engine/matcher/` → `internal/matcher/`
  - [ ] Move `internal/core/engine/operation/` → `internal/rules/`
  - [ ] Move `internal/core/logging/` → `internal/logging/`
  - [ ] Move `internal/core/messaging/template/` → `internal/template/`
  - [ ] Move `internal/core/messaging/context/` files into `internal/template/`
  - [ ] Delete empty `internal/core/` directory tree

- [ ] **Phase 2: Flatten platform layer**
  - [ ] Move `internal/platform/claude/` → `internal/claude/`
  - [ ] Move `internal/platform/database/` → `internal/database/`
  - [ ] Move `internal/platform/storage/` → `internal/storage/`
  - [ ] Merge `internal/platform/state/` into `internal/storage/`
  - [ ] Delete empty `internal/platform/` directory

- [ ] **Phase 3: Flatten infrastructure layer**
  - [ ] Move `internal/infrastructure/constants/` → `internal/constants/`
  - [ ] Move `internal/infrastructure/project/` → `internal/project/`
  - [ ] Delete empty `internal/infrastructure/` directory

- [ ] **Phase 4: Rename for clarity**
  - [ ] Rename `internal/cli/` → `internal/app/`
  - [ ] Update all imports from `cli` to `app`

- [ ] **Phase 5: Update imports**
  - [ ] Update all import paths throughout codebase
  - [ ] Update any documentation referencing old package paths
  - [ ] Run tests to ensure no broken imports

### Benefits
- **Reduced nesting**: Max 2 levels (`internal/claude/api/`) instead of 4
- **Easier navigation**: Related code is grouped by domain
- **Go-idiomatic**: Follows standard Go project organization patterns
- **Clearer purpose**: Package names directly indicate their responsibility
- **Less refactoring**: Mostly just moving directories, minimal code changes