package source

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"

	"bauer/internal/figma"
	"bauer/internal/gdocs"
)

// Manager holds all registered source adapters and orchestrates fetching.
type Manager struct {
	credentialsPath string
}

// NewManager creates a Manager. credentialsPath is for Google Docs auth.
func NewManager(credentialsPath string) *Manager {
	return &Manager{credentialsPath: credentialsPath}
}

// Fetch runs the Google Docs adapter (and later, the Figma adapter when it exists).
// Returns a SourceBundle combining all source outputs.
func (m *Manager) Fetch(ctx context.Context, req Request) (*SourceBundle, error) {
	bundle := &SourceBundle{}

	if req.DocID != "" {
		gdocsClient, err := gdocs.NewClient(ctx, m.credentialsPath)
		if err != nil {
			return nil, fmt.Errorf("gdocs client init: %w", err)
		}
		result, err := gdocsClient.ProcessDocument(ctx, req.DocID)
		if err != nil {
			return nil, fmt.Errorf("gdocs fetch: %w", err)
		}
		bundle.Document = result
	}

	return bundle, nil
}

// FetchFigma fetches and normalizes Figma design data.
// It downloads screenshots to screenshotDir.
func (m *Manager) FetchFigma(ctx context.Context, client *figma.Client, ref *figma.LinkRef, screenshotDir string) (*figma.NormalizedDesign, error) {
	meta, err := client.GetMeta(ctx, ref.FileKey)
	if err != nil {
		return nil, fmt.Errorf("fetching figma metadata: %w", err)
	}

	nodeIDs := []string{}
	if ref.NodeID != "" {
		nodeIDs = []string{ref.NodeID}
	}

	var nodes *figma.NodesResponse
	if len(nodeIDs) == 0 {
		slog.Warn("whole-file Figma link — no specific node requested, skipping node fetch")
		nodes = &figma.NodesResponse{}
	} else {
		nodes, err = client.GetNodes(ctx, ref.FileKey, nodeIDs)
		if err != nil {
			return nil, fmt.Errorf("fetching figma nodes: %w", err)
		}
	}

	comments, err := client.GetComments(ctx, ref.FileKey)
	if err != nil {
		return nil, fmt.Errorf("fetching figma comments: %w", err)
	}

	// Request screenshots for the specified node(s)
	screenshotPaths := map[string]string{}
	if len(nodeIDs) > 0 {
		imageURLs, err := client.GetImages(ctx, ref.FileKey, nodeIDs)
		if err != nil {
			// Non-fatal: log and continue without screenshots
			slog.Warn("could not fetch figma screenshots", slog.Any("error", err))
		} else {
			for nodeID, imgURL := range imageURLs {
				if imgURL == "" {
					continue
				}
				safeNodeID := sanitizeNodeID(nodeID)
				destPath := filepath.Join(screenshotDir, fmt.Sprintf("shot-node-%s.png", safeNodeID))
				if err := client.DownloadImage(ctx, imgURL, destPath); err != nil {
					slog.Warn("could not download screenshot", slog.String("node_id", nodeID), slog.Any("error", err))
					continue
				}
				screenshotPaths[nodeID] = destPath
			}
		}
	}

	return figma.Normalize(ref.FileKey, ref.NodeID, meta, nodes, comments, screenshotPaths), nil
}

// sanitizeNodeID removes unsafe characters from a node ID for use in filenames.
var nodeIDSafePattern = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

func sanitizeNodeID(nodeID string) string {
	return nodeIDSafePattern.ReplaceAllString(nodeID, "_")
}
