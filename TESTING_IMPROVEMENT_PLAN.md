# Testing Framework Improvement Plan

**Generated:** 2025-01-25  
**Status:** In Progress  
**Overall Progress:** 0/47 items completed (0%)

## Executive Summary

This document outlines a comprehensive plan to improve the testing framework for the bumpers project. The audit revealed multiple critical issues beyond just logging problems, including unused test infrastructure, poor assertion patterns, race conditions, and missing test categories.

**Key Issues Identified:**
- 4 different logger initialization systems with duplicate code
- Comprehensive test helpers created but never used (0% adoption)
- No modern assertion framework (1194 basic t.Errorf calls)
- 19 tests can't run in parallel due to global state issues
- Missing benchmarks, fuzz tests, and proper test categorization
- Inconsistent error handling and resource management

---

## Phase 1: Foundation (Immediate - Week 1)

### 1.1 Logger Consolidation
**Priority:** Critical | **Estimated:** 4 hours

- [ ] **Create centralized test logger helper** 
  - **Context:** Currently 3 files have identical `setupTest()` functions with `loggerInitOnce`
  - **Action:** Create `internal/testutil/logger.go` with single `InitTestLogger(t *testing.T)` function
  - **Files to update:** `cmd/bumpers/main_test.go`, `internal/ai/generator_test.go`, `internal/cli/app_test.go`
  - **Notes:** *(Update when completed)*

- [ ] **Add logger initialization to missing test packages**
  - **Context:** 27/30 test files have no logger initialization
  - **Action:** Add `testutil.InitTestLogger(t)` to each test package's setup
  - **Files affected:** All test files except the 3 that already have it
  - **Notes:** *(Update when completed)*

- [ ] **Consolidate logger production code**
  - **Context:** 4 different init methods: `Init()`, `InitLogger()`, `InitTest()`, `InitWithProjectContext()`
  - **Action:** Reduce to 2: production and test variants
  - **File:** `internal/logger/logger.go`
  - **Notes:** *(Update when completed)*

### 1.2 Test Utilities Package
**Priority:** Critical | **Estimated:** 6 hours

- [ ] **Create internal/testutil package structure**
  ```
  internal/testutil/
  ├── logger.go      # Centralized test logger setup
  ├── assertions.go  # Custom assertion helpers
  ├── fixtures.go    # Test data builders
  ├── claude.go      # Claude mock utilities
  ├── filesystem.go  # Filesystem test helpers
  └── database.go    # Database test utilities
  ```
  - **Notes:** *(Update when completed)*

- [ ] **Implement basic assertion helpers**
  - **Context:** Replace verbose error checking patterns
  - **Functions to create:**
    - `AssertNoError(t, err, msg)`
    - `AssertEqual(t, expected, actual)`
    - `AssertContains(t, haystack, needle)`
    - `AssertTrue/False(t, condition, msg)`
  - **Notes:** *(Update when completed)*

- [ ] **Create test data builders**
  - **Context:** Currently hardcoded test strings everywhere
  - **Builders needed:**
    - `NewTestConfig()` with options pattern
    - `NewTestHookEvent()` 
    - `NewTestGenerateRequest()`
  - **Notes:** *(Update when completed)*

### 1.3 Adopt Modern Assertion Framework
**Priority:** High | **Estimated:** 3 hours

- [ ] **Add testify dependency**
  - **Command:** `go get github.com/stretchr/testify`
  - **Context:** Will provide better diff output and reduce boilerplate
  - **Notes:** *(Update when completed)*

- [ ] **Convert high-traffic test files first**
  - **Priority files:** `internal/cli/app_test.go`, `internal/config/config_test.go`
  - **Pattern:** Replace `if x != y { t.Errorf(...) }` with `assert.Equal(t, x, y)`
  - **Benefits:** Better error messages, diff output, less code
  - **Notes:** *(Update when completed)*

### 1.4 Fix Resource Management
**Priority:** High | **Estimated:** 2 hours

- [ ] **Replace defer func() with t.Cleanup()**
  - **Context:** Found 20+ instances of manual defer cleanup
  - **Pattern:** Replace `defer func() { /* cleanup */ }()` with `t.Cleanup(func() { /* cleanup */ })`
  - **Benefits:** Automatic cleanup on test failure, better error handling
  - **Files affected:** All files with defer cleanup patterns
  - **Notes:** *(Update when completed)*

