package gdocs

import (
	"context"
	"fmt"

	"google.golang.org/api/docs/v1"
)

// Suggestion view modes accepted by the Google Docs API's documents.get endpoint.
// See https://developers.google.com/docs/api/reference/rest/v1/documents/get#suggestionsviewmode
const (
	// Use this to fetch the document with inline suggestions visible.
	SuggestionsInline = "SUGGESTIONS_INLINE"

	// Use this to read the document's clean, original text.
	PreviewWithoutSuggestions = "PREVIEW_WITHOUT_SUGGESTIONS"

	// Use this to read the document as it would look post-approval.
	PreviewSuggestionsAccepted = "PREVIEW_SUGGESTIONS_ACCEPTED"
)

// FetchDocumentWithMode fetches a document using the given suggestions view mode.
// Supports docID parameter with optional tab ID (e.g., "docID?tab=tabID").
// Sets includeTabsContent=true to fetch all tabs in the document.
func (c *Client) FetchDocumentWithMode(ctx context.Context, docID, viewMode string) (*docs.Document, error) {
	actualDocID, _ := ParseDocIDAndTab(docID)

	doc, err := c.Docs.Documents.Get(actualDocID).
		SuggestionsViewMode(viewMode).
		IncludeTabsContent(true).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch document: %w", err)
	}
	return doc, nil
}

// DocumentContent holds the parsed structure of a Google Doc without any
// suggestion processing. It is the content-only counterpart to ProcessingResult.
type DocumentContent struct {
	DocumentTitle string             `json:"document_title"`
	DocumentID    string             `json:"document_id"`
	TabID         string             `json:"tab_id,omitempty"`
	Metadata      *MetadataTable     `json:"metadata,omitempty"`
	Structure     *DocumentStructure `json:"structure"`
}
