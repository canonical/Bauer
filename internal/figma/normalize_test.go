package figma_test

import (
	"testing"

	"bauer/internal/figma"
)

func ptr(s string) *string { return &s }

func makeMeta() *figma.FileMeta {
	return &figma.FileMeta{
		Name:         "Test File",
		LastModified: "2024-01-15T10:00:00Z",
		Version:      "99",
	}
}

func TestNormalize_EmptyChildren(t *testing.T) {
	nodes := &figma.NodesResponse{
		Nodes: map[string]figma.NodeEntry{
			"1:1": {
				Document: figma.DocumentNode{
					ID:   "1:1",
					Name: "Frame",
					Type: "FRAME",
				},
			},
		},
	}
	design := figma.Normalize("fileKey", "1:1", makeMeta(), nodes, nil, nil)

	if len(design.Anchors) != 1 {
		t.Fatalf("expected 1 anchor, got %d", len(design.Anchors))
	}
	if design.Anchors[0].NearestText != nil {
		t.Errorf("expected nil NearestText, got %v", design.Anchors[0].NearestText)
	}
	if design.Anchors[0].ComponentIDs != nil {
		t.Errorf("expected nil ComponentIDs, got %v", design.Anchors[0].ComponentIDs)
	}
}

func TestNormalize_NoCommentsNoScreenshots(t *testing.T) {
	nodes := &figma.NodesResponse{Nodes: map[string]figma.NodeEntry{}}
	design := figma.Normalize("fileKey", "", makeMeta(), nodes, nil, nil)

	if len(design.Comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(design.Comments))
	}
	if len(design.Screenshots) != 0 {
		t.Errorf("expected 0 screenshots, got %d", len(design.Screenshots))
	}
}

func TestNormalize_ResolvedComments(t *testing.T) {
	resolvedAt := "2024-01-16T12:00:00Z"
	comments := &figma.CommentsResponse{
		Comments: []figma.Comment{
			{
				ID:         "c1",
				Message:    "unresolved",
				User:       figma.CommentUser{Handle: "alice"},
				CreatedAt:  "2024-01-15T10:00:00Z",
				ResolvedAt: nil,
			},
			{
				ID:         "c2",
				Message:    "resolved",
				User:       figma.CommentUser{Handle: "bob"},
				CreatedAt:  "2024-01-15T11:00:00Z",
				ResolvedAt: &resolvedAt,
			},
		},
	}
	nodes := &figma.NodesResponse{Nodes: map[string]figma.NodeEntry{}}
	design := figma.Normalize("fileKey", "", makeMeta(), nodes, comments, nil)

	if len(design.Comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(design.Comments))
	}

	// Find by ID
	byID := map[string]figma.DesignComment{}
	for _, c := range design.Comments {
		byID[c.ID] = c
	}

	if byID["c1"].Resolved {
		t.Errorf("c1 should not be resolved")
	}
	if !byID["c2"].Resolved {
		t.Errorf("c2 should be resolved")
	}
}

func TestNormalize_WholeFileFetch(t *testing.T) {
	nodes := &figma.NodesResponse{
		Nodes: map[string]figma.NodeEntry{
			"1:1": {Document: figma.DocumentNode{ID: "1:1", Name: "Frame A", Type: "FRAME"}},
			"2:2": {Document: figma.DocumentNode{ID: "2:2", Name: "Frame B", Type: "FRAME"}},
		},
	}
	// Whole-file fetch: requestedNodeID == ""
	design := figma.Normalize("fileKey", "", makeMeta(), nodes, nil, nil)

	if len(design.Anchors) != 2 {
		t.Errorf("expected 2 anchors for whole-file fetch, got %d", len(design.Anchors))
	}
	if design.RootNodeID != "" {
		t.Errorf("RootNodeID should be empty for whole-file fetch, got %q", design.RootNodeID)
	}
}

func TestNormalize_NodeSpecificFetch(t *testing.T) {
	nodes := &figma.NodesResponse{
		Nodes: map[string]figma.NodeEntry{
			"1:42": {Document: figma.DocumentNode{ID: "1:42", Name: "My Frame", Type: "FRAME"}},
			"2:99": {Document: figma.DocumentNode{ID: "2:99", Name: "Other Frame", Type: "FRAME"}},
		},
	}
	design := figma.Normalize("fileKey", "1:42", makeMeta(), nodes, nil, nil)

	if len(design.Anchors) != 1 {
		t.Fatalf("expected 1 anchor for node-specific fetch, got %d", len(design.Anchors))
	}
	if design.Anchors[0].NodeID != "1:42" {
		t.Errorf("NodeID = %q, want %q", design.Anchors[0].NodeID, "1:42")
	}
	if design.RootNodeID != "1:42" {
		t.Errorf("RootNodeID = %q, want %q", design.RootNodeID, "1:42")
	}
}

