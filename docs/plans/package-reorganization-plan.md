# Package Reorganization Plan

**Status**: 🔵 Planning  
**Created**: 2025-08-26  
**Last Updated**: 2025-08-26  

## Executive Summary

This plan addresses the growing width of the `internal/` package structure and large file sizes by introducing a hierarchical organization that groups related functionality, splits oversized files, and establishes clearer boundaries between components.

## Current State Analysis

### Package Metrics

| Package | Files | Total Lines | Largest File | Lines | Issues |
|---------|-------|-------------|--------------|-------|--------|
| `cli` | 2 | ~850 | app.go | 708 | Orchestrator doing too much |
| `config` | 2 | ~400 | config.go | 340 | Mixed responsibilities |
| `transcript` | 2 | ~350 | reader.go | 279 | Complex parsing logic |
| `claude/settings` | 3 | ~350 | hooks.go | 255 | Event handling mixed with config |
| `ai` | 1 | ~150 | - | - | Could be part of platform |
| `matcher` | 1 | ~100 | - | - | Core business logic |
| `hooks` | 1 | ~120 | - | - | Event processing |
| `logger` | 1 | ~80 | - | - | Cross-cutting concern |
| `template` | 1 | ~100 | - | - | Part of messaging |
| `context` | 1 | ~80 | - | - | Part of messaging |

### Dependency Analysis

**High Coupling** (imports 5+ packages):
- `cli/app.go` → 9 internal packages
- `config/config.go` → 5 internal packages

**Widely Imported** (imported by 5+ packages):
- `constants` → Used everywhere
- `filesystem` → Used by 6 packages
- `config` → Used by 5 packages

## Reorganization Plan

### Phase 1: Directory Structure Creation

#### 1.1 Core Business Logic (`internal/core/`)

- [ ] **Create base structure**
  ```
  internal/core/
  ├── engine/
  ├── rules/
  └── messaging/
  ```

- [ ] **Engine package** (`internal/core/engine/`)
  - [ ] Create `processor.go` - Main rule processing engine
    - Extract from `cli/app.go` lines 200-500
    - Define `Engine` interface
    - Implement `Process(event HookEvent) (Response, error)`
  - [ ] Create `hooks.go` - Hook event handling
    - Move from `internal/hooks/`
    - Add event type validation
  - [ ] Create `matcher.go` - Pattern matching logic
    - Move from `internal/matcher/`
    - Optimize regex compilation

- [ ] **Rules package** (`internal/core/rules/`)
  - [ ] Create `types.go` - Core rule types
    - Extract from `config/config.go` lines 20-120
    - Define `Rule`, `Command` interfaces
  - [ ] Create `loader.go` - Configuration loading
    - Extract from `config/config.go` lines 150-250
    - Add validation during load
  - [ ] Create `validator.go` - Rule validation
    - Extract from `config/config.go` lines 260-340
    - Add comprehensive validation rules

- [ ] **Messaging package** (`internal/core/messaging/`)
  - [ ] Move `internal/template/` → `internal/core/messaging/templates/`
  - [ ] Move `internal/context/` → `internal/core/messaging/context/`
  - [ ] Create `generator.go` - Message generation
    - Extract from `cli/app.go` lines 550-700
    - Define `MessageGenerator` interface

#### 1.2 Platform Integrations (`internal/platform/`)

- [ ] **Create base structure**
  ```
  internal/platform/
  ├── claude/
  ├── filesystem/
  └── storage/
  ```

- [ ] **Claude platform** (`internal/platform/claude/`)
  - [ ] Move `internal/claude/` → `internal/platform/claude/`
  - [ ] Move `internal/ai/` → `internal/platform/claude/api/`
  - [ ] Create `internal/platform/claude/transcript/`
    - [ ] Create `parser.go` - JSON parsing logic
      - Extract from `transcript/reader.go` lines 50-200
    - [ ] Create `extractor.go` - Content extraction
      - Extract from `transcript/reader.go` lines 201-279
    - [ ] Create `types.go` - Transcript types
      - Extract from `transcript/reader.go` lines 1-49

- [ ] **Move existing platform packages**
  - [ ] Move `internal/filesystem/` → `internal/platform/filesystem/`
  - [ ] Move `internal/storage/` → `internal/platform/storage/`

#### 1.3 Infrastructure (`internal/infrastructure/`)

- [ ] **Create base structure**
  ```
  internal/infrastructure/
  ├── logging/
  ├── config/
  ├── constants/
  └── project/
  ```

- [ ] **Move infrastructure packages**
  - [ ] Move `internal/logger/` → `internal/infrastructure/logging/`
  - [ ] Move `internal/constants/` → `internal/infrastructure/constants/`
  - [ ] Move `internal/project/` → `internal/infrastructure/project/`
  - [ ] Create `internal/infrastructure/config/`
    - [ ] Create `loader.go` - Generic config loading utilities
    - [ ] Create `watcher.go` - Config file watching (future)

