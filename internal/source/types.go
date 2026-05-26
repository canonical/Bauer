package source

import (
	"bauer/internal/figma"
	"bauer/internal/gdocs"
)

// SourceBundle is the normalized combined output from all source adapters.
// It is what the orchestrator operates on — not raw gdocs or figma types directly.
type SourceBundle struct {
	// Document is the Google Docs extraction result. Always present when a DocID is set.
	Document *gdocs.ProcessingResult `json:"document,omitempty"`
	// Design holds the optional Figma normalized design output.
	// It is nil when no --figma-url was supplied.
	Design *figma.NormalizedDesign `json:"design,omitempty"`
}
