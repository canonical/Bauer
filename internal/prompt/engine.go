package prompt

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"bauer/internal/gdocs"
	"bauer/internal/source/mapping"
)

//go:embed templates/page-refresh-instructions.md
var pageRefreshInstructionsTemplate string

//go:embed templates/copy-docs-instructions.md
var copyDocsInstructionsTemplate string

//go:embed templates/vanilla-patterns.md
var vanillaPatterns string

//go:embed templates/figma-context.md
var figmaContextTemplate string

// Engine handles prompt generation for Copilot
type Engine struct {
	// UsePageRefresh determines which instruction template to use
	UsePageRefresh bool
}

// PromptData contains all data needed to render a complete prompt
type PromptData struct {
	// Document metadata
	DocumentTitle string

	// Target file from metadata
	SuggestedURL string

	// Chunking information
	ChunkNumber   int
	TotalChunks   int
	LocationCount int

	// Location-grouped suggestions for this chunk (raw JSON)
	SuggestionsJSON string

	// FigmaContextJSON is serialised figma enrichment — empty string when no Figma URL was supplied.
	FigmaContextJSON string

	// FigmaURL is the optional Figma URL for MCP guidance block.
	FigmaURL string
}

// ChunkResult contains the rendered prompt and metadata for a chunk
type ChunkResult struct {
	ChunkNumber   int
	Content       string
	Filename      string
	LocationCount int
}

// NewEngine creates a new prompt engine
func NewEngine(usePageRefresh bool) (*Engine, error) {
	return &Engine{
		UsePageRefresh: usePageRefresh,
	}, nil
}

// ChunkLocations splits location groups into the desired number of chunks
// chunkSize is the desired number of chunks to create, not locations per chunk
func ChunkLocations(groups []gdocs.LocationGroupedSuggestions, desiredChunks int) [][]gdocs.LocationGroupedSuggestions {
	if desiredChunks <= 0 {
		desiredChunks = 1
	}

	totalLocations := len(groups)

	// Handle edge cases
	if totalLocations == 0 {
		return [][]gdocs.LocationGroupedSuggestions{{}}
	}

	// If desired chunks is greater than or equal to total locations,
	// create one chunk per location
	if desiredChunks >= totalLocations {
		var chunks [][]gdocs.LocationGroupedSuggestions
		for _, group := range groups {
			chunks = append(chunks, []gdocs.LocationGroupedSuggestions{group})
		}
		return chunks
	}

	// Calculate locations per chunk (rounded up to ensure all locations are included)
	locationsPerChunk := (totalLocations + desiredChunks - 1) / desiredChunks

	var chunks [][]gdocs.LocationGroupedSuggestions

	for i := 0; i < totalLocations; i += locationsPerChunk {
		end := i + locationsPerChunk
		if end > totalLocations {
			end = totalLocations
		}
		chunks = append(chunks, groups[i:end])
	}

	return chunks
}

// RenderChunk generates a complete prompt for a single chunk
func (e *Engine) RenderChunk(data PromptData) (string, error) {
	var buf bytes.Buffer

	// Write instructions with template variable substitution
	// Select template based on page refresh mode
	instructions := copyDocsInstructionsTemplate
	if e.UsePageRefresh {
		instructions = pageRefreshInstructionsTemplate
	}
	instructions = replaceVar(instructions, "DocumentTitle", data.DocumentTitle)
	instructions = replaceVar(instructions, "SuggestedURL", data.SuggestedURL)
	instructions = replaceVar(instructions, "ChunkNumber", fmt.Sprintf("%d", data.ChunkNumber))
	instructions = replaceVar(instructions, "TotalChunks", fmt.Sprintf("%d", data.TotalChunks))

	buf.WriteString(instructions)
	buf.WriteString("\n\n")

	// Append Vanilla patterns reference (before the data)
	buf.WriteString("---\n\n")
	buf.WriteString(vanillaPatterns)
	buf.WriteString("\n\n")

	// Write raw JSON suggestions (last, as the data to process)
	buf.WriteString("---\n\n")
	buf.WriteString("# Suggestions Data\n\n")
	buf.WriteString("The following is the JSON array of location-grouped suggestions to implement.\n")
	buf.WriteString("Process each location one by one, applying all suggestions for that location before moving to the next.\n\n")
	buf.WriteString("```json\n")
	buf.WriteString(data.SuggestionsJSON)
	buf.WriteString("\n```\n")

	// Append figma context section when design data is present
	if data.FigmaContextJSON != "" {
		var ctx figmaChunkContext
		if err := json.Unmarshal([]byte(data.FigmaContextJSON), &ctx); err != nil {
			return "", fmt.Errorf("parsing figma context JSON: %w", err)
		}
		ctx.FigmaURL = data.FigmaURL
		tmpl, err := template.New("figma-context").Parse(figmaContextTemplate)
		if err != nil {
			return "", fmt.Errorf("parsing figma context template: %w", err)
		}
		buf.WriteString("\n\n---\n\n")
		if err := tmpl.Execute(&buf, ctx); err != nil {
			return "", fmt.Errorf("rendering figma context: %w", err)
		}
	}

	return buf.String(), nil
}

