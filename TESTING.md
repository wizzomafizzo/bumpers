# Testing Guide

This document explains how to write and run tests in the bumpers project, including best practices and available utilities.

## Quick Start

```bash
# Run all tests
just test

# Run only unit tests (fastest)
just test-unit

# Run integration tests (with external dependencies)
just test-integration

# Run end-to-end tests (slowest)
just test-e2e

# Run specific package tests
just test-unit ./internal/matcher
just test-integration ./internal/ai
```

Arguments to `just test*` are passed directly to `go test` so you can use the same flags.

## Test Categories

Tests are organized into three categories using build tags:

### Unit Tests (`*_test.go`)
- Fast tests that don't require external dependencies
- Test individual functions and methods in isolation
- Should use mocks for external dependencies
- Must be parallelizable with `t.Parallel()`

### Integration Tests (`*_integration_test.go`)
- Test components working together
- May use databases, file systems, or other local resources
- Build tag: `//go:build integration`

### End-to-End Tests (`*_e2e_test.go`) 
- Full system tests with real external dependencies
- Test complete user workflows
- Build tag: `//go:build e2e`

## Test Utilities

### Logger Initialization

Always initialize the test logger in your tests:

```go
func TestExample(t *testing.T) {
    testutil.InitTestLogger(t)
    t.Parallel()
    
    // Your test code here
}
```

### Assertions

Use the custom assertion helpers for cleaner tests:

```go
import "github.com/wizzomafizzo/bumpers/internal/testutil"

func TestExample(t *testing.T) {
    result, err := SomeFunction()
    
    testutil.AssertNoError(t, err, "SomeFunction should not error")
    testutil.AssertEqual(t, "expected", result)
    testutil.AssertTrue(t, len(result) > 0, "result should not be empty")
}
```

Available assertion helpers:
- `AssertNoError(t, err, msg)` - Assert no error occurred
- `AssertError(t, err, msg)` - Assert an error occurred  
- `AssertEqual(t, expected, actual)` - Assert values are equal
- `AssertEqualMsg(t, expected, actual, msg)` - Assert with custom message
- `AssertContains(t, haystack, needle)` - Assert string contains substring
- `AssertNotContains(t, haystack, needle)` - Assert string doesn't contain substring
- `AssertTrue(t, condition, msg)` - Assert condition is true
- `AssertFalse(t, condition, msg)` - Assert condition is false
- `AssertNil(t, value)` - Assert value is nil
- `AssertNotNil(t, value)` - Assert value is not nil
- `AssertLen(t, collection, expectedLen)` - Assert collection length
- `AssertEmpty(t, collection)` - Assert collection is empty
- `AssertNotEmpty(t, collection)` - Assert collection is not empty

### Resource Cleanup

Use `t.Cleanup()` for proper resource management:

```go
func TestWithTempFile(t *testing.T) {
    file, err := os.CreateTemp("", "test-*.txt")
    testutil.AssertNoError(t, err, "should create temp file")
    
    t.Cleanup(func() {
        os.Remove(file.Name())
    })
    
    // Use file in test
}
```

### Resource Leak Detection

For tests that create goroutines or other resources:

```go
func TestWithGoroutines(t *testing.T) {
    defer testutil.VerifyNoLeaks(t)
    
    // Test code that might leak goroutines
}
```

## Claude Test Helpers

The project includes comprehensive helpers for testing Claude AI interactions.

### Quick Setup with Defaults

```go
func TestAIGeneration(t *testing.T) {
    mock := claude.SetupMockLauncherWithDefaults()
    // Mock is pre-configured with common response patterns
    
    // Your test code using the mock
    result, err := GenerateWithMock(mock, "go test command")
    
    claude.AssertMockCalled(t, mock, 1)
    claude.AssertMockCalledWithPattern(t, mock, ".*go test.*")
}
```

### Custom Mock Responses

```go
func TestCustomResponses(t *testing.T) {
    responses := map[string]string{
        ".*deploy.*": "Consider running tests before deploying",
        ".*secret.*": "Use environment variables for secrets",
    }
    
    mock := claude.NewMockLauncherWithResponses(responses)
    // Use mock in your tests
}
```

### Mock Binary for Integration Tests

```go
func TestWithMockBinary(t *testing.T) {
    mockPath := claude.SetupMockClaudeBinary(t)
    claude.UseMockClaudePath(t, mockPath)
    
    // Now claude commands will use the mock binary
}
```

### Mock Assertion Helpers

