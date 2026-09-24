package gdocs

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestGroupActionableSuggestions_EmptyInput tests handling of empty input
func TestGroupActionableSuggestions_EmptyInput(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{},
	}

	result := GroupActionableSuggestions([]ActionableSuggestion{}, structure)

	if len(result) != 0 {
		t.Errorf("Expected empty result for empty input, got %d location groups", len(result))
	}
}

// TestGroupActionableSuggestions_SingleSuggestion tests that single suggestions are converted correctly
func TestGroupActionableSuggestions_SingleSuggestion(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "Hello world! ", StartIndex: 0, EndIndex: 13},
			{ID: "text-2", Text: "This is a test.", StartIndex: 13, EndIndex: 28},
		},
	}

	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.1",
			Anchor: SuggestionAnchor{
				PrecedingText: "Hello ",
				FollowingText: "! This",
			},
			Change: SuggestionChange{
				Type:         "replace",
				OriginalText: "world",
				NewText:      "universe",
			},
			Verification: SuggestionVerification{
				TextBeforeChange: "Hello world! This",
				TextAfterChange:  "Hello universe! This",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{
				StartIndex: 6,
				EndIndex:   11,
			},
		},
	}

	result := GroupActionableSuggestions(suggestions, structure)

	if len(result) != 1 {
		t.Fatalf("Expected 1 location group, got %d", len(result))
	}

	if len(result[0].Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion in location group, got %d", len(result[0].Suggestions))
	}

	want := GroupedActionableSuggestion{
		ID: "suggest.1",
		Anchor: SuggestionAnchor{
			PrecedingText: "Hello ",
			FollowingText: "! This",
		},
		Change: SuggestionChange{
			Type:         "replace",
			OriginalText: "world",
			NewText:      "universe",
		},
		Verification: SuggestionVerification{
			TextBeforeChange: "Hello world! This",
			TextAfterChange:  "Hello universe! This",
		},
		Position: struct {
			StartIndex int64 `json:"start_index"`
			EndIndex   int64 `json:"end_index"`
		}{
			StartIndex: 6,
			EndIndex:   11,
		},
		AtomicChanges: []SuggestionChange{
			{
				Type:         "replace",
				OriginalText: "world",
				NewText:      "universe",
			},
		},
		AtomicCount: 1,
	}

	if diff := cmp.Diff(want, result[0].Suggestions[0]); diff != "" {
		t.Errorf("GroupedActionableSuggestion mismatch (-want +got):\n%s", diff)
	}

	wantLocation := SuggestionLocation{
		Section: "Body",
	}
	if diff := cmp.Diff(wantLocation, result[0].Location); diff != "" {
		t.Errorf("Location mismatch (-want +got):\n%s", diff)
	}
}

// TestGroupActionableSuggestions_MultipleUnrelatedSuggestions tests that unrelated suggestions stay separate
func TestGroupActionableSuggestions_MultipleUnrelatedSuggestions(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "First sentence. ", StartIndex: 0, EndIndex: 16},
			{ID: "text-2", Text: "Second sentence.", StartIndex: 16, EndIndex: 32},
		},
	}

	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.1",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "Hello ",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 0, EndIndex: 0},
		},
		{
			ID: "suggest.2",
			Change: SuggestionChange{
				Type:         "delete",
				OriginalText: "sentence",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 23, EndIndex: 31},
		},
	}

	result := GroupActionableSuggestions(suggestions, structure)

	if len(result) != 1 {
		t.Fatalf("Expected 1 location group, got %d", len(result))
	}

	if len(result[0].Suggestions) != 2 {
		t.Fatalf("Expected 2 suggestions in location group, got %d", len(result[0].Suggestions))
	}

	// Verify both are treated as single suggestions
	for i, grouped := range result[0].Suggestions {
		if grouped.AtomicCount != 1 {
			t.Errorf("Suggestion %d: Expected AtomicCount 1, got %d", i, grouped.AtomicCount)
		}
	}
}

