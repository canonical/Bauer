package gdocs

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/docs/v1"
)

// FetchDocument fetches the document with suggestions inline.
func (c *Client) FetchDocument(ctx context.Context, docID string) (*docs.Document, error) {
	// Use SUGGESTIONS_INLINE to see suggestions marked in the content
	doc, err := c.Docs.Documents.Get(docID).
		SuggestionsViewMode("SUGGESTIONS_INLINE").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch document: %w", err)
	}
	return doc, nil
}

// ExtractSuggestions walks through the document content and extracts all suggestions.
// TODO this and all sub functions can be made concurrent for speed
// TODO add recursion depth control on this and sub functions
func ExtractSuggestions(doc *docs.Document) []Suggestion {
	var suggestions []Suggestion

	if doc.Body != nil {
		for _, elem := range doc.Body.Content {
			processStructuralElement(elem, &suggestions)
		}
	}

	for _, header := range doc.Headers {
		if header.Content != nil {
			for _, elem := range header.Content {
				processStructuralElement(elem, &suggestions)
			}
		}
	}

	for _, footer := range doc.Footers {
		if footer.Content != nil {
			for _, elem := range footer.Content {
				processStructuralElement(elem, &suggestions)
			}
		}
	}

	return suggestions
}

// BuildDocumentStructure builds a comprehensive structure of the document.
// TODO this should be combined with ExtractSuggestions to avoid multiple traversals of the same document
func BuildDocumentStructure(doc *docs.Document) *DocumentStructure {
	structure := &DocumentStructure{
		Headings:     []DocumentHeading{},
		Tables:       []TableRange{},
		TextElements: []TextElementWithPosition{},
	}

	var fullTextBuilder strings.Builder

	if doc.Body == nil || doc.Body.Content == nil {
		return structure
	}

	var lastParagraphText string
	var textElementCounter int
	var tableCounter int
	var headingCounter int

	for _, elem := range doc.Body.Content {
		// Extract headings
		if heading := extractHeading(elem, headingCounter+1); heading != nil {
			headingCounter++
			structure.Headings = append(structure.Headings, *heading)
		}

		// Extract all text elements with positions (including from headings)
		if elem.Paragraph != nil {
			var paraText strings.Builder
			for _, paraElem := range elem.Paragraph.Elements {
				if paraElem.TextRun != nil {
					textElementCounter++
					structure.TextElements = append(structure.TextElements, TextElementWithPosition{
						ID:         fmt.Sprintf("text-%d", textElementCounter),
						Text:       paraElem.TextRun.Content,
						StartIndex: paraElem.StartIndex,
						EndIndex:   paraElem.EndIndex,
					})
					fullTextBuilder.WriteString(paraElem.TextRun.Content)
					paraText.WriteString(paraElem.TextRun.Content)
				}
			}
			lastParagraphText = strings.TrimSpace(paraText.String())
		}

		// Extract table structure
		if elem.Table != nil {
			tableCounter++
			tableRange := TableRange{
				ID:            fmt.Sprintf("table-%d", tableCounter),
				Title:         lastParagraphText,
				StartIndex:    elem.StartIndex,
				EndIndex:      elem.EndIndex,
				RowRanges:     []RowRange{},
				ColumnHeaders: []string{},
			}

			for rowIdx, row := range elem.Table.TableRows {
				rowRange := RowRange{
					StartIndex: row.StartIndex,
					EndIndex:   row.EndIndex,
					CellRanges: []CellRange{},
				}

				for _, cell := range row.TableCells {
					cellText := extractCellText(cell)
					firstLine := cellText
					if idx := strings.Index(cellText, "\n"); idx != -1 {
						firstLine = cellText[:idx]
					}
					if len(firstLine) > 50 {
						firstLine = firstLine[:50] + "..."
					}

					cellRange := CellRange{
						StartIndex: cell.StartIndex,
						EndIndex:   cell.EndIndex,
						Text:       cellText,
						FirstLine:  firstLine,
					}
					rowRange.CellRanges = append(rowRange.CellRanges, cellRange)

					if rowIdx == 0 {
						tableRange.ColumnHeaders = append(tableRange.ColumnHeaders, firstLine)
					}

					for _, cellContent := range cell.Content {
						if cellContent.Paragraph != nil {
							for _, paraElem := range cellContent.Paragraph.Elements {
								if paraElem.TextRun != nil {
									textElementCounter++
									structure.TextElements = append(structure.TextElements, TextElementWithPosition{
										ID:         fmt.Sprintf("text-%d", textElementCounter),
										Text:       paraElem.TextRun.Content,
										StartIndex: paraElem.StartIndex,
										EndIndex:   paraElem.EndIndex,
									})
									fullTextBuilder.WriteString(paraElem.TextRun.Content)
								}
							}
						}
					}
				}
				tableRange.RowRanges = append(tableRange.RowRanges, rowRange)
			}
			structure.Tables = append(structure.Tables, tableRange)
		}

		if elem.Paragraph == nil {
			lastParagraphText = ""
		}
	}

	structure.FullText = fullTextBuilder.String()
	return structure
}