// GenerateAllChunks creates prompts for all chunks and saves them to files
func (e *Engine) GenerateAllChunks(
	result *gdocs.ProcessingResult,
	chunkSize int,
	outputDir string,
) ([]ChunkResult, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Chunk the location groups (simple slicing)
	chunks := ChunkLocations(result.GroupedSuggestions, chunkSize)
	totalChunks := len(chunks)

	// Extract suggested URL from metadata
	suggestedURL := ""
	if result.Metadata != nil {
		suggestedURL = result.Metadata.SuggestedUrl
	}

	var results []ChunkResult

	// Generate prompt for each chunk
	for i, chunk := range chunks {
		chunkNum := i + 1

		// Marshal chunk to JSON
		chunkJSON, err := json.MarshalIndent(chunk, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal chunk %d to JSON: %w", chunkNum, err)
		}

		// Build prompt data
		data := PromptData{
			DocumentTitle:   result.DocumentTitle,
			SuggestedURL:    suggestedURL,
			ChunkNumber:     chunkNum,
			TotalChunks:     totalChunks,
			LocationCount:   len(chunk),
			SuggestionsJSON: string(chunkJSON),
		}

		// Render the chunk
		content, err := e.RenderChunk(data)
		if err != nil {
			return nil, fmt.Errorf("failed to render chunk %d: %w", chunkNum, err)
		}

		// Generate filename
		filename := fmt.Sprintf("chunk-%d-of-%d.md", chunkNum, totalChunks)
		filepath := filepath.Join(outputDir, filename)

		// Write to file
		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write chunk %d to file: %w", chunkNum, err)
		}

		results = append(results, ChunkResult{
			ChunkNumber:   chunkNum,
			Content:       content,
			Filename:      filepath,
			LocationCount: len(chunk),
		})
	}

	return results, nil
}

// figmaChunkContext is the data structure serialized into FigmaContextJSON.
// FigmaURL is NOT serialized; it is set at render time from PromptData.FigmaURL
// to power the optional MCP guidance block in the figma-context template.
type figmaChunkContext struct {
	Anchors     []mapping.DesignAnchorRef  `json:"anchors,omitempty"`
	Screenshots []string                   `json:"screenshots,omitempty"`
	Comments    []mapping.DesignCommentRef `json:"comments,omitempty"`
	FigmaURL    string                     `json:"-"`
}

