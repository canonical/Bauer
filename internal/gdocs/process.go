package gdocs

import (
	"context"
	"fmt"
	"log/slog"
)

// ProcessingResult contains all extracted data from a Google Doc.
type ProcessingResult struct {
	DocumentTitle         string                       `json:"document_title"`
	DocumentID            string                       `json:"document_id"`
	Metadata              *MetadataTable               `json:"metadata,omitempty"`
	ActionableSuggestions []ActionableSuggestion       `json:"actionable_suggestions"`
	GroupedSuggestions    []LocationGroupedSuggestions `json:"grouped_suggestions"`
	Comments              []Comment                    `json:"comments"`
}

// ProcessDocument fetches a document and extracts all relevant information.
// It orchestrates the fetching, extraction, and structuring of data.
// Supports docID parameter with optional tab ID (e.g., "docID?tab=tabID")
func (c *Client) ProcessDocument(ctx context.Context, docID string) (*ProcessingResult, error) {
	slog.Info("Fetching document content...", slog.String("doc_id", docID))
	fmt.Printf("Fetching document %s...\n", docID)

	// Parse tab ID from docID if provided
	actualDocID, tabID := ParseDocIDAndTab(docID)
	if tabID != "" {
		slog.Info("Tab ID specified for filtering", slog.String("tab_id", tabID))
		fmt.Printf("Filtering suggestions to tab: %s\n", tabID)
	}

	doc, err := c.FetchDocument(ctx, actualDocID)
	if err != nil {
		slog.Error("Failed to fetch document", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to fetch document: %w", err)
	}

	slog.Info("Document fetched successfully",
		slog.String("title", doc.Title),
		slog.String("document_id", doc.DocumentId),
	)
	fmt.Printf("Successfully fetched document: %s\n", doc.Title)

	// Extract Suggestions (with optional tab filtering)
	suggestions := ExtractSuggestions(doc, tabID)
	slog.Info("Suggestions extracted", slog.Int("count", len(suggestions)))
	if tabID != "" {
		slog.Info("Suggestions filtered by tab", slog.String("tab_id", tabID))
	}

	// Extract Metadata
	metadata := ExtractMetadataTable(doc)
	if metadata != nil {
		slog.Info("Metadata table extracted", slog.Int("field_count", len(metadata.Raw)))
	}

	// Build Document Structure
	docStructure := BuildDocumentStructure(doc)
	slog.Info("Document structure built",
		slog.Int("headings", len(docStructure.Headings)),
		slog.Int("tables", len(docStructure.Tables)),
	)

	// Build Actionable Suggestions
	actionableSuggestions := BuildActionableSuggestions(suggestions, docStructure, metadata)
	slog.Info("Extracted actionable suggestions", slog.Int("field_count", len(actionableSuggestions)))

	// Group Actionable Suggestions
	groupedSuggestions := GroupActionableSuggestions(actionableSuggestions, docStructure)
	slog.Info("Grouped actionable suggestions", slog.Int("location_groups", len(groupedSuggestions)))

	return &ProcessingResult{
		DocumentTitle:         doc.Title,
		DocumentID:            doc.DocumentId,
		Metadata:              metadata,
		ActionableSuggestions: actionableSuggestions,
		GroupedSuggestions:    groupedSuggestions,
		Comments:              nil,
	}, nil
}