// TestGroupActionableSuggestions_GroupedReplacement tests the main use case: grouping insert+delete into replace
func TestGroupActionableSuggestions_GroupedReplacement(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "TEMPLATE HERO\n", StartIndex: 783, EndIndex: 797},
			{ID: "text-2", Text: "Yyour full stack for AI infrastructure", StartIndex: 797, EndIndex: 835},
			{ID: "text-3", Text: " at scale\n", StartIndex: 835, EndIndex: 845},
		},
	}

	// Simulating the example from the user: "Yyour" -> "Build your"
	// The positions are now contiguous: 797->797, 797->798, 798->798
	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.r3eqy31u1iac",
			Anchor: SuggestionAnchor{
				PrecedingText: "TEMPLATE HERO\n",
				FollowingText: "Yyour full stack",
			},
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "Build ",
			},
			Verification: SuggestionVerification{
				TextBeforeChange: "TEMPLATE HERO\nYyour full stack",
				TextAfterChange:  "TEMPLATE HERO\nBuild Yyour full stack",
			},
			Location: SuggestionLocation{
				Section: "Body",
				InTable: true,
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 797, EndIndex: 797},
		},
		{
			ID: "suggest.r3eqy31u1iac",
			Anchor: SuggestionAnchor{
				PrecedingText: "TEMPLATE HERO\nBuild ",
				FollowingText: "your full stack",
			},
			Change: SuggestionChange{
				Type:         "delete",
				OriginalText: "Y",
			},
			Verification: SuggestionVerification{
				TextBeforeChange: "TEMPLATE HERO\nBuild Yyour full stack",
				TextAfterChange:  "TEMPLATE HERO\nBuild your full stack",
			},
			Location: SuggestionLocation{
				Section: "Body",
				InTable: true,
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 797, EndIndex: 798},
		},
		{
			ID: "suggest.r3eqy31u1iac",
			Anchor: SuggestionAnchor{
				PrecedingText: "TEMPLATE HERO\nBuild Y",
				FollowingText: "our full stack",
			},
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "y",
			},
			Verification: SuggestionVerification{
				TextBeforeChange: "TEMPLATE HERO\nBuild Your full stack",
				TextAfterChange:  "TEMPLATE HERO\nBuild Yyour full stack",
			},
			Location: SuggestionLocation{
				Section: "Body",
				InTable: true,
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 798, EndIndex: 798},
		},
	}

	result := GroupActionableSuggestions(suggestions, structure)

	if len(result) != 1 {
		t.Fatalf("Expected 1 location group, got %d", len(result))
	}

	if len(result[0].Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion in location group, got %d", len(result[0].Suggestions))
	}

	grouped := result[0].Suggestions[0]

	// Verify basic properties
	if grouped.ID != "suggest.r3eqy31u1iac" {
		t.Errorf("Expected ID 'suggest.r3eqy31u1iac', got '%s'", grouped.ID)
	}

	if grouped.AtomicCount != 3 {
		t.Errorf("Expected AtomicCount 3, got %d", grouped.AtomicCount)
	}

	if grouped.Change.Type != "replace" {
		t.Errorf("Expected merged change type 'replace', got '%s'", grouped.Change.Type)
	}

	if grouped.Change.OriginalText != "Y" {
		t.Errorf("Expected original text 'Y', got '%s'", grouped.Change.OriginalText)
	}

	expectedNewText := "Build y"
	if grouped.Change.NewText != expectedNewText {
		t.Errorf("Expected new text '%s', got '%s'", expectedNewText, grouped.Change.NewText)
	}

	if !strings.Contains(grouped.Anchor.FollowingText, "your full stack") {
		t.Errorf("Expected following text to contain 'your full stack', got '%s'", grouped.Anchor.FollowingText)
	}

	if grouped.Position.StartIndex != 797 || grouped.Position.EndIndex != 798 {
		t.Errorf("Expected position 797-798, got %d-%d", grouped.Position.StartIndex, grouped.Position.EndIndex)
	}

	if !result[0].Location.InTable {
		t.Error("Expected InTable to be true")
	}
}

// TestGroupActionableSuggestions_PureInsertion tests grouping multiple insertions
func TestGroupActionableSuggestions_PureInsertion(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "Hello world", StartIndex: 0, EndIndex: 11},
		},
	}

	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.insert1",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "beautiful ",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 6, EndIndex: 6},
		},
		{
			ID: "suggest.insert1",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "amazing ",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 6, EndIndex: 6},
		},
	}

	result := GroupActionableSuggestions(suggestions, structure)

	if len(result) != 1 {
		t.Fatalf("Expected 1 location group, got %d", len(result))
	}

	if len(result[0].Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion in location group, got %d", len(result[0].Suggestions))
	}

	wantChange := SuggestionChange{
		Type:         "insert",
		OriginalText: "",
		NewText:      "beautiful amazing ",
	}

	if diff := cmp.Diff(wantChange, result[0].Suggestions[0].Change); diff != "" {
		t.Errorf("Change mismatch (-want +got):\n%s", diff)
	}

	if result[0].Suggestions[0].AtomicCount != 2 {
		t.Errorf("Expected AtomicCount 2, got %d", result[0].Suggestions[0].AtomicCount)
	}
}

// TestGroupActionableSuggestions_PureDeletion tests grouping multiple deletions
func TestGroupActionableSuggestions_PureDeletion(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "Hello beautiful amazing world", StartIndex: 0, EndIndex: 29},
		},
	}

	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.delete1",
			Change: SuggestionChange{
				Type:         "delete",
				OriginalText: "beautiful ",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 6, EndIndex: 16},
		},
		{
			ID: "suggest.delete1",
			Change: SuggestionChange{
				Type:         "delete",
				OriginalText: "amazing ",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 16, EndIndex: 24},
		},
	}

	result := GroupActionableSuggestions(suggestions, structure)

	if len(result) != 1 {
		t.Fatalf("Expected 1 location group, got %d", len(result))
	}

	if len(result[0].Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion in location group, got %d", len(result[0].Suggestions))
	}

	wantChange := SuggestionChange{
		Type:         "delete",
		OriginalText: "beautiful amazing ",
		NewText:      "",
	}

	if diff := cmp.Diff(wantChange, result[0].Suggestions[0].Change); diff != "" {
		t.Errorf("Change mismatch (-want +got):\n%s", diff)
	}

	if result[0].Suggestions[0].AtomicCount != 2 {
		t.Errorf("Expected AtomicCount 2, got %d", result[0].Suggestions[0].AtomicCount)
	}
}