```go
func TestMockAssertions(t *testing.T) {
    mock := claude.SetupMockLauncherWithDefaults()
    
    // Run your code that should call the mock
    GenerateResponse(mock, "some input")
    
    // Verify the mock was used correctly
    claude.AssertMockCalled(t, mock, 1)
    claude.AssertMockCalledWithPattern(t, mock, ".*some input.*")
}

func TestNoMockCalls(t *testing.T) {
    mock := claude.SetupMockLauncherWithDefaults()
    
    // Code that shouldn't call AI
    result := ProcessWithoutAI("input")
    
    claude.AssertMockNotCalled(t, mock)
}
```

## Common Test Patterns

### Table-Driven Tests

Use table-driven tests for multiple similar test cases:

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "valid", false},
        {"empty input", "", true},
        {"invalid format", "bad-format", true},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()

            err := Validate(tc.input)

            if tc.wantErr {
                testutil.AssertError(t, err, "should have error for "+tc.name)
            } else {
                testutil.AssertNoError(t, err, "should not have error for "+tc.name)
            }
        })
    }
}
```

### Benchmark Tests

Add benchmarks for performance-critical code:

```go
func BenchmarkFunction(b *testing.B) {
    // Setup
    input := setupBenchmarkData()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = FunctionToBenchmark(input)
    }
}
```

### Fuzz Tests

Add fuzz tests for input validation:

```go
func FuzzParser(f *testing.F) {
    // Add seed inputs
    f.Add(`{"valid": "json"}`)
    f.Add(`{"key": "value"}`)
    
    f.Fuzz(func(t *testing.T, input string) {
        // Test that parser doesn't panic on any input
        _, _ = ParseJSON(input)
    })
}
```

## Test Data

Store test data in the `testdata/` directory:

```
testdata/
├── mock-claude.sh          # Mock Claude binary for integration tests
├── transcript-*.jsonl      # Sample transcript files
├── config-examples/        # Sample configuration files
└── expected-outputs/       # Expected test outputs
```

Access test data in tests:

```go
func TestWithTestData(t *testing.T) {
    content, err := os.ReadFile("testdata/sample-config.yml")
    testutil.AssertNoError(t, err, "should read test data")
    
    // Use content in test
}
```

## Performance Testing

Run benchmarks:

```bash
# Run all benchmarks
just test-unit -bench=.

# Run specific benchmark
just test-unit -bench=BenchmarkMatcher ./internal/matcher
```

Run fuzz tests:

```bash
# Run fuzz test
just test-unit -fuzz=FuzzParser -fuzztime=30s ./internal/parser
```

## Mutation Testing

Mutation testing evaluates the quality of your tests by introducing small changes (mutations) to the code and checking if tests catch them. High code coverage doesn't guarantee high test quality - mutation testing reveals gaps.

### Setup

Install the mutation testing tool (one-time setup):

```bash
go install github.com/avito-tech/go-mutesting/cmd/go-mutesting@latest
```

### Usage

Run mutation testing on specific packages:

```bash
# Test mutation score for matcher package
go-mutesting --exec-timeout=30 ./internal/matcher

# Test config package
go-mutesting --exec-timeout=30 ./internal/config

# Run with verbose output
go-mutesting --exec-timeout=30 --verbose ./internal/matcher
```

### Interpreting Results

- **Coverage %**: Percentage of code executed by tests
- **Mutation Score %**: Percentage of mutations caught by tests
- **Good mutation scores**: 70%+ indicates strong test quality
- **Low scores**: Tests may miss edge cases or error conditions

**Example results from this project:**
- Matcher package: 90.9% coverage → 69.2% mutation score
- Config package: 89.5% coverage → 78.4% mutation score

**Key insight**: High coverage doesn't guarantee catching bugs. Mutation testing helps identify weak test cases that need improvement.

## Best Practices

1. **Always use `testutil.InitTestLogger(t)` in test functions**
2. **Use `t.Parallel()` for unit tests that can run in parallel** 
3. **Use `t.Cleanup()` instead of defer for resource cleanup**
4. **Prefer table-driven tests for multiple similar test cases**
5. **Use the Claude test helpers instead of manual mock setup**
6. **Add benchmarks for performance-critical paths**
7. **Add fuzz tests for input validation functions**
8. **Keep test data in `testdata/` directory**
9. **Use build tags to separate test categories**
10. **Verify no resource leaks with `testutil.VerifyNoLeaks(t)`**

## TDD Integration

The project uses TDD guard integration through the `just` commands. Always use:

- `just test` instead of `go test`
- `just test-unit` for fast unit tests
- `just test-integration` for integration tests  
- `just test-e2e` for end-to-end tests

This provides better error messages, coverage reporting, and prevents common testing mistakes.