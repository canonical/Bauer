package figma

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const baseURL = "https://api.figma.com/v1"

// Client is the Figma REST API client.
// It never logs the token.
type Client struct {
	token string
	http  *http.Client
}

// NewClient creates a new Figma client with the given personal access token.
func NewClient(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
	}
}

// NewClientWithHTTP creates a Client with a custom HTTP client. Use for testing only.
func NewClientWithHTTP(token string, httpClient *http.Client) *Client {
	return &Client{token: token, http: httpClient}
}

// GetMeta fetches file name, last-modified date, and version.
// Docs: https://developers.figma.com/docs/rest-api/file-endpoints/
func (c *Client) GetMeta(ctx context.Context, fileKey string) (*FileMeta, error) {
	return doGet[FileMeta](ctx, c, fmt.Sprintf("%s/files/%s/meta", baseURL, fileKey))
}

// GetNodes fetches specific nodes (frames, layers) and their children.
// If nodeIDs is empty, returns nothing useful — callers should always provide at least one ID.
// Docs: https://developers.figma.com/docs/rest-api/file-endpoints/
func (c *Client) GetNodes(ctx context.Context, fileKey string, nodeIDs []string) (*NodesResponse, error) {
	if len(nodeIDs) == 0 {
		return &NodesResponse{}, nil
	}
	ids := url.QueryEscape(strings.Join(nodeIDs, ","))
	return doGet[NodesResponse](ctx, c,
		fmt.Sprintf("%s/files/%s/nodes?ids=%s", baseURL, fileKey, ids))
}

// GetComments fetches all comments from the file, with text in markdown format.
// Docs: https://developers.figma.com/docs/rest-api/comments-endpoints/
func (c *Client) GetComments(ctx context.Context, fileKey string) (*CommentsResponse, error) {
	return doGet[CommentsResponse](ctx, c,
		fmt.Sprintf("%s/files/%s/comments?as_md=true", baseURL, fileKey))
}

// GetImages requests rendered screenshot URLs for the given node IDs at 2x scale.
// Returns a map of nodeID → pre-signed URL. URLs expire quickly; download immediately.
// Docs: https://developers.figma.com/docs/rest-api/file-endpoints/#get-images-endpoint
func (c *Client) GetImages(ctx context.Context, fileKey string, nodeIDs []string) (map[string]string, error) {
	if len(nodeIDs) == 0 {
		return map[string]string{}, nil
	}
	ids := url.QueryEscape(strings.Join(nodeIDs, ","))
	resp, err := doGet[imagesResponse](ctx, c,
		fmt.Sprintf("%s/images/%s?ids=%s&format=png&scale=2", baseURL, fileKey, ids))
	if err != nil {
		return nil, err
	}
	if resp.Images == nil {
		return map[string]string{}, nil
	}
	return resp.Images, nil
}

// DownloadImage downloads a pre-signed image URL (no auth needed) to destPath.
// The file is created with 0o644 permissions.
func (c *Client) DownloadImage(ctx context.Context, presignedURL, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, presignedURL, nil)
	if err != nil {
		return fmt.Errorf("creating download request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("downloading image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("image download failed: status %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating image file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("writing image: %w", err)
	}
	return nil
}

// doGet performs a GET request to the Figma API endpoint and decodes the JSON response.
// It returns a clear error for non-200 responses. The token is never logged.
func doGet[T any](ctx context.Context, c *Client, endpoint string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("figma API request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// success path; continue
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, fmt.Errorf("figma API authentication failed (status %d): check BAUER_FIGMA_TOKEN", resp.StatusCode)
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("figma API rate limit exceeded (status 429): retry after a delay")
	case http.StatusNotFound:
		return nil, fmt.Errorf("figma resource not found (status 404): check the file key and node ID")
	default:
		return nil, fmt.Errorf("figma API error: status %d for %s", resp.StatusCode, endpoint)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding figma API response: %w", err)
	}
	return &result, nil
}