// TestGroupActionableSuggestions_StyleChange tests style-only changes
func TestGroupActionableSuggestions_StyleChange(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "Hello world", StartIndex: 0, EndIndex: 11},
		},
	}

	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.style1",
			Change: SuggestionChange{
				Type:         "style",
				OriginalText: "Hello",
				NewText:      "Hello",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 0, EndIndex: 5},
		},
	}

	result := GroupActionableSuggestions(suggestions, structure)

	if len(result) != 1 {
		t.Fatalf("Expected 1 location group, got %d", len(result))
	}

	if len(result[0].Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion in location group, got %d", len(result[0].Suggestions))
	}

	wantChange := SuggestionChange{
		Type:         "style",
		OriginalText: "Hello",
		NewText:      "Hello",
	}

	if diff := cmp.Diff(wantChange, result[0].Suggestions[0].Change); diff != "" {
		t.Errorf("Change mismatch (-want +got):\n%s", diff)
	}
}

// TestGroupActionableSuggestions_NonContiguous tests that non-contiguous suggestions with same ID stay separate
func TestGroupActionableSuggestions_NonContiguous(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "First paragraph. ", StartIndex: 0, EndIndex: 17},
			{ID: "text-2", Text: "Some filler text here. ", StartIndex: 17, EndIndex: 40},
			{ID: "text-3", Text: "Second paragraph.", StartIndex: 40, EndIndex: 57},
		},
	}

	// Same ID but non-contiguous positions (gap between them)
	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.same",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "A",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 5, EndIndex: 5},
		},
		{
			ID: "suggest.same",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "B",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 45, EndIndex: 45}, // Far from first suggestion
		},
	}

	result := GroupActionableSuggestions(suggestions, structure)

	// Should be in 1 location group but treated as 2 separate suggestions since they're not contiguous
	if len(result) != 1 {
		t.Fatalf("Expected 1 location group, got %d", len(result))
	}

	if len(result[0].Suggestions) != 2 {
		t.Fatalf("Expected 2 suggestions in location group (non-contiguous), got %d", len(result[0].Suggestions))
	}

	for i, grouped := range result[0].Suggestions {
		if grouped.AtomicCount != 1 {
			t.Errorf("Suggestion %d: Expected AtomicCount 1, got %d", i, grouped.AtomicCount)
		}
	}
}

// TestGroupActionableSuggestions_VerificationContent tests that verification texts are constructed correctly
func TestGroupActionableSuggestions_VerificationContent(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "Before ", StartIndex: 0, EndIndex: 7},
			{ID: "text-2", Text: "oldtext", StartIndex: 7, EndIndex: 14},
			{ID: "text-3", Text: " after", StartIndex: 14, EndIndex: 20},
		},
	}

	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.verify",
			Change: SuggestionChange{
				Type:         "delete",
				OriginalText: "old",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 7, EndIndex: 10},
		},
		{
			ID: "suggest.verify",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "new",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 10, EndIndex: 10},
		},
	}

	result := GroupActionableSuggestions(suggestions, structure)

	if len(result) != 1 {
		t.Fatalf("Expected 1 location group, got %d", len(result))
	}

	if len(result[0].Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion in location group, got %d", len(result[0].Suggestions))
	}

	// Verify verification texts contain expected content
	checks := []struct {
		text     string
		field    string
		expected string
	}{
		{result[0].Suggestions[0].Verification.TextBeforeChange, "TextBeforeChange", "old"},
		{result[0].Suggestions[0].Verification.TextAfterChange, "TextAfterChange", "new"},
		{result[0].Suggestions[0].Verification.TextBeforeChange, "TextBeforeChange", "Before"},
		{result[0].Suggestions[0].Verification.TextAfterChange, "TextAfterChange", "after"},
	}

	for _, check := range checks {
		if !containsText(check.text, check.expected) {
			t.Errorf("Expected %s to contain '%s', got '%s'", check.field, check.expected, check.text)
		}
	}
}

// TestGroupActionableSuggestions_SortedOutput tests that output is sorted by position
func TestGroupActionableSuggestions_SortedOutput(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "A B C D E", StartIndex: 0, EndIndex: 9},
		},
	}

	// Provide suggestions out of order
	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.3",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "Z",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 8, EndIndex: 8},
		},
		{
			ID: "suggest.1",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "X",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 2, EndIndex: 2},
		},
		{
			ID: "suggest.2",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "Y",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 5, EndIndex: 5},
		},
	}

	result := GroupActionableSuggestions(suggestions, structure)

	if len(result) != 1 {
		t.Fatalf("Expected 1 location group, got %d", len(result))
	}

	if len(result[0].Suggestions) != 3 {
		t.Fatalf("Expected 3 suggestions in location group, got %d", len(result[0].Suggestions))
	}

	// Verify they're sorted by position
	if result[0].Suggestions[0].Position.StartIndex != 2 {
		t.Errorf("First suggestion should start at 2, got %d", result[0].Suggestions[0].Position.StartIndex)
	}
	if result[0].Suggestions[1].Position.StartIndex != 5 {
		t.Errorf("Second suggestion should start at 5, got %d", result[0].Suggestions[1].Position.StartIndex)
	}
	if result[0].Suggestions[2].Position.StartIndex != 8 {
		t.Errorf("Third suggestion should start at 8, got %d", result[0].Suggestions[2].Position.StartIndex)
	}
}

