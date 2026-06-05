package gdocs

type Suggestion struct {
	ID         string `json:"id"`
	Type       string `json:"type"` // "insertion", "deletion", "text_style_change", or "link"
	Content    string `json:"content"`
	StartIndex int64  `json:"start_index"`
	EndIndex   int64  `json:"end_index"`

	// Link URLs, populated for Type "link" and for insertions carrying a link.
	// Not serialized; they feed BuildActionableSuggestions.
	OldURL string `json:"-"` // current link URL, empty if not previously linked
	NewURL string `json:"-"` // suggested link URL, empty if link removed
	TabID      string `json:"tab_id,omitempty"` // Tab ID if suggestion belongs to a specific tab
}

// DocumentHeading represents a heading in the document with its position.
// Used to determine which section a suggestion belongs to.
type DocumentHeading struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	Level      int    `json:"level"` // 1-6 for HEADING_1 through HEADING_6
	StartIndex int64  `json:"start_index"`
	EndIndex   int64  `json:"end_index"`
}

// TableLocation describes where within a table a suggestion is located
type TableLocation struct {
	TableIndex   int    `json:"table_index"`   // Which table (1-based)
	TableID      string `json:"table_id"`      // Unique ID of the table
	TableTitle   string `json:"table_title"`   // Title of the table if available
	RowIndex     int    `json:"row_index"`     // Row number (1-based)
	ColumnIndex  int    `json:"column_index"`  // Column number (1-based)
	ColumnHeader string `json:"column_header"` // Header of this column if available
	RowHeader    string `json:"row_header"`    // First cell of this row if available
}

// SuggestionLocation provides context about where in the document a suggestion is located.
// This is metadata for verification, not for finding the text.
type SuggestionLocation struct {
	Section       string         `json:"section"`                  // "Body", "Header", "Footer"
	ParentHeading string         `json:"parent_heading,omitempty"` // Nearest heading above
	HeadingLevel  int            `json:"heading_level,omitempty"`  // Level of parent heading (1-6)
	InTable       bool           `json:"in_table"`
	Table         *TableLocation `json:"table,omitempty"` // Table details if in a table
	InMetadata    bool           `json:"in_metadata"`     // True if in the metadata table
}

// SuggestionAnchor contains the exact text before and after a suggestion.
// Used by LLMs to locate where to apply the change in HTML/text content.
// These are NOT truncated - they contain enough context to uniquely identify the location.
type SuggestionAnchor struct {
	// PrecedingText is the exact text immediately before the suggestion point.
	// For insertions: text before where new content should be inserted.
	// For deletions: text before the content to be deleted.
	PrecedingText string `json:"preceding_text"`

	// FollowingText is the exact text immediately after the suggestion point.
	// For insertions: text after where new content should be inserted.
	// For deletions: text after the content to be deleted.
	FollowingText string `json:"following_text"`
}

// SuggestionChange describes exactly what text change should be made.
type SuggestionChange struct {
	// Type is the operation: "insert", "delete", "replace",
	// "link_add", "link_change", or "link_remove".
	Type string `json:"type"`

	// OriginalText is the text currently in the document (empty for pure insertions)
	OriginalText string `json:"original_text,omitempty"`

	// NewText is the text that should replace/be inserted (empty for pure deletions)
	NewText string `json:"new_text,omitempty"`

	// LinkText is the visible text a link wraps (link_* operations only).
	// Named link_text, not anchor_text, to avoid overloading "anchor" (which
	// already means the preceding/following locator text in SuggestionAnchor).
	LinkText string `json:"link_text,omitempty"`

	// OldURL is the previous link target (empty for link_add).
	OldURL string `json:"old_url,omitempty"`

	// NewURL is the new link target (empty for link_remove).
	NewURL string `json:"new_url,omitempty"`
}

// SuggestionVerification shows the before/after state for validation.
type SuggestionVerification struct {
	// TextBeforeChange shows what the text looks like before applying the suggestion
	TextBeforeChange string `json:"text_before_change"`

	// TextAfterChange shows what the text should look like after applying the suggestion
	TextAfterChange string `json:"text_after_change"`
}

// ActionableSuggestion provides all context needed for an LLM to find and apply a suggestion.
//
// Design principles for machine-readability:
// 1. Anchor texts are NOT truncated - they contain exact strings for matching
// 2. Change is explicit about the operation (insert/delete/replace)
// 3. Verification allows confirming the change was applied correctly
// 4. Location is metadata for context, not for finding text
//
// To apply a suggestion to HTML:
// 1. Search for anchor.preceding_text + change.original_text + anchor.following_text
// 2. Replace change.original_text with change.new_text
// 3. Verify result matches verification.text_after_change
type ActionableSuggestion struct {
	// ID is the unique suggestion identifier from Google Docs
	ID string `json:"id"`

	// Anchor contains exact text before/after for locating where to apply the change
	Anchor SuggestionAnchor `json:"anchor"`

	// Change describes exactly what modification to make
	Change SuggestionChange `json:"change"`

	// Verification provides before/after text for validating the change
	Verification SuggestionVerification `json:"verification"`

	// Location provides contextual metadata (section, table, etc.) for human verification
	Location SuggestionLocation `json:"location"`

	// Position contains character indices in the original Google Doc (for reference only)
	Position struct {
		StartIndex int64 `json:"start_index"`
		EndIndex   int64 `json:"end_index"`
	} `json:"position"`
}

