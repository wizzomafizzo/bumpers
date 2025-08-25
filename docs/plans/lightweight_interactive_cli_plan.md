# Lightweight Interactive CLI Implementation Plan

## Overview

Replace the TUI wizard approach with a minimal, Unix-like interactive CLI using `liner` for input and `fatih/color` for styling. This provides natural text editing, custom key bindings (Tab for AI generation), and simple selection menus without TUI overhead.

## Research Findings

### Rejected Heavy Solutions
- **charmbracelet/huh, bubbles, pterm** - Full TUI frameworks, too heavy
- **survey** - Good but brings more features than needed
- **promptui** - Decent but still more opinionated than needed

### Selected Minimal Stack
- **[peterh/liner](https://github.com/peterh/liner)** - Pure Go readline-like line editor
  - Only depends on `golang.org/x/term`
  - ~1000 lines of code, compiles to ~1MB
  - Custom key bindings for Tab → AI generation
  - Cross-platform arrow keys, Home, End, Backspace
  - NOT a TUI - just line editing

- **[fatih/color](https://github.com/fatih/color)** - Simple ANSI color library
  - Most popular Go color library (7,400+ stars)
  - Zero dependencies
  - Simple API: `color.Cyan("text")`

## Implementation Design

### Core Interactive Components

#### 1. Enhanced Text Input with AI Generation
```go
func AITextInput(prompt string) (string, error) {
    line := liner.NewLiner()
    defer line.Close()
    
    line.SetCtrlCAborts(true) // Esc/Ctrl+C to cancel
    
    // Custom Tab handler for AI pattern generation
    line.SetKeyBindings(map[string]liner.Action{
        "\t": func(l *liner.State) {
            current := l.Line()
            color.Yellow("\nGenerating pattern for: '%s'...\n", current)
            
            // Call AI or pattern generator
            generated := GeneratePattern(current)
            
            color.Green("Generated: %s\n", generated)
            l.SetLine(generated)
            l.Refresh()
        },
    })
    
    // Colored prompt
    coloredPrompt := color.CyanString(prompt)
    return line.Prompt(coloredPrompt)
}
```

#### 2. Quick Select Menu (Letter Shortcuts)
```go
func QuickSelect(prompt string, options map[string]string) (string, error) {
    line := liner.NewLiner()
    defer line.Close()
    
    // Display menu with colored shortcuts
    color.Cyan("%s\n", prompt)
    keys := make([]string, 0, len(options))
    for key, desc := range options {
        color.Yellow("  [%s] %s\n", key, desc)
        keys = append(keys, key)
    }
    
    // Single character input loop
    for {
        input, err := line.Prompt(color.GreenString("> "))
        if err != nil {
            return "", err
        }
        
        if len(input) == 1 {
            if choice, ok := options[strings.ToLower(input)]; ok {
                return choice, nil
            }
        }
        
        color.Red("Invalid choice. Press one of: %s\n", strings.Join(keys, ", "))
    }
}
```

#### 3. Multi-line Text Input (Messages)
```go
func MultiLineInput(prompt string) (string, error) {
    line := liner.NewLiner()
    defer line.Close()
    
    color.Cyan("%s (Press Enter twice when done)\n", prompt)
    
    var lines []string
    emptyLineCount := 0
    
    for {
        input, err := line.Prompt(color.YellowString("  "))
        if err != nil {
            return "", err
        }
        
        if input == "" {
            emptyLineCount++
            if emptyLineCount >= 2 {
                break
            }
        } else {
            emptyLineCount = 0
        }
        
        lines = append(lines, input)
    }
    
    return strings.Join(lines, "\n"), nil
}
```

### Command Structure

#### Rule Pattern Helper
```bash
bumpers rule pattern "rm -rf /"
# Output: Generated pattern: ^rm\s+-rf\s+/.*$

bumpers rule pattern --explain "^go test"
# Output: Pattern breakdown:
#   ^ - Start of line
#   go - Literal "go"
#   \s+ - One or more whitespace
#   test - Literal "test"
```

#### Rule Testing
```bash
bumpers rule test "^rm.*-rf" "rm -rf /tmp/file"
# Output: ✅ Pattern matches!

bumpers rule test "^go test" "go build"
# Output: ❌ Pattern does not match
```

#### Interactive Rule Creation
```bash
bumpers rule add --interactive
```

Flow:
1. **Command Pattern** (with Tab for AI generation)
   ```
   Enter command to block (Tab to generate pattern):
   > rm -rf /    [TAB]
   Generating pattern for: 'rm -rf /'...
   Generated: ^rm\s+-rf\s+/.*$
   > ^rm\s+-rf\s+/.*$
   ```

2. **Tool Selection** (Quick select)
   ```
   Which tools should this rule apply to?
     [b] Bash only (default)
     [a] All tools  
     [e] Edit tools (Write, Edit, MultiEdit)
     [c] Custom regex
   > b
   ```

3. **Help Message** (Multi-line if needed)
   ```
   Helpful message to show when blocked:
   > Use 'just clean' or specify exact files to delete safely
   ```

4. **AI Generation** (Quick select)
   ```
   Generate AI responses?
     [o] off (default)
     [n] once - generate one time
     [s] session - cache for session  
     [a] always - regenerate each time
   > n
   ```

5. **Confirmation**
   ```
   Rule preview:
   Pattern: ^rm\s+-rf\s+/.*$
   Tools: ^Bash$
   Message: Use 'just clean' or specify exact files to delete safely
   Generate: once
   
   Add this rule? [Y/n] y
   ✅ Rule added to bumpers.yml
   ```

#### Non-Interactive Flags
```bash
bumpers rule add \
  --pattern "^rm\s+-rf" \
  --message "Use safer deletion commands" \
  --tools "^Bash$" \
  --generate "off"
```

### Pattern Generation Logic

```go
func GeneratePattern(command string) string {
    // Start with escaped literal
    pattern := regexp.QuoteMeta(command)
    
    // Smart replacements for common cases
    replacements := map[string]string{
        "\\ ":     "\\s+",    // Flexible whitespace
        "\\*":     ".*",      // Keep wildcards  
        "\\?":     ".",       // Keep single char wildcards
        "\\.":     "\\.",     // Keep literal dots
        "\\|":     "|",       // Keep alternation
    }
    
    for old, new := range replacements {
        pattern = strings.ReplaceAll(pattern, old, new)
    }
    
    // Add anchors for exact commands (heuristic)
    if !strings.Contains(pattern, "|") && !strings.HasPrefix(pattern, "^") {
        // Full line match for simple commands
        if isSimpleCommand(command) {
            pattern = "^" + pattern + "$"
        } else {
            // Prefix match for complex commands
            pattern = "^" + pattern
        }
    }
    
    return pattern
}

func isSimpleCommand(cmd string) bool {
    // Simple heuristic: no pipes, redirects, or complex operators
    complex := []string{"|", ">", "<", "&&", "||", ";", "$", "`"}
    for _, op := range complex {
        if strings.Contains(cmd, op) {
            return false
        }
    }
    return true
}
```

### File Structure

```
cmd/bumpers/
├── rule.go              # Rule management commands
├── rule_test.go         # Tests

internal/
├── prompt/              # Interactive input components
│   ├── input.go         # AITextInput, MultiLineInput
│   ├── select.go        # QuickSelect  
│   └── prompt_test.go   # Tests
├── patterns/            # Pattern generation
│   ├── generator.go     # GeneratePattern logic
│   └── generator_test.go # Tests  
└── config/
    └── rules.go         # Rule YAML manipulation (existing)

docs/
├── lightweight-interactive-cli-plan.md  # This document
└── ui-patterns.md       # UI interaction patterns
```

### Dependencies

**Add:**
```go
github.com/peterh/liner v1.2.2
github.com/fatih/color v1.16.0
```

**Remove:**
- Any charmbracelet/huh or TUI dependencies

### Benefits

1. **Lightweight**: ~2MB total dependency size vs 10MB+ for TUI
2. **Fast**: Single-key selections, immediate feedback
3. **Unix-like**: Follows CLI conventions, works in scripts
4. **Natural**: Standard text editing (arrows, home, end)
5. **Customizable**: Tab for AI, colored output, flexible prompts
6. **Testable**: Simple functions, easy to unit test
7. **Maintainable**: ~300 lines vs 1000+ for TUI wizard

### Migration Strategy

1. **Phase 1**: Implement basic interactive add command
2. **Phase 2**: Add pattern generation and testing commands  
3. **Phase 3**: Enhance with AI integration
4. **Phase 4**: Add advanced features (history, templates)

### Testing Strategy

- Unit tests for pattern generation logic
- Interactive tests with liner mocking  
- Integration tests for full rule creation flow
- Cross-platform testing (Windows, Linux, macOS)

## Usage Examples

### Quick Rule Creation
```bash
# Generate pattern from example
bumpers rule pattern "go test ./..."
# Output: ^go\s+test\s+\./\.\.\.$ 

# Test the pattern
bumpers rule test "^go\s+test" "go test ./internal"
# Output: ✅ Matches

# Add rule non-interactively  
bumpers rule add --pattern "^go\s+test" --message "Use 'just test' instead"
```

### Interactive Flow
```bash
bumpers rule add -i

# Natural conversation-style prompts:
Enter command to block (Tab to generate pattern): go test
[Tab pressed]
Generated: ^go\s+test.*$

Tools to apply rule to:
  [b] Bash only
  [a] All tools
> b

Message when blocked: Use 'just test' for TDD guard integration

Generate AI responses:
  [o] off  
  [n] once
> n

✅ Rule added successfully!
```

This approach provides the interactive experience you wanted while staying true to CLI principles and avoiding TUI complexity.