#### 1.4 Testing Improvements (`internal/testing/`)

- [ ] **Rename and restructure**
  - [ ] Move `internal/testutil/` → `internal/testing/`
  - [ ] Create subdirectories:
    - [ ] `assertions/` - Test assertions
    - [ ] `fixtures/` - Test data and fixtures
    - [ ] `mocks/` - Mock implementations
    - [ ] `helpers/` - Common test helpers

### Phase 2: File Splitting

#### 2.1 Split `cli/app.go` (708 lines)

- [ ] **Line 1-100**: Keep in `cli/app.go` - Command definitions
- [ ] **Line 101-200**: Move to `cli/commands.go` - Command handlers
- [ ] **Line 201-500**: Move to `core/engine/processor.go` - Processing logic
- [ ] **Line 501-550**: Move to `core/engine/validator.go` - Validation
- [ ] **Line 551-700**: Move to `core/messaging/generator.go` - Message generation
- [ ] **Line 701-708**: Keep in `cli/app.go` - Main entry point

**Notes**: 
- Keep CLI package focused on command-line interface only
- Move all business logic to core packages
- Use dependency injection for core services

#### 2.2 Split `config/config.go` (340 lines)

- [ ] **Line 1-50**: Move to `core/rules/types.go` - Type definitions
- [ためLine 51-120**: Move to `core/rules/command.go` - Command types
- [ ] **Line 121-150**: Move to `infrastructure/config/utils.go` - Utilities
- [ ] **Line 151-250**: Move to `core/rules/loader.go` - Loading logic
- [ ] **Line 251-340**: Move to `core/rules/validator.go` - Validation

**Notes**:
- Separate concerns: types, loading, validation
- Keep config package for backward compatibility (facades)

#### 2.3 Split `transcript/reader.go` (279 lines)

- [ ] **Line 1-49**: Move to `platform/claude/transcript/types.go`
- [ ] **Line 50-150**: Move to `platform/claude/transcript/parser.go`
- [ ] **Line 151-200**: Move to `platform/claude/transcript/decoder.go`
- [ ] **Line 201-279**: Move to `platform/claude/transcript/extractor.go`

**Notes**:
- Create clear separation between parsing and extraction
- Add unit tests for each component

#### 2.4 Split `claude/settings/hooks.go` (255 lines)

- [ ] **Line 1-50**: Move to `platform/claude/settings/types.go`
- [ ] **Line 51-150**: Keep in `platform/claude/settings/hooks.go`
- [ ] **Line 151-255**: Move to `platform/claude/settings/events.go`

**Notes**:
- Separate event definitions from hook management
- Consider event-driven architecture for future

### Phase 3: Interface Definitions

#### 3.1 Core Interfaces

- [ ] **Create `internal/core/interfaces.go`**
  ```go
  type Engine interface {
      Process(event HookEvent) (Response, error)
      Validate(config Config) error
  }
  
  type RuleProcessor interface {
      Match(input string) (bool, *Rule)
      Execute(rule *Rule, context Context) Response
  }
  
  type MessageGenerator interface {
      Generate(template string, context Context) (string, error)
      GenerateAI(prompt string, context Context) (string, error)
  }
  ```

#### 3.2 Platform Interfaces

- [ ] **Create `internal/platform/interfaces.go`**
  ```go
  type Claude interface {
      Launch(args []string) error
      GetSettings() (*Settings, error)
      UpdateHooks(hooks []Hook) error
  }
  
  type Storage interface {
      Get(key string) ([]byte, error)
      Set(key string, value []byte) error
      Delete(key string) error
  }
  ```

#### 3.3 Infrastructure Interfaces

- [ ] **Create `internal/infrastructure/interfaces.go`**
  ```go
  type Logger interface {
      Debug(msg string, fields ...Field)
      Info(msg string, fields ...Field)
      Error(msg string, fields ...Field)
  }
  
  type ConfigLoader interface {
      Load(path string) (*Config, error)
      Watch(path string, callback func(*Config))
  }
  ```

### Phase 4: Import Updates

#### 4.1 Update Import Paths

- [ ] **Create migration script** (`scripts/migrate-imports.sh`)
  - [ ] Map old paths to new paths
  - [ ] Use `gofmt` to update imports
  - [ ] Verify no broken imports

- [ ] **Package-by-package migration**
  - [ ] Update `cmd/bumpers/` imports
  - [ ] Update test file imports
  - [ ] Update integration test imports
  - [ ] Update e2e test imports

#### 4.2 Compatibility Layer

- [ ] **Create facades for backward compatibility**
  - [ ] Keep old package paths as facades (deprecated)
  - [ ] Forward calls to new packages
  - [ ] Add deprecation notices

### Phase 5: Testing & Validation

#### 5.1 Test Updates

- [ ] **Update test imports**
  - [ ] Unit tests
  - [ ] Integration tests
  - [ ] E2E tests

- [ ] **Add new tests**
  - [ ] Test new interfaces
  - [ ] Test split components
  - [ ] Test facades work correctly

#### 5.2 Validation Checklist

- [ ] All tests pass: `just test`
- [ ] Coverage maintained: `just coverage`
- [ ] Linting passes: `just lint`
- [ ] E2E tests pass: `just test-e2e`
- [ ] No circular dependencies
- [ ] Documentation updated

### Phase 6: Documentation

#### 6.1 Update Documentation

- [ ] Update `CLAUDE.md` with new structure
- [ ] Update `TESTING.md` with new test organization
- [ ] Create architecture diagram
- [ ] Update package godoc comments

#### 6.2 Migration Guide

- [ ] Create `docs/MIGRATION.md`
  - [ ] List all moved packages
  - [ ] Provide import update examples
  - [ ] Document breaking changes (if any)

## Benefits Achieved

### Quantitative Improvements

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Packages in internal/ | 15 (flat) | 4 top-level | 73% reduction |
| Largest file (lines) | 708 | <300 | 58% reduction |
| Average file size | ~150 | ~100 | 33% reduction |
| Import depth | 1 level | 3 levels | Better organization |
| Circular dependencies | Unknown | 0 | Eliminated |

### Qualitative Improvements

1. **Clearer Organization**
   - Related functionality grouped together
   - Obvious where new features belong
   - Easier to navigate codebase

2. **Better Testability**
   - Smaller, focused units
   - Clear interfaces for mocking
   - Isolated components

3. **Improved Maintainability**
   - Single responsibility principle
   - Reduced coupling
   - Clear boundaries

4. **Enhanced Extensibility**
   - Plugin-friendly architecture
   - Clear extension points
   - Modular design

## Risk Mitigation

### Potential Risks

1. **Breaking Changes**
   - Mitigation: Facade layer for compatibility
   - Fallback: Git revert if issues

2. **Test Failures**
   - Mitigation: Phase-by-phase migration
   - Fallback: Fix tests incrementally

3. **Import Cycles**
   - Mitigation: Clear hierarchy prevents cycles
   - Fallback: Refactor if detected

## Timeline Estimate

| Phase | Duration | Dependencies |
|-------|----------|--------------|
| Phase 1: Structure | 2 hours | None |
| Phase 2: Splitting | 3 hours | Phase 1 |
| Phase 3: Interfaces | 1 hour | Phase 2 |
| Phase 4: Imports | 2 hours | Phase 3 |
| Phase 5: Testing | 2 hours | Phase 4 |
| Phase 6: Docs | 1 hour | Phase 5 |
| **Total** | **~11 hours** | |

## Progress Tracking

### Overall Progress: 0% Complete

- [ ] Phase 1: Directory Structure (0/15 tasks)
- [ ] Phase 2: File Splitting (0/14 tasks)
- [ ] Phase 3: Interface Definitions (0/3 tasks)
- [ ] Phase 4: Import Updates (0/6 tasks)
- [ ] Phase 5: Testing & Validation (0/8 tasks)
- [ ] Phase 6: Documentation (0/5 tasks)

**Total Tasks**: 0/51 completed

### Notes Section

#### 2025-08-26
- Initial plan created
- Identified 15 packages to reorganize
- Found 4 files >250 lines needing splitting
- Proposed 3-level hierarchy: core, platform, infrastructure

#### Future Considerations
- Consider extracting AI functionality into plugin
- Evaluate need for event-driven architecture
- Consider adding metrics/telemetry package
- Potential for extracting CLI into separate module

## Implementation Commands

```bash
# Phase 1: Create new structure
mkdir -p internal/core/{engine,rules,messaging}
mkdir -p internal/platform/claude/{api,transcript,settings}
mkdir -p internal/infrastructure/{logging,config,constants,project}
mkdir -p internal/testing/{assertions,fixtures,mocks,helpers}

# Phase 2: Move and split files (example)
git mv internal/matcher/matcher.go internal/core/engine/matcher.go
git mv internal/hooks/hooks.go internal/core/engine/hooks.go

# Phase 4: Update imports
find . -name "*.go" -exec sed -i 's|internal/matcher|internal/core/engine|g' {} \;

# Phase 5: Run tests
just test
just coverage
just lint
```

## Sign-off

- [ ] Plan reviewed by team
- [ ] Approach approved
- [ ] Timeline accepted
- [ ] Resources allocated

---

*This is a living document. Update progress as tasks are completed and add notes for any deviations from the plan.*