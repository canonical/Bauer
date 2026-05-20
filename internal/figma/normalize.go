package figma

import "time"

// Normalize converts raw Figma API responses into a NormalizedDesign.
//
// requestedNodeID is the node ID from the user's LinkRef (empty for whole-file).
// screenshotPaths maps nodeID → local file path after download.
//
// All comments are included — including resolved ones. Resolved status is indicated
// by the Resolved field. Filtering resolved comments from prompt context happens
// in the mapping layer, not here.
func Normalize(
	fileKey string,
	requestedNodeID string,
	meta *FileMeta,
	nodes *NodesResponse,
	comments *CommentsResponse,
	screenshotPaths map[string]string,
) *NormalizedDesign {
	design := &NormalizedDesign{
		FileKey:      fileKey,
		RootNodeID:   requestedNodeID,
		Version:      meta.Version,
		LastModified: meta.LastModified,
	}

	// Normalize nodes into anchors.
	// IMPORTANT: if a specific node was requested, look it up by ID.
	// Do NOT range over the map for the primary lookup — Go map iteration is
	// randomized, which would cause nondeterministic results when multiple nodes
	// are returned.
	if requestedNodeID != "" {
		if entry, ok := nodes.Nodes[requestedNodeID]; ok {
			design.Anchors = extractAnchors(requestedNodeID, &entry.Document, nil)
		}
	} else {
		// Whole-file fetch: collect anchors from every node returned.
		// Use a sorted iteration order if stability matters for tests.
		for nodeID, entry := range nodes.Nodes {
			design.Anchors = append(design.Anchors, extractAnchors(nodeID, &entry.Document, nil)...)
		}
	}

	// Normalize comments — include all (resolved and unresolved).
	if comments != nil {
		for _, c := range comments.Comments {
			design.Comments = append(design.Comments, DesignComment{
				ID:        c.ID,
				NodeID:    c.ClientMeta.NodeID,
				Message:   c.Message,
				Author:    c.User.Handle,
				CreatedAt: c.CreatedAt,
				Resolved:  c.ResolvedAt != nil,
			})
		}
	}

	// Map screenshot paths to ScreenshotArtifact records.
	now := time.Now().UTC().Format(time.RFC3339)
	for nodeID, localPath := range screenshotPaths {
		design.Screenshots = append(design.Screenshots, ScreenshotArtifact{
			NodeID:    nodeID,
			LocalPath: localPath,
			Scale:     2,
			FetchedAt: now,
		})
	}

	return design
}

// extractAnchors recursively extracts DesignAnchor values from a document node subtree.
// path is the breadcrumb from the root to the current node (used for NodePath).
func extractAnchors(nodeID string, doc *DocumentNode, path []string) []DesignAnchor {
	currentPath := append(append([]string{}, path...), doc.Name)
	anchor := DesignAnchor{
		NodeID:   nodeID,
		NodeName: doc.Name,
		NodePath: currentPath,
	}

	for _, child := range doc.Children {
		switch child.Type {
		case "TEXT":
			if child.Characters != "" {
				anchor.NearestText = append(anchor.NearestText, child.Characters)
			}
		case "INSTANCE":
			if child.ComponentID != "" {
				anchor.ComponentIDs = append(anchor.ComponentIDs, child.ComponentID)
			}
		}
	}

	return []DesignAnchor{anchor}
}
