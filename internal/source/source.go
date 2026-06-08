package source

import "context"

// Adapter is the interface any upstream data source must implement.
type Adapter interface {
	// Name returns the adapter's identifier (e.g. "gdocs", "figma").
	Name() string
	// Fetch retrieves data from the source. Returns the raw source-specific payload.
	Fetch(ctx context.Context, req Request) (any, error)
}

// Request contains the parameters for fetching from sources.
type Request struct {
	DocID    string // Google Doc ID
	FigmaURL string // Optional Figma file/node URL
}
