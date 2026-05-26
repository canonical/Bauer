package source

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

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
		fmt.Printf("warning: whole-file Figma link — no specific node requested, skipping node fetch\n")
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
			fmt.Printf("warning: could not fetch figma screenshots: %v\n", err)
		} else {
			for nodeID, imgURL := range imageURLs {
				if imgURL == "" {
					continue
				}
				destPath := filepath.Join(screenshotDir, fmt.Sprintf("shot-node-%s.png", strings.ReplaceAll(nodeID, ":", "-")))
				if err := client.DownloadImage(ctx, imgURL, destPath); err != nil {
					fmt.Printf("warning: could not download screenshot for node %s: %v\n", nodeID, err)
					continue
				}
				screenshotPaths[nodeID] = destPath
			}
		}
	}

	return figma.Normalize(ref.FileKey, ref.NodeID, meta, nodes, comments, screenshotPaths), nil
}
