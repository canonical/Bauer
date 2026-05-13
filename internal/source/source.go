// Package source defines the source abstraction layer for Bauer. Source adapters
// fetch data from upstream systems (Google Docs, Figma, etc.) and the Manager
// combines them into a normalized SourceBundle consumed by the orchestrator.
package source

import "context"

// Adapter is the interface for any upstream source (Google Docs, Figma, etc.).
// Defined here because the Manager (consumer) lives in the same package.
type Adapter interface {
	// Name returns a human-readable identifier for this source (e.g. "gdocs", "figma").
	Name() string

	// Fetch retrieves data from the upstream source. The returned value is
	// source-specific; callers type-assert based on the adapter they provided.
	Fetch(ctx context.Context, req Request) (any, error)
}

// Request carries the parameters needed to fetch from sources.
type Request struct {
	DocID           string // Google Doc ID (required for gdocs)
	CredentialsPath string // Path to Google service account JSON (required for gdocs)
	FigmaURL        string // Figma URL (optional; empty when no Figma context)
}