// TestGroupActionableSuggestions_DifferentLocations tests that suggestions in different locations are grouped separately
func TestGroupActionableSuggestions_DifferentLocations(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "Text in body. ", StartIndex: 0, EndIndex: 14},
			{ID: "text-2", Text: "Text in table.", StartIndex: 100, EndIndex: 114},
			{ID: "text-3", Text: "Text under heading.", StartIndex: 200, EndIndex: 219},
		},
	}

	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.1",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "Body ",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 0, EndIndex: 0},
		},
		{
			ID: "suggest.2",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "Table ",
			},
			Location: SuggestionLocation{
				Section: "Body",
				InTable: true,
				Table: &TableLocation{
					TableID:    "table-1",
					TableIndex: 1,
				},
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 100, EndIndex: 100},
		},
		{
			ID: "suggest.3",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "Heading ",
			},
			Location: SuggestionLocation{
				Section:       "Body",
				ParentHeading: "Introduction",
				HeadingLevel:  1,
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 200, EndIndex: 200},
		},
	}

	result := GroupActionableSuggestions(suggestions, structure)

	// Should have 3 location groups since all are in different locations
	if len(result) != 3 {
		t.Fatalf("Expected 3 location groups, got %d", len(result))
	}

	// Verify each location group has 1 suggestion
	for i, locationGroup := range result {
		if len(locationGroup.Suggestions) != 1 {
			t.Errorf("Location group %d: Expected 1 suggestion, got %d", i, len(locationGroup.Suggestions))
		}
	}

	// Verify the first location is body (no table, no heading)
	if result[0].Location.Section != "Body" || result[0].Location.InTable || result[0].Location.ParentHeading != "" {
		t.Errorf("First location should be plain Body section")
	}

	// Verify the second location is in a table
	if !result[1].Location.InTable || result[1].Location.Table == nil {
		t.Errorf("Second location should be in a table")
	}

	// Verify the third location has a parent heading
	if result[2].Location.ParentHeading != "Introduction" {
		t.Errorf("Third location should be under 'Introduction' heading, got '%s'", result[2].Location.ParentHeading)
	}
}

// TestGroupSuggestionsByID_EmptyInput tests handling of empty input
func TestGroupSuggestionsByID_EmptyInput(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{},
	}

	result := groupSuggestionsByID([]ActionableSuggestion{}, structure)

	if len(result) != 0 {
		t.Errorf("Expected empty result for empty input, got %d suggestions", len(result))
	}
}

// TestGroupSuggestionsByID_SingleSuggestion tests that single suggestions are converted correctly
func TestGroupSuggestionsByID_SingleSuggestion(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "Hello world! ", StartIndex: 0, EndIndex: 13},
			{ID: "text-2", Text: "This is a test.", StartIndex: 13, EndIndex: 28},
		},
	}

	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.1",
			Anchor: SuggestionAnchor{
				PrecedingText: "Hello ",
				FollowingText: "! This",
			},
			Change: SuggestionChange{
				Type:         "replace",
				OriginalText: "world",
				NewText:      "universe",
			},
			Verification: SuggestionVerification{
				TextBeforeChange: "Hello world! This",
				TextAfterChange:  "Hello universe! This",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{
				StartIndex: 6,
				EndIndex:   11,
			},
		},
	}

	result := groupSuggestionsByID(suggestions, structure)

	if len(result) != 1 {
		t.Fatalf("Expected 1 grouped suggestion, got %d", len(result))
	}

	want := GroupedActionableSuggestion{
		ID: "suggest.1",
		Anchor: SuggestionAnchor{
			PrecedingText: "Hello ",
			FollowingText: "! This",
		},
		Change: SuggestionChange{
			Type:         "replace",
			OriginalText: "world",
			NewText:      "universe",
		},
		Verification: SuggestionVerification{
			TextBeforeChange: "Hello world! This",
			TextAfterChange:  "Hello universe! This",
		},
		Position: struct {
			StartIndex int64 `json:"start_index"`
			EndIndex   int64 `json:"end_index"`
		}{
			StartIndex: 6,
			EndIndex:   11,
		},
		AtomicChanges: []SuggestionChange{
			{
				Type:         "replace",
				OriginalText: "world",
				NewText:      "universe",
			},
		},
		AtomicCount: 1,
	}

	if diff := cmp.Diff(want, result[0]); diff != "" {
		t.Errorf("GroupedActionableSuggestion mismatch (-want +got):\n%s", diff)
	}
}

// TestGroupSuggestionsByID_MultipleUnrelatedSuggestions tests that unrelated suggestions stay separate
func TestGroupSuggestionsByID_MultipleUnrelatedSuggestions(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "First sentence. ", StartIndex: 0, EndIndex: 16},
			{ID: "text-2", Text: "Second sentence.", StartIndex: 16, EndIndex: 32},
		},
	}

	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.1",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "Hello ",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 0, EndIndex: 0},
		},
		{
			ID: "suggest.2",
			Change: SuggestionChange{
				Type:         "delete",
				OriginalText: "sentence",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 23, EndIndex: 31},
		},
	}

	result := groupSuggestionsByID(suggestions, structure)

	if len(result) != 2 {
		t.Fatalf("Expected 2 grouped suggestions, got %d", len(result))
	}

	// Verify both are treated as single suggestions
	for i, grouped := range result {
		if grouped.AtomicCount != 1 {
			t.Errorf("Suggestion %d: Expected AtomicCount 1, got %d", i, grouped.AtomicCount)
		}
	}
}

