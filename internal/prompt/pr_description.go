package prompt

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"

	"bauer/internal/gdocs"
)

//go:embed templates/pr-description.md
var prDescriptionTemplate string

// BuildPRDescription renders a PR description that references the prompt templates
// and summarizes extracted suggestions for Copilot execution.
func BuildPRDescription(result *gdocs.ProcessingResult, chunks []ChunkResult, usePageRefresh bool) string {
	if result == nil {
		return "# @copilot Apply BAU Suggestions\n\nNo extraction result was available."
	}

	instructionsTemplatePath := "internal/prompt/templates/copy-docs-instructions.md"
	mode := "copy-docs"
	if usePageRefresh {
		instructionsTemplatePath = "internal/prompt/templates/page-refresh-instructions.md"
		mode = "page-refresh"
	}

	suggestedURL := ""
	if result.Metadata != nil {
		suggestedURL = result.Metadata.SuggestedUrl
	}

	insertCount, deleteCount, replaceCount := summarizeSuggestionTypes(result.ActionableSuggestions)

	chunkList := "- No chunk files were generated."
	if len(chunks) > 0 {
		var lines []string
		for _, chunk := range chunks {
			chunkFile := filepath.ToSlash(chunk.Filename)
			lines = append(lines, fmt.Sprintf("- `%s`", chunkFile))
		}
		chunkList = strings.Join(lines, "\n")
	}

	body := prDescriptionTemplate
	body = replaceVar(body, "DocumentTitle", result.DocumentTitle)
	body = replaceVar(body, "DocumentID", result.DocumentID)
	body = replaceVar(body, "SuggestedURL", suggestedURL)
	body = replaceVar(body, "Mode", mode)
	body = replaceVar(body, "InstructionsTemplatePath", instructionsTemplatePath)
	body = replaceVar(body, "PatternsTemplatePath", "internal/prompt/templates/vanilla-patterns.md")
	body = replaceVar(body, "ChunkFiles", chunkList)
	body = replaceVar(body, "LocationCount", fmt.Sprintf("%d", len(result.GroupedSuggestions)))
	body = replaceVar(body, "SuggestionCount", fmt.Sprintf("%d", len(result.ActionableSuggestions)))
	body = replaceVar(body, "InsertCount", fmt.Sprintf("%d", insertCount))
	body = replaceVar(body, "DeleteCount", fmt.Sprintf("%d", deleteCount))
	body = replaceVar(body, "ReplaceCount", fmt.Sprintf("%d", replaceCount))

	return body
}

// BuildIssueDescription renders an issue description for Copilot.
// Full parse JSON is posted in follow-up issue comments to avoid body size limits.
func BuildIssueDescription(result *gdocs.ProcessingResult, chunks []ChunkResult, usePageRefresh bool) string {
	body := BuildPRDescription(result, chunks, usePageRefresh)

	var b strings.Builder
	b.WriteString(body)
	b.WriteString("\n\n## Parsed Output (Machine Readable)\n\n")
	b.WriteString("Full parse JSON is posted in the issue comments below as chunked `json` blocks.\n")
	b.WriteString("Process chunks in order (`Part 1/N`, `Part 2/N`, ...).\n")

	return b.String()
}

// BuildIssueJSONComments splits parse JSON into comment-safe chunks.
func BuildIssueJSONComments(parseResultJSON string) []string {
	const maxJSONPerComment = 45000
	if parseResultJSON == "" {
		return []string{}
	}

	parts := splitByLimit(parseResultJSON, maxJSONPerComment)
	comments := make([]string, 0, len(parts))
	for i, part := range parts {
		comments = append(comments,
			fmt.Sprintf("### Parsed Output Part %d/%d\n\n```json\n%s\n```", i+1, len(parts), part),
		)
	}

	return comments
}

func splitByLimit(s string, max int) []string {
	if len(s) <= max {
		return []string{s}
	}

	parts := []string{}
	for len(s) > 0 {
		if len(s) <= max {
			parts = append(parts, s)
			break
		}

		cut := max
		if idx := strings.LastIndex(s[:max], "\n"); idx > 0 {
			cut = idx
		}
		parts = append(parts, s[:cut])
		s = s[cut:]
	}

	return parts
}

func summarizeSuggestionTypes(suggestions []gdocs.ActionableSuggestion) (insertCount, deleteCount, replaceCount int) {
	for _, s := range suggestions {
		switch s.Change.Type {
		case "insert":
			insertCount++
		case "delete":
			deleteCount++
		case "replace":
			replaceCount++
		}
	}
	return insertCount, deleteCount, replaceCount
}
