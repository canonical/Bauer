package mapping

import (
	"testing"

	"bauer/internal/figma"
	"bauer/internal/gdocs"
)

func makeGroup(heading, section string, suggestions ...gdocs.GroupedActionableSuggestion) gdocs.LocationGroupedSuggestions {
	return gdocs.LocationGroupedSuggestions{
		Location: gdocs.SuggestionLocation{
			Section:       section,
			ParentHeading: heading,
		},
		Suggestions: suggestions,
	}
}

func makeSuggestion(originalText string) gdocs.GroupedActionableSuggestion {
	return gdocs.GroupedActionableSuggestion{
		ID: "test-id",
		Change: gdocs.SuggestionChange{
			Type:         "replace",
			OriginalText: originalText,
			NewText:      "new text",
		},
		AtomicCount: 1,
	}
}

func TestBuild_NilDesign_AllNone(t *testing.T) {
	r := &Resolver{}
	groups := []gdocs.LocationGroupedSuggestions{
		makeGroup("Heading One", "Body"),
		makeGroup("Heading Two", "Body"),
	}

	chunks := r.Build(groups, nil)

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	for i, chunk := range chunks {
		if chunk.Mapping.Method != "none" {
			t.Errorf("chunk[%d]: expected method 'none', got %q", i, chunk.Mapping.Method)
		}
		if chunk.Mapping.Confidence != 0 {
			t.Errorf("chunk[%d]: expected confidence 0, got %f", i, chunk.Mapping.Confidence)
		}
		if len(chunk.DesignAnchors) != 0 {
			t.Errorf("chunk[%d]: expected no anchors, got %d", i, len(chunk.DesignAnchors))
		}
		if len(chunk.ScreenshotPaths) != 0 {
			t.Errorf("chunk[%d]: expected no screenshots, got %d", i, len(chunk.ScreenshotPaths))
		}
		if len(chunk.Comments) != 0 {
			t.Errorf("chunk[%d]: expected no comments, got %d", i, len(chunk.Comments))
		}
	}
}

func TestBuild_UserSuppliedNodeID_URLMethod(t *testing.T) {
	r := &Resolver{}
	groups := []gdocs.LocationGroupedSuggestions{
		makeGroup("Any Heading", "Body"),
	}
	design := &figma.NormalizedDesign{
		FileKey:    "file123",
		RootNodeID: "1:23",
		Anchors: []figma.DesignAnchor{
			{NodeID: "1:23", NodeName: "Login Frame"},
			{NodeID: "4:56", NodeName: "Other Frame"},
		},
	}

	chunks := r.Build(groups, design)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	chunk := chunks[0]
	if chunk.Mapping.Method != "url" {
		t.Errorf("expected method 'url', got %q", chunk.Mapping.Method)
	}
	if chunk.Mapping.Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", chunk.Mapping.Confidence)
	}
	if chunk.Mapping.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", chunk.Mapping.Status)
	}
	if len(chunk.DesignAnchors) != 1 {
		t.Fatalf("expected 1 anchor, got %d", len(chunk.DesignAnchors))
	}
	if chunk.DesignAnchors[0].NodeID != "1:23" {
		t.Errorf("expected node ID '1:23', got %q", chunk.DesignAnchors[0].NodeID)
	}
}

func TestBuild_TextMatching_TextMethod(t *testing.T) {
	r := &Resolver{}
	// Use a heading with distinct tokens that appear in the figma anchor's NearestText
	groups := []gdocs.LocationGroupedSuggestions{
		makeGroup("Dashboard Analytics Overview", "Body",
			makeSuggestion("revenue chart monthly"),
		),
	}
	design := &figma.NormalizedDesign{
		FileKey: "file123",
		Anchors: []figma.DesignAnchor{
			{
				NodeID:      "10:1",
				NodeName:    "Unrelated Frame",
				NearestText: []string{"dashboard", "analytics", "overview", "revenue", "chart", "monthly"},
			},
		},
	}

	chunks := r.Build(groups, design)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	chunk := chunks[0]
	if chunk.Mapping.Method != "text" {
		t.Errorf("expected method 'text', got %q", chunk.Mapping.Method)
	}
	if chunk.Mapping.Confidence <= 0.50 {
		t.Errorf("expected confidence > 0.50, got %f", chunk.Mapping.Confidence)
	}
	if chunk.Mapping.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", chunk.Mapping.Status)
	}
}

func TestBuild_FrameNameMatching_NameMethod(t *testing.T) {
	r := &Resolver{}
	// Use heading tokens that overlap ≥50% with frame name tokens
	groups := []gdocs.LocationGroupedSuggestions{
		makeGroup("Settings Panel", "Body"),
	}
	design := &figma.NormalizedDesign{
		FileKey: "file123",
		Anchors: []figma.DesignAnchor{
			// NearestText is empty so text matching won't fire
			{NodeID: "20:5", NodeName: "Settings Panel"},
		},
	}

	chunks := r.Build(groups, design)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	chunk := chunks[0]
	if chunk.Mapping.Method != "name" {
		t.Errorf("expected method 'name', got %q", chunk.Mapping.Method)
	}
	if chunk.Mapping.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", chunk.Mapping.Status)
	}
}