// TestGroupSuggestionsByID_GroupedReplacement tests the main use case: grouping insert+delete into replace
func TestGroupSuggestionsByID_GroupedReplacement(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "TEMPLATE HERO\n", StartIndex: 783, EndIndex: 797},
			{ID: "text-2", Text: "Yyour full stack for AI infrastructure", StartIndex: 797, EndIndex: 835},
			{ID: "text-3", Text: " at scale\n", StartIndex: 835, EndIndex: 845},
		},
	}

	// Simulating the example from the user: "Yyour" -> "Build your"
	// The positions are now contiguous: 797->797, 797->798, 798->798
	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.r3eqy31u1iac",
			Anchor: SuggestionAnchor{
				PrecedingText: "TEMPLATE HERO\n",
				FollowingText: "Yyour full stack",
			},
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "Build ",
			},
			Verification: SuggestionVerification{
				TextBeforeChange: "TEMPLATE HERO\nYyour full stack",
				TextAfterChange:  "TEMPLATE HERO\nBuild Yyour full stack",
			},
			Location: SuggestionLocation{
				Section: "Body",
				InTable: true,
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 797, EndIndex: 797},
		},
		{
			ID: "suggest.r3eqy31u1iac",
			Anchor: SuggestionAnchor{
				PrecedingText: "TEMPLATE HERO\nBuild ",
				FollowingText: "your full stack",
			},
			Change: SuggestionChange{
				Type:         "delete",
				OriginalText: "Y",
			},
			Verification: SuggestionVerification{
				TextBeforeChange: "TEMPLATE HERO\nBuild Yyour full stack",
				TextAfterChange:  "TEMPLATE HERO\nBuild your full stack",
			},
			Location: SuggestionLocation{
				Section: "Body",
				InTable: true,
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 797, EndIndex: 798},
		},
		{
			ID: "suggest.r3eqy31u1iac",
			Anchor: SuggestionAnchor{
				PrecedingText: "TEMPLATE HERO\nBuild Y",
				FollowingText: "our full stack",
			},
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "y",
			},
			Verification: SuggestionVerification{
				TextBeforeChange: "TEMPLATE HERO\nBuild Your full stack",
				TextAfterChange:  "TEMPLATE HERO\nBuild Yyour full stack",
			},
			Location: SuggestionLocation{
				Section: "Body",
				InTable: true,
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 798, EndIndex: 798},
		},
	}

	result := groupSuggestionsByID(suggestions, structure)

	if len(result) != 1 {
		t.Fatalf("Expected 1 grouped suggestion, got %d", len(result))
	}

	// Verify basic properties
	if result[0].ID != "suggest.r3eqy31u1iac" {
		t.Errorf("Expected ID 'suggest.r3eqy31u1iac', got '%s'", result[0].ID)
	}

	if result[0].AtomicCount != 3 {
		t.Errorf("Expected AtomicCount 3, got %d", result[0].AtomicCount)
	}

	if result[0].Change.Type != "replace" {
		t.Errorf("Expected merged change type 'replace', got '%s'", result[0].Change.Type)
	}

	if result[0].Change.OriginalText != "Y" {
		t.Errorf("Expected original text 'Y', got '%s'", result[0].Change.OriginalText)
	}

	expectedNewText := "Build y"
	if result[0].Change.NewText != expectedNewText {
		t.Errorf("Expected new text '%s', got '%s'", expectedNewText, result[0].Change.NewText)
	}

	if !strings.Contains(result[0].Anchor.FollowingText, "your full stack") {
		t.Errorf("Expected following text to contain 'your full stack', got '%s'", result[0].Anchor.FollowingText)
	}

	if result[0].Position.StartIndex != 797 || result[0].Position.EndIndex != 798 {
		t.Errorf("Expected position 797-798, got %d-%d", result[0].Position.StartIndex, result[0].Position.EndIndex)
	}

}

// TestGroupSuggestionsByID_NonContiguous tests that non-contiguous suggestions with same ID stay separate
func TestGroupSuggestionsByID_NonContiguous(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "First paragraph. ", StartIndex: 0, EndIndex: 17},
			{ID: "text-2", Text: "Some filler text here. ", StartIndex: 17, EndIndex: 40},
			{ID: "text-3", Text: "Second paragraph.", StartIndex: 40, EndIndex: 57},
		},
	}

	// Same ID but non-contiguous positions (gap between them)
	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.same",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "A",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 5, EndIndex: 5},
		},
		{
			ID: "suggest.same",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "B",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 45, EndIndex: 45}, // Far from first suggestion
		},
	}

	result := groupSuggestionsByID(suggestions, structure)

	// Should be treated as separate since they're not contiguous
	if len(result) != 2 {
		t.Fatalf("Expected 2 separate suggestions (non-contiguous), got %d", len(result))
	}

	for i, grouped := range result {
		if grouped.AtomicCount != 1 {
			t.Errorf("Suggestion %d: Expected AtomicCount 1, got %d", i, grouped.AtomicCount)
		}
	}
}

