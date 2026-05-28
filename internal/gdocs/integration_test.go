package gdocs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"google.golang.org/api/docs/v1"
)

func TestFullExtractionIntegration(t *testing.T) {
	// 1. Load the mock API response fixture
	fixturePath := filepath.Join("..", "..", "test_fixtures", "doc_api_response.json")
	docJSON, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("Failed to read fixture file %s: %v", fixturePath, err)
	}

	var doc docs.Document
	if err := json.Unmarshal(docJSON, &doc); err != nil {
		t.Fatalf("Failed to unmarshal fixture into docs.Document: %v", err)
	}

	// 2. Run the extraction pipeline (mimicking ProcessDocument without network calls)

	// Step A: Extract Suggestions
	suggestions := ExtractSuggestions(&doc, "")

	// Step B: Extract Metadata
	metadata := ExtractMetadataTable(&doc)

	// Step C: Build Document Structure
	docStructure := BuildDocumentStructure(&doc)

	// Step D: Build Actionable Suggestions
	actionableSuggestions := BuildActionableSuggestions(suggestions, docStructure, metadata)

	// Construct the result object
	// Note: We are mocking comments as empty since the fixture is only for the Docs API response
	actualResult := ProcessingResult{
		DocumentTitle:         doc.Title,
		DocumentID:            doc.DocumentId,
		Metadata:              metadata,
		ActionableSuggestions: actionableSuggestions,
		Comments:              []Comment{},
	}

	// 3. Load the expected output fixture
	expectedPath := filepath.Join("..", "..", "test_fixtures", "expected_output.json")
	expectedJSON, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read fixture file %s: %v", expectedPath, err)
	}

	var expectedResult ProcessingResult
	if err := json.Unmarshal(expectedJSON, &expectedResult); err != nil {
		t.Fatalf("Failed to unmarshal expected output: %v", err)
	}

	// 4. Compare Actual vs Expected
	// We'll verify sections individually to provide better failure messages

	// Verify Metadata
	if !reflect.DeepEqual(actualResult.Metadata, expectedResult.Metadata) {
		t.Errorf("Metadata mismatch.\nGot: %+v\nWant: %+v", actualResult.Metadata, expectedResult.Metadata)
	}

	// Verify Actionable Suggestions
	if len(actualResult.ActionableSuggestions) != len(expectedResult.ActionableSuggestions) {
		t.Fatalf("Actionable suggestions count mismatch. Got %d, want %d",
			len(actualResult.ActionableSuggestions), len(expectedResult.ActionableSuggestions))
	}

	for i := range actualResult.ActionableSuggestions {
		got := actualResult.ActionableSuggestions[i]
		want := expectedResult.ActionableSuggestions[i]

		// Check IDs
		if got.ID != want.ID {
			t.Errorf("Suggestion[%d] ID mismatch. Got %s, want %s", i, got.ID, want.ID)
		}

		// Check Change
		if !reflect.DeepEqual(got.Change, want.Change) {
			t.Errorf("Suggestion[%d] Change mismatch.\nGot: %+v\nWant: %+v", i, got.Change, want.Change)
		}

		// Check Anchor
		if got.Anchor.PrecedingText != want.Anchor.PrecedingText {
			t.Errorf("Suggestion[%d] PrecedingText mismatch.\nGot: %q\nWant: %q", i, got.Anchor.PrecedingText, want.Anchor.PrecedingText)
		}
		if got.Anchor.FollowingText != want.Anchor.FollowingText {
			t.Errorf("Suggestion[%d] FollowingText mismatch.\nGot: %q\nWant: %q", i, got.Anchor.FollowingText, want.Anchor.FollowingText)
		}

		// Check Verification
		if !reflect.DeepEqual(got.Verification, want.Verification) {
			t.Errorf("Suggestion[%d] Verification mismatch.\nGot: %+v\nWant: %+v", i, got.Verification, want.Verification)
		}

		// Check Location
		// We explicitly ignore the table pointer address in comparison, checking content
		if got.Location.Section != want.Location.Section {
			t.Errorf("Suggestion[%d] Location.Section mismatch", i)
		}
		if got.Location.ParentHeading != want.Location.ParentHeading {
			t.Errorf("Suggestion[%d] Location.ParentHeading mismatch", i)
		}
		if got.Location.HeadingLevel != want.Location.HeadingLevel {
			t.Errorf("Suggestion[%d] Location.HeadingLevel mismatch", i)
		}
	}

	// Verify Titles/IDs
	if actualResult.DocumentTitle != expectedResult.DocumentTitle {
		t.Errorf("Title mismatch. Got %s, want %s", actualResult.DocumentTitle, expectedResult.DocumentTitle)
	}
	if actualResult.DocumentID != expectedResult.DocumentID {
		t.Errorf("ID mismatch. Got %s, want %s", actualResult.DocumentID, expectedResult.DocumentID)
	}
}

