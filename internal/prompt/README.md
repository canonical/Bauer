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

// Create engine
engine, _ := prompt.NewEngine()

// Generate prompts
chunks, _ := engine.GenerateAllChunks(
    result,      // *gdocs.ProcessingResult
    "bauer-output", // output directory
)

// Access results
for _, chunk := range chunks {
    fmt.Printf("Chunk %d: %s (%d locations)\n", 
        chunk.ChunkNumber, chunk.Filename, chunk.LocationCount)
}
```

## Key Features

- **Single prompt output**: All suggestion locations are rendered into one prompt
- **Embedded templates**: Templates bundled in binary via `go:embed`
- **Raw JSON output**: Suggestions embedded as JSON for Copilot to parse
- **Standalone prompt**: The generated prompt is complete and self-contained

## Output Structure

Generated file: `chunk-1-of-1.md`

Each file contains:
1. **Instructions**: Context, file path resolution, how to apply changes
2. **JSON Data**: Array of location-grouped suggestions with schema
3. **Patterns**: Vanilla Framework pattern reference

## Data Structures

```go
type PromptData struct {
    DocumentTitle   string  // For context
    SuggestedURL    string  // Target file path
    ChunkNumber     int     // Current chunk number
    TotalChunks     int     // Total chunks
    LocationCount   int     // Locations in this chunk
    SuggestionsJSON string  // Raw JSON suggestions
}

type ChunkResult struct {
    ChunkNumber   int
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
    - Lists generated chunk files
    - Includes suggestion and change-type summary for Copilot execution
    - Used as the base for issue descriptions in parse-and-issue workflow mode


## String Replacement

Simple variable substitution without template engine:

```go
// Replaces {{.Variable}} with value
instructions = replaceVar(instructions, "DocumentTitle", data.DocumentTitle)
instructions = replaceVar(instructions, "ChunkNumber", "1")
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
- Chunk rendering
- File generation
- String replacement

## Usage Notes

- User must run from target repository (CWD = repo)
- User must checkout correct branch before running
- Output directory created automatically if missing
