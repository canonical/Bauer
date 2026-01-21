package prompt

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"bauer/internal/gdocs"
)

//go:embed templates/instructions.md
var instructionsTemplate string

//go:embed templates/vanilla-patterns.md
var vanillaPatterns string

// Engine handles prompt generation for Copilot
type Engine struct{}

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
}

// ChunkResult contains the rendered prompt and metadata for a chunk
type ChunkResult struct {
	ChunkNumber   int
	Content       string
	Filename      string
	LocationCount int
}

// NewEngine creates a new prompt engine
func NewEngine() (*Engine, error) {
	return &Engine{}, nil
}

// ChunkLocations splits location groups into chunks based on location count (not suggestion count)
func ChunkLocations(groups []gdocs.LocationGroupedSuggestions, chunkSize int) [][]gdocs.LocationGroupedSuggestions {
	if chunkSize <= 0 {
		chunkSize = 10
	}

	var chunks [][]gdocs.LocationGroupedSuggestions

	for i := 0; i < len(groups); i += chunkSize {
		end := i + chunkSize
		if end > len(groups) {
			end = len(groups)
		}
		chunks = append(chunks, groups[i:end])
	}

	// Handle case where there are no groups
	if len(chunks) == 0 {
		return [][]gdocs.LocationGroupedSuggestions{{}}
	}

	return chunks
}

// RenderChunk generates a complete prompt for a single chunk
func (e *Engine) RenderChunk(data PromptData) (string, error) {
	var buf bytes.Buffer

	// Write instructions with template variable substitution
	instructions := instructionsTemplate
	instructions = replaceVar(instructions, "DocumentTitle", data.DocumentTitle)
	instructions = replaceVar(instructions, "SuggestedURL", data.SuggestedURL)
	instructions = replaceVar(instructions, "ChunkNumber", fmt.Sprintf("%d", data.ChunkNumber))
	instructions = replaceVar(instructions, "TotalChunks", fmt.Sprintf("%d", data.TotalChunks))

	buf.WriteString(instructions)
	buf.WriteString("\n\n")

	// Write raw JSON suggestions
	buf.WriteString("# Suggestions Data\n\n")
	buf.WriteString("The following is the JSON array of location-grouped suggestions to implement.\n")
	buf.WriteString("Process each location one by one, applying all suggestions for that location before moving to the next.\n\n")
	buf.WriteString("```json\n")
	buf.WriteString(data.SuggestionsJSON)
	buf.WriteString("\n```\n\n")

	// Append Vanilla patterns reference
	buf.WriteString("---\n\n")
	buf.WriteString("# Vanilla Framework Patterns Reference\n\n")
	buf.WriteString(vanillaPatterns)

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
