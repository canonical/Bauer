package prompt

import (
	"testing"

	"bauer/internal/gdocs"
)

func TestBuildPRDescription_IncludesTemplateReferences(t *testing.T) {
	result := &gdocs.ProcessingResult{
		DocumentTitle: "Copy of example.com/page",
		DocumentID:    "doc-123",
		Metadata: &gdocs.MetadataTable{
			SuggestedUrl: "example.com/page",
		},
		GroupedSuggestions: []gdocs.LocationGroupedSuggestions{
			{},
			{},
		},
		ActionableSuggestions: []gdocs.ActionableSuggestion{
			{Change: gdocs.SuggestionChange{Type: "insert"}},
			{Change: gdocs.SuggestionChange{Type: "delete"}},
			{Change: gdocs.SuggestionChange{Type: "replace"}},
		},
	}

	body := BuildPRDescription(result, false)

	expected := []string{
		"@copilot",
		"internal/prompt/templates/copy-docs-instructions.md",
		"internal/prompt/templates/vanilla-patterns.md",
		"Grouped locations: 2",
		"Atomic actionable suggestions: 3",
	}

	for _, e := range expected {
		if !contains(body, e) {
			t.Fatalf("expected PR description to contain %q", e)
		}
	}
}

func TestBuildPRDescription_PageRefreshTemplateReference(t *testing.T) {
	result := &gdocs.ProcessingResult{
		DocumentTitle: "Doc",
		DocumentID:    "doc-abc",
	}

	body := BuildPRDescription(result, true)
	if !contains(body, "internal/prompt/templates/page-refresh-instructions.md") {
		t.Fatalf("expected page refresh template reference in PR description")
	}
}
