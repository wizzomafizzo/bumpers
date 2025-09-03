# Session State - Database Locking Issue Resolution

## Current Status
Working on resolving database locking issues in the bumpers project test suite. Made significant progress but discovered a deeper architectural issue.

## Issues Resolved
1. ✅ **Import alias inconsistency**: Fixed `app_session_test.go:14` to use consistent `ai "github.com/wizzomafizzo/bumpers/internal/platform/claude/api"` import pattern
2. ✅ **SessionManager dependency injection**: Added `SetCacheForTesting(cachePath, projectRoot string)` method to `DefaultSessionManager` struct
3. ✅ **ClearSessionCache refactor**: Modified `ClearSessionCache()` method to use injected cache path when available instead of always creating new cache instances

## Current Problem
Database locking issue persists because **both the cache system and state manager are trying to open the same bbolt database file simultaneously**:

- **Cache system**: `internal/platform/claude/api/cache.go:27` calls `bbolt.Open(dbPath, 0o600, nil)`
- **State manager**: `internal/platform/state/manager.go:18` calls `bbolt.Open(dbPath, 0o600, nil)`
- **Same database path**: Both use `cachePath` from `storage.GetCachePath()`

## Technical Details

### File Locations
- **SessionManager**: `/home/callan/dev/bumpers/internal/cli/session_manager.go`
- **App creation**: `/home/callan/dev/bumpers/internal/cli/app.go:117` (createStateManager function)
- **State manager**: `/home/callan/dev/bumpers/internal/platform/state/manager.go:18`
- **Cache**: `/home/callan/dev/bumpers/internal/platform/claude/api/cache.go:27`

### Key Code Changes Made
1. **SessionManager.go changes**:
   ```go
   // Added method
   func (s *DefaultSessionManager) SetCacheForTesting(cachePath, projectRoot string) {
       s.cachePath = cachePath
       s.aiHelper.projectRoot = projectRoot
   }

   // Modified ClearSessionCache to use injected path
   func (s *DefaultSessionManager) ClearSessionCache(ctx context.Context) error {
       var cachePath string
       var err error
       
       // Use explicit cache path for testing, otherwise use XDG-compliant cache path
       if s.cachePath != "" {
           cachePath = s.cachePath
       } else {
           storageManager := storage.New(s.getFileSystem())
           cachePath, err = storageManager.GetCachePath()
           // ... rest of method
   ```

2. **Test changes**: `TestProcessSessionStartClearsSessionCache` now uses dependency injection approach

### Current Stack Traces Show
- Multiple tests failing with `bbolt.flock` timeouts
- All failing at `NewApp()` -> `createStateManager()` -> `state.NewManager()` -> `bbolt.Open()`
- State manager and cache trying to open same database file concurrently

### Test Success
- `TestProcessSessionStartClearsSessionCache` now passes when run individually
- Other session tests still fail due to state/cache database conflict

## Root Cause Analysis
The issue is architectural: both cache and state systems independently create bbolt database connections to the same file path. bbolt doesn't support concurrent access to the same database file.

## Current Todo Status
1. [completed] Run just lint fix to auto-fix linting issues
2. [completed] Fix missing ai import in app_session_test.go  
3. [completed] Run just test-unit to check unit tests
4. [in_progress] Fix database locking issue - same database used by cache and state manager
5. [pending] Refactor to share single bbolt instance between cache and state manager

## Working Solution Direction
Was about to modify `app.go:117` to make state manager use a different database file:
```go
// Create state manager with separate database file to avoid conflicts with cache
stateDBPath := strings.Replace(cachePath, ".db", "_state.db", 1)
stateManager, err := state.NewManager(stateDBPath, projectID)
```

## Testing Commands Used
- `just lint fix` - passes
- `just test-unit -timeout=10s ./internal/cli -run TestProcessSessionStartClearsSessionCache` - passes individually
- `just test-unit -timeout=30s ./internal/cli -run "TestProcessSessionStart"` - fails with database locking

## Key Files Modified
- `/home/callan/dev/bumpers/internal/cli/session_manager.go` - Added dependency injection
- `/home/callan/dev/bumpers/internal/cli/app_session_test.go` - Updated test to use injection