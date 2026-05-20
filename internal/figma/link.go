package figma

import (
	"fmt"
	"net/url"
	"regexp"
)

var figmaFilePattern = regexp.MustCompile(`figma\.com/(?:file|design)/([A-Za-z0-9_-]+)`)

// LinkRef holds the parsed result of a Figma URL.
type LinkRef struct {
	FileKey string // opaque file key from the URL path
	NodeID  string // URL-decoded node ID, e.g. "1:42". Empty for whole-file links.
	RawURL  string
}

// ParseLink extracts the file key and optional node ID from a Figma link.
// Accepts /file/ and /design/ URL patterns.
// Returns a clear error for non-Figma URLs.
func ParseLink(rawURL string) (*LinkRef, error) {
	matches := figmaFilePattern.FindStringSubmatch(rawURL)
	if len(matches) < 2 {
		return nil, fmt.Errorf("not a valid Figma link: %q (expected figma.com/file/... or figma.com/design/...)", rawURL)
	}
	ref := &LinkRef{FileKey: matches[1], RawURL: rawURL}

	u, err := url.Parse(rawURL)
	if err == nil {
		// url.Query().Get() URL-decodes automatically: "1%3A42" → "1:42"
		ref.NodeID = u.Query().Get("node-id")
	}
	return ref, nil
}
