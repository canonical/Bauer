package source

import "bauer/internal/gdocs"

// SourceBundle is the normalized combined output from all source adapters.
// It is what the orchestrator operates on — not raw gdocs or figma types directly.
type SourceBundle struct {
	// Document is the Google Docs extraction result. Nil when DocID is empty or extraction fails.
	Document *gdocs.ProcessingResult `json:"document,omitempty"`
	// Design holds the optional Figma normalized design output.
	// It is nil when no --figma-url was supplied. Will be *figma.NormalizedDesign
	// once the figma package lands in a later branch.
	Design any `json:"design,omitempty"`
}
