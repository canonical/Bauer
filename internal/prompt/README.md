# Prompt Generation Package

Generates structured prompts for GitHub Copilot from Google Docs feedback.

## Overview

This package takes processed document feedback and creates markdown files containing:
- Instructions for applying changes
- Suggestions as JSON data
- Vanilla Framework pattern reference

## Quick Start

```go
import "bauer/internal/prompt"

// Create engine (pass true for page-refresh mode)
engine, _ := prompt.NewEngine(false)

// Generate the prompt
result, _ := engine.GeneratePrompt(
    docResult,      // *gdocs.ProcessingResult
    "bauer-output", // output directory
)

// Access result
fmt.Printf("%s (%d locations)\n", result.Filename, result.LocationCount)
```

## Key Features

- **Single prompt output**: All suggestion locations are rendered into one prompt file, `prompt.md`
- **Embedded templates**: Templates bundled in binary via `go:embed`
- **Raw JSON output**: Suggestions embedded as JSON for Copilot to parse
- **Standalone prompt**: The generated prompt is complete and self-contained

## Data Structures

```go
type PromptData struct {
    DocumentTitle   string  // For context
    SuggestedURL    string  // Target file path
    LocationCount   int     // Number of location groups included
    SuggestionsJSON string  // Raw JSON suggestions
}

type PromptResult struct {
    Content       string
    Filename      string
    LocationCount int
}
```

## Templates

Templates in `templates/`:

1. **`copy-docs-instructions.md`**: Main instructions for copy-doc updates
   - Project context (Vanilla Framework, Jinja2)
   - File path resolution rules
   - How to apply changes (insert/delete/replace)
   - JSON schema documentation
   - Error handling guidance

2. **`page-refresh-instructions.md`**: Main instructions for page-refresh updates
    - Same schema-driven workflow with page-refresh-specific execution notes

3. **`vanilla-patterns.md`**: Pattern reference
   - Hero, Equal Heights, Text Spotlight, etc.
   - Usage examples and parameters

4. **`pr-description.md`**: Shared issue/PR body template
    - References instruction/pattern templates
    - Includes suggestion and change-type summary for Copilot execution
    - Used as the base for issue descriptions in parse-and-issue workflow mode


## String Replacement

Simple variable substitution without template engine:

```go
// Replaces {{.Variable}} with value
instructions = replaceVar(instructions, "DocumentTitle", data.DocumentTitle)
instructions = replaceVar(instructions, "SuggestedURL", data.SuggestedURL)
```

No `html/template` needed - just string operations for clarity and simplicity.

## File Path Resolution

Template includes algorithm for URL → file path:

- `ubuntu.com/desktop/features` → `templates/desktop/features.html`
- `ubuntu.com/desktop` → `templates/desktop/index.html`
- Creates necessary directories

## Testing

```bash
go test ./internal/prompt/... -v
```

Tests cover:
- Engine initialization
- Prompt rendering
- File generation
- String replacement

## Usage Notes

- User must run from target repository (CWD = repo)
- User must checkout correct branch before running
- Output directory created automatically if missing