- [ ] **Add goleak for resource leak detection**
  - **Command:** `go get go.uber.org/goleak`
  - **Usage:** Add `defer goleak.VerifyNone(t)` to tests with resources
  - **Focus:** Database connections, file handles, goroutines
  - **Notes:** *(Update when completed)*

---

## Phase 2: Organization (Week 2)

### 2.1 Test Categorization
**Priority:** Medium | **Estimated:** 4 hours

- [ ] **Separate test types by naming convention**
  - **Current:** All tests in `*_test.go`
  - **New structure:**
    - `*_test.go` - Unit tests (fast, no I/O)
    - `*_integration_test.go` - Integration tests (database, filesystem)
    - `*_e2e_test.go` - End-to-end tests (full system)
  - **Files to rename:** TBD based on test analysis
  - **Notes:** *(Update when completed)*

- [ ] **Add build tags for test isolation**
  ```go
  //go:build integration
  //go:build !race
  //go:build e2e
  ```
  - **Context:** Allow running specific test categories
  - **Benefits:** Faster CI, better test isolation
  - **Notes:** *(Update when completed)*

- [ ] **Update justfile with test categories**
  ```bash
  just test-unit           # Unit tests only
  just test-integration    # Integration tests  
  just test-e2e           # End-to-end tests
  just test-all           # All tests
  ```
  - **Notes:** *(Update when completed)*

### 2.2 Use Existing Test Infrastructure
**Priority:** Critical | **Estimated:** 2 hours

- [ ] **Document existing claude test helpers**
  - **Context:** `internal/claude/testing_helpers.go` has 115 lines of unused helpers
  - **Unused functions:**
    - `SetupMockLauncherWithDefaults()` - 0 usages
    - `SetupMockClaudeBinary()` - 0 usages
    - `AssertMockCalled()`, `AssertMockCalledWithPattern()` - 0 usages
  - **Action:** Create usage examples and migrate existing tests
  - **Notes:** *(Update when completed)*

- [ ] **Migrate existing AI tests to use helpers**
  - **Files to update:** 
    - `internal/ai/generator_test.go` (5 tests manually creating mocks)
    - `internal/cli/app_test.go` (tests with AI integration)
  - **Pattern:** Replace `mock := claude.NewMockLauncher()` with `mock := claude.SetupMockLauncherWithDefaults()`
  - **Notes:** *(Update when completed)*

- [ ] **Utilize testdata directory**
  - **Existing:** `testdata/mock-claude.sh` and transcript files
  - **Missing:** Golden files, config fixtures, expected outputs
  - **Action:** Create comprehensive test data structure
  - **Notes:** *(Update when completed)*

### 2.3 Fix Race Conditions
**Priority:** High | **Estimated:** 6 hours

- [ ] **Audit global state dependencies**
  - **Context:** 19 tests disabled parallel execution
  - **Common issues:**
    - Working directory changes
    - Global logger state
    - Environment variables
    - os.Args modifications
  - **Notes:** *(Update when completed)*

- [ ] **Isolate working directory changes**
  - **Pattern:** Use `t.TempDir()` instead of changing global working directory
  - **Files affected:** Tests with `//nolint:paralleltest // changes working directory`
  - **Count:** ~8 test functions
  - **Notes:** *(Update when completed)*

- [ ] **Fix global state modifications**
  - **Logger state:** Use test-specific logger instances
  - **Environment:** Use `t.Setenv()` instead of `os.Setenv()`
  - **Args:** Mock command args instead of modifying `os.Args`
  - **Notes:** *(Update when completed)*

---

## Phase 3: Quality Improvements (Week 3)

### 3.1 Add Missing Test Types
**Priority:** Medium | **Estimated:** 8 hours

- [ ] **Create benchmark tests**
  - **Current:** 0 benchmark functions
  - **Targets for benchmarking:**
    - `internal/matcher` pattern matching
    - `internal/template` rendering
    - `internal/ai` cache lookups
    - Config file parsing
  - **Template:**
    ```go
    func BenchmarkMatcherEvaluate(b *testing.B) {
        // Setup
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            // Operation to benchmark
        }
    }
    ```
  - **Notes:** *(Update when completed)*

