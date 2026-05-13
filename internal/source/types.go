package source

import "bauer/internal/gdocs"

// SourceBundle is the normalized combined output from all configured sources.
// The orchestrator depends on this type, not on any individual source package.
type SourceBundle struct {
	// Document holds the Google Docs extraction result. Always populated
	// after a successful fetch (gdocs is the primary text source).
	Document *gdocs.ProcessingResult `json:"document,omitempty"`

	// Design is reserved for Figma integration (spec 002). Nil until T2F lands.
	// When populated, it will hold a *figma.NormalizedDesign.
	Design any `json:"design,omitempty"`
}