func TestNormalize_TextExtraction(t *testing.T) {
	nodes := &figma.NodesResponse{
		Nodes: map[string]figma.NodeEntry{
			"1:1": {
				Document: figma.DocumentNode{
					ID:   "1:1",
					Name: "Frame",
					Type: "FRAME",
					Children: []figma.DocumentNode{
						{ID: "1:2", Name: "Title", Type: "TEXT", Characters: "Hello World"},
						{ID: "1:3", Name: "Subtitle", Type: "TEXT", Characters: "Sub text"},
						{ID: "1:4", Name: "Empty", Type: "TEXT", Characters: ""},
					},
				},
			},
		},
	}
	design := figma.Normalize("fileKey", "1:1", makeMeta(), nodes, nil, nil)

	if len(design.Anchors) != 1 {
		t.Fatalf("expected 1 anchor, got %d", len(design.Anchors))
	}
	anchor := design.Anchors[0]
	if len(anchor.NearestText) != 2 {
		t.Errorf("expected 2 NearestText entries (empty excluded), got %d: %v", len(anchor.NearestText), anchor.NearestText)
	}
	if anchor.NearestText[0] != "Hello World" {
		t.Errorf("NearestText[0] = %q, want %q", anchor.NearestText[0], "Hello World")
	}
}

func TestNormalize_ComponentIDExtraction(t *testing.T) {
	nodes := &figma.NodesResponse{
		Nodes: map[string]figma.NodeEntry{
			"1:1": {
				Document: figma.DocumentNode{
					ID:   "1:1",
					Name: "Frame",
					Type: "FRAME",
					Children: []figma.DocumentNode{
						{ID: "1:2", Name: "Button", Type: "INSTANCE", ComponentID: "comp-abc"},
						{ID: "1:3", Name: "Icon", Type: "INSTANCE", ComponentID: "comp-xyz"},
						{ID: "1:4", Name: "NoComp", Type: "INSTANCE", ComponentID: ""},
					},
				},
			},
		},
	}
	design := figma.Normalize("fileKey", "1:1", makeMeta(), nodes, nil, nil)

	anchor := design.Anchors[0]
	if len(anchor.ComponentIDs) != 2 {
		t.Errorf("expected 2 ComponentIDs (empty excluded), got %d: %v", len(anchor.ComponentIDs), anchor.ComponentIDs)
	}
}

func TestNormalize_Screenshots(t *testing.T) {
	nodes := &figma.NodesResponse{Nodes: map[string]figma.NodeEntry{}}
	screenshotPaths := map[string]string{
		"1:42": "/tmp/shot-node-1-42.png",
	}
	design := figma.Normalize("fileKey", "", makeMeta(), nodes, nil, screenshotPaths)

	if len(design.Screenshots) != 1 {
		t.Fatalf("expected 1 screenshot, got %d", len(design.Screenshots))
	}
	shot := design.Screenshots[0]
	if shot.NodeID != "1:42" {
		t.Errorf("NodeID = %q, want %q", shot.NodeID, "1:42")
	}
	if shot.LocalPath != "/tmp/shot-node-1-42.png" {
		t.Errorf("LocalPath = %q, want %q", shot.LocalPath, "/tmp/shot-node-1-42.png")
	}
	if shot.Scale != 2 {
		t.Errorf("Scale = %d, want 2", shot.Scale)
	}
}

func TestNormalize_MetaFields(t *testing.T) {
	nodes := &figma.NodesResponse{Nodes: map[string]figma.NodeEntry{}}
	design := figma.Normalize("myFileKey", "3:7", makeMeta(), nodes, nil, nil)

	if design.FileKey != "myFileKey" {
		t.Errorf("FileKey = %q, want %q", design.FileKey, "myFileKey")
	}
	if design.Version != "99" {
		t.Errorf("Version = %q, want %q", design.Version, "99")
	}
	if design.LastModified != "2024-01-15T10:00:00Z" {
		t.Errorf("LastModified = %q, want %q", design.LastModified, "2024-01-15T10:00:00Z")
	}
}