// TestGroupSuggestionsByID_SortedOutput tests that output is sorted by position
func TestGroupSuggestionsByID_SortedOutput(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "A B C D E", StartIndex: 0, EndIndex: 9},
		},
	}

	// Provide suggestions out of order
	suggestions := []ActionableSuggestion{
		{
			ID: "suggest.3",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "Z",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 8, EndIndex: 8},
		},
		{
			ID: "suggest.1",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "X",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 2, EndIndex: 2},
		},
		{
			ID: "suggest.2",
			Change: SuggestionChange{
				Type:    "insert",
				NewText: "Y",
			},
			Location: SuggestionLocation{
				Section: "Body",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 5, EndIndex: 5},
		},
	}

	result := groupSuggestionsByID(suggestions, structure)

	if len(result) != 3 {
		t.Fatalf("Expected 3 suggestions, got %d", len(result))
	}

	// Verify they're sorted by position
	if result[0].Position.StartIndex != 2 {
		t.Errorf("First suggestion should start at 2, got %d", result[0].Position.StartIndex)
	}
	if result[1].Position.StartIndex != 5 {
		t.Errorf("Second suggestion should start at 5, got %d", result[1].Position.StartIndex)
	}
	if result[2].Position.StartIndex != 8 {
		t.Errorf("Third suggestion should start at 8, got %d", result[2].Position.StartIndex)
	}
}

// TestResolveGroupedConflicts_NestedConflict tests that large deletions swallow small nested edits
func TestResolveGroupedConflicts_NestedConflict(t *testing.T) {
	// Setup: A large deletion and a tiny replacement inside it
	groups := []LocationGroupedSuggestions{
		{
			Location: SuggestionLocation{Section: "Body"},
			Suggestions: []GroupedActionableSuggestion{
				{
					ID: "suggest.big_delete",
					Change: SuggestionChange{
						Type:         "delete",
						OriginalText: "Ubuntu Core 24 is amazing and should be deleted.",
					},
					Position: struct {
						StartIndex int64 `json:"start_index"`
						EndIndex   int64 `json:"end_index"`
					}{StartIndex: 1600, EndIndex: 1650},
				},
				{
					ID: "suggest.small_fix",
					Change: SuggestionChange{
						Type:         "replace",
						OriginalText: "24",
						NewText:      "2024",
					},
					Position: struct {
						StartIndex int64 `json:"start_index"`
						EndIndex   int64 `json:"end_index"`
					}{StartIndex: 1612, EndIndex: 1614}, // This is INSIDE the big delete
				},
			},
		},
	}

	resolved := ResolveGroupedConflicts(groups)

	if len(resolved[0].Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion after resolution, got %d", len(resolved[0].Suggestions))
	}

	winner := resolved[0].Suggestions[0]
	if winner.ID != "suggest.big_delete" {
		t.Errorf("The large deletion should have won, but got %s", winner.ID)
	}
}

// TestResolveGroupedConflicts_PartialOverlap tests resolution when ranges partially touch
func TestResolveGroupedConflicts_PartialOverlap(t *testing.T) {
	groups := []LocationGroupedSuggestions{
		{
			Location: SuggestionLocation{Section: "Body"},
			Suggestions: []GroupedActionableSuggestion{
				{
					ID:     "suggest.left", // Indices 10 to 30
					Change: SuggestionChange{Type: "delete"},
					Position: struct {
						StartIndex int64 `json:"start_index"`
						EndIndex   int64 `json:"end_index"`
					}{StartIndex: 10, EndIndex: 30},
				},
				{
					ID:     "suggest.right", // Indices 25 to 40 (Overlaps 25-30)
					Change: SuggestionChange{Type: "delete"},
					Position: struct {
						StartIndex int64 `json:"start_index"`
						EndIndex   int64 `json:"end_index"`
					}{StartIndex: 25, EndIndex: 40},
				},
			},
		},
	}

	resolved := ResolveGroupedConflicts(groups)

	// In size-based logic, 10-30 (Size 20) beats 25-40 (Size 15)
	if len(resolved[0].Suggestions) != 1 {
		t.Errorf("Expected 1 suggestion, got %d", len(resolved[0].Suggestions))
	}

	if resolved[0].Suggestions[0].ID != "suggest.left" {
		t.Errorf("Larger range (suggest.left) should have won")
	}
}

