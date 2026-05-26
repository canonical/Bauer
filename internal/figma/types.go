package figma

// FileMeta is the raw response from GET /v1/files/:key/meta
type FileMeta struct {
	Name         string `json:"name"`
	LastModified string `json:"lastModified"`
	Version      string `json:"version"`
}

// DocumentNode represents a Figma document node (frame, layer, text, etc.)
type DocumentNode struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Characters  string         `json:"characters,omitempty"`  // TEXT nodes only
	ComponentID string         `json:"componentId,omitempty"` // INSTANCE nodes only
	Children    []DocumentNode `json:"children,omitempty"`
}

// NodeEntry is a single node entry in NodesResponse.Nodes map
type NodeEntry struct {
	Document DocumentNode `json:"document"`
}

// NodesResponse is the raw response from GET /v1/files/:key/nodes
type NodesResponse struct {
	Name         string               `json:"name"`
	LastModified string               `json:"lastModified"`
	Nodes        map[string]NodeEntry `json:"nodes"`
}

// CommentUser is the author of a comment
type CommentUser struct {
	Handle string `json:"handle"`
	Name   string `json:"name"`
}

// CommentClientMeta holds positional metadata for a comment
type CommentClientMeta struct {
	NodeID string `json:"node_id,omitempty"`
}

// Comment is a single Figma comment
type Comment struct {
	ID         string            `json:"id"`
	Message    string            `json:"message"`
	ClientMeta CommentClientMeta `json:"client_meta"`
	CreatedAt  string            `json:"created_at"`
	User       CommentUser       `json:"user"`
	ParentID   string            `json:"parent_id,omitempty"`
	ResolvedAt *string           `json:"resolved_at,omitempty"` // nil if not resolved
}

// CommentsResponse is the raw response from GET /v1/files/:key/comments
type CommentsResponse struct {
	Comments []Comment `json:"comments"`
}

// imagesResponse is the raw response from GET /v1/images/:key
type imagesResponse struct {
	Err    interface{}       `json:"err"`
	Images map[string]string `json:"images"`
}

// NormalizedDesign is the Bauer-owned representation of a fetched Figma design.
// This is the canonical type all downstream code uses — never the raw API types.
type NormalizedDesign struct {
	FileKey      string               `json:"file_key"`
	RootNodeID   string               `json:"root_node_id"` // from the LinkRef.NodeID
	Version      string               `json:"version"`
	LastModified string               `json:"last_modified"`
	Anchors      []DesignAnchor       `json:"anchors"`
	Comments     []DesignComment      `json:"comments"`
	Screenshots  []ScreenshotArtifact `json:"screenshots"`
}

// DesignAnchor represents a Figma node used as a design reference point.
type DesignAnchor struct {
	NodeID       string   `json:"node_id"`
	NodeName     string   `json:"node_name"`
	NodePath     []string `json:"node_path,omitempty"`
	NearestText  []string `json:"nearest_text,omitempty"`
	ComponentIDs []string `json:"component_ids,omitempty"`
}

// DesignComment is a normalized Figma comment.
type DesignComment struct {
	ID        string `json:"id"`
	NodeID    string `json:"node_id,omitempty"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
	Resolved  bool   `json:"resolved"`
}

// ScreenshotArtifact records a downloaded screenshot.
type ScreenshotArtifact struct {
	NodeID    string `json:"node_id"`
	LocalPath string `json:"local_path"`
	Scale     int    `json:"scale"`
	FetchedAt string `json:"fetched_at"`
}
