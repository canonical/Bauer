package source

import "bauer/internal/gdocs"

// SourceBundle is the normalized combined output from all configured sources.
// The orchestrator depends on this type, not on any individual source package.
// Fields may be nil depending on which adapters were configured and whether
// their fetch succeeded — callers must check for nil before accessing.
type SourceBundle struct {
	// Document holds the Google Docs extraction result.
	// Nil when no gdocs adapter was configured or the adapter returned
	// an unrecognised result type (see Manager.Fetch).
	Document *gdocs.ProcessingResult `json:"document,omitempty"`

	// Design is reserved for Figma integration (spec 002). Nil until T2F lands.
	// When populated, it will hold a *figma.NormalizedDesign.
	Design any `json:"design,omitempty"`
}
