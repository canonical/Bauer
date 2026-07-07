package prompt

import (
	"os"
	"testing"

	"bauer/internal/gdocs"
)

func TestNewEngine(t *testing.T) {
	engine, err := NewEngine(false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	if engine == nil {
		t.Fatal("NewEngine() returned nil engine")
	}
}

func TestRenderPrompt(t *testing.T) {
	engine, err := NewEngine(false)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	data := PromptData{
		DocumentTitle:   "Test Document",
		SuggestedURL:    "ubuntu.com/test/page",
		LocationCount:   2,
		SuggestionsJSON: `[{"location":{"section":"Body"},"suggestions":[{"id":"test-1"}]}]`,
	}

	content, err := engine.RenderPrompt(data)
	if err != nil {
		t.Fatalf("RenderPrompt() failed: %v", err)
	}

	// Verify content contains expected sections
	expectedStrings := []string{
		"BAU Copy Update Implementation Instructions",
		"Test Document",
		"ubuntu.com/test/page",
		"Suggestions Data",
		"```json",
		"Vanilla Framework Patterns Reference",
	}

	for _, expected := range expectedStrings {
		if !contains(content, expected) {
			t.Errorf("Rendered content missing expected string: %q", expected)
		}
	}
}

func TestRenderPromptWithPageRefresh(t *testing.T) {
	// Test with PageRefresh enabled
	engine, err := NewEngine(true)
	if err != nil {
		t.Fatalf("Failed to create engine with PageRefresh: %v", err)
	}

	data := PromptData{
		DocumentTitle:   "Test Document",
		SuggestedURL:    "ubuntu.com/test/page",
		LocationCount:   1,
		SuggestionsJSON: `[{"location":{"section":"Body"},"suggestions":[{"id":"test-1"}]}]`,
	}

	content, err := engine.RenderPrompt(data)
	if err != nil {
		t.Fatalf("RenderPrompt() with PageRefresh failed: %v", err)
	}

	// Verify content still contains expected sections
	// (Both templates should have the same structure for now)
	expectedStrings := []string{
		"BAU Page Refresh Implementation Instructions",
		"Test Document",
		"ubuntu.com/test/page",
		"Suggestions Data",
		"```json",
		"Vanilla Framework Patterns Reference",
	}

	for _, expected := range expectedStrings {
		if !contains(content, expected) {
			t.Errorf("Rendered content with PageRefresh missing expected string: %q", expected)
		}
	}

	// Test with PageRefresh disabled
	engineNormal, err := NewEngine(false)
	if err != nil {
		t.Fatalf("Failed to create engine without PageRefresh: %v", err)
	}

	contentNormal, err := engineNormal.RenderPrompt(data)
	if err != nil {
		t.Fatalf("RenderPrompt() without PageRefresh failed: %v", err)
	}

	// For now, both templates are identical, so content should be the same
	// In the future, they may differ
	if len(content) == 0 || len(contentNormal) == 0 {
		t.Error("Rendered content should not be empty")
	}
}

func TestGeneratePrompt(t *testing.T) {
	engine, err := NewEngine(false)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Create temporary output directory
	tmpDir := t.TempDir()

	result := &gdocs.ProcessingResult{
		DocumentTitle: "Test Document",
		DocumentID:    "test-456",
		Metadata: &gdocs.MetadataTable{
			SuggestedUrl: "ubuntu.com/test/page",
		},
		GroupedSuggestions: []gdocs.LocationGroupedSuggestions{
			{
				Location:    gdocs.SuggestionLocation{Section: "Body"},
				Suggestions: makeTestSuggestions(5),
			},
			{
				Location:    gdocs.SuggestionLocation{Section: "Body"},
				Suggestions: makeTestSuggestions(8),
			},
			{
				Location:    gdocs.SuggestionLocation{Section: "Body"},
				Suggestions: makeTestSuggestions(3),
			},
		},
	}

	promptResult, err := engine.GeneratePrompt(
		result,
		tmpDir,
	)
	if err != nil {
		t.Fatalf("GeneratePrompt() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(promptResult.Filename); os.IsNotExist(err) {
		t.Errorf("Prompt file not created: %s", promptResult.Filename)
	}

	// Verify file content is not empty
	content, err := os.ReadFile(promptResult.Filename)
	if err != nil {
		t.Errorf("Failed to read prompt file: %v", err)
	}
	if len(content) == 0 {
		t.Errorf("Prompt file is empty: %s", promptResult.Filename)
	}

	// All location groups are rendered into the single prompt.
	totalOriginal := len(result.GroupedSuggestions)
	if promptResult.LocationCount != totalOriginal {
		t.Errorf("Location count mismatch: prompt=%d, original=%d", promptResult.LocationCount, totalOriginal)
	}

}

func TestReplaceVar(t *testing.T) {
	tests := []struct {
		name     string
		template string
		key      string
		value    string
		expected string
	}{
		{
			name:     "single replacement",
			template: "Hello {{.Name}}!",
			key:      "Name",
			value:    "World",
			expected: "Hello World!",
		},
		{
			name:     "multiple replacements",
			template: "{{.Greeting}} {{.Name}}, {{.Greeting}} again!",
			key:      "Greeting",
			value:    "Hi",
			expected: "Hi {{.Name}}, Hi again!",
		},
		{
			name:     "no replacement",
			template: "Hello World",
			key:      "Name",
			value:    "Test",
			expected: "Hello World",
		},
		{
			name:     "empty value",
			template: "Value: {{.Value}}",
			key:      "Value",
			value:    "",
			expected: "Value: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceVar(tt.template, tt.key, tt.value)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Helper functions

func makeTestSuggestions(count int) []gdocs.GroupedActionableSuggestion {
	suggestions := make([]gdocs.GroupedActionableSuggestion, count)
	for i := range count {
		suggestions[i] = gdocs.GroupedActionableSuggestion{
			ID: string(rune('a' + i)),
			Anchor: gdocs.SuggestionAnchor{
				PrecedingText: "before",
				FollowingText: "after",
			},
			Change: gdocs.SuggestionChange{
				Type:    "insert",
				NewText: "test",
			},
			Verification: gdocs.SuggestionVerification{
				TextBeforeChange: "before after",
				TextAfterChange:  "before test after",
			},
			AtomicCount: 1,
		}
	}
	return suggestions
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