// TestMetadataSuggestionsSurviveProcessingFlow verifies metadata-table suggestions are preserved
// through extraction, actionable conversion, and location grouping.
func TestMetadataSuggestionsSurviveProcessingFlow(t *testing.T) {
	const (
		metadataSuggestionID = "meta-ins-1"
		bodySuggestionID     = "body-ins-1"
	)

	doc := buildMetadataFlowTestDocument(metadataSuggestionID, bodySuggestionID)

	suggestions := ExtractSuggestions(doc, "")
	if len(suggestions) < 2 {
		t.Fatalf("Expected at least 2 suggestions, got %d", len(suggestions))
	}

	metadata := ExtractMetadataTable(doc)
	if metadata == nil {
		t.Fatal("Expected metadata to be extracted, got nil")
	}

	docStructure := BuildDocumentStructure(doc)
	actionableSuggestions := BuildActionableSuggestions(suggestions, docStructure, metadata)
	groupedSuggestions := GroupActionableSuggestions(actionableSuggestions, docStructure)

	if len(groupedSuggestions) == 0 {
		t.Fatal("Expected grouped suggestions, got none")
	}

	if !hasMetadataLocation(groupedSuggestions) {
		t.Fatal("Expected at least one grouped suggestion location with in_metadata=true")
	}

	if !hasSuggestionID(groupedSuggestions, metadataSuggestionID) {
		t.Fatalf("Expected metadata suggestion ID %q to survive grouping", metadataSuggestionID)
	}

	if !hasSuggestionID(groupedSuggestions, bodySuggestionID) {
		t.Fatalf("Expected non-metadata suggestion ID %q to be present", bodySuggestionID)
	}
}

func buildMetadataFlowTestDocument(metadataSuggestionID, bodySuggestionID string) *docs.Document {
	return &docs.Document{
		Title:      "Metadata Flow Test",
		DocumentId: "metadata-flow-doc",
		Body: &docs.Body{
			Content: []*docs.StructuralElement{
				{
					StartIndex: 1,
					EndIndex:   150,
					Table: &docs.Table{
						TableRows: []*docs.TableRow{
							{
								StartIndex: 2,
								EndIndex:   30,
								TableCells: []*docs.TableCell{
									{
										StartIndex: 3,
										EndIndex:   15,
										Content: []*docs.StructuralElement{
											paragraphWithText(3, 11, "Metadata"),
										},
									},
									{
										StartIndex: 16,
										EndIndex:   29,
										Content: []*docs.StructuralElement{
											paragraphWithText(16, 17, "\n"),
										},
									},
								},
							},
							{
								StartIndex: 31,
								EndIndex:   90,
								TableCells: []*docs.TableCell{
									{
										StartIndex: 32,
										EndIndex:   50,
										Content: []*docs.StructuralElement{
											paragraphWithText(32, 42, "Page title"),
										},
									},
									{
										StartIndex: 51,
										EndIndex:   89,
										Content: []*docs.StructuralElement{
											paragraphWithText(51, 65, "Better title", metadataSuggestionID),
										},
									},
								},
							},
						},
					},
				},
				{
					StartIndex: 151,
					EndIndex:   210,
					Paragraph: &docs.Paragraph{
						Elements: []*docs.ParagraphElement{
							{
								StartIndex: 151,
								EndIndex:   166,
								TextRun: &docs.TextRun{
									Content:               "Body change",
									SuggestedInsertionIds: []string{bodySuggestionID},
								},
							},
						},
					},
				},
			},
		},
	}
}

func paragraphWithText(startIndex, endIndex int64, text string, suggestedInsertionIDs ...string) *docs.StructuralElement {
	textRun := &docs.TextRun{Content: text}
	if len(suggestedInsertionIDs) > 0 {
		textRun.SuggestedInsertionIds = suggestedInsertionIDs
	}

	return &docs.StructuralElement{
		Paragraph: &docs.Paragraph{
			Elements: []*docs.ParagraphElement{
				{
					StartIndex: startIndex,
					EndIndex:   endIndex,
					TextRun:    textRun,
				},
			},
		},
	}
}

func hasMetadataLocation(groups []LocationGroupedSuggestions) bool {
	for _, group := range groups {
		if group.Location.InMetadata {
			return true
		}
	}

	return false
}

func hasSuggestionID(groups []LocationGroupedSuggestions, suggestionID string) bool {
	for _, group := range groups {
		for _, suggestion := range group.Suggestions {
			if suggestion.ID == suggestionID {
				return true
			}
		}
	}

	return false
}