// TestAreContiguous tests the contiguity checking function
func TestAreContiguous(t *testing.T) {
	tests := []struct {
		name        string
		suggestions []ActionableSuggestion
		expected    bool
	}{
		{
			name:        "empty slice",
			suggestions: []ActionableSuggestion{},
			expected:    true,
		},
		{
			name: "single suggestion",
			suggestions: []ActionableSuggestion{
				{Position: struct {
					StartIndex int64 `json:"start_index"`
					EndIndex   int64 `json:"end_index"`
				}{StartIndex: 0, EndIndex: 5}},
			},
			expected: true,
		},
		{
			name: "adjacent suggestions",
			suggestions: []ActionableSuggestion{
				{Position: struct {
					StartIndex int64 `json:"start_index"`
					EndIndex   int64 `json:"end_index"`
				}{StartIndex: 0, EndIndex: 5}},
				{Position: struct {
					StartIndex int64 `json:"start_index"`
					EndIndex   int64 `json:"end_index"`
				}{StartIndex: 5, EndIndex: 10}},
			},
			expected: true,
		},
		{
			name: "overlapping suggestions",
			suggestions: []ActionableSuggestion{
				{Position: struct {
					StartIndex int64 `json:"start_index"`
					EndIndex   int64 `json:"end_index"`
				}{StartIndex: 0, EndIndex: 7}},
				{Position: struct {
					StartIndex int64 `json:"start_index"`
					EndIndex   int64 `json:"end_index"`
				}{StartIndex: 5, EndIndex: 10}},
			},
			expected: true,
		},
		{
			name: "gap of 1 (allowed)",
			suggestions: []ActionableSuggestion{
				{Position: struct {
					StartIndex int64 `json:"start_index"`
					EndIndex   int64 `json:"end_index"`
				}{StartIndex: 0, EndIndex: 5}},
				{Position: struct {
					StartIndex int64 `json:"start_index"`
					EndIndex   int64 `json:"end_index"`
				}{StartIndex: 6, EndIndex: 10}},
			},
			expected: true,
		},
		{
			name: "gap of 2 (not contiguous)",
			suggestions: []ActionableSuggestion{
				{Position: struct {
					StartIndex int64 `json:"start_index"`
					EndIndex   int64 `json:"end_index"`
				}{StartIndex: 0, EndIndex: 5}},
				{Position: struct {
					StartIndex int64 `json:"start_index"`
					EndIndex   int64 `json:"end_index"`
				}{StartIndex: 7, EndIndex: 10}},
			},
			expected: false,
		},
		{
			name: "large gap (not contiguous)",
			suggestions: []ActionableSuggestion{
				{Position: struct {
					StartIndex int64 `json:"start_index"`
					EndIndex   int64 `json:"end_index"`
				}{StartIndex: 0, EndIndex: 5}},
				{Position: struct {
					StartIndex int64 `json:"start_index"`
					EndIndex   int64 `json:"end_index"`
				}{StartIndex: 100, EndIndex: 105}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := areContiguous(tt.suggestions)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for test '%s'", tt.expected, result, tt.name)
			}
		})
	}
}