// BuildActionableSuggestions converts raw suggestions into actionable suggestions with full context.
func BuildActionableSuggestions(suggestions []Suggestion, structure *DocumentStructure, metadata *MetadataTable) []ActionableSuggestion {
	actionable := make([]ActionableSuggestion, 0, len(suggestions))
	const anchorLength = 80

	for _, sugg := range suggestions {
		as := ActionableSuggestion{
			ID: sugg.ID,
		}

		as.Position.StartIndex = sugg.StartIndex
		as.Position.EndIndex = sugg.EndIndex

		as.Location = SuggestionLocation{
			Section: "Body",
		}

		if metadata != nil && sugg.StartIndex >= metadata.TableStartIndex && sugg.EndIndex <= metadata.TableEndIndex {
			as.Location.InMetadata = true
		}

		parentHeading, headingLevel := findParentHeading(structure, sugg.StartIndex)
		// if sugg.ID == "suggest.r3eqy31u1iac" {
		// 	fmt.Printf("\n\n SUSPECT \n\n PARENT: %v -- level: %v \n\n", parentHeading, headingLevel)
		// }
		as.Location.ParentHeading = parentHeading
		as.Location.HeadingLevel = headingLevel

		tableLoc := findTableLocation(structure, sugg.StartIndex)
		if tableLoc != nil {
			as.Location.InTable = true
			as.Location.Table = tableLoc
		}
		// if sugg.ID == "suggest.r3eqy31u1iac" {
		// 	fmt.Printf("\n\n SUSPECT 1 \n\n TABLE LOC:\n %v \n\n ", tableLoc)
		// }

		precedingText, followingText := getTextAround(structure, sugg.StartIndex, sugg.EndIndex, anchorLength)
		// if sugg.ID == "suggest.r3eqy31u1iac" {
		// 	fmt.Printf("\n\n SUSPECT 2 \n\n PRECEDING:\n %v \n\n --FOLLOWING:\n\n %v \n\n", precedingText, followingText)
		// }
		as.Anchor = SuggestionAnchor{
			PrecedingText: precedingText,
			FollowingText: followingText,
		}

		switch sugg.Type {
		case "insertion":
			as.Change = SuggestionChange{
				Type:         "insert",
				OriginalText: "",
				NewText:      sugg.Content,
			}
			as.Verification = SuggestionVerification{
				TextBeforeChange: precedingText + followingText,
				TextAfterChange:  precedingText + sugg.Content + followingText,
			}

		case "deletion":
			as.Change = SuggestionChange{
				Type:         "delete",
				OriginalText: sugg.Content,
				NewText:      "",
			}
			as.Verification = SuggestionVerification{
				TextBeforeChange: precedingText + sugg.Content + followingText,
				TextAfterChange:  precedingText + followingText,
			}

		// TODO we need to mention the exact style change, this is currently not helpful at all
		case "text_style_change":
			as.Change = SuggestionChange{
				Type:         "style",
				OriginalText: sugg.Content,
				NewText:      sugg.Content,
			}
			as.Verification = SuggestionVerification{
				TextBeforeChange: precedingText + sugg.Content + followingText,
				TextAfterChange:  precedingText + sugg.Content + followingText,
			}
		}

		actionable = append(actionable, as)
	}

	return actionable
}

