package prompt

import (
	_ "embed"
	"fmt"
	"strings"

	"bauer/internal/gdocs"
)

//go:embed templates/pr-description.md
var prDescriptionTemplate string

// BuildPRDescription renders a PR description that references the prompt templates
// and summarizes extracted suggestions for Copilot execution.
func BuildPRDescription(result *gdocs.ProcessingResult, usePageRefresh bool) string {
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

	tabIDLine := ""
	if strings.TrimSpace(result.TabID) != "" {
		tabIDLine = fmt.Sprintf("- Tab ID: %s", result.TabID)
	}

	documentURL := buildGoogleDocURL(result.DocumentID, result.TabID)

	insertCount, deleteCount, replaceCount := summarizeSuggestionTypes(result.ActionableSuggestions)

	body := prDescriptionTemplate
	body = replaceVar(body, "DocumentTitle", result.DocumentTitle)
	body = replaceVar(body, "DocumentID", result.DocumentID)
	body = replaceVar(body, "DocumentURL", documentURL)
	body = replaceVar(body, "TabIDLine", tabIDLine)
	body = replaceVar(body, "SuggestedURL", suggestedURL)
	body = replaceVar(body, "Mode", mode)
	body = replaceVar(body, "InstructionsTemplatePath", instructionsTemplatePath)
	body = replaceVar(body, "PatternsTemplatePath", "internal/prompt/templates/vanilla-patterns.md")
	body = replaceVar(body, "LocationCount", fmt.Sprintf("%d", len(result.GroupedSuggestions)))
	body = replaceVar(body, "SuggestionCount", fmt.Sprintf("%d", len(result.ActionableSuggestions)))
	body = replaceVar(body, "InsertCount", fmt.Sprintf("%d", insertCount))
	body = replaceVar(body, "DeleteCount", fmt.Sprintf("%d", deleteCount))
	body = replaceVar(body, "ReplaceCount", fmt.Sprintf("%d", replaceCount))

	return body
}

// BuildIssueDescription renders an issue description for Copilot.
// The full parse JSON is linked in the "Prompt Source" section appended by the workflow.
func BuildIssueDescription(result *gdocs.ProcessingResult, usePageRefresh bool) string {
	body := BuildPRDescription(result, usePageRefresh)

	var b strings.Builder
	b.WriteString(body)
	b.WriteString("\n\n## Parsed Output (Machine Readable)\n\n")
	b.WriteString("The full parse JSON is available via the branch-backed prompt file links in the \"Prompt Source\" section below.\n")

	return b.String()
}

// BuildIssueJSONComments splits parse JSON into comment-safe parts.
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

func buildGoogleDocURL(documentID, tabID string) string {
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return ""
	}

	if strings.HasPrefix(documentID, "https://docs.google.com/document/d/") {
		return documentID
	}

	if strings.TrimSpace(tabID) != "" {
		return fmt.Sprintf("https://docs.google.com/document/d/%s/edit?tab=%s", documentID, tabID)
	}

	return fmt.Sprintf("https://docs.google.com/document/d/%s/edit", documentID)
}