// GenerateChunksFromResolved creates one PromptData per batch of resolved chunks.
// chunkSize controls how many ResolvedChunks are combined into one prompt.
// When FigmaContextJSON is non-empty, the rendered prompt includes the figma-context section.
func (e *Engine) GenerateChunksFromResolved(
	docTitle, suggestedURL, figmaURL string,
	chunks []mapping.ResolvedChunk,
	chunkSize int,
) ([]PromptData, error) {
	batches := batchResolvedChunks(chunks, chunkSize)
	result := make([]PromptData, len(batches))

	for i, batch := range batches {
		var locations []gdocs.LocationGroupedSuggestions
		for _, rc := range batch {
			locations = append(locations, rc.Locations...)
		}

		suggestionsJSON, err := json.Marshal(locations)
		if err != nil {
			return nil, fmt.Errorf("marshaling suggestions for chunk %d: %w", i+1, err)
		}

		figmaJSON, err := buildFigmaContextJSON(batch)
		if err != nil {
			return nil, fmt.Errorf("building figma context for chunk %d: %w", i+1, err)
		}

		result[i] = PromptData{
			DocumentTitle:    docTitle,
			SuggestedURL:     suggestedURL,
			ChunkNumber:      i + 1,
			TotalChunks:      len(batches),
			LocationCount:    len(locations),
			SuggestionsJSON:  string(suggestionsJSON),
			FigmaContextJSON: figmaJSON,
			FigmaURL:         figmaURL,
		}
	}
	return result, nil
}

func buildFigmaContextJSON(batch []mapping.ResolvedChunk) (string, error) {
	ctx := figmaChunkContext{}
	for _, rc := range batch {
		ctx.Anchors = append(ctx.Anchors, rc.DesignAnchors...)
		ctx.Screenshots = append(ctx.Screenshots, rc.ScreenshotPaths...)
		ctx.Comments = append(ctx.Comments, rc.Comments...)
	}

	if len(ctx.Anchors) == 0 && len(ctx.Screenshots) == 0 && len(ctx.Comments) == 0 {
		return "", nil
	}

	b, err := json.Marshal(ctx)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func batchResolvedChunks(chunks []mapping.ResolvedChunk, size int) [][]mapping.ResolvedChunk {
	if size <= 0 {
		size = 1
	}
	var batches [][]mapping.ResolvedChunk
	for i := 0; i < len(chunks); i += size {
		end := i + size
		if end > len(chunks) {
			end = len(chunks)
		}
		batches = append(batches, chunks[i:end])
	}
	return batches
}

// replaceVar is a simple string replacement helper for template variables
func replaceVar(template, key, value string) string {
	placeholder := "{{." + key + "}}"
	var result bytes.Buffer

	for {
		idx := indexOf(template, placeholder)
		if idx == -1 {
			result.WriteString(template)
			break
		}
		result.WriteString(template[:idx])
		result.WriteString(value)
		template = template[idx+len(placeholder):]
	}
	return result.String()
}

// indexOf finds the index of a substring
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
// RenderChunksFromResolved generates figma-aware prompt files from pre-resolved chunks.
// It is used when --figma-url is supplied so that Figma design context is embedded in
// each prompt. outputDir is created if it does not exist.
// The returned ChunkResults contain Filenames suitable for agent execution.
func (e *Engine) RenderChunksFromResolved(
        docTitle, suggestedURL, figmaURL string,
        chunks []mapping.ResolvedChunk,
        chunkSize int,
        outputDir string,
) ([]ChunkResult, error) {
        if err := os.MkdirAll(outputDir, 0755); err != nil {
                return nil, fmt.Errorf("creating output directory %q: %w", outputDir, err)
        }

        promptDatas, err := e.GenerateChunksFromResolved(docTitle, suggestedURL, figmaURL, chunks, chunkSize)
        if err != nil {
                return nil, err
        }

        results := make([]ChunkResult, len(promptDatas))
        for i, pd := range promptDatas {
                content, err := e.RenderChunk(pd)
                if err != nil {
                        return nil, fmt.Errorf("rendering chunk %d: %w", i+1, err)
                }
                fname := fmt.Sprintf("chunk-%d-of-%d.md", pd.ChunkNumber, pd.TotalChunks)
                fpath := filepath.Join(outputDir, fname)
                if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
                        return nil, fmt.Errorf("writing chunk %d to file: %w", i+1, err)
                }
                results[i] = ChunkResult{
                        ChunkNumber:   pd.ChunkNumber,
                        Content:       content,
                        Filename:      fpath,
                        LocationCount: pd.LocationCount,
                }
        }
        return results, nil
}