// TestMergeChanges tests the change merging logic
func TestMergeChanges(t *testing.T) {
	tests := []struct {
		name         string
		suggestions  []ActionableSuggestion
		expectedType string
		expectedOrig string
		expectedNew  string
	}{
		{
			name: "pure insertion",
			suggestions: []ActionableSuggestion{
				{Change: SuggestionChange{Type: "insert", NewText: "hello"}},
			},
			expectedType: "insert",
			expectedOrig: "",
			expectedNew:  "hello",
		},
		{
			name: "pure deletion",
			suggestions: []ActionableSuggestion{
				{Change: SuggestionChange{Type: "delete", OriginalText: "goodbye"}},
			},
			expectedType: "delete",
			expectedOrig: "goodbye",
			expectedNew:  "",
		},
		{
			name: "insert then delete (replacement)",
			suggestions: []ActionableSuggestion{
				{Change: SuggestionChange{Type: "insert", NewText: "new"}},
				{Change: SuggestionChange{Type: "delete", OriginalText: "old"}},
			},
			expectedType: "replace",
			expectedOrig: "old",
			expectedNew:  "new",
		},
		{
			name: "delete then insert (replacement)",
			suggestions: []ActionableSuggestion{
				{Change: SuggestionChange{Type: "delete", OriginalText: "old"}},
				{Change: SuggestionChange{Type: "insert", NewText: "new"}},
			},
			expectedType: "replace",
			expectedOrig: "old",
			expectedNew:  "new",
		},
		{
			name: "multiple insertions",
			suggestions: []ActionableSuggestion{
				{Change: SuggestionChange{Type: "insert", NewText: "hello "}},
				{Change: SuggestionChange{Type: "insert", NewText: "world"}},
			},
			expectedType: "insert",
			expectedOrig: "",
			expectedNew:  "hello world",
		},
		{
			name: "multiple deletions",
			suggestions: []ActionableSuggestion{
				{Change: SuggestionChange{Type: "delete", OriginalText: "foo "}},
				{Change: SuggestionChange{Type: "delete", OriginalText: "bar"}},
			},
			expectedType: "delete",
			expectedOrig: "foo bar",
			expectedNew:  "",
		},
		{
			name: "complex replacement (insert, delete, insert)",
			suggestions: []ActionableSuggestion{
				{Change: SuggestionChange{Type: "insert", NewText: "Build "}},
				{Change: SuggestionChange{Type: "delete", OriginalText: "Y"}},
				{Change: SuggestionChange{Type: "insert", NewText: "y"}},
			},
			expectedType: "replace",
			expectedOrig: "Y",
			expectedNew:  "Build y",
		},
		{
			name: "style change only",
			suggestions: []ActionableSuggestion{
				{Change: SuggestionChange{Type: "style", OriginalText: "text", NewText: "text"}},
			},
			expectedType: "style",
			expectedOrig: "text",
			expectedNew:  "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeChanges(tt.suggestions)

			if result.Type != tt.expectedType {
				t.Errorf("Expected type '%s', got '%s'", tt.expectedType, result.Type)
			}
			if result.OriginalText != tt.expectedOrig {
				t.Errorf("Expected original text '%s', got '%s'", tt.expectedOrig, result.OriginalText)
			}
			if result.NewText != tt.expectedNew {
				t.Errorf("Expected new text '%s', got '%s'", tt.expectedNew, result.NewText)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsText(text, substr string) bool {
	return len(text) > 0 && len(substr) > 0 && (text == substr || strings.Contains(text, substr))
}

func TestMergeChanges_PreservesLinkOnInsert(t *testing.T) {
	// A replace built from delete + a linked insert (same ID) must keep new_url.
	suggestions := []ActionableSuggestion{
		{ID: "x", Change: SuggestionChange{Type: "delete", OriginalText: "old"}},
		{ID: "x", Change: SuggestionChange{Type: "insert", NewText: "new", NewURL: "/u"}},
	}
	got := mergeChanges(suggestions)
	if got.Type != "replace" || got.OriginalText != "old" || got.NewText != "new" || got.NewURL != "/u" {
		t.Errorf("merge dropped text/link metadata: %+v", got)
	}
}

func TestMergeChanges_PureLinkKeepsType(t *testing.T) {
	// Two link atoms (a link spanning two runs) with no text change: merged result
	// keeps the link type+URLs and concatenates link_text across the segments.
	suggestions := []ActionableSuggestion{
		{ID: "x", Change: SuggestionChange{Type: "link_change", LinkText: "Con", OldURL: "/a", NewURL: "/b"}},
		{ID: "x", Change: SuggestionChange{Type: "link_change", LinkText: "tact", OldURL: "/a", NewURL: "/b"}},
	}
	got := mergeChanges(suggestions)
	if got.Type != "link_change" || got.LinkText != "Contact" || got.OldURL != "/a" || got.NewURL != "/b" {
		t.Errorf("pure-link merge wrong: %+v", got)
	}
}

func TestGroupActionableSuggestions_LinkChange(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "Contact us", StartIndex: 7, EndIndex: 17},
		},
	}

	suggestions := []ActionableSuggestion{
		{
			ID: "link-1",
			Change: SuggestionChange{
				Type:     "link_change",
				LinkText: "Contact us",
				OldURL:   "/old",
				NewURL:   "/new",
			},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 7, EndIndex: 17},
		},
	}

	groups := GroupActionableSuggestions(suggestions, structure)
	if len(groups) != 1 || len(groups[0].Suggestions) != 1 {
		t.Fatalf("expected 1 group with 1 suggestion, got %#v", groups)
	}
	c := groups[0].Suggestions[0].Change
	if c.Type != "link_change" || c.LinkText != "Contact us" || c.OldURL != "/old" || c.NewURL != "/new" {
		t.Errorf("link fields not preserved through grouping: %+v", c)
	}
}

func TestGroupActionableSuggestions_MultiRunLink(t *testing.T) {
	// A single link suggestion spanning two text runs produces two link atoms with
	// the same ID at contiguous positions, so they get merged. The merged link_text
	// must be the full text (not just the last segment), and the verification strings
	// must include the linked text (a link change does not alter visible text).
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "Before ", StartIndex: 0, EndIndex: 7},
			{ID: "text-2", Text: "Con", StartIndex: 7, EndIndex: 10},
			{ID: "text-3", Text: "tact us", StartIndex: 10, EndIndex: 17},
			{ID: "text-4", Text: " after", StartIndex: 17, EndIndex: 23},
		},
	}

	suggestions := []ActionableSuggestion{
		{
			ID:     "link-1",
			Change: SuggestionChange{Type: "link_change", LinkText: "Con", OldURL: "/old", NewURL: "/new"},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 7, EndIndex: 10},
		},
		{
			ID:     "link-1",
			Change: SuggestionChange{Type: "link_change", LinkText: "tact us", OldURL: "/old", NewURL: "/new"},
			Position: struct {
				StartIndex int64 `json:"start_index"`
				EndIndex   int64 `json:"end_index"`
			}{StartIndex: 10, EndIndex: 17},
		},
	}

	groups := GroupActionableSuggestions(suggestions, structure)
	if len(groups) != 1 || len(groups[0].Suggestions) != 1 {
		t.Fatalf("expected 1 group with 1 merged suggestion, got %#v", groups)
	}
	g := groups[0].Suggestions[0]
	if g.Change.Type != "link_change" || g.Change.LinkText != "Contact us" {
		t.Errorf("merged link_text wrong: %+v", g.Change)
	}
	if g.Change.OldURL != "/old" || g.Change.NewURL != "/new" {
		t.Errorf("merged URLs wrong: %+v", g.Change)
	}
	if !strings.Contains(g.Verification.TextBeforeChange, "Contact us") ||
		!strings.Contains(g.Verification.TextAfterChange, "Contact us") {
		t.Errorf("verification omits linked text: before=%q after=%q",
			g.Verification.TextBeforeChange, g.Verification.TextAfterChange)
	}
}
