# Test Failure Analysis: Shared Cache State Issue

## Problem Summary

CLI package tests are failing intermittently when run as a suite, despite individual tests passing and showing correct mock behavior. Tests expecting "Mock response" from mocked Claude API calls sometimes receive real API responses instead.

## Root Cause

**Shared Global Cache Between Parallel Tests**

All tests share the same global cache file at `~/.local/share/bumpers/cache.db` (via XDG specification). This creates race conditions where one test's cached results interfere with another test's execution.

## Technical Details

### Cache Path Resolution
- Cache path determined by `storage.GetCachePath()` → `xdg.DataHome/bumpers/cache.db`
- All tests use the same global cache file regardless of test isolation
- BBolt database file is shared across all parallel test executions

### AI Generation Flow
1. Test calls `processAIGeneration()` or `processAIGenerationGeneric()`
2. Generator checks cache first (`generator.GenerateMessage()` lines 65-73)
3. If cache entry exists and not expired → returns cached result (bypasses mock)
4. If no cache entry → calls launcher (mock or real) and caches result

### Race Condition Scenario
1. **Test A**: Uses real Claude API, stores result in shared cache
2. **Test B**: Expects mock response, but finds Test A's cached real API result
3. **Test B**: Returns cached real API response instead of "Mock response"
4. **Test B**: Fails assertion expecting "Mock response"

## Affected Tests

Tests with `generate: "always"` mode that expect mocked responses:

- `TestProcessHookWithAIGeneration` (app_test.go:2618)
- `TestProcessHookAIGenerationRequired` (app_test.go:2700)
- `TestProcessHookDangerousCommand` (app_test.go:390)
- `TestProcessUserPromptWithCommandGeneration` (app_test.go:1607)
- `TestProcessSessionStartWithAIGeneration` (app_test.go:2383)

## Evidence

### Debug Log Analysis
```
{"level":"debug","mode":"always","original":"Use 'just test' instead","time":"...","message":"AI generation from fresh Claude call"}
app_test.go:2673: DEBUG: Actual result: "Mock response"
```

This sequence shows:
1. Real Claude API call logged ("AI generation from fresh Claude call")
2. But result is "Mock response" 
3. Indicates cache lookup returned real API result, then mock was called separately

### Test Behavior
- Individual tests pass: Cache is empty or isolated
- Test suite fails: Tests contaminate each other's cache
- Intermittent failures: Depends on test execution order
- Mock setup is correct: All affected tests properly call `SetMockLauncher()`

## Impact

- Tests are flaky and unreliable in CI/CD
- False positive failures hide real issues
- Parallel test execution breaks test isolation
- Mock functionality appears broken when it's actually working correctly

## Solution Requirements

**DO NOT** use environment variable solutions (like setting `XDG_DATA_HOME`) as they are also global and would create race conditions between parallel tests.

**REQUIRED**: Implement cache path dependency injection:
1. Add `cachePath` field to `App` struct
2. Modify AI generation methods to use injected cache path when available
3. Update test setup to provide unique cache paths per test
4. Maintain backward compatibility for production usage

## Files Requiring Changes

- `internal/cli/app.go`: Add cache path injection
- `internal/cli/app_test.go`: Update test setup with unique cache paths
- `internal/platform/claude/api/generator.go`: Accept custom cache path
- `internal/platform/storage/storage.go`: Support cache path override

## Verification

After fix implementation:
1. Run full test suite multiple times - should pass consistently
2. Verify individual tests still pass
3. Confirm mock setup is working properly
4. Check that production functionality remains unchanged