// ExtractMetadataTable extracts the metadata table from the beginning of the document.
func ExtractMetadataTable(doc *docs.Document) *MetadataTable {
	if doc.Body == nil || doc.Body.Content == nil {
		return nil
	}

	var firstTable *docs.Table
	var tableStartIndex, tableEndIndex int64

	for _, elem := range doc.Body.Content {
		if elem.Table != nil {
			firstTable = elem.Table
			tableStartIndex = elem.StartIndex
			tableEndIndex = elem.EndIndex
			break
		}
	}

	if firstTable == nil {
		return nil
	}

	// Validate that this is a metadata table by checking the first row, first column
	if len(firstTable.TableRows) > 0 && len(firstTable.TableRows[0].TableCells) > 0 {
		firstCellText := extractCellText(firstTable.TableRows[0].TableCells[0])
		if !strings.EqualFold(firstCellText, "Metadata") {
			return nil
		}
	} else {
		return nil
	}

	metadata := &MetadataTable{
		Raw:             make(map[string]string),
		TableStartIndex: tableStartIndex,
		TableEndIndex:   tableEndIndex,
	}

	for _, row := range firstTable.TableRows {
		if len(row.TableCells) < 2 {
			continue
		}

		key := extractCellText(row.TableCells[0])
		value := extractCellText(row.TableCells[1])

		if key == "" || strings.EqualFold(key, "Metadata") {
			continue
		}

		metadata.Raw[key] = value

		keyLower := strings.ToLower(key)
		if strings.Contains(keyLower, "page title") || (strings.Contains(keyLower, "title") && !strings.Contains(keyLower, "description")) {
			metadata.PageTitle = value
		} else if strings.Contains(keyLower, "page description") || strings.Contains(keyLower, "description") {
			metadata.PageDescription = value
		} else if strings.Contains(keyLower, "url") || strings.Contains(keyLower, "page url") {
			metadata.SuggestedUrl = value
		}
	}

	if len(metadata.Raw) == 0 {
		return nil
	}

	return metadata
}

// Helper functions

// processStructuralElement recursively processes a structural element (paragraph, table, TOC)
// to find and extract suggestions.
func processStructuralElement(elem *docs.StructuralElement, suggestions *[]Suggestion) {
	if elem == nil {
		return
	}

	if elem.Paragraph != nil {
		processParagraph(elem.Paragraph, suggestions)
	}
	if elem.Table != nil {
		processTable(elem.Table, suggestions)
	}
	if elem.TableOfContents != nil && elem.TableOfContents.Content != nil {
		for _, tocElem := range elem.TableOfContents.Content {
			processStructuralElement(tocElem, suggestions)
		}
	}
}

// processParagraph iterates through paragraph elements to extract suggestions.
func processParagraph(para *docs.Paragraph, suggestions *[]Suggestion) {
	if para == nil {
		return
	}
	for _, paraElem := range para.Elements {
		processParagraphElement(paraElem, suggestions)
	}
}

// processTable iterates through table rows and cells to extract suggestions recursively.
func processTable(table *docs.Table, suggestions *[]Suggestion) {
	if table == nil {
		return
	}
	for _, row := range table.TableRows {
		for _, cell := range row.TableCells {
			for _, cellContent := range cell.Content {
				processStructuralElement(cellContent, suggestions)
			}
		}
	}
}

// processParagraphElement inspects a single paragraph element (TextRun) for suggested insertions,
// deletions, or text style changes.
func processParagraphElement(paraElem *docs.ParagraphElement, suggestions *[]Suggestion) {
	if paraElem.TextRun != nil {
		tr := paraElem.TextRun

		if len(tr.SuggestedInsertionIds) > 0 {
			for _, suggID := range tr.SuggestedInsertionIds {
				*suggestions = append(*suggestions, Suggestion{
					ID:         suggID,
					Type:       "insertion",
					Content:    tr.Content,
					StartIndex: paraElem.StartIndex,
					EndIndex:   paraElem.EndIndex,
				})
			}
		}

		if len(tr.SuggestedDeletionIds) > 0 {
			for _, suggID := range tr.SuggestedDeletionIds {
				*suggestions = append(*suggestions, Suggestion{
					ID:         suggID,
					Type:       "deletion",
					Content:    tr.Content,
					StartIndex: paraElem.StartIndex,
					EndIndex:   paraElem.EndIndex,
				})
			}
		}

		if tr.SuggestedTextStyleChanges != nil {
			for suggID := range tr.SuggestedTextStyleChanges {
				*suggestions = append(*suggestions, Suggestion{
					ID:         suggID,
					Type:       "text_style_change",
					Content:    tr.Content,
					StartIndex: paraElem.StartIndex,
					EndIndex:   paraElem.EndIndex,
				})
			}
		}
	}
}