// GroupedActionableSuggestion represents one or more atomic suggestions that belong together.
// When Google Docs returns a replacement operation, it breaks it into multiple atomic
// insert/delete operations with the same suggestion ID. This type groups those together
// into a single logical suggestion for easier consumption by LLMs.
type GroupedActionableSuggestion struct {
	// ID is the unique suggestion identifier from Google Docs (shared across all atomic parts)
	ID string `json:"id"`

	// Anchor contains exact text before/after for locating where to apply the change
	// Uses larger context (120 chars) to account for multi-part changes
	Anchor SuggestionAnchor `json:"anchor"`

	// Change describes the complete, merged modification to make
	Change SuggestionChange `json:"change"`

	// Verification provides before/after text for validating the complete change
	Verification SuggestionVerification `json:"verification"`

	// Position spans the entire range of all atomic changes
	Position struct {
		StartIndex int64 `json:"start_index"`
		EndIndex   int64 `json:"end_index"`
	} `json:"position"`

	// AtomicChanges preserves the individual operations for debugging/reference
	AtomicChanges []SuggestionChange `json:"atomic_changes,omitempty"`

	// AtomicCount indicates how many operations were merged (1 for non-grouped suggestions)
	AtomicCount int `json:"atomic_count"`
}

// LocationGroupedSuggestions represents suggestions grouped first by location, then by suggestion ID.
// This structure makes it easier to process suggestions in a logical order - handling all
// suggestions in one location before moving to the next.
type LocationGroupedSuggestions struct {
	// Location provides contextual metadata for this group
	Location SuggestionLocation `json:"location"`

	// Suggestions contains all grouped suggestions for this location
	Suggestions []GroupedActionableSuggestion `json:"suggestions"`
}

// DocumentStructure holds the parsed structure of the document for context lookups
type DocumentStructure struct {
	Headings     []DocumentHeading         `json:"headings"`
	Tables       []TableRange              `json:"tables"`
	FullText     string                    `json:"full_text"`     // Complete document text
	TextElements []TextElementWithPosition `json:"text_elements"` // All text with positions
}

// TableRange represents a table's position in the document
type TableRange struct {
	ID            string     `json:"id"`              // Unique ID for the table
	Title         string     `json:"title,omitempty"` // Text immediately above the table
	StartIndex    int64      `json:"start_index"`
	EndIndex      int64      `json:"end_index"`
	RowRanges     []RowRange `json:"row_ranges"`
	ColumnHeaders []string   `json:"column_headers"` // Headers from first row if available
}

// RowRange represents a row's position within a table
type RowRange struct {
	StartIndex int64       `json:"start_index"`
	EndIndex   int64       `json:"end_index"`
	CellRanges []CellRange `json:"cell_ranges"`
}

// CellRange represents a cell's position within a row
type CellRange struct {
	StartIndex int64  `json:"start_index"`
	EndIndex   int64  `json:"end_index"`
	Text       string `json:"text"`       // Full text of cell
	FirstLine  string `json:"first_line"` // First line only (for display)
}

// TextElementWithPosition stores text content with its document position
type TextElementWithPosition struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	StartIndex int64  `json:"start_index"`
	EndIndex   int64  `json:"end_index"`
}

// Comment represents a comment on the document (from Drive API)
type Comment struct {
	ID              string   `json:"id"`
	Author          string   `json:"author"`
	AuthorEmail     string   `json:"author_email"`
	Content         string   `json:"content"`
	QuotedContent   string   `json:"quoted_content"` // The text this comment refers to
	CreatedTime     string   `json:"created_time"`
	ModifiedTime    string   `json:"modified_time"`
	Resolved        bool     `json:"resolved"`
	Replies         []Reply  `json:"replies,omitempty"`
	MentionedEmails []string `json:"mentioned_emails,omitempty"`
}

// Reply represents a reply to a comment
type Reply struct {
	ID          string `json:"id"`
	Author      string `json:"author"`
	AuthorEmail string `json:"author_email"`
	Content     string `json:"content"`
	CreatedTime string `json:"created_time"`
}

// MetadataTable represents the metadata table at the beginning of a document.
// This table contains key-value pairs with document metadata such as title,
// description, and other custom fields defined by the document template.
//
// Structure expected:
//
//	| Metadata                          |                                    |
//	| Page title (60 characters max)    | Ubuntu on AWS                      |
//	| Page description (160 chars max)  | Ubuntu is the operating system...  |
//	| ...                               | ...                                |
type MetadataTable struct {
	// Raw contains all key-value pairs from the metadata table
	Raw map[string]string `json:"raw"`

	// Commonly used fields extracted for convenience
	PageTitle       string `json:"page_title,omitempty"`
	PageDescription string `json:"page_description,omitempty"`
	SuggestedUrl    string `json:"suggested_url,omitempty"`

	// TableStartIndex is the character position where the metadata table starts
	TableStartIndex int64 `json:"table_start_index"`
	// TableEndIndex is the character position where the metadata table ends
	TableEndIndex int64 `json:"table_end_index"`
}
