# TODO - Code Quality Improvements

This document tracks minor code quality issues identified during the testing framework audit (2025-08-25).

## 1. Code Duplication in Fixtures (Minor)

**Location**: `internal/testutil/fixtures.go:13-29` and `46-66`

**Issue**: Project root detection logic is duplicated between `LoadTestdataFile()` and `GetTestdataPath()` functions.

**Code Pattern**:
```go
// Duplicated in both functions:
wd, err := os.Getwd()
// ... error handling ...
projectRoot := wd
for {
    if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
        break
    }
    parent := filepath.Dir(projectRoot)
    if parent == projectRoot {
        t.Fatal("Could not find project root (go.mod)")
    }
    projectRoot = parent
}
```

**Solution**: Extract to helper function:
```go
func findProjectRoot(t *testing.T) string {
    // Move common logic here
}
```

**Impact**: Low - just maintenance overhead
**Priority**: Low

## 2. Limited Generic Support in Assertions (Minor)

**Location**: `internal/testutil/assertions.go:89-110`

**Issue**: `AssertLen`, `AssertEmpty`, `AssertNotEmpty` functions only support slices (`[]T`), not maps or strings.

**Current Limitation**:
```go
func AssertLen[T any](t *testing.T, items []T, expectedLen int, msg string)
func AssertEmpty[T any](t *testing.T, items []T, msg string)
func AssertNotEmpty[T any](t *testing.T, items []T, msg string)
```

**Solution**: Use interface constraints to support multiple types:
```go
type Lengthable interface {
    ~[]any | ~map[any]any | ~string
}

func AssertLen[T Lengthable](t *testing.T, items T, expectedLen int, msg string)
```

**Impact**: Low - constrains utility function usage
**Priority**: Low

## 3. Global State in TestUtil Logger (Minor)

**Location**: `internal/testutil/logger.go:12,22`

**Issue**: Uses global `sync.Once` variable and modifies global `log.Logger` state, which could theoretically cause issues in complex test scenarios.

**Current Pattern**:
```go
var loggerInitOnce sync.Once  // Global state

func InitTestLogger(t *testing.T) {
    loggerInitOnce.Do(func() {
        log.Logger = zerolog.New(io.Discard)  // Modifies global
    })
}
```

**Consideration**: This pattern is acceptable for test utilities and solves the import cycle problem effectively. The global state is intentional to prevent race conditions in parallel tests.

**Impact**: Low - acceptable trade-off for test utilities
**Priority**: Very Low (may not need fixing)

---

## Notes

- All issues identified are **minor** and don't affect functionality
- The testing framework implementation is **excellent** overall
- These items are maintenance improvements, not critical fixes
- Consider addressing during regular refactoring cycles