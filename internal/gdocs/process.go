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
func (c *Client) ProcessDocument(ctx context.Context, docID string) (*ProcessingResult, error) {
	slog.Info("Fetching document content...", slog.String("doc_id", docID))
	fmt.Printf("Fetching document %s...\n", docID)

	doc, err := c.FetchDocument(ctx, docID)
	if err != nil {
		slog.Error("Failed to fetch document", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to fetch document: %w", err)
	}

	slog.Info("Document fetched successfully",
		slog.String("title", doc.Title),
		slog.String("document_id", doc.DocumentId),
	)
	fmt.Printf("Successfully fetched document: %s\n", doc.Title)

	// Extract Suggestions
	suggestions := ExtractSuggestions(doc)
	slog.Info("Suggestions extracted", slog.Int("count", len(suggestions)))

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

	// Filter out suggestions in the metadata table
	// TODO need to remove this filtering and add instructions on how exactly to approach metadata fields
	groupedSuggestions = FilterMetadataSuggestions(groupedSuggestions)
	slog.Info("Filtered metadata suggestions", slog.Int("location_groups_remaining", len(groupedSuggestions)))

	return &ProcessingResult{
		DocumentTitle:         doc.Title,
		DocumentID:            doc.DocumentId,
		Metadata:              metadata,
		ActionableSuggestions: actionableSuggestions,
		GroupedSuggestions:    groupedSuggestions,
		Comments:              nil,
	}, nil
}