- [ ] **Add fuzz tests for input validation**
  - **Targets:**
    - Hook JSON parsing (`internal/hooks`)
    - Config YAML parsing (`internal/config`)
    - Pattern matching (`internal/matcher`)
  - **Template:**
    ```go
    func FuzzHookParser(f *testing.F) {
        f.Add(`{"tool_input": {"command": "test"}}`)
        f.Fuzz(func(t *testing.T, input string) {
            _, _ = hooks.ParseInput(strings.NewReader(input))
        })
    }
    ```
  - **Notes:** *(Update when completed)*

- [ ] **Create example tests for documentation**
  - **Context:** 0 example functions currently
  - **Targets:** Public APIs, complex usage patterns
  - **Benefits:** Executable documentation, usage examples
  - **Notes:** *(Update when completed)*

### 3.2 Improve Table-Driven Tests
**Priority:** Medium | **Estimated:** 4 hours

- [ ] **Convert repetitive tests to table-driven**
  - **Current:** Only 5/30 files use table-driven tests
  - **Candidates:** Tests with similar structure, multiple scenarios
  - **Pattern:**
    ```go
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        // Test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
    ```
  - **Notes:** *(Update when completed)*

- [ ] **Standardize table test structure**
  - **Consistent naming:** `name`, `input`, `expected`, `wantErr`
  - **Subtests:** Always use `t.Run()` for table tests
  - **Error testing:** Consistent error handling patterns
  - **Notes:** *(Update when completed)*

### 3.3 Contract Testing
**Priority:** Low | **Estimated:** 3 hours

- [ ] **Define interface contracts**
  - **Interfaces to test:**
    - `filesystem.FileSystem`
    - `ai.MessageGenerator` 
    - `claude.Launcher`
  - **Contract tests:** Ensure all implementations behave consistently
  - **Notes:** *(Update when completed)*

- [ ] **Test all interface implementations**
  - **FileSystem:** Both `OSFileSystem` and `MemoryFileSystem`
  - **Pattern:** Shared test suite for interface implementations
  - **Notes:** *(Update when completed)*

---

## Phase 4: Advanced Testing (Week 4)

### 4.1 Coverage Analysis
**Priority:** Medium | **Estimated:** 3 hours

- [ ] **Set up per-package coverage reporting**
  - **Current:** 73.3% overall coverage
  - **Goal:** >80% per package, identify coverage gaps
  - **Tool:** `go tool cover -html=coverage.out`
  - **CI Integration:** Coverage gates, trend tracking
  - **Notes:** *(Update when completed)*

- [ ] **Identify untested code paths**
  - **Focus:** Error handling, edge cases
  - **Tools:** Coverage reports, mutation testing
  - **Action:** Add tests for uncovered paths
  - **Notes:** *(Update when completed)*

### 4.2 Property-Based Testing
**Priority:** Low | **Estimated:** 4 hours

- [ ] **Add property-based tests for invariants**
  - **Tool:** `github.com/leanovate/gopter`
  - **Targets:**
    - Config serialization/deserialization
    - Pattern matching consistency
    - Cache behavior properties
  - **Notes:** *(Update when completed)*

### 4.3 Mutation Testing
**Priority:** Low | **Estimated:** 2 hours

- [ ] **Set up mutation testing**
  - **Tool:** `go-mutesting`
  - **Purpose:** Test the quality of tests themselves
  - **Action:** Identify weak test cases
  - **Notes:** *(Update when completed)*

---

## Phase 5: Documentation & Process (Week 5)

### 5.1 Test Documentation
**Priority:** Medium | **Estimated:** 3 hours

- [ ] **Create TESTING.md guide**
  - **Content:**
    - Testing best practices for the project
    - How to use test utilities
    - Test categories and when to use them
    - Mock setup patterns
    - Common pitfalls and solutions
  - **Notes:** *(Update when completed)*

- [ ] **Document test helper usage**
  - **Focus:** The unused `claude/testing_helpers.go` functions
  - **Examples:** Code snippets showing proper usage
  - **Integration:** Link from main README
  - **Notes:** *(Update when completed)*

- [ ] **Add package-level test documentation**
  - **Pattern:** `doc.go` files explaining test approach
  - **Content:** What each package tests, special considerations
  - **Notes:** *(Update when completed)*

