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
    10,          // chunk size (locations per chunk)
    "bauer-output", // output directory
)

// Access results
for _, chunk := range chunks {
    fmt.Printf("Chunk %d: %s (%d locations)\n", 
        chunk.ChunkNumber, chunk.Filename, chunk.LocationCount)
}
```

## Key Features

- **Location-based chunking**: Splits suggestions into manageable chunks by location count
- **Embedded templates**: Templates bundled in binary via `go:embed`
- **Raw JSON output**: Suggestions embedded as JSON for Copilot to parse
- **Single file per chunk**: Each chunk is a complete, standalone prompt

## How Chunking Works

Simple array slicing by location count:

```go
// 23 locations with chunk size 10:
// Chunk 1: locations 0-9   (10 locations)
// Chunk 2: locations 10-19 (10 locations)
// Chunk 3: locations 20-22 (3 locations)

chunks := ChunkLocations(groups, chunkSize)
```

## Output Structure

Generated files: `chunk-{N}-of-{TOTAL}.md`

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

Three embedded templates in `templates/`:

1. **`instructions.md`**: Main instructions for Copilot
   - Project context (Vanilla Framework, Jinja2)
   - File path resolution rules
   - How to apply changes (insert/delete/replace)
   - JSON schema documentation
   - Error handling guidance

2. **`vanilla-patterns.md`**: Pattern reference
   - Hero, Equal Heights, Text Spotlight, etc.
   - Usage examples and parameters


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
- Location-based chunking
- Chunk rendering
- File generation
- String replacement

## Usage Notes

- User must run from target repository (CWD = repo)
- User must checkout correct branch before running
- Chunk size is location count, not suggestion count
- Output directory created automatically if missing
