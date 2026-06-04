package orchestrator

import (
	"bauer/internal/gdocs"
	"time"
)

// ParseResultMetadata contains metadata about the parsing operation
type ParseResultMetadata struct {
	DocumentTitle      string        `json:"document_title"`
	DocumentID         string        `json:"document_id"`
	TabID              string        `json:"tab_id,omitempty"`
	ExtractionTime     time.Time     `json:"extraction_time"`
	ExtractionDuration time.Duration `json:"extraction_duration"`
	ProcessingDuration time.Duration `json:"processing_duration"`
	TotalDuration      time.Duration `json:"total_duration"`
}

// FileMapping describes where changes should be applied in a file
type FileMapping struct {
	// Suggested file path from document metadata (optional)
	SuggestedFile string `json:"suggested_file,omitempty"`

	// Source reference from document (e.g., URL, page title)
	SourceReference string `json:"source_reference,omitempty"`

	// Number of suggestions for this file
	SuggestionCount int `json:"suggestion_count"`

	// IDs of suggestions that apply to this file
	SuggestionIDs []string `json:"suggestion_ids"`
}

// SimplifiedActionableSuggestion is a machine-optimized version with file reference and reduced redundancy
type SimplifiedActionableSuggestion struct {
	// ID is the unique suggestion identifier from Google Docs
	ID string `json:"id"`

	// File is the target file where this change should be applied (added for agent efficiency)
	File string `json:"file"`

	// Anchor contains exact text before/after for locating where to apply the change
	Anchor gdocs.SuggestionAnchor `json:"anchor"`

	// Change describes exactly what modification to make
	Change gdocs.SuggestionChange `json:"change"`

	// Verification allows confirming the change was applied correctly
	Verification gdocs.SuggestionVerification `json:"verification"`
}

// SuggestionSummaryByFile shows statistics per file
type SuggestionSummaryByFile struct {
	Insertions   int `json:"insertions"`
	Deletions    int `json:"deletions"`
	Replacements int `json:"replacements"`
}

// SuggestionSummaryByType shows statistics by operation type
type SuggestionSummaryByType struct {
	Insert  int `json:"insert"`
	Delete  int `json:"delete"`
	Replace int `json:"replace"`
}

// ParseResultSummary provides an at-a-glance overview for agents
type ParseResultSummary struct {
	TotalSuggestions int                                `json:"total_suggestions"`
	TotalFiles       int                                `json:"total_files"`
	ByFile           map[string]SuggestionSummaryByFile `json:"by_file"`
	ByType           SuggestionSummaryByType            `json:"by_type"`
}

// ParseResult contains all parsed data optimized for agent readability
type ParseResult struct {
	// Metadata about the parsing operation
	Metadata ParseResultMetadata `json:"metadata"`

	// Document-level metadata from Google Doc
	DocumentMetadata *gdocs.MetadataTable `json:"document_metadata,omitempty"`

	// Summary provides at-a-glance statistics for agent planning
	Summary ParseResultSummary `json:"summary"`

	// File mappings - where changes should be applied
	FileMappings map[string]*FileMapping `json:"file_mappings"`

	// Simplified actionable suggestions (without redundant location data)
	// Each suggestion includes its target file for quick lookup
	ActionableSuggestions []SimplifiedActionableSuggestion `json:"actionable_suggestions"`

	// Suggestions grouped by location in document (for context/verification)
	GroupedSuggestions []gdocs.LocationGroupedSuggestions `json:"grouped_suggestions"`

	// All comments from the document
	Comments []gdocs.Comment `json:"comments"`
}