### 5.2 CI/CD Integration
**Priority:** Medium | **Estimated:** 2 hours

- [ ] **Separate CI test stages**
  - **Stages:** Unit → Integration → E2E
  - **Parallel:** Run unit tests in parallel
  - **Fast feedback:** Fail fast on unit test failures
  - **Notes:** *(Update when completed)*

- [ ] **Add test quality gates**
  - **Coverage:** Minimum thresholds per package
  - **Performance:** Benchmark regression detection
  - **Quality:** Test duration limits
  - **Notes:** *(Update when completed)*

### 5.3 Developer Experience
**Priority:** Medium | **Estimated:** 2 hours

- [ ] **Create test templates/snippets**
  - **IDE integration:** VS Code snippets for common test patterns
  - **Templates:** Boilerplate for different test types
  - **Notes:** *(Update when completed)*

- [ ] **Add pre-commit hooks for test quality**
  - **Checks:**
    - AI tests use mocks
    - Tests have proper cleanup
    - No hardcoded test data
  - **Tool:** `pre-commit` or custom git hooks
  - **Notes:** *(Update when completed)*

---

## Quick Wins (Do First!)

These items provide immediate value with minimal effort:

- [ ] **✅ HIGH IMPACT: Use existing claude test helpers** (30 min)
  - Replace manual mock creation with `SetupMockLauncherWithDefaults()`
  - **Files:** `internal/ai/generator_test.go`, `internal/cli/app_test.go`

- [ ] **✅ HIGH IMPACT: Consolidate duplicate setupTest functions** (15 min)  
  - Create single `testutil.InitTestLogger()` function
  - **Benefits:** Removes 30+ lines of duplicate code

- [ ] **✅ MEDIUM IMPACT: Add testify assertions** (45 min)
  - Convert verbose error checking to `assert.Equal()`, `assert.NoError()`
  - **Start with:** Most frequently failing tests

- [ ] **✅ MEDIUM IMPACT: Fix resource cleanup** (30 min)
  - Replace `defer func()` with `t.Cleanup()`
  - **Benefits:** Proper cleanup on test failures

- [ ] **✅ LOW IMPACT: Document test best practices** (20 min)
  - Create basic TESTING.md with common patterns
  - **Content:** Mock usage, cleanup patterns, test organization

---

## Metrics Dashboard

Track progress with these metrics:

### Current State (Baseline)
- **Test Files:** 30 total
- **Test Functions:** ~150+ (estimated)
- **Assertion Framework:** Basic `t.Errorf()` only
- **Mock Usage:** Manual creation only
- **Parallel Tests:** ~20% can't run parallel
- **Coverage:** 73.3% overall
- **Benchmarks:** 0
- **Fuzz Tests:** 0
- **Example Tests:** 0

### Target State
- **Assertion Framework:** testify/assert adoption >50%
- **Mock Usage:** Helper functions used in >80% of AI tests
- **Parallel Tests:** >95% can run parallel
- **Coverage:** >80% per package
- **Benchmarks:** 5+ critical path benchmarks
- **Test Categories:** Unit/Integration/E2E separation
- **Documentation:** Comprehensive TESTING.md guide

### Progress Tracking
Update these numbers as work progresses:

- **Items Completed:** 0/47 (0%)
- **Phase 1 (Foundation):** 0/8 items
- **Phase 2 (Organization):** 0/8 items  
- **Phase 3 (Quality):** 0/8 items
- **Phase 4 (Advanced):** 0/5 items
- **Phase 5 (Documentation):** 0/6 items
- **Quick Wins:** 0/5 items

---

## Notes & Updates

*This section should be updated as work progresses with notes about:*
- *Blockers encountered*
- *Lessons learned* 
- *Scope changes*
- *Additional issues discovered*
- *Completion timestamps*

### Completed Items Log
*Format: [Date] Item - Notes*

### Issues Encountered
*Format: [Date] Issue - Resolution*

### Scope Changes  
*Format: [Date] Change - Reason*

---

## References

- **Test Files Analyzed:** All 30 `*_test.go` files
- **Code Patterns Identified:** 1194 basic assertions, 19 non-parallel tests
- **Existing Infrastructure:** `testdata/` directory, `claude/testing_helpers.go`
- **Related Documentation:** `CLAUDE.md`, `justfile` test commands