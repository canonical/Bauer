package mapping

import "bauer/internal/gdocs"

// ResolvedChunk is a group of suggestion locations enriched with figma design context.
// It is the unit that the prompt engine receives and renders into a PromptData.
// When no figma URL was supplied, DesignAnchors, ScreenshotPaths, and Comments are empty.
type ResolvedChunk struct {
	Locations       []gdocs.LocationGroupedSuggestions `json:"locations"`
	DesignAnchors   []DesignAnchorRef                  `json:"design_anchors,omitempty"`
	ScreenshotPaths []string                            `json:"screenshot_paths,omitempty"`
	Comments        []DesignCommentRef                  `json:"comments,omitempty"`
	Mapping         MappingMetadata                     `json:"mapping"`
}

// DesignAnchorRef is a lightweight reference to a matched Figma node.
type DesignAnchorRef struct {
	FileKey  string `json:"file_key"`
	NodeID   string `json:"node_id"`
	NodeName string `json:"node_name"`
}

// DesignCommentRef is a lightweight reference to a matched Figma comment.
type DesignCommentRef struct {
	CommentID string `json:"comment_id"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	NodeID    string `json:"node_id"`
}

// MappingMetadata describes how a suggestion group was matched to a figma anchor.
type MappingMetadata struct {
	Method     string  `json:"method"`     // "url", "cache", "text", "name", "fallback", or "none"
	Confidence float64 `json:"confidence"` // 0.0 to 1.0; 0 for the "none" case
	Status     string  `json:"status"`     // "healthy", "stale", "unresolved", or "none"
}