// extractHeading attempts to extract heading info from a structural element.
// Returns nil if the element is not a heading.
func extractHeading(elem *docs.StructuralElement, headingCounter int) *DocumentHeading {
	if elem.Paragraph == nil || elem.Paragraph.ParagraphStyle == nil {
		return nil
	}

	para := elem.Paragraph
	namedStyle := para.ParagraphStyle.NamedStyleType
	headingLevel := 0
	switch namedStyle {
	case "HEADING_1":
		headingLevel = 1
	case "HEADING_2":
		headingLevel = 2
	case "HEADING_3":
		headingLevel = 3
	case "HEADING_4":
		headingLevel = 4
	case "HEADING_5":
		headingLevel = 5
	case "HEADING_6":
		headingLevel = 6
	}

	if headingLevel == 0 {
		return nil
	}

	var headingText strings.Builder
	for _, paraElem := range para.Elements {
		if paraElem.TextRun != nil {
			headingText.WriteString(paraElem.TextRun.Content)
		}
	}

	return &DocumentHeading{
		ID:         fmt.Sprintf("heading-%d", headingCounter),
		Text:       strings.TrimSpace(headingText.String()),
		Level:      headingLevel,
		StartIndex: elem.StartIndex,
		EndIndex:   elem.EndIndex,
	}
}

// extractCellText extracts all text content from a table cell.
// It traverses all paragraphs and text runs within the cell and concatenates their content.
// Newlines are trimmed from the final result.
func extractCellText(cell *docs.TableCell) string {
	var builder strings.Builder

	if cell == nil || cell.Content == nil {
		return ""
	}

	for _, elem := range cell.Content {
		if elem.Paragraph != nil {
			for _, paraElem := range elem.Paragraph.Elements {
				if paraElem.TextRun != nil {
					builder.WriteString(paraElem.TextRun.Content)
				}
			}
		}
	}

	return strings.TrimSpace(builder.String())
}

// findParentHeading finds the nearest heading that comes before the given position.
// It returns the heading text and its level.
func findParentHeading(structure *DocumentStructure, position int64) (string, int) {
	var parentHeading string
	var headingLevel int

	for _, heading := range structure.Headings {
		if heading.StartIndex < position {
			parentHeading = heading.Text
			headingLevel = heading.Level
		} else {
			break
		}
	}

	return parentHeading, headingLevel
}

// findTableLocation determines if a position is within a table and returns its location details.
func findTableLocation(structure *DocumentStructure, position int64) *TableLocation {
	for tableIdx, table := range structure.Tables {
		if position >= table.StartIndex && position <= table.EndIndex {
			loc := &TableLocation{
				TableIndex: tableIdx + 1,
				TableID:    table.ID,
				TableTitle: table.Title,
			}

			for rowIdx, row := range table.RowRanges {
				if position >= row.StartIndex && position <= row.EndIndex {
					loc.RowIndex = rowIdx + 1

					if len(row.CellRanges) > 0 {
						loc.RowHeader = row.CellRanges[0].FirstLine
					}

					for colIdx, cell := range row.CellRanges {
						if position >= cell.StartIndex && position <= cell.EndIndex {
							loc.ColumnIndex = colIdx + 1

							if colIdx < len(table.ColumnHeaders) {
								loc.ColumnHeader = table.ColumnHeaders[colIdx]
							}
							break
						}
					}
					break
				}
			}

			return loc
		}
	}

	return nil
}

// getTextAround extracts text before and after a given position.
// Handles partial text extraction from elements that span the positions.
// The anchorLength parameter controls how much context to include.
func getTextAround(structure *DocumentStructure, startIndex, endIndex int64, anchorLength int) (before, after string) {
	var beforeBuilder strings.Builder
	var afterBuilder strings.Builder

	for _, elem := range structure.TextElements {
		// Text before startIndex
		if elem.EndIndex <= startIndex {
			beforeBuilder.WriteString(elem.Text)
		} else if elem.StartIndex < startIndex {
			// Element spans the start position - extract the portion before startIndex
			charsToTake := startIndex - elem.StartIndex
			if charsToTake > 0 && charsToTake <= int64(len(elem.Text)) {
				beforeBuilder.WriteString(elem.Text[:charsToTake])
			}
		}

		// Text after endIndex
		if elem.StartIndex >= endIndex {
			afterBuilder.WriteString(elem.Text)
		} else if elem.EndIndex > endIndex {
			// Element spans the end position - extract the portion after endIndex
			offsetIntoElement := endIndex - elem.StartIndex
			if offsetIntoElement >= 0 && offsetIntoElement < int64(len(elem.Text)) {
				afterBuilder.WriteString(elem.Text[offsetIntoElement:])
			}
		}
	}

	beforeText := beforeBuilder.String()
	afterText := afterBuilder.String()

	// Truncate to anchor length
	if len(beforeText) > anchorLength {
		before = beforeText[len(beforeText)-anchorLength:]
	} else {
		before = beforeText
	}

	if len(afterText) > anchorLength {
		after = afterText[:anchorLength]
	} else {
		after = afterText
	}

	return before, after
}
