package testutil

// TableTestStandards demonstrates the recommended structure for table-driven tests
// in the bumpers project. Use these patterns for consistency across the codebase.

/*
STANDARD TABLE TEST STRUCTURE:

tests := []struct {
    name    string    // REQUIRED: Test case description
    input   InputType // REQUIRED: Primary input data
    want    WantType  // REQUIRED for return values: Expected result
    wantErr bool      // REQUIRED for error testing: Expected error state
    setup   func()    // OPTIONAL: Test case setup function
    cleanup func()    // OPTIONAL: Test case cleanup function
}{
    {
        name:    "descriptive test case name",
        input:   validInput,
        want:    expectedOutput,
        wantErr: false,
    },
    {
        name:    "error case description",
        input:   invalidInput,
        want:    zeroValue, // Use appropriate zero value
        wantErr: true,
    },
}

FIELD NAMING CONVENTIONS:
- name: Always "name" (not "testName", "description", etc.)
- input: Use "input" for single input values
- For multiple inputs, use descriptive names:
  - command, pattern, config, etc.
- want: Always "want" for expected results (not "expected", "result", etc.)
- wantErr: Always "wantErr" for error expectations (not "expectErr", "hasErr", etc.)

TEST EXECUTION PATTERN:
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        t.Parallel() // Include unless test modifies global state

        if tt.setup != nil {
            tt.setup()
        }
        if tt.cleanup != nil {
            defer tt.cleanup()
        }

        got, err := FunctionUnderTest(tt.input)

        if (err != nil) != tt.wantErr {
            t.Errorf("FunctionUnderTest() error = %v, wantErr %v", err, tt.wantErr)
            return
        }

        if !tt.wantErr && !equal(got, tt.want) {
            t.Errorf("FunctionUnderTest() = %v, want %v", got, tt.want)
        }
    })
}

EXAMPLES OF GOOD NAMING:

// Single input/output
tests := []struct {
    name    string
    input   string
    want    bool
    wantErr bool
}{...}

// Multiple specific inputs
tests := []struct {
    name    string
    pattern string
    command string
    want    bool
}{...}

// Complex test cases
tests := []struct {
    name     string
    config   Config
    input    string
    want     Result
    wantErr  bool
    setup    func()
    cleanup  func()
}{...}
*/
