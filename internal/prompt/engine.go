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

//go:embed templates/page-refresh-instructions.md
var pageRefreshInstructionsTemplate string

//go:embed templates/copy-docs-instructions.md
var copyDocsInstructionsTemplate string

//go:embed templates/vanilla-patterns.md
var vanillaPatterns string

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

	// Number of location groups included in the prompt
	LocationCount int

	// Location-grouped suggestions (raw JSON)
	SuggestionsJSON string
}

// PromptResult contains the rendered prompt and metadata
type PromptResult struct {
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

// RenderPrompt generates a complete prompt from the provided data
func (e *Engine) RenderPrompt(data PromptData) (string, error) {
	var buf bytes.Buffer

	// Write instructions with template variable substitution
	// Select template based on page refresh mode
	instructions := copyDocsInstructionsTemplate
	if e.UsePageRefresh {
		instructions = pageRefreshInstructionsTemplate
	}
	instructions = replaceVar(instructions, "DocumentTitle", data.DocumentTitle)
	instructions = replaceVar(instructions, "SuggestedURL", data.SuggestedURL)

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

	return buf.String(), nil
}

// GeneratePrompt renders the prompt for all suggestions and saves it to a file
func (e *Engine) GeneratePrompt(
	result *gdocs.ProcessingResult,
	outputDir string,
) (PromptResult, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return PromptResult{}, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Extract suggested URL from metadata
	suggestedURL := ""
	if result.Metadata != nil {
		suggestedURL = result.Metadata.SuggestedUrl
	}

	// Marshal all location groups to JSON
	suggestionsJSON, err := json.MarshalIndent(result.GroupedSuggestions, "", "  ")
	if err != nil {
		return PromptResult{}, fmt.Errorf("failed to marshal suggestions to JSON: %w", err)
	}

	// Build prompt data
	data := PromptData{
		DocumentTitle:   result.DocumentTitle,
		SuggestedURL:    suggestedURL,
		LocationCount:   len(result.GroupedSuggestions),
		SuggestionsJSON: string(suggestionsJSON),
	}

	// Render the prompt
	content, err := e.RenderPrompt(data)
	if err != nil {
		return PromptResult{}, fmt.Errorf("failed to render prompt: %w", err)
	}

	// Write to file
	outputPath := filepath.Join(outputDir, "prompt.md")
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return PromptResult{}, fmt.Errorf("failed to write prompt to file: %w", err)
	}

	return PromptResult{
		Content:       content,
		Filename:      outputPath,
		LocationCount: len(result.GroupedSuggestions),
	}, nil
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