func TestBuild_Fallback_UnresolvedStatus(t *testing.T) {
	r := &Resolver{}
	// heading has no tokens matching figma frame name or nearest text
	groups := []gdocs.LocationGroupedSuggestions{
		makeGroup("Completely Different Topic", "Body"),
	}
	design := &figma.NormalizedDesign{
		FileKey: "file123",
		Anchors: []figma.DesignAnchor{
			{NodeID: "30:1", NodeName: "XYZ Frame"},
		},
	}

	chunks := r.Build(groups, design)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	chunk := chunks[0]
	if chunk.Mapping.Method != "fallback" {
		t.Errorf("expected method 'fallback', got %q", chunk.Mapping.Method)
	}
	if chunk.Mapping.Confidence != 0.50 {
		t.Errorf("expected confidence 0.50, got %f", chunk.Mapping.Confidence)
	}
	if chunk.Mapping.Status != "unresolved" {
		t.Errorf("expected status 'unresolved', got %q", chunk.Mapping.Status)
	}
}

func TestBuild_NoAnchors_NoneUnresolved(t *testing.T) {
	r := &Resolver{}
	groups := []gdocs.LocationGroupedSuggestions{
		makeGroup("Some Heading", "Body"),
	}
	design := &figma.NormalizedDesign{
		FileKey: "file123",
		Anchors: []figma.DesignAnchor{}, // no anchors
	}

	chunks := r.Build(groups, design)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	chunk := chunks[0]
	if chunk.Mapping.Method != "none" {
		t.Errorf("expected method 'none', got %q", chunk.Mapping.Method)
	}
	if chunk.Mapping.Status != "unresolved" {
		t.Errorf("expected status 'unresolved', got %q", chunk.Mapping.Status)
	}
}

func TestBuild_ResolvedCommentsExcluded(t *testing.T) {
	r := &Resolver{}
	groups := []gdocs.LocationGroupedSuggestions{
		makeGroup("Any Heading", "Body"),
	}
	design := &figma.NormalizedDesign{
		FileKey:    "file123",
		RootNodeID: "1:1",
		Anchors: []figma.DesignAnchor{
			{NodeID: "1:1", NodeName: "Frame"},
		},
		Comments: []figma.DesignComment{
			{ID: "c1", NodeID: "1:1", Message: "resolved comment", Author: "alice", Resolved: true},
			{ID: "c2", NodeID: "1:1", Message: "open comment", Author: "bob", Resolved: false},
		},
	}

	chunks := r.Build(groups, design)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	comments := chunks[0].Comments
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment (unresolved only), got %d", len(comments))
	}
	if comments[0].CommentID != "c2" {
		t.Errorf("expected comment ID 'c2', got %q", comments[0].CommentID)
	}
	if comments[0].Author != "bob" {
		t.Errorf("expected author 'bob', got %q", comments[0].Author)
	}
}

func TestBuild_UnresolvedCommentsIncluded(t *testing.T) {
	r := &Resolver{}
	groups := []gdocs.LocationGroupedSuggestions{
		makeGroup("Any Heading", "Body"),
	}
	design := &figma.NormalizedDesign{
		FileKey:    "file123",
		RootNodeID: "1:1",
		Anchors: []figma.DesignAnchor{
			{NodeID: "1:1", NodeName: "Frame"},
		},
		Comments: []figma.DesignComment{
			{ID: "c3", NodeID: "1:1", Message: "first open", Author: "carol", Resolved: false},
			{ID: "c4", NodeID: "1:1", Message: "second open", Author: "dave", Resolved: false},
		},
	}

	chunks := r.Build(groups, design)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	comments := chunks[0].Comments
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
}

func TestBuild_ScreenshotsAttachedToCorrectAnchor(t *testing.T) {
	r := &Resolver{}
	groups := []gdocs.LocationGroupedSuggestions{
		makeGroup("Any Heading", "Body"),
	}
	design := &figma.NormalizedDesign{
		FileKey:    "file123",
		RootNodeID: "1:1",
		Anchors: []figma.DesignAnchor{
			{NodeID: "1:1", NodeName: "Frame A"},
			{NodeID: "2:2", NodeName: "Frame B"},
		},
		Screenshots: []figma.ScreenshotArtifact{
			{NodeID: "1:1", LocalPath: "/tmp/frame-a.png", Scale: 2},
			{NodeID: "2:2", LocalPath: "/tmp/frame-b.png", Scale: 2},
		},
	}

	chunks := r.Build(groups, design)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	// URL match → only node 1:1 is the anchor
	paths := chunks[0].ScreenshotPaths
	if len(paths) != 1 {
		t.Fatalf("expected 1 screenshot for matched anchor, got %d", len(paths))
	}
	if paths[0] != "/tmp/frame-a.png" {
		t.Errorf("expected '/tmp/frame-a.png', got %q", paths[0])
	}
}

func TestBuild_EmptyGroups(t *testing.T) {
	r := &Resolver{}
	chunks := r.Build(nil, nil)
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for nil input, got %d", len(chunks))
	}
}
