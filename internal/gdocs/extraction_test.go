package gdocs

import (
	"testing"

	"google.golang.org/api/docs/v1"
)

// Helper to create a pointer to int64
func int64Ptr(i int64) *int64 {
	return &i
}

func TestExtractSuggestions(t *testing.T) {
	doc := &docs.Document{
		Body: &docs.Body{
			Content: []*docs.StructuralElement{
				{
					StartIndex: 1,
					EndIndex:   20,
					Paragraph: &docs.Paragraph{
						Elements: []*docs.ParagraphElement{
							{
								StartIndex: 1,
								EndIndex:   10,
								TextRun: &docs.TextRun{
									Content:               "Insertion",
									SuggestedInsertionIds: []string{"ins-1"},
								},
							},
							{
								StartIndex: 10,
								EndIndex:   20,
								TextRun: &docs.TextRun{
									Content:              "Deletion",
									SuggestedDeletionIds: []string{"del-1"},
								},
							},
						},
					},
				},
				{
					StartIndex: 21,
					EndIndex:   30,
					Paragraph: &docs.Paragraph{
						Elements: []*docs.ParagraphElement{
							{
								StartIndex: 21,
								EndIndex:   30,
								TextRun: &docs.TextRun{
									Content: "StyleChange",
									SuggestedTextStyleChanges: map[string]docs.SuggestedTextStyle{
										"style-1": {},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	suggestions := ExtractSuggestions(doc)

	if len(suggestions) != 3 {
		t.Fatalf("Expected 3 suggestions, got %d", len(suggestions))
	}

	// Verify Insertion
	foundIns := false
	for _, s := range suggestions {
		if s.ID == "ins-1" {
			foundIns = true
			if s.Type != "insertion" {
				t.Errorf("Expected insertion type, got %s", s.Type)
			}
			if s.Content != "Insertion" {
				t.Errorf("Expected content 'Insertion', got %s", s.Content)
			}
			if s.StartIndex != 1 || s.EndIndex != 10 {
				t.Errorf("Incorrect indices for insertion: %d-%d", s.StartIndex, s.EndIndex)
			}
		}
	}
	if !foundIns {
		t.Error("Insertion suggestion not found")
	}

	// Verify Deletion
	foundDel := false
	for _, s := range suggestions {
		if s.ID == "del-1" {
			foundDel = true
			if s.Type != "deletion" {
				t.Errorf("Expected deletion type, got %s", s.Type)
			}
		}
	}
	if !foundDel {
		t.Error("Deletion suggestion not found")
	}

	// Verify Style Change
	foundStyle := false
	for _, s := range suggestions {
		if s.ID == "style-1" {
			foundStyle = true
			if s.Type != "text_style_change" {
				t.Errorf("Expected text_style_change type, got %s", s.Type)
			}
		}
	}
	if !foundStyle {
		t.Error("Style change suggestion not found")
	}
}

func TestExtractMetadataTable(t *testing.T) {
	tests := []struct {
		name       string
		doc        *docs.Document
		wantTitle  string
		wantDesc   string
		wantFields int
		wantNil    bool
	}{
		{
			name: "Valid Metadata Table",
			doc: &docs.Document{
				Body: &docs.Body{
					Content: []*docs.StructuralElement{
						{
							StartIndex: 1,
							EndIndex:   100,
							Table: &docs.Table{
								TableRows: []*docs.TableRow{
									{
										TableCells: []*docs.TableCell{
											{Content: createContent("Metadata")},
											{Content: createContent("")},
										},
									},
									{
										TableCells: []*docs.TableCell{
											{Content: createContent("Page Title")},
											{Content: createContent("My Title")},
										},
									},
									{
										TableCells: []*docs.TableCell{
											{Content: createContent("Page Description")},
											{Content: createContent("My Description")},
										},
									},
									{
										TableCells: []*docs.TableCell{
											{Content: createContent("Custom Field")},
											{Content: createContent("Custom Value")},
										},
									},
								},
							},
						},
					},
				},
			},
			wantTitle:  "My Title",
			wantDesc:   "My Description",
			wantFields: 3,
			wantNil:    false,
		},
		{
			name: "No Table",
			doc: &docs.Document{
				Body: &docs.Body{
					Content: []*docs.StructuralElement{
						{
							Paragraph: &docs.Paragraph{},
						},
					},
				},
			},
			wantNil: true,
		},
		{
			name: "Table without enough columns",
			doc: &docs.Document{
				Body: &docs.Body{
					Content: []*docs.StructuralElement{
						{
							Table: &docs.Table{
								TableRows: []*docs.TableRow{
									{
										TableCells: []*docs.TableCell{
											{Content: createContent("Single Column")},
										},
									},
								},
							},
						},
					},
				},
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractMetadataTable(tt.doc)
			if tt.wantNil {
				if got != nil {
					t.Error("Expected nil metadata, got struct")
				}
				return
			}

			if got == nil {
				t.Fatal("Expected metadata, got nil")
			}

			if got.PageTitle != tt.wantTitle {
				t.Errorf("PageTitle = %s, want %s", got.PageTitle, tt.wantTitle)
			}
			if got.PageDescription != tt.wantDesc {
				t.Errorf("PageDescription = %s, want %s", got.PageDescription, tt.wantDesc)
			}
			if len(got.Raw) != tt.wantFields {
				t.Errorf("Raw fields count = %d, want %d", len(got.Raw), tt.wantFields)
			}
		})
	}
}

func TestBuildDocumentStructure(t *testing.T) {
	doc := &docs.Document{
		Body: &docs.Body{
			Content: []*docs.StructuralElement{
				// Heading 1
				{
					StartIndex: 1,
					EndIndex:   10,
					Paragraph: &docs.Paragraph{
						ParagraphStyle: &docs.ParagraphStyle{NamedStyleType: "HEADING_1"},
						Elements: []*docs.ParagraphElement{
							{TextRun: &docs.TextRun{Content: "Heading 1"}},
						},
					},
				},
				// Normal Text
				{
					StartIndex: 11,
					EndIndex:   20,
					Paragraph: &docs.Paragraph{
						ParagraphStyle: &docs.ParagraphStyle{NamedStyleType: "NORMAL_TEXT"},
						Elements: []*docs.ParagraphElement{
							{
								StartIndex: 11,
								EndIndex:   20,
								TextRun:    &docs.TextRun{Content: "Some text"},
							},
						},
					},
				},
				// Table
				{
					StartIndex: 21,
					EndIndex:   50,
					Table: &docs.Table{
						TableRows: []*docs.TableRow{
							{
								StartIndex: 22,
								EndIndex:   30,
								TableCells: []*docs.TableCell{
									{
										StartIndex: 23,
										EndIndex:   29,
										Content:    createContent("Cell 1"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	structure := BuildDocumentStructure(doc)

	// Verify Headings
	if len(structure.Headings) != 1 {
		t.Fatalf("Expected 1 heading, got %d", len(structure.Headings))
	}
	if structure.Headings[0].Text != "Heading 1" {
		t.Errorf("Expected heading 'Heading 1', got '%s'", structure.Headings[0].Text)
	}
	if structure.Headings[0].Level != 1 {
		t.Errorf("Expected heading level 1, got %d", structure.Headings[0].Level)
	}

	// Verify Tables
	if len(structure.Tables) != 1 {
		t.Fatalf("Expected 1 table, got %d", len(structure.Tables))
	}

	// Verify TextElements
	// 1 from Heading, 1 from Normal Text, 1 from Table Cell = 3
	if len(structure.TextElements) != 3 {
		t.Errorf("Expected 3 text elements, got %d", len(structure.TextElements))
	}

	expectedText := "Heading 1Some textCell 1"
	if structure.FullText != expectedText {
		t.Errorf("Expected full text '%s', got '%s'", expectedText, structure.FullText)
	}
}

func TestBuildActionableSuggestions(t *testing.T) {
	// Setup a document structure with text: "Start [INSERT] End"
	// "Start " is indices 0-6
	// "End" is indices 6-9
	// Suggestion is at index 6

	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "Start ", StartIndex: 0, EndIndex: 6},
			{ID: "text-2", Text: "End", StartIndex: 6, EndIndex: 9},
		},
		Headings: []DocumentHeading{
			{Text: "My Heading", Level: 1, StartIndex: 0, EndIndex: 5},
		},
	}

	suggestions := []Suggestion{
		{
			ID:         "sugg-1",
			Type:       "insertion",
			Content:    "INSERT ",
			StartIndex: 6,
			EndIndex:   6, // Point insertion
		},
	}

	actionable := BuildActionableSuggestions(suggestions, structure, nil)

	if len(actionable) != 1 {
		t.Fatalf("Expected 1 actionable suggestion, got %d", len(actionable))
	}

	as := actionable[0]

	// Verify Anchor (only newlines/tabs trimmed, spaces preserved)
	if as.Anchor.PrecedingText != "Start " {
		t.Errorf("Expected PrecedingText 'Start ', got '%s'", as.Anchor.PrecedingText)
	}
	if as.Anchor.FollowingText != "End" {
		t.Errorf("Expected FollowingText 'End', got '%s'", as.Anchor.FollowingText)
	}

	// Verify Change
	if as.Change.Type != "insert" {
		t.Errorf("Expected change type 'insert', got '%s'", as.Change.Type)
	}
	if as.Change.NewText != "INSERT " {
		t.Errorf("Expected NewText 'INSERT ', got '%s'", as.Change.NewText)
	}

	// Verify Location context
	if as.Location.ParentHeading != "My Heading" {
		t.Errorf("Expected ParentHeading 'My Heading', got '%s'", as.Location.ParentHeading)
	}
}

// Helper to create basic content structure for tests
func createContent(text string) []*docs.StructuralElement {
	return []*docs.StructuralElement{
		{
			Paragraph: &docs.Paragraph{
				Elements: []*docs.ParagraphElement{
					{
						TextRun: &docs.TextRun{
							Content: text,
						},
					},
				},
			},
		},
	}
}

// TestBuildActionableSuggestions_FilterStyleChanges verifies that style changes are completely filtered out
func TestBuildActionableSuggestions_FilterStyleChanges(t *testing.T) {
	structure := &DocumentStructure{
		TextElements: []TextElementWithPosition{
			{ID: "text-1", Text: "Normal text ", StartIndex: 0, EndIndex: 12},
			{ID: "text-2", Text: "bold text", StartIndex: 12, EndIndex: 21},
			{ID: "text-3", Text: " more text", StartIndex: 21, EndIndex: 31},
		},
		Headings: []DocumentHeading{},
	}

	suggestions := []Suggestion{
		{
			ID:         "sugg-insert",
			Type:       "insertion",
			Content:    "INSERT",
			StartIndex: 12,
			EndIndex:   12,
		},
		{
			ID:         "sugg-delete",
			Type:       "deletion",
			Content:    "bold text",
			StartIndex: 12,
			EndIndex:   21,
		},
		{
			ID:         "sugg-style-1",
			Type:       "text_style_change",
			Content:    "bold text",
			StartIndex: 12,
			EndIndex:   21,
		},
		{
			ID:         "sugg-style-2",
			Type:       "text_style_change",
			Content:    "more text",
			StartIndex: 22,
			EndIndex:   31,
		},
	}

	actionable := BuildActionableSuggestions(suggestions, structure, nil)

	// Should only have 2 actionable suggestions (insertion and deletion)
	// Style changes should be completely filtered out
	if len(actionable) != 2 {
		t.Fatalf("Expected 2 actionable suggestions (style changes filtered), got %d", len(actionable))
	}

	// Verify we only have insertion and deletion, no style changes
	hasInsertion := false
	hasDeletion := false
	for _, as := range actionable {
		if as.ID == "sugg-insert" {
			hasInsertion = true
			if as.Change.Type != "insert" {
				t.Errorf("Expected change type 'insert', got '%s'", as.Change.Type)
			}
		}
		if as.ID == "sugg-delete" {
			hasDeletion = true
			if as.Change.Type != "delete" {
				t.Errorf("Expected change type 'delete', got '%s'", as.Change.Type)
			}
		}
		// Verify no style change suggestions made it through
		if as.ID == "sugg-style-1" || as.ID == "sugg-style-2" {
			t.Errorf("Style change suggestion %s should have been filtered out", as.ID)
		}
	}

	if !hasInsertion {
		t.Error("Insertion suggestion was incorrectly filtered out")
	}
	if !hasDeletion {
		t.Error("Deletion suggestion was incorrectly filtered out")
	}
}

// TestGetTextAround tests the text extraction around a position with various edge cases
func TestGetTextAround(t *testing.T) {
	tests := []struct {
		name         string
		structure    *DocumentStructure
		startIndex   int64
		endIndex     int64
		anchorLength int
		wantBefore   string
		wantAfter    string
		description  string
	}{
		{
			name: "basic extraction - insertion point",
			structure: &DocumentStructure{
				TextElements: []TextElementWithPosition{
					{ID: "text-1", Text: "Hello ", StartIndex: 0, EndIndex: 6},
					{ID: "text-2", Text: "World", StartIndex: 6, EndIndex: 11},
				},
			},
			startIndex:   6,
			endIndex:     6,
			anchorLength: 80,
			wantBefore:   "Hello ",
			wantAfter:    "World",
			description:  "Insertion point between two elements",
		},
		{
			name: "partial text extraction - element spans start",
			structure: &DocumentStructure{
				TextElements: []TextElementWithPosition{
					{ID: "text-1", Text: "Hello World!", StartIndex: 0, EndIndex: 12},
				},
			},
			startIndex:   6,
			endIndex:     11,
			anchorLength: 80,
			wantBefore:   "Hello ",
			wantAfter:    "!",
			description:  "Single element spans both positions",
		},
		{
			name: "partial text extraction - element spans end",
			structure: &DocumentStructure{
				TextElements: []TextElementWithPosition{
					{ID: "text-1", Text: "Start ", StartIndex: 0, EndIndex: 6},
					{ID: "text-2", Text: "Middle End", StartIndex: 6, EndIndex: 16},
				},
			},
			startIndex:   6,
			endIndex:     12,
			anchorLength: 80,
			wantBefore:   "Start ",
			wantAfter:    " End",
			description:  "Second element spans end position",
		},
		{
			name: "anchor length limiting",
			structure: &DocumentStructure{
				TextElements: []TextElementWithPosition{
					{ID: "text-1", Text: "This is a very long text before the suggestion point. ", StartIndex: 0, EndIndex: 55},
					{ID: "text-2", Text: "This is a very long text after the suggestion point.", StartIndex: 55, EndIndex: 107},
				},
			},
			startIndex:   55,
			endIndex:     55,
			anchorLength: 10,
			wantBefore:   "on point. ",
			wantAfter:    "This is a ",
			description:  "Anchor length limits output to 10 chars",
		},
		{
			name: "multiple elements before and after",
			structure: &DocumentStructure{
				TextElements: []TextElementWithPosition{
					{ID: "text-1", Text: "Part1 ", StartIndex: 0, EndIndex: 6},
					{ID: "text-2", Text: "Part2 ", StartIndex: 6, EndIndex: 12},
					{ID: "text-3", Text: "Part3 ", StartIndex: 12, EndIndex: 18},
					{ID: "text-4", Text: "Part4", StartIndex: 18, EndIndex: 23},
				},
			},
			startIndex:   12,
			endIndex:     18,
			anchorLength: 80,
			wantBefore:   "Part1 Part2 ",
			wantAfter:    "Part4",
			description:  "Multiple elements concatenated",
		},
		{
			name: "element entirely within suggestion range",
			structure: &DocumentStructure{
				TextElements: []TextElementWithPosition{
					{ID: "text-1", Text: "Before ", StartIndex: 0, EndIndex: 7},
					{ID: "text-2", Text: "ToDelete", StartIndex: 7, EndIndex: 15},
					{ID: "text-3", Text: " After", StartIndex: 15, EndIndex: 21},
				},
			},
			startIndex:   7,
			endIndex:     15,
			anchorLength: 80,
			wantBefore:   "Before ",
			wantAfter:    " After",
			description:  "Middle element is within suggestion range (deleted)",
		},
		{
			name: "empty structure",
			structure: &DocumentStructure{
				TextElements: []TextElementWithPosition{},
			},
			startIndex:   0,
			endIndex:     0,
			anchorLength: 80,
			wantBefore:   "",
			wantAfter:    "",
			description:  "No text elements",
		},
		{
			name: "position at document start",
			structure: &DocumentStructure{
				TextElements: []TextElementWithPosition{
					{ID: "text-1", Text: "Start text", StartIndex: 0, EndIndex: 10},
				},
			},
			startIndex:   0,
			endIndex:     0,
			anchorLength: 80,
			wantBefore:   "",
			wantAfter:    "Start text",
			description:  "Insertion at very start of document",
		},
		{
			name: "position at document end",
			structure: &DocumentStructure{
				TextElements: []TextElementWithPosition{
					{ID: "text-1", Text: "End text", StartIndex: 0, EndIndex: 8},
				},
			},
			startIndex:   8,
			endIndex:     8,
			anchorLength: 80,
			wantBefore:   "End text",
			wantAfter:    "",
			description:  "Insertion at very end of document",
		},
		{
			name: "whitespace trimming",
			structure: &DocumentStructure{
				TextElements: []TextElementWithPosition{
					{ID: "text-1", Text: "\n\t Before \n\t", StartIndex: 0, EndIndex: 13},
					{ID: "text-2", Text: "\n\t After \n\t", StartIndex: 13, EndIndex: 24},
				},
			},
			startIndex:   13,
			endIndex:     13,
			anchorLength: 80,
			wantBefore:   "\n\t Before \n\t",
			wantAfter:    "\n\t After \n\t",
			description:  "No trimming applied, full text returned",
		},
		{
			name: "partial extraction mid-element",
			structure: &DocumentStructure{
				TextElements: []TextElementWithPosition{
					{ID: "text-1", Text: "0123456789", StartIndex: 0, EndIndex: 10},
				},
			},
			startIndex:   3,
			endIndex:     7,
			anchorLength: 80,
			wantBefore:   "012",
			wantAfter:    "789",
			description:  "Extract portions from middle of element",
		},
		{
			name: "replacement spanning multiple elements",
			structure: &DocumentStructure{
				TextElements: []TextElementWithPosition{
					{ID: "text-1", Text: "AAA", StartIndex: 0, EndIndex: 3},
					{ID: "text-2", Text: "BBB", StartIndex: 3, EndIndex: 6},
					{ID: "text-3", Text: "CCC", StartIndex: 6, EndIndex: 9},
					{ID: "text-4", Text: "DDD", StartIndex: 9, EndIndex: 12},
				},
			},
			startIndex:   3,
			endIndex:     9,
			anchorLength: 80,
			wantBefore:   "AAA",
			wantAfter:    "DDD",
			description:  "Replacement spans BBB and CCC, which should not appear in anchors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before, after := getTextAround(tt.structure, tt.startIndex, tt.endIndex, tt.anchorLength)

			if before != tt.wantBefore {
				t.Errorf("PrecedingText mismatch\n  want: %q\n  got:  %q\n  desc: %s",
					tt.wantBefore, before, tt.description)
			}

			if after != tt.wantAfter {
				t.Errorf("FollowingText mismatch\n  want: %q\n  got:  %q\n  desc: %s",
					tt.wantAfter, after, tt.description)
			}
		})
	}
}
