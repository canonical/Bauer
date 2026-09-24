package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"bauer/internal/gdocs"
)

func main() {
	credentialsPath := flag.String("credentials", "credentials.json", "Path to the Google service account JSON key file")
	docID := flag.String("doc", "", "Document ID, optionally with a tab (e.g. \"docID?tab=tabID\")")
	viewMode := flag.String("mode", gdocs.PreviewWithoutSuggestions,
		"Suggestions view mode: PREVIEW_WITHOUT_SUGGESTIONS, PREVIEW_SUGGESTIONS_ACCEPTED, or SUGGESTIONS_INLINE")
	outPath := flag.String("out", "doc-extract.json", "Path to the JSON file to write the extraction result to")
	flag.Parse()

	if *docID == "" {
		log.Fatal("missing required -doc flag")
	}

	ctx := context.Background()

	// 1. Authenticate.
	client, err := gdocs.NewClient(ctx, *credentialsPath)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	// 2. Split the docID into the raw ID + optional tab. The extractors need the
	//    tabID separately (pass "" to process all tabs).
	_, tabID := gdocs.ParseDocIDAndTab(*docID)

	// 3. Fetch the raw document with the chosen view mode (no suggestion markers).
	doc, err := client.FetchDocumentWithMode(ctx, *docID, *viewMode)
	if err != nil {
		log.Fatalf("failed to fetch document: %v", err)
	}

	// 4. Run the individual, suggestion-agnostic extractors.
	structure := gdocs.BuildDocumentStructure(doc, tabID)
	metadata := gdocs.ExtractMetadataTable(doc, tabID)

	// 5. Assemble the result and write it to a JSON file.
	result := gdocs.DocumentContent{
		DocumentTitle: doc.Title,
		DocumentID:    doc.DocumentId,
		TabID:         tabID,
		Metadata:      metadata,
		Structure:     structure,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("failed to encode result: %v", err)
	}

	if err := os.WriteFile(*outPath, data, 0o644); err != nil {
		log.Fatalf("failed to write %s: %v", *outPath, err)
	}

	fmt.Printf("Wrote extraction result for %q to %s\n", doc.Title, *outPath)